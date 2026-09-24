package integration_test

import (
	"errors"
	"strings"

	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// These integration tests exercise options.WithLimits' MaxObjectDecodedBytes
// cap end-to-end against the shared Gitea testcontainer. This is the
// decompression-bomb defense: unlike the wire caps, it bounds an object's
// *decoded* (post-inflation) size, so it catches a small, highly compressible
// payload that would slip past any byte-count cap on the compressed response.
//
// Every blob here is built from a repeated byte (strings.Repeat), the exact
// opposite of incompressibleBytes: it inflates to megabytes but travels as a
// few hundred bytes on the wire, so only a decoded-size cap can stop it. We
// leave the wire caps at their zero value so the decoded cap is provably the
// thing that trips.
var _ = Describe("Decoded-object size cap (decompression-bomb protection)", func() {
	Context("MaxObjectDecodedBytes cap", func() {
		It("returns ObjectTooLargeError from GetBlob when a compressible blob inflates past the cap", func() {
			const cap = 4096
			const decodedSize = 1 << 20 // 1 MiB decoded, ~KB on the wire

			By("Setting up a client with a tight decoded-object cap and no wire caps")
			cappedClient, _, local, _ := QuickSetup(options.WithLimits(options.Limits{
				MaxObjectDecodedBytes: cap,
			}))

			By("Pushing a blob that inflates to 1 MiB but compresses to a few hundred bytes")
			payload := strings.Repeat("a", decodedSize)
			Expect(local.CreateFile("bomb.txt", payload)).To(Succeed())
			_, err := local.Git("add", "bomb.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add compressible bomb blob")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			blobHashOutput, err := local.Git("rev-parse", "HEAD:bomb.txt")
			Expect(err).NotTo(HaveOccurred())
			blobHash, err := hash.FromHex(strings.TrimSpace(blobHashOutput))
			Expect(err).NotTo(HaveOccurred())

			By("GetBlob must reject on the declared decoded size, before inflating it")
			_, err = cappedClient.GetBlob(ctx, blobHash)
			Expect(err).To(HaveOccurred())
			var tooLarge *protocol.ObjectTooLargeError
			Expect(errors.As(err, &tooLarge)).To(BeTrue(),
				"expected *protocol.ObjectTooLargeError, got %T: %v", err, err)
			Expect(tooLarge.Size).To(Equal(int64(decodedSize)))
			Expect(tooLarge.Limit).To(Equal(int64(cap)))
			By("and the sentinel must survive the wrap for errors.Is callers")
			Expect(errors.Is(err, protocol.ErrObjectTooLarge)).To(BeTrue())
		})

		It("does not interfere with an object that inflates within the cap", func() {
			const decodedSize = 1 << 20 // 1 MiB

			By("Setting up a client with a decoded cap generously above the blob")
			client, _, local, _ := QuickSetup(options.WithLimits(options.Limits{
				MaxObjectDecodedBytes: 8 * 1024 * 1024, // 8 MiB, well above the blob
			}))

			payload := strings.Repeat("b", decodedSize)
			Expect(local.CreateFile("ok.txt", payload)).To(Succeed())
			_, err := local.Git("add", "ok.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add within-cap blob")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			blobHashOutput, err := local.Git("rev-parse", "HEAD:ok.txt")
			Expect(err).NotTo(HaveOccurred())
			blobHash, err := hash.FromHex(strings.TrimSpace(blobHashOutput))
			Expect(err).NotTo(HaveOccurred())

			blob, err := client.GetBlob(ctx, blobHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(blob.Content).To(HaveLen(decodedSize))
		})
	})

	Context("Default options (no WithLimits)", func() {
		It("keeps decoded-object protection on at the built-in default", func() {
			// The security invariant: the zero value of Limits disables the wire
			// caps but NOT the decoded cap, which falls back to the built-in
			// MaxUnpackedObjectSize (10 MiB). A blob inflating past that must be
			// rejected even though the embedder configured no limits at all.
			const decodedSize = protocol.MaxUnpackedObjectSize + (1 << 20) // 11 MiB

			By("Running with no WithLimits configured")
			client, _, local, _ := QuickSetup() // no WithLimits

			By("Pushing a blob that inflates past the built-in 10 MiB default")
			payload := strings.Repeat("c", decodedSize)
			Expect(local.CreateFile("huge.txt", payload)).To(Succeed())
			_, err := local.Git("add", "huge.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add over-default blob")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			blobHashOutput, err := local.Git("rev-parse", "HEAD:huge.txt")
			Expect(err).NotTo(HaveOccurred())
			blobHash, err := hash.FromHex(strings.TrimSpace(blobHashOutput))
			Expect(err).NotTo(HaveOccurred())

			By("GetBlob must still reject, carrying the built-in default as the limit")
			_, err = client.GetBlob(ctx, blobHash)
			Expect(err).To(HaveOccurred())
			var tooLarge *protocol.ObjectTooLargeError
			Expect(errors.As(err, &tooLarge)).To(BeTrue(),
				"expected *protocol.ObjectTooLargeError, got %T: %v", err, err)
			Expect(tooLarge.Size).To(Equal(int64(decodedSize)))
			Expect(tooLarge.Limit).To(Equal(int64(protocol.MaxUnpackedObjectSize)))
			Expect(errors.Is(err, protocol.ErrObjectTooLarge)).To(BeTrue())
		})
	})
})
