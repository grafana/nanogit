package integration_test

import (
	"errors"
	"strings"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol/hash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// These tests exercise push flows with WithoutPushSideBand enabled against a
// real Git server. They verify that disabling the "side-band-64k" capability
// on receive-pack does not regress functional behavior for CreateRef,
// UpdateRef, DeleteRef, or StagedWriter.Push.
//
// The option exists as a workaround for servers (notably some GitLab
// configurations) that wrap report-status in side-band channel 1, which
// nanogit's parser did not unwrap before matching "ng"/"unpack" prefixes —
// causing rejections from push rules/pre-receive hooks to be silently
// swallowed. Disabling side-band forces the server to send report-status
// unwrapped, making failure detection reliable.
var _ = Describe("Push without side-band capability", func() {
	author := nanogit.Author{Name: "Test Author", Email: "test@example.com", Time: time.Now()}
	committer := nanogit.Committer{Name: "Test Committer", Email: "test@example.com", Time: time.Now()}

	Context("CreateRef / UpdateRef / DeleteRef", func() {
		var (
			client      nanogit.Client
			local       *gittest.LocalRepo
			firstCommit hash.Hash
		)

		BeforeEach(func() {
			By("Setting up repo with WithoutPushSideBand")
			client, _, local, _ = QuickSetup(options.WithoutPushSideBand())

			By("Publishing initial main branch")
			_, err := local.Git("branch", "-M", "main")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "-u", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			firstCommitStr, err := local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			firstCommit, err = hash.FromHex(firstCommitStr)
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates, updates, and deletes a ref end-to-end", func() {
			refName := "refs/heads/no-sideband-ref-lifecycle"

			By("Creating the branch ref")
			Expect(client.CreateRef(ctx, nanogit.Ref{Name: refName, Hash: firstCommit})).To(Succeed())

			created, err := client.GetRef(ctx, refName)
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Hash).To(Equal(firstCommit))

			By("Making a new commit on main and pushing it")
			_, err = local.Git("commit", "--allow-empty", "-m", "no-sideband: second commit")
			Expect(err).NotTo(HaveOccurred())
			newHashStr, err := local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			newHash, err := hash.FromHex(newHashStr)
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Updating the ref to the new commit")
			Expect(client.UpdateRef(ctx, nanogit.Ref{Name: refName, Hash: newHash})).To(Succeed())

			updated, err := client.GetRef(ctx, refName)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Hash).To(Equal(newHash))

			By("Deleting the ref")
			Expect(client.DeleteRef(ctx, refName)).To(Succeed())

			_, err = client.GetRef(ctx, refName)
			var notFoundErr *nanogit.RefNotFoundError
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &notFoundErr)).To(BeTrue())
		})
	})

	Context("StagedWriter push", func() {
		It("pushes a new commit to an existing branch", func() {
			client, _, local, _ := QuickSetup(options.WithoutPushSideBand())

			_, err := local.Git("branch", "-M", "main")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "-u", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			mainRef, err := client.GetRef(ctx, "refs/heads/main")
			Expect(err).NotTo(HaveOccurred())

			writer, err := client.NewStagedWriter(ctx, mainRef)
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.CreateBlob(ctx, "no-sideband.txt", []byte("hello from nanogit"))
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "no-sideband: add file", author, committer)
			Expect(err).NotTo(HaveOccurred())

			Expect(writer.Push(ctx)).To(Succeed())

			By("Confirming the commit reached the remote and contains the file")
			_, err = local.Git("fetch", "origin", "main")
			Expect(err).NotTo(HaveOccurred())
			remoteTree, err := local.Git("ls-tree", "-r", "--name-only", "origin/main")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Split(strings.TrimSpace(remoteTree), "\n")).To(ContainElement("no-sideband.txt"))
		})

		// This is the exact flow that produced empty branches for GitLab users
		// in grafana/grafana: ensureBranchExists creates a new branch via
		// CreateRef pointing at the source branch HEAD, and then the staged
		// writer pushes a new commit onto that branch. When the Push was
		// silently swallowed (side-band-wrapped "ng" packet), the branch was
		// left pointing at the source HEAD with no new commit — exactly the
		// "empty branch" symptom.
		It("creates a new branch then pushes a commit onto it", func() {
			client, _, local, _ := QuickSetup(options.WithoutPushSideBand())

			_, err := local.Git("branch", "-M", "main")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "-u", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			mainRef, err := client.GetRef(ctx, "refs/heads/main")
			Expect(err).NotTo(HaveOccurred())

			branchName := "refs/heads/no-sideband-new-branch"
			By("Creating a new branch ref pointing at main's HEAD")
			Expect(client.CreateRef(ctx, nanogit.Ref{Name: branchName, Hash: mainRef.Hash})).To(Succeed())

			By("Staging a blob, committing, and pushing to the new branch")
			writer, err := client.NewStagedWriter(ctx, nanogit.Ref{Name: branchName, Hash: mainRef.Hash})
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.CreateBlob(ctx, "from-new-branch.txt", []byte("committed on a fresh branch"))
			Expect(err).NotTo(HaveOccurred())

			commit, err := writer.Commit(ctx, "no-sideband: add file on new branch", author, committer)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit).NotTo(BeNil())

			Expect(writer.Push(ctx)).To(Succeed())

			By("Verifying the new branch advanced past main (no empty branch)")
			remoteRef, err := client.GetRef(ctx, branchName)
			Expect(err).NotTo(HaveOccurred())
			Expect(remoteRef.Hash).NotTo(Equal(mainRef.Hash),
				"branch must advance past the source ref; if it equals main, the push was silently swallowed")
			Expect(remoteRef.Hash).To(Equal(commit.Hash))

			By("Confirming the staged file is present on the new branch in the remote")
			_, err = local.Git("fetch", "origin", strings.TrimPrefix(branchName, "refs/heads/"))
			Expect(err).NotTo(HaveOccurred())
			remoteTree, err := local.Git("ls-tree", "-r", "--name-only",
				"origin/"+strings.TrimPrefix(branchName, "refs/heads/"))
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Split(strings.TrimSpace(remoteTree), "\n")).To(ContainElement("from-new-branch.txt"))
		})
	})

	Context("Default behavior (side-band enabled) is unchanged", func() {
		// Sanity check that the baseline flow still works without the option
		// and that the same new-branch flow also succeeds. This is a
		// regression guard for clients that do not opt into the workaround.
		It("creates a branch and pushes onto it with side-band enabled", func() {
			client, _, local, _ := QuickSetup() // no WithoutPushSideBand

			_, err := local.Git("branch", "-M", "main")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("push", "-u", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			mainRef, err := client.GetRef(ctx, "refs/heads/main")
			Expect(err).NotTo(HaveOccurred())

			branchName := "refs/heads/default-sideband-new-branch"
			Expect(client.CreateRef(ctx, nanogit.Ref{Name: branchName, Hash: mainRef.Hash})).To(Succeed())

			writer, err := client.NewStagedWriter(ctx, nanogit.Ref{Name: branchName, Hash: mainRef.Hash})
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.CreateBlob(ctx, "default.txt", []byte("default side-band flow"))
			Expect(err).NotTo(HaveOccurred())
			commit, err := writer.Commit(ctx, "default: add file on new branch", author, committer)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			remoteRef, err := client.GetRef(ctx, branchName)
			Expect(err).NotTo(HaveOccurred())
			Expect(remoteRef.Hash).To(Equal(commit.Hash))
		})
	})
})
