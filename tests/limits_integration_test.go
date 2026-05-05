package integration_test

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol/client"
	"github.com/grafana/nanogit/protocol/hash"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// incompressibleBytes returns n bytes of cryptographically random hex —
// content that survives the zlib pass inside the packfile, so the wire
// size stays close to len(payload). Tests that need to cap the response
// below the blob's payload size MUST use this; a string of repeated
// characters will compress to tens of bytes regardless of length.
func incompressibleBytes(n int) string {
	buf := make([]byte, n/2)
	_, err := rand.Read(buf)
	Expect(err).NotTo(HaveOccurred())
	return hex.EncodeToString(buf)
}

// These integration tests exercise options.WithLimits end-to-end against
// the shared Gitea testcontainer. Writes are performed via the git CLI
// (local.Git ...) so the cap only governs the nanogit read path under test.
// We assert that:
//   - a configured cap surfaces *client.ErrResponseTooLarge through the
//     public Client API, not just at the internal reader layer
//   - a generously-configured cap does not interfere with normal reads
//   - the zero-value Limits keeps fetching unbounded, so existing
//     embedders are not silently capped by this change
var _ = Describe("Byte limits (DoS protection)", func() {
	Context("SingleObjectFetch cap", func() {
		It("returns ErrResponseTooLarge from GetBlob when the blob exceeds the cap", func() {
			By("Setting up a client with a tight single-object cap")
			cappedClient, _, local, _ := QuickSetup(options.WithLimits(options.Limits{
				SingleObjectFetch: 256,
			}))

			By("Pushing a blob whose post-zlib size is comfortably larger than the cap")
			// Use incompressible data: zlib in the packfile would
			// otherwise shrink a string of repeated characters to
			// tens of bytes regardless of input length, and the cap
			// would never trip. 32 KiB of random hex stays close to
			// 32 KiB on the wire.
			payload := incompressibleBytes(32 * 1024)
			Expect(local.CreateFile("big.txt", payload)).To(Succeed())
			_, err := local.Git("add", "big.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add big blob")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			blobHashOutput, err := local.Git("rev-parse", "HEAD:big.txt")
			Expect(err).NotTo(HaveOccurred())
			blobHash, err := hash.FromHex(strings.TrimSpace(blobHashOutput))
			Expect(err).NotTo(HaveOccurred())

			By("GetBlob must surface *client.ErrResponseTooLarge")
			_, err = cappedClient.GetBlob(ctx, blobHash)
			Expect(err).To(HaveOccurred())
			var tooLarge *client.ErrResponseTooLarge
			Expect(errors.As(err, &tooLarge)).To(BeTrue(),
				"expected *client.ErrResponseTooLarge, got %T: %v", err, err)
			Expect(tooLarge.Op).To(Equal("fetch"))
			Expect(tooLarge.Limit).To(Equal(int64(256)))
		})

		It("does not interfere with normal-sized blobs", func() {
			By("Setting up a client with a generous single-object cap")
			client, _, local, _ := QuickSetup(options.WithLimits(options.Limits{
				SingleObjectFetch: 10 * 1024 * 1024, // 10 MiB
			}))

			payload := []byte("hello world")
			Expect(local.CreateFile("ok.txt", string(payload))).To(Succeed())
			_, err := local.Git("add", "ok.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add small blob")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			blobHashOutput, err := local.Git("rev-parse", "HEAD:ok.txt")
			Expect(err).NotTo(HaveOccurred())
			blobHash, err := hash.FromHex(strings.TrimSpace(blobHashOutput))
			Expect(err).NotTo(HaveOccurred())

			blob, err := client.GetBlob(ctx, blobHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(blob.Content).To(Equal(payload))
		})
	})

	Context("MultiObjectFetch cap", func() {
		It("returns ErrResponseTooLarge from GetFlatTree when the tree exceeds the cap", func() {
			// GetFlatTree's initial fetch is shallow + blob:none, so
			// the response (commit + root tree) is on the order of a
			// few hundred bytes after zlib. We need a cap below that
			// shape to reliably trip; 128 bytes is below even a
			// minimal commit + tree pair.
			By("Setting up a client with a very tight multi-object cap")
			cappedClient, _, local, _ := QuickSetup(options.WithLimits(options.Limits{
				MultiObjectFetch: 128,
			}))

			By("Creating a tree with several files so the packfile is non-trivial")
			Expect(local.CreateDirPath("dir")).To(Succeed())
			for i := range 5 {
				name := fmt.Sprintf("dir/f%d.txt", i)
				// Incompressible per-file payload so the packfile
				// wire size scales with the file size rather than
				// collapsing to near-zero under zlib.
				Expect(local.CreateFile(name, incompressibleBytes(4*1024))).To(Succeed())
			}
			_, err := local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Multi-file commit")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			commitHashOutput, err := local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			commitHash, err := hash.FromHex(strings.TrimSpace(commitHashOutput))
			Expect(err).NotTo(HaveOccurred())

			_, err = cappedClient.GetFlatTree(ctx, commitHash)
			Expect(err).To(HaveOccurred())
			var tooLarge *client.ErrResponseTooLarge
			Expect(errors.As(err, &tooLarge)).To(BeTrue(),
				"expected *client.ErrResponseTooLarge, got %T: %v", err, err)
			Expect(tooLarge.Op).To(Equal("fetch"))
			Expect(tooLarge.Limit).To(Equal(int64(128)))
		})
	})

	Context("RefsMetadata cap", func() {
		It("returns ErrResponseTooLarge from ListRefs when the ref list exceeds the cap", func() {
			By("Setting up a client with a tight refs cap")
			cappedClient, _, local, _ := QuickSetup(options.WithLimits(options.Limits{
				RefsMetadata: 64,
			}))

			By("Creating extra refs so the ls-refs response is non-trivial")
			Expect(local.CreateFile("a.txt", "a")).To(Succeed())
			_, err := local.Git("add", "a.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "seed")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())
			for i := range 3 {
				branch := fmt.Sprintf("limit-test-%d", i)
				_, err = local.Git("branch", branch)
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("push", "origin", branch)
				Expect(err).NotTo(HaveOccurred())
			}

			_, err = cappedClient.ListRefs(ctx)
			Expect(err).To(HaveOccurred())
			var tooLarge *client.ErrResponseTooLarge
			Expect(errors.As(err, &tooLarge)).To(BeTrue(),
				"expected *client.ErrResponseTooLarge, got %T: %v", err, err)
			Expect(tooLarge.Op).To(Equal("ls-refs"))
			Expect(tooLarge.Limit).To(Equal(int64(64)))
		})
	})

	Context("Default options", func() {
		It("preserves unbounded behavior so existing embedders see no change", func() {
			By("Running with no WithLimits configured")
			client, _, local, _ := QuickSetup() // no WithLimits

			By("Pushing a blob larger than any reasonable accidental cap")
			payload := incompressibleBytes(256 * 1024)
			Expect(local.CreateFile("default.txt", payload)).To(Succeed())
			_, err := local.Git("add", "default.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Default-options blob")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			blobHashOutput, err := local.Git("rev-parse", "HEAD:default.txt")
			Expect(err).NotTo(HaveOccurred())
			blobHash, err := hash.FromHex(strings.TrimSpace(blobHashOutput))
			Expect(err).NotTo(HaveOccurred())

			blob, err := client.GetBlob(ctx, blobHash)
			Expect(err).NotTo(HaveOccurred(),
				"default options must not cap any operation")
			Expect(blob.Content).To(HaveLen(len(payload)))
		})
	})
})
