package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/protocol/hash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Integration tests for https://github.com/grafana/grafana/issues/123891
//
// When a Git Sync repository contains a submodule, editing a dashboard
// through Grafana causes nanogit's StagedWriter to push a commit that
// silently deletes the submodule entry from any tree that gets rebuilt.
//
// Root cause: NewStagedWriter loads the working tree via GetFlatTree,
// which (correctly, on the read path) filters out gitlink entries
// (mode 0o160000) — see tree.go around the `entry.FileMode == 0o160000`
// check. The writer's `treeEntries` map therefore never contains
// submodule paths. At commit time, every directory whose contents changed
// is marked dirty (and the root tree is *always* marked dirty for any
// modification — see addMissingOrStaleTreeEntries in writer.go). Dirty
// trees get rebuilt from `treeEntries` via collectDirectChildren, so any
// dirty tree that previously held a submodule sibling drops that
// submodule entry on the floor.
//
// The submodule survives only when its parent directory tree is *not*
// rebuilt — i.e., when nothing under that parent changed. So these tests
// deliberately exercise the two scenarios the bug actually breaks:
//
//   1. Submodule at the repository root: ANY edit always re-marks the
//      root tree dirty, so the submodule is dropped.
//   2. Submodule as a sibling of the modified blob (same parent
//      directory): that directory is dirty, so the submodule sibling
//      gets dropped.
var _ = Describe("Writer Operations with Submodules", func() {
	var (
		testAuthor = nanogit.Author{
			Name:  "Test Author",
			Email: "test@example.com",
			Time:  time.Now(),
		}
		testCommitter = nanogit.Committer{
			Name:  "Test Committer",
			Email: "test@example.com",
			Time:  time.Now(),
		}
	)

	createWriterFromHead := func(ctx context.Context, client nanogit.Client, local *gittest.LocalRepo) nanogit.StagedWriter {
		headOutput, err := local.Git("rev-parse", "HEAD")
		Expect(err).NotTo(HaveOccurred())
		currentHash, err := hash.FromHex(headOutput)
		Expect(err).NotTo(HaveOccurred())

		writer, err := client.NewStagedWriter(ctx, nanogit.Ref{
			Name: "refs/heads/main",
			Hash: currentHash,
		})
		Expect(err).NotTo(HaveOccurred())
		return writer
	}

	// pushSubmoduleSource creates a fresh remote repository with a single
	// commit and returns a usable URL for `git submodule add`.
	pushSubmoduleSource := func(user *gittest.User) string {
		subRepo, err := gitServer.CreateRepo(ctx, gittest.RandomRepoName(), user)
		Expect(err).NotTo(HaveOccurred())

		subLocal, err := gittest.NewLocalRepo(ctx,
			gittest.WithRepoLogger(logger),
			gittest.WithGitTrace(),
		)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			Expect(subLocal.Cleanup()).To(Succeed())
		})
		_, err = subLocal.Git("config", "user.name", user.Username)
		Expect(err).NotTo(HaveOccurred())
		_, err = subLocal.Git("config", "user.email", user.Email)
		Expect(err).NotTo(HaveOccurred())
		_, err = subLocal.Git("remote", "add", "origin", subRepo.AuthURL)
		Expect(err).NotTo(HaveOccurred())
		Expect(subLocal.CreateFile("lib.txt", "library content")).To(Succeed())
		_, err = subLocal.Git("add", ".")
		Expect(err).NotTo(HaveOccurred())
		_, err = subLocal.Git("commit", "-m", "Initial submodule commit")
		Expect(err).NotTo(HaveOccurred())
		_, err = subLocal.Git("branch", "-M", "main")
		Expect(err).NotTo(HaveOccurred())
		_, err = subLocal.Git("push", "origin", "main", "--force")
		Expect(err).NotTo(HaveOccurred())

		return subRepo.AuthURL
	}

	// addSubmodule mounts a submodule at submodulePath in `local`, commits
	// it, pushes, and returns the gitlink hash recorded in the parent tree.
	addSubmodule := func(local *gittest.LocalRepo, submoduleURL, submodulePath string) string {
		// `protocol.file.allow=always` lets `git submodule add` source
		// from a file:// or http URL inside testcontainers.
		_, err := local.Git("-c", "protocol.file.allow=always",
			"submodule", "add", submoduleURL, submodulePath)
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("add", ".")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("commit", "-m", "Add submodule at "+submodulePath)
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("branch", "-M", "main")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("push", "origin", "main", "--force")
		Expect(err).NotTo(HaveOccurred())

		gitlinkLine, err := local.Git("ls-tree", "HEAD", submodulePath)
		Expect(err).NotTo(HaveOccurred())
		gitlinkLine = strings.TrimSpace(gitlinkLine)
		Expect(gitlinkLine).To(HavePrefix("160000 commit "),
			"submodule entry at %q should be a gitlink before nanogit modifications", submodulePath)
		fields := strings.Fields(gitlinkLine)
		Expect(len(fields)).To(BeNumerically(">=", 3))
		return fields[2]
	}

	// expectSubmodulePreserved asserts that, after pulling the latest
	// commit produced by nanogit, the submodule entry is still present
	// at the expected path with the expected gitlink hash. Failure here
	// is the bug from grafana/grafana#123891: nanogit silently dropped
	// the submodule.
	expectSubmodulePreserved := func(local *gittest.LocalRepo, submodulePath, expectedHash string) {
		_, err := local.Git("fetch", "origin", "main")
		Expect(err).NotTo(HaveOccurred())

		// Use ls-tree -r --full-tree on origin/main so we read the tree
		// straight off the just-pushed commit instead of relying on a
		// merge in the local working tree.
		out, err := local.Git("ls-tree", "-r", "--full-tree", "origin/main")
		Expect(err).NotTo(HaveOccurred())

		var found bool
		for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}
			// fields = [mode, type, hash, path]
			if fields[3] != submodulePath {
				continue
			}
			found = true
			Expect(fields[0]).To(Equal("160000"),
				"submodule at %q should keep gitlink mode 160000; nanogit rewrote it (grafana/grafana#123891)",
				submodulePath)
			Expect(fields[1]).To(Equal("commit"),
				"submodule at %q should keep object type 'commit'; nanogit rewrote the entry", submodulePath)
			Expect(fields[2]).To(Equal(expectedHash),
				"submodule gitlink hash at %q should be unchanged across the nanogit-authored commit",
				submodulePath)
		}
		Expect(found).To(BeTrue(),
			"submodule entry at %q must still exist in the new commit; nanogit dropped it (grafana/grafana#123891).\nFull tree:\n%s",
			submodulePath, out)
	}

	// The next three Contexts each set up a fresh repo with a submodule
	// in a position that will force the writer to rebuild the parent
	// tree of the submodule:
	//   - "submodule at repository root" → root tree is always dirty
	//   - "submodule as sibling of edited file" → that directory is dirty

	Context("Submodule at the repository root", func() {
		var (
			client      nanogit.Client
			local       *gittest.LocalRepo
			gitlinkHash string
		)

		BeforeEach(func() {
			var user *gittest.User
			client, _, local, user = QuickSetup()
			subURL := pushSubmoduleSource(user)

			By("Creating a dashboard inside dashboards/")
			Expect(local.CreateDirPath("dashboards")).To(Succeed())
			Expect(local.CreateFile("dashboards/home.json", `{"title":"home","version":1}`)).To(Succeed())
			_, err := local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add dashboards")
			Expect(err).NotTo(HaveOccurred())

			By("Mounting the submodule at the repository root")
			gitlinkHash = addSubmodule(local, subURL, "thirdparty")
		})

		It("should preserve the root submodule when updating a dashboard", func() {
			writer := createWriterFromHead(ctx, client, local)

			_, err := writer.UpdateBlob(ctx, "dashboards/home.json",
				[]byte(`{"title":"home","version":2}`))
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Update dashboard", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			expectSubmodulePreserved(local, "thirdparty", gitlinkHash)
		})

		It("should preserve the root submodule when creating a new dashboard", func() {
			writer := createWriterFromHead(ctx, client, local)

			_, err := writer.CreateBlob(ctx, "dashboards/new.json",
				[]byte(`{"title":"new","version":1}`))
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Add new dashboard", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			expectSubmodulePreserved(local, "thirdparty", gitlinkHash)
		})

		It("should preserve the root submodule when deleting a dashboard", func() {
			writer := createWriterFromHead(ctx, client, local)

			_, err := writer.DeleteBlob(ctx, "dashboards/home.json")
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Remove dashboard", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			expectSubmodulePreserved(local, "thirdparty", gitlinkHash)

			// Pull the new commit into the working tree to confirm the
			// dashboard delete actually landed alongside the preserved
			// submodule.
			_, err = local.Git("pull", "origin", "main")
			Expect(err).NotTo(HaveOccurred())
			_, statErr := os.Stat(filepath.Join(local.Path, "dashboards/home.json"))
			Expect(os.IsNotExist(statErr)).To(BeTrue(),
				"deleted dashboard should not exist on disk after pull")
		})
	})

	Context("Submodule as a sibling of the edited file", func() {
		var (
			client      nanogit.Client
			local       *gittest.LocalRepo
			gitlinkHash string
		)

		BeforeEach(func() {
			var user *gittest.User
			client, _, local, user = QuickSetup()
			subURL := pushSubmoduleSource(user)

			By("Creating dashboards/home.json and committing it")
			Expect(local.CreateDirPath("dashboards")).To(Succeed())
			Expect(local.CreateFile("dashboards/home.json", `{"title":"home","version":1}`)).To(Succeed())
			_, err := local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add dashboard")
			Expect(err).NotTo(HaveOccurred())

			By("Mounting the submodule alongside the dashboard at dashboards/lib")
			gitlinkHash = addSubmodule(local, subURL, "dashboards/lib")
		})

		It("should preserve the sibling submodule when updating the dashboard", func() {
			writer := createWriterFromHead(ctx, client, local)

			_, err := writer.UpdateBlob(ctx, "dashboards/home.json",
				[]byte(`{"title":"home","version":2}`))
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Update dashboard", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			expectSubmodulePreserved(local, "dashboards/lib", gitlinkHash)
		})

		It("should preserve the sibling submodule when creating a new dashboard", func() {
			writer := createWriterFromHead(ctx, client, local)

			_, err := writer.CreateBlob(ctx, "dashboards/new.json",
				[]byte(`{"title":"new","version":1}`))
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Add new dashboard", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			expectSubmodulePreserved(local, "dashboards/lib", gitlinkHash)
		})

		It("should preserve the sibling submodule when deleting the dashboard", func() {
			writer := createWriterFromHead(ctx, client, local)

			_, err := writer.DeleteBlob(ctx, "dashboards/home.json")
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Remove dashboard", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			expectSubmodulePreserved(local, "dashboards/lib", gitlinkHash)
		})
	})

	// Regression tests for the two writer-reuse hazards Copilot flagged
	// during review of the original fix: the writer is reused across
	// pushes, so any in-memory submodule entry that was removed from the
	// tree by a previous push must NOT come back on a later commit.
	// Without the post-Push reconciliation the stale gitlink re-emits
	// from collectDirectChildren and resurrects a submodule the user
	// just removed.

	Context("Submodule replaced then the replacement is deleted", func() {
		It("should not resurrect the submodule across a follow-up push on the same writer", func() {
			client, _, local, user := QuickSetup()
			subURL := pushSubmoduleSource(user)

			By("Mounting the submodule at the repository root")
			_ = addSubmodule(local, subURL, "thirdparty")

			writer := createWriterFromHead(ctx, client, local)

			By("First commit: overwrite the submodule path with a regular blob")
			// CreateBlob succeeds at a submodule path because treeEntries
			// (filtered) doesn't contain it; this is the realistic shape
			// of "user replaces the submodule with a regular file".
			_, err := writer.CreateBlob(ctx, "thirdparty",
				[]byte("plain file replacing the submodule"))
			Expect(err).NotTo(HaveOccurred())
			_, err = writer.Commit(ctx, "Replace submodule with a regular file",
				testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			By("Verifying the first push replaced the submodule with a blob")
			_, err = local.Git("fetch", "origin", "main")
			Expect(err).NotTo(HaveOccurred())
			out, err := local.Git("ls-tree", "--full-tree", "origin/main", "thirdparty")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.TrimSpace(out)).To(HavePrefix("100644 blob "),
				"after the first push, thirdparty should be a regular blob, not a submodule")

			By("Second commit (same writer): delete the replacement blob")
			_, err = writer.DeleteBlob(ctx, "thirdparty")
			Expect(err).NotTo(HaveOccurred())
			_, err = writer.Commit(ctx, "Remove the replacement file",
				testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			By("Verifying the submodule was NOT resurrected by the second push")
			_, err = local.Git("fetch", "origin", "main")
			Expect(err).NotTo(HaveOccurred())
			out, err = local.Git("ls-tree", "--full-tree", "origin/main", "thirdparty")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.TrimSpace(out)).To(BeEmpty(),
				"thirdparty must not exist after the second push; "+
					"a 160000 entry here means a stale submoduleEntries cache "+
					"resurrected the gitlink the previous commit replaced")
		})
	})

	Context("Submodule's parent directory is deleted then recreated", func() {
		It("should not resurrect the submodule when the parent dir is recreated on the same writer", func() {
			client, _, local, user := QuickSetup()
			subURL := pushSubmoduleSource(user)

			By("Creating dashboards/home.json so the dir exists for the submodule mount")
			Expect(local.CreateDirPath("dashboards")).To(Succeed())
			Expect(local.CreateFile("dashboards/home.json", `{"title":"home"}`)).To(Succeed())
			_, err := local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add dashboard")
			Expect(err).NotTo(HaveOccurred())

			By("Mounting the submodule at dashboards/lib")
			_ = addSubmodule(local, subURL, "dashboards/lib")

			writer := createWriterFromHead(ctx, client, local)

			By("First commit: delete the entire dashboards/ tree (submodule and all)")
			_, err = writer.DeleteTree(ctx, "dashboards")
			Expect(err).NotTo(HaveOccurred())
			_, err = writer.Commit(ctx, "Remove dashboards directory",
				testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			By("Verifying the first push wiped dashboards/ entirely")
			_, err = local.Git("fetch", "origin", "main")
			Expect(err).NotTo(HaveOccurred())
			afterDelete, err := local.Git("ls-tree", "-r", "--full-tree", "origin/main")
			Expect(err).NotTo(HaveOccurred())
			Expect(afterDelete).NotTo(ContainSubstring("dashboards/"),
				"dashboards/ should be gone after the first push;\nfull tree:\n%s", afterDelete)

			By("Second commit (same writer): recreate a file under dashboards/")
			_, err = writer.CreateBlob(ctx, "dashboards/new.json",
				[]byte(`{"title":"new"}`))
			Expect(err).NotTo(HaveOccurred())
			_, err = writer.Commit(ctx, "Recreate dashboards with a new file",
				testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			By("Verifying the submodule was NOT resurrected when dashboards/ was rebuilt")
			_, err = local.Git("fetch", "origin", "main")
			Expect(err).NotTo(HaveOccurred())
			afterRecreate, err := local.Git("ls-tree", "-r", "--full-tree", "origin/main")
			Expect(err).NotTo(HaveOccurred())
			Expect(afterRecreate).NotTo(ContainSubstring("dashboards/lib"),
				"dashboards/lib must not be in the new tree; a 160000 entry "+
					"here means an orphaned submoduleEntries cache resurrected "+
					"the gitlink whose parent dir was deleted in the previous "+
					"push.\nfull tree:\n%s", afterRecreate)
			Expect(afterRecreate).To(ContainSubstring("\tdashboards/new.json"),
				"the second commit's blob must have actually landed; "+
					"otherwise the test isn't exercising the rebuild path.\nfull tree:\n%s",
				afterRecreate)
		})
	})

	// In-session DeleteTree(parent) + CreateBlob(parent/x) BEFORE any
	// Push: the resurrection happens at Commit time, so the post-Push
	// reconciliation can't catch it on its own. The fix needs to drop the
	// affected submoduleEntries inside DeleteTree itself.
	Context("Submodule's parent is deleted and recreated within a single commit", func() {
		It("should not resurrect the submodule on a single commit that wipes and rebuilds the parent", func() {
			client, _, local, user := QuickSetup()
			subURL := pushSubmoduleSource(user)

			By("Creating dashboards/home.json so the dir exists for the submodule mount")
			Expect(local.CreateDirPath("dashboards")).To(Succeed())
			Expect(local.CreateFile("dashboards/home.json", `{"title":"home"}`)).To(Succeed())
			_, err := local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add dashboard")
			Expect(err).NotTo(HaveOccurred())

			By("Mounting the submodule at dashboards/lib")
			_ = addSubmodule(local, subURL, "dashboards/lib")

			writer := createWriterFromHead(ctx, client, local)

			By("Staging DeleteTree + CreateBlob inside the same commit")
			_, err = writer.DeleteTree(ctx, "dashboards")
			Expect(err).NotTo(HaveOccurred())
			_, err = writer.CreateBlob(ctx, "dashboards/new.json",
				[]byte(`{"title":"new"}`))
			Expect(err).NotTo(HaveOccurred())
			_, err = writer.Commit(ctx, "Wipe and rebuild dashboards in one commit",
				testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			By("Verifying the single pushed commit does not contain the gitlink")
			_, err = local.Git("fetch", "origin", "main")
			Expect(err).NotTo(HaveOccurred())
			pushed, err := local.Git("ls-tree", "-r", "--full-tree", "origin/main")
			Expect(err).NotTo(HaveOccurred())
			Expect(pushed).NotTo(ContainSubstring("dashboards/lib"),
				"dashboards/lib must not be in the pushed commit; a 160000 entry "+
					"here means collectDirectChildren re-emitted a stale gitlink "+
					"during the commit's tree rebuild (Push-time prune is too "+
					"late to catch this single-commit shape).\nfull tree:\n%s", pushed)
			Expect(pushed).To(ContainSubstring("\tdashboards/new.json"),
				"the new blob must be in the pushed commit; otherwise the test "+
					"isn't exercising the rebuild path.\nfull tree:\n%s", pushed)
		})
	})

	// Type-aware ancestor check: if the submodule's parent is replaced
	// by a blob (not a tree), the submodule is unreachable in the new
	// commit, but a key-presence-only ancestor check would keep it
	// cached and risk resurrection on a later commit that turns the
	// blob back into a directory.
	Context("Submodule's parent is replaced by a blob", func() {
		It("should not resurrect the submodule when a parent-as-blob is later turned back into a tree", func() {
			client, _, local, user := QuickSetup()
			subURL := pushSubmoduleSource(user)

			By("Creating an initial dashboards/home.json")
			Expect(local.CreateDirPath("dashboards")).To(Succeed())
			Expect(local.CreateFile("dashboards/home.json", `{"title":"home"}`)).To(Succeed())
			_, err := local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Add dashboard")
			Expect(err).NotTo(HaveOccurred())

			By("Mounting the submodule at dashboards/lib")
			_ = addSubmodule(local, subURL, "dashboards/lib")

			writer := createWriterFromHead(ctx, client, local)

			By("Push 1: replace the dashboards directory with a blob at the same path")
			_, err = writer.DeleteTree(ctx, "dashboards")
			Expect(err).NotTo(HaveOccurred())
			_, err = writer.CreateBlob(ctx, "dashboards",
				[]byte("dashboards is a regular file now"))
			Expect(err).NotTo(HaveOccurred())
			_, err = writer.Commit(ctx, "Replace dashboards/ with a blob",
				testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			_, err = local.Git("fetch", "origin", "main")
			Expect(err).NotTo(HaveOccurred())
			afterReplace, err := local.Git("ls-tree", "-r", "--full-tree", "origin/main")
			Expect(err).NotTo(HaveOccurred())
			Expect(afterReplace).To(MatchRegexp(`(?m)^100644 blob \S+\tdashboards$`),
				"after Push 1 dashboards must be a blob (mode 100644).\nfull tree:\n%s", afterReplace)

			By("Push 2 on the same writer: turn dashboards back into a directory by writing inside it")
			_, err = writer.DeleteBlob(ctx, "dashboards")
			Expect(err).NotTo(HaveOccurred())
			_, err = writer.CreateBlob(ctx, "dashboards/new.json",
				[]byte(`{"title":"new"}`))
			Expect(err).NotTo(HaveOccurred())
			_, err = writer.Commit(ctx, "dashboards back as a directory",
				testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			_, err = local.Git("fetch", "origin", "main")
			Expect(err).NotTo(HaveOccurred())
			afterRebuild, err := local.Git("ls-tree", "-r", "--full-tree", "origin/main")
			Expect(err).NotTo(HaveOccurred())
			Expect(afterRebuild).NotTo(ContainSubstring("dashboards/lib"),
				"dashboards/lib must not appear in Push 2's commit; a 160000 entry "+
					"here means the Push-time ancestor check accepted a non-tree "+
					"entry as a valid parent and let the orphaned gitlink "+
					"survive into the next rebuild.\nfull tree:\n%s", afterRebuild)
		})
	})

	// DeleteTree(ctx, "") and DeleteTree(ctx, ".") take an early-return
	// fast path that installs an empty root tree without running the
	// per-path submodule prune. A follow-up write on the same writer
	// would otherwise rebuild the root from a cleared treeEntries plus
	// a still-populated submoduleEntries and re-emit every original
	// gitlink. Fixed by clearing submoduleEntries inside that branch.
	Context("Whole-repo wipe via DeleteTree('') leaves no cached submodules behind", func() {
		It("should not resurrect a root-level submodule on a follow-up write after DeleteTree('')", func() {
			client, _, local, user := QuickSetup()
			subURL := pushSubmoduleSource(user)

			By("Mounting a submodule at the repository root")
			_ = addSubmodule(local, subURL, "thirdparty")

			writer := createWriterFromHead(ctx, client, local)

			By("Push 1: wipe the entire tree via DeleteTree('')")
			_, err := writer.DeleteTree(ctx, "")
			Expect(err).NotTo(HaveOccurred())
			_, err = writer.Commit(ctx, "Wipe the repository", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			By("Push 2 on the same writer: add a new file at the root")
			_, err = writer.CreateBlob(ctx, "fresh.txt", []byte("hello"))
			Expect(err).NotTo(HaveOccurred())
			_, err = writer.Commit(ctx, "Add fresh file after wipe", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())
			Expect(writer.Push(ctx)).To(Succeed())

			By("Verifying the submodule was NOT resurrected by the follow-up write")
			_, err = local.Git("fetch", "origin", "main")
			Expect(err).NotTo(HaveOccurred())
			afterFollowUp, err := local.Git("ls-tree", "-r", "--full-tree", "origin/main")
			Expect(err).NotTo(HaveOccurred())
			Expect(afterFollowUp).NotTo(ContainSubstring("\tthirdparty"),
				"thirdparty must not exist in Push 2's tree; a 160000 entry "+
					"here means the DeleteTree('') fast path skipped the "+
					"submodule cache prune and the follow-up rebuild re-emitted "+
					"the gitlink from a stale cache.\nfull tree:\n%s", afterFollowUp)
		})
	})
})
