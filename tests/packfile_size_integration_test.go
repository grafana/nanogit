package integration_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Packfile object size limits", func() {
	var (
		client nanogit.Client
		local  *gittest.LocalRepo
	)

	BeforeEach(func() {
		client, _, local, _ = QuickSetup()
	})

	It("should read an object that inflates to exactly the maximum size without corrupting other objects in the pack", func() {
		By("Creating a blob whose inflated size is exactly the maximum")
		maxContent := string(bytes.Repeat([]byte("a"), protocol.MaxUnpackedObjectSize))
		Expect(local.CreateFile("max.bin", maxContent)).To(Succeed())

		By("Creating additional files packed alongside it")
		Expect(local.CreateFile("before.txt", "before content")).To(Succeed())
		Expect(local.CreateFile("after.txt", "after content")).To(Succeed())
		Expect(local.CreateFile("nested/deep.txt", "nested content")).To(Succeed())

		By("Committing and pushing")
		_, err := local.Git("add", ".")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("commit", "-m", "Add max-size object with neighbours")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("push", "origin", "main", "--force")
		Expect(err).NotTo(HaveOccurred())

		output, err := local.Git("rev-parse", "HEAD")
		Expect(err).NotTo(HaveOccurred())
		commitHash, err := hash.FromHex(output)
		Expect(err).NotTo(HaveOccurred())

		By("Cloning so every blob arrives in a single pack")
		tempDir := GinkgoT().TempDir()
		_, err = client.Clone(ctx, nanogit.CloneOptions{Path: tempDir, Hash: commitHash})
		Expect(err).NotTo(HaveOccurred())

		By("Verifying the max-size object is intact")
		got, err := os.ReadFile(filepath.Join(tempDir, "max.bin"))
		Expect(err).NotTo(HaveOccurred())
		Expect(len(got)).To(Equal(protocol.MaxUnpackedObjectSize))
		Expect(bytes.Equal(got, []byte(maxContent))).To(BeTrue())

		By("Verifying neighbouring objects are not misaligned")
		for path, want := range map[string]string{
			"before.txt":      "before content",
			"after.txt":       "after content",
			"nested/deep.txt": "nested content",
		} {
			content, err := os.ReadFile(filepath.Join(tempDir, path))
			Expect(err).NotTo(HaveOccurred(), "path: %q", path)
			Expect(string(content)).To(Equal(want), "path: %q", path)
		}
	})

	It("should read a max-size blob directly via GetBlob", func() {
		By("Creating and pushing a blob of exactly the maximum size")
		maxContent := string(bytes.Repeat([]byte("b"), protocol.MaxUnpackedObjectSize))
		Expect(local.CreateFile("max.bin", maxContent)).To(Succeed())
		_, err := local.Git("add", "max.bin")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("commit", "-m", "Add max-size blob")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("push", "origin", "main", "--force")
		Expect(err).NotTo(HaveOccurred())

		output, err := local.Git("rev-parse", "HEAD:max.bin")
		Expect(err).NotTo(HaveOccurred())
		blobHash, err := hash.FromHex(output)
		Expect(err).NotTo(HaveOccurred())

		By("Fetching it through nanogit")
		blob, err := client.GetBlob(ctx, blobHash)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(blob.Content)).To(Equal(protocol.MaxUnpackedObjectSize))
		Expect(bytes.Equal(blob.Content, []byte(maxContent))).To(BeTrue())
		Expect(blob.Hash).To(Equal(blobHash))
	})

	It("should reject a blob larger than the maximum unpacked object size", func() {
		By("Creating and pushing a blob exceeding the maximum size")
		tooLarge := string(bytes.Repeat([]byte("c"), protocol.MaxUnpackedObjectSize+1))
		Expect(local.CreateFile("toobig.bin", tooLarge)).To(Succeed())
		_, err := local.Git("add", "toobig.bin")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("commit", "-m", "Add oversized blob")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("push", "origin", "main", "--force")
		Expect(err).NotTo(HaveOccurred())

		output, err := local.Git("rev-parse", "HEAD:toobig.bin")
		Expect(err).NotTo(HaveOccurred())
		blobHash, err := hash.FromHex(output)
		Expect(err).NotTo(HaveOccurred())

		By("Fetching it through nanogit")
		_, err = client.GetBlob(ctx, blobHash)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, protocol.ErrObjectTooLarge)).To(BeTrue())
	})
})
