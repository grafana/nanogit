package integration_test

import (
	"context"
	"errors"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("References", func() {
	Context("ListRefs operations", func() {
		var (
			client      nanogit.Client
			local       *helpers.LocalGitRepo
			firstCommit hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Getting initial commit hash")
			firstCommitStr := local.Git("rev-parse", "HEAD")
			var err error
			firstCommit, err = hash.FromHex(firstCommitStr)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up branches and tags")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")
			local.Git("branch", "test-branch")
			local.Git("push", "origin", "test-branch", "--force")
			local.Git("tag", "v1.0.0")
			local.Git("push", "origin", "v1.0.0", "--force")
		})

		It("should list all references", func() {
			refs, err := client.ListRefs(context.Background())
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
			local       *helpers.LocalGitRepo
			firstCommit hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Getting initial commit hash")
			firstCommitStr := local.Git("rev-parse", "HEAD")
			var err error
			firstCommit, err = hash.FromHex(firstCommitStr)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up branches and tags")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")
			local.Git("branch", "test-branch")
			local.Git("push", "origin", "test-branch", "--force")
			local.Git("tag", "v1.0.0")
			local.Git("push", "origin", "v1.0.0", "--force")
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
				ref, err := client.GetRef(context.Background(), wantRef.Name)
				Expect(err).NotTo(HaveOccurred(), "GetRef failed for %s", wantRef.Name)
				Expect(ref.Name).To(Equal(wantRef.Name))
				Expect(ref.Hash).To(Equal(firstCommit))
			}
		})

		It("should return error for non-existent ref", func() {
			_, err := client.GetRef(context.Background(), "refs/heads/non-existent")

			var notFoundErr *nanogit.RefNotFoundError
			Expect(errors.As(err, &notFoundErr)).To(BeTrue())
			Expect(notFoundErr.RefName).To(Equal("refs/heads/non-existent"))
		})
	})

	Context("CreateRef operations", func() {
		var (
			client      nanogit.Client
			local       *helpers.LocalGitRepo
			firstCommit hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Getting initial commit hash")
			firstCommitStr := local.Git("rev-parse", "HEAD")
			var err error
			firstCommit, err = hash.FromHex(firstCommitStr)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up main branch")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")
		})

		It("should create branch ref", func() {
			By("Creating new branch ref")
			err := client.CreateRef(context.Background(), nanogit.Ref{Name: "refs/heads/new-branch", Hash: firstCommit})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the created ref")
			ref, err := client.GetRef(context.Background(), "refs/heads/new-branch")
			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Hash).To(Equal(firstCommit))
		})

		It("should create tag ref", func() {
			By("Creating new tag ref")
			err := client.CreateRef(context.Background(), nanogit.Ref{Name: "refs/tags/v2.0.0", Hash: firstCommit})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the created tag")
			ref, err := client.GetRef(context.Background(), "refs/tags/v2.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Hash).To(Equal(firstCommit))
		})
	})

	Context("UpdateRef operations", func() {
		var (
			client      nanogit.Client
			local       *helpers.LocalGitRepo
			firstCommit hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Getting initial commit hash")
			firstCommitStr := local.Git("rev-parse", "HEAD")
			var err error
			firstCommit, err = hash.FromHex(firstCommitStr)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up main branch")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")
		})

		It("should update existing ref", func() {
			By("Creating ref for update test")
			err := client.CreateRef(context.Background(), nanogit.Ref{Name: "refs/heads/update-test", Hash: firstCommit})
			Expect(err).NotTo(HaveOccurred())

			By("Creating a new commit")
			local.Git("commit", "--allow-empty", "-m", "new commit")
			newHashStr := local.Git("rev-parse", "HEAD")
			newHash, err := hash.FromHex(newHashStr)
			Expect(err).NotTo(HaveOccurred())
			local.Git("push", "origin", "main", "--force")

			By("Updating ref to point to new commit")
			err = client.UpdateRef(context.Background(), nanogit.Ref{Name: "refs/heads/update-test", Hash: newHash})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the update")
			ref, err := client.GetRef(context.Background(), "refs/heads/update-test")
			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Hash).To(Equal(newHash))
		})
	})

	Context("DeleteRef operations", func() {
		var (
			client      nanogit.Client
			local       *helpers.LocalGitRepo
			firstCommit hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Getting initial commit hash")
			firstCommitStr := local.Git("rev-parse", "HEAD")
			var err error
			firstCommit, err = hash.FromHex(firstCommitStr)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up main branch")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")
		})

		It("should delete branch ref", func() {
			By("Creating ref for delete test")
			err := client.CreateRef(context.Background(), nanogit.Ref{Name: "refs/heads/delete-test", Hash: firstCommit})
			Expect(err).NotTo(HaveOccurred())

			By("Deleting the ref")
			err = client.DeleteRef(context.Background(), "refs/heads/delete-test")
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ref is deleted")
			_, err = client.GetRef(context.Background(), "refs/heads/delete-test")
			var notFoundErr *nanogit.RefNotFoundError
			Expect(errors.As(err, &notFoundErr)).To(BeTrue())
			Expect(notFoundErr.RefName).To(Equal("refs/heads/delete-test"))
		})

		It("should delete tag ref", func() {
			By("Creating tag for delete test")
			err := client.CreateRef(context.Background(), nanogit.Ref{Name: "refs/tags/delete-test", Hash: firstCommit})
			Expect(err).NotTo(HaveOccurred())

			By("Deleting the tag")
			err = client.DeleteRef(context.Background(), "refs/tags/delete-test")
			Expect(err).NotTo(HaveOccurred())

			By("Verifying tag is deleted")
			_, err = client.GetRef(context.Background(), "refs/tags/delete-test")
			var notFoundErr *nanogit.RefNotFoundError
			Expect(errors.As(err, &notFoundErr)).To(BeTrue())
			Expect(notFoundErr.RefName).To(Equal("refs/tags/delete-test"))
		})
	})

	Context("Integration workflow", func() {
		var (
			client      nanogit.Client
			local       *helpers.LocalGitRepo
			firstCommit hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Getting initial commit hash")
			firstCommitStr := local.Git("rev-parse", "HEAD")
			var err error
			firstCommit, err = hash.FromHex(firstCommitStr)
			Expect(err).NotTo(HaveOccurred())

			By("Setting up main branch")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")
		})

		It("should complete full ref lifecycle", func() {
			refName := "refs/heads/integration-flow"

			By("Creating ref for integration flow")
			err := client.CreateRef(context.Background(), nanogit.Ref{Name: refName, Hash: firstCommit})
			Expect(err).NotTo(HaveOccurred())

			By("Getting created ref")
			ref, err := client.GetRef(context.Background(), refName)
			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Hash).To(Equal(firstCommit))

			By("Creating new commit for update")
			local.Git("commit", "--allow-empty", "-m", "integration flow commit")
			newHashStr := local.Git("rev-parse", "HEAD")
			newHash, err := hash.FromHex(newHashStr)
			Expect(err).NotTo(HaveOccurred())
			local.Git("push", "origin", "main", "--force")

			By("Updating ref to new commit")
			err = client.UpdateRef(context.Background(), nanogit.Ref{Name: refName, Hash: newHash})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ref update")
			ref, err = client.GetRef(context.Background(), refName)
			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Hash).To(Equal(newHash))

			By("Deleting ref")
			err = client.DeleteRef(context.Background(), refName)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ref deletion")
			_, err = client.GetRef(context.Background(), refName)
			var notFoundErr *nanogit.RefNotFoundError
			Expect(errors.As(err, &notFoundErr)).To(BeTrue())
			Expect(notFoundErr.RefName).To(Equal(refName))
		})
	})
})
