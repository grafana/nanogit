package integration_test

import (
	"errors"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("References", func() {
	Context("ListRefs operations", func() {
		var (
			client      nanogit.Client
			local       *LocalGitRepo
			firstCommit hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Getting initial commit hash")
			firstCommitStr := gitNoError(local, "rev-parse", "HEAD")
			var err error
			firstCommit, err = hash.FromHex(firstCommitStr)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up branches and tags")
			gitNoError(local, "branch", "-M", "main")
			gitNoError(local, "push", "-u", "origin", "main", "--force")
			gitNoError(local, "branch", "test-branch")
			gitNoError(local, "push", "origin", "test-branch", "--force")
			gitNoError(local, "tag", "v1.0.0")
			gitNoError(local, "push", "origin", "v1.0.0", "--force")
		})

		It("should list all references", func() {
			refs, err := client.ListRefs(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(refs).To(HaveLen(4), "should have 4 references")

			wantRefs := []nanogit.Ref{
				{Name: "HEAD", Hash: firstCommit},
				{Name: "refs/heads/main", Hash: firstCommit},
				{Name: "refs/heads/test-branch", Hash: firstCommit},
				{Name: "refs/tags/v1.0.0", Hash: firstCommit},
			}

			Expect(refs).To(ConsistOf(wantRefs))
		})
	})

	Context("GetRef operations", func() {
		var (
			client      nanogit.Client
			local       *LocalGitRepo
			firstCommit hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Getting initial commit hash")
			firstCommitStr := gitNoError(local, "rev-parse", "HEAD")
			var err error
			firstCommit, err = hash.FromHex(firstCommitStr)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up branches and tags")
			gitNoError(local, "branch", "-M", "main")
			gitNoError(local, "push", "-u", "origin", "main", "--force")
			gitNoError(local, "branch", "test-branch")
			gitNoError(local, "push", "origin", "test-branch", "--force")
			gitNoError(local, "tag", "v1.0.0")
			gitNoError(local, "push", "origin", "v1.0.0", "--force")
		})

		It("should get existing refs", func() {
			wantRefs := []nanogit.Ref{
				{Name: "HEAD", Hash: firstCommit},
				{Name: "refs/heads/main", Hash: firstCommit},
				{Name: "refs/heads/test-branch", Hash: firstCommit},
				{Name: "refs/tags/v1.0.0", Hash: firstCommit},
			}

			By("Getting refs one by one")
			for _, wantRef := range wantRefs {
				ref, err := client.GetRef(ctx, wantRef.Name)
				Expect(err).NotTo(HaveOccurred(), "GetRef failed for %s", wantRef.Name)
				Expect(ref.Name).To(Equal(wantRef.Name))
				Expect(ref.Hash).To(Equal(firstCommit))
			}
		})

		It("should return error for non-existent ref", func() {
			_, err := client.GetRef(ctx, "refs/heads/non-existent")

			var notFoundErr *nanogit.RefNotFoundError
			Expect(errors.As(err, &notFoundErr)).To(BeTrue())
			Expect(notFoundErr.RefName).To(Equal("refs/heads/non-existent"))
		})

		It("should handle ambiguous ref prefix by exact matching", func() {
			By("Creating refs with similar prefixes")
			gitNoError(local, "branch", "test")
			gitNoError(local, "push", "origin", "test", "--force")
			gitNoError(local, "branch", "test-longer")
			gitNoError(local, "push", "origin", "test-longer", "--force")

			By("Getting specific ref should work despite prefix ambiguity")
			ref, err := client.GetRef(ctx, "refs/heads/test")
			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Name).To(Equal("refs/heads/test"))
			Expect(ref.Hash).To(Equal(firstCommit))

			By("Getting the longer ref should also work")
			ref, err = client.GetRef(ctx, "refs/heads/test-longer")
			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Name).To(Equal("refs/heads/test-longer"))
			Expect(ref.Hash).To(Equal(firstCommit))
		})
	})

	Context("CreateRef operations", func() {
		var (
			client      nanogit.Client
			local       *LocalGitRepo
			firstCommit hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Getting initial commit hash")
			firstCommitStr := gitNoError(local, "rev-parse", "HEAD")
			var err error
			firstCommit, err = hash.FromHex(firstCommitStr)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up main branch")
			gitNoError(local, "branch", "-M", "main")
			gitNoError(local, "push", "-u", "origin", "main", "--force")
		})

		It("should create branch ref", func() {
			By("Creating new branch ref")
			err := client.CreateRef(ctx, nanogit.Ref{Name: "refs/heads/new-branch", Hash: firstCommit})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the created ref")
			ref, err := client.GetRef(ctx, "refs/heads/new-branch")
			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Hash).To(Equal(firstCommit))
		})

		It("should create tag ref", func() {
			By("Creating new tag ref")
			err := client.CreateRef(ctx, nanogit.Ref{Name: "refs/tags/v2.0.0", Hash: firstCommit})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the created tag")
			ref, err := client.GetRef(ctx, "refs/tags/v2.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Hash).To(Equal(firstCommit))
		})
	})

	Context("UpdateRef operations", func() {
		var (
			client      nanogit.Client
			local       *LocalGitRepo
			firstCommit hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Getting initial commit hash")
			firstCommitStr := gitNoError(local, "rev-parse", "HEAD")
			var err error
			firstCommit, err = hash.FromHex(firstCommitStr)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up main branch")
			gitNoError(local, "branch", "-M", "main")
			gitNoError(local, "push", "-u", "origin", "main", "--force")
		})

		It("should update existing ref", func() {
			By("Creating ref for update test")
			err := client.CreateRef(ctx, nanogit.Ref{Name: "refs/heads/update-test", Hash: firstCommit})
			Expect(err).NotTo(HaveOccurred())

			By("Creating a new commit")
			gitNoError(local, "commit", "--allow-empty", "-m", "new commit")
			newHashStr := gitNoError(local, "rev-parse", "HEAD")
			newHash, err := hash.FromHex(newHashStr)
			Expect(err).NotTo(HaveOccurred())
			gitNoError(local, "push", "origin", "main", "--force")

			By("Updating ref to point to new commit")
			err = client.UpdateRef(ctx, nanogit.Ref{Name: "refs/heads/update-test", Hash: newHash})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the update")
			ref, err := client.GetRef(ctx, "refs/heads/update-test")
			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Hash).To(Equal(newHash))
		})
	})

	Context("DeleteRef operations", func() {
		var (
			client      nanogit.Client
			local       *LocalGitRepo
			firstCommit hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Getting initial commit hash")
			firstCommitStr := gitNoError(local, "rev-parse", "HEAD")
			var err error
			firstCommit, err = hash.FromHex(firstCommitStr)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up main branch")
			gitNoError(local, "branch", "-M", "main")
			gitNoError(local, "push", "-u", "origin", "main", "--force")
		})

		It("should delete branch ref", func() {
			By("Creating ref for delete test")
			err := client.CreateRef(ctx, nanogit.Ref{Name: "refs/heads/delete-test", Hash: firstCommit})
			Expect(err).NotTo(HaveOccurred())

			By("Deleting the ref")
			err = client.DeleteRef(ctx, "refs/heads/delete-test")
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ref is deleted")
			_, err = client.GetRef(ctx, "refs/heads/delete-test")
			var notFoundErr *nanogit.RefNotFoundError
			Expect(errors.As(err, &notFoundErr)).To(BeTrue())
			Expect(notFoundErr.RefName).To(Equal("refs/heads/delete-test"))
		})

		It("should delete tag ref", func() {
			By("Creating tag for delete test")
			err := client.CreateRef(ctx, nanogit.Ref{Name: "refs/tags/delete-test", Hash: firstCommit})
			Expect(err).NotTo(HaveOccurred())

			By("Deleting the tag")
			err = client.DeleteRef(ctx, "refs/tags/delete-test")
			Expect(err).NotTo(HaveOccurred())

			By("Verifying tag is deleted")
			_, err = client.GetRef(ctx, "refs/tags/delete-test")
			var notFoundErr *nanogit.RefNotFoundError
			Expect(errors.As(err, &notFoundErr)).To(BeTrue())
			Expect(notFoundErr.RefName).To(Equal("refs/tags/delete-test"))
		})
	})

	Context("Integration workflow", func() {
		var (
			client      nanogit.Client
			local       *LocalGitRepo
			firstCommit hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Getting initial commit hash")
			firstCommitStr := gitNoError(local, "rev-parse", "HEAD")
			var err error
			firstCommit, err = hash.FromHex(firstCommitStr)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up main branch")
			gitNoError(local, "branch", "-M", "main")
			gitNoError(local, "push", "-u", "origin", "main", "--force")
		})

		It("should complete full ref lifecycle", func() {
			refName := "refs/heads/integration-flow"

			By("Creating ref for integration flow")
			err := client.CreateRef(ctx, nanogit.Ref{Name: refName, Hash: firstCommit})
			Expect(err).NotTo(HaveOccurred())

			By("Getting created ref")
			ref, err := client.GetRef(ctx, refName)
			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Hash).To(Equal(firstCommit))

			By("Creating new commit for update")
			gitNoError(local, "commit", "--allow-empty", "-m", "integration flow commit")
			newHashStr := gitNoError(local, "rev-parse", "HEAD")
			newHash, err := hash.FromHex(newHashStr)
			Expect(err).NotTo(HaveOccurred())
			gitNoError(local, "push", "origin", "main", "--force")

			By("Updating ref to new commit")
			err = client.UpdateRef(ctx, nanogit.Ref{Name: refName, Hash: newHash})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ref update")
			ref, err = client.GetRef(ctx, refName)
			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Hash).To(Equal(newHash))

			By("Deleting ref")
			err = client.DeleteRef(ctx, refName)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ref deletion")
			_, err = client.GetRef(ctx, refName)
			var notFoundErr *nanogit.RefNotFoundError
			Expect(errors.As(err, &notFoundErr)).To(BeTrue())
			Expect(notFoundErr.RefName).To(Equal(refName))
		})
	})
})
