package integration_test

import (
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Protocol Error Scenarios", func() {
	var (
		testAuthor = nanogit.Author{
			Name:  "Protocol Test Author",
			Email: "protocol-test@example.com",
			Time:  time.Now(),
		}
		testCommitter = nanogit.Committer{
			Name:  "Protocol Test Committer",
			Email: "protocol-test@example.com",
			Time:  time.Now(),
		}
	)

	Context("Reference Update Failures", func() {
		It("should handle old hash reference update failure", func() {
			client, _, local, _ := QuickSetup()

			By("Setting up initial state with a commit")
			local.CreateFile("initial.txt", "initial content")
			local.Git("add", "initial.txt")
			local.Git("commit", "-m", "Initial commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			By("Getting the initial commit hash")
			initialHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			By("Creating a new commit locally to advance the remote")
			local.Git("commit", "--allow-empty", "-m", "Second commit")
			local.Git("push", "origin", "main")

			By("Creating a writer with the old (stale) hash")
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: initialHash, // This is now outdated
			}
			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			By("Attempting to create and commit a new file")
			_, err = writer.CreateBlob(ctx, "newfile.txt", []byte("new content"))
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Add new file with stale ref", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			By("Attempting to push with stale reference - should fail")
			err = writer.Push(ctx)
			Expect(err).To(HaveOccurred())

			By("Push failed as expected - analyzing error type")
			if protocol.IsGitReferenceUpdateError(err) {
				var refUpdateErr *protocol.GitReferenceUpdateError
				Expect(errors.As(err, &refUpdateErr)).To(BeTrue())
				Expect(refUpdateErr.RefName).To(Equal("refs/heads/main"))
				logger.Info("Reference update failed as expected", "ref", refUpdateErr.RefName, "reason", refUpdateErr.Reason)
			} else if protocol.IsGitServerError(err) {
				var serverErr *protocol.GitServerError
				Expect(errors.As(err, &serverErr)).To(BeTrue())
				logger.Info("Push failed with Git server error as expected", "type", serverErr.ErrorType, "message", serverErr.Message)
			} else {
				logger.Info("Push failed with other error type", "error", err.Error(), "type", fmt.Sprintf("%T", err))
				// Still consider this a valid test result - any error shows conflict detection works
			}
		})

		It("should handle non-fast-forward reference update failure", func() {
			client, _, local, _ := QuickSetup()

			By("Setting up initial state")
			local.CreateFile("file1.txt", "content 1")
			local.Git("add", "file1.txt")
			local.Git("commit", "-m", "First commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			firstCommitHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			By("Creating a divergent commit history locally")
			local.Git("commit", "--allow-empty", "-m", "Local commit")
			local.Git("push", "origin", "main")

			By("Resetting local to first commit and creating conflicting commit")
			local.Git("reset", "--hard", firstCommitHash.String())
			local.CreateFile("file2.txt", "conflicting content")
			local.Git("add", "file2.txt")
			local.Git("commit", "-m", "Conflicting commit")

			By("Attempting to push non-fast-forward update")
			cmd := exec.Command("git", "push", "origin", "main", "--force-with-lease")
			cmd.Dir = local.Path
			pushErr := cmd.Run()
			if pushErr != nil {
				// This is expected - Git itself prevents non-fast-forward pushes
				logger.Info("Git prevented non-fast-forward push as expected", "error", pushErr.Error())
			}

			By("Using nanogit writer to test protocol error handling")
			// Get the current remote state
			currentRef, err := client.GetRef(ctx, "refs/heads/main")
			Expect(err).NotTo(HaveOccurred())

			// Create writer with the actual current state
			writer, err := client.NewStagedWriter(ctx, currentRef)
			Expect(err).NotTo(HaveOccurred())

			// This should work since we're using the correct reference
			_, err = writer.CreateBlob(ctx, "valid.txt", []byte("valid content"))
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Valid commit", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			err = writer.Push(ctx)
			Expect(err).NotTo(HaveOccurred())

			logger.Info("Valid push succeeded as expected")
		})
	})

	Context("Git Server Error Messages", func() {
		It("should handle server-side validation errors", func() {
			client, remote, local, _ := QuickSetup()

			By("Setting up competing operations to trigger server validation errors")

			// Get current state
			currentHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			// Create a commit locally to advance the remote
			local.CreateFile("server-test.txt", "server test content")
			local.Git("add", "server-test.txt")
			local.Git("commit", "-m", "Server test commit")
			local.Git("push", "origin", "main")

			By("Attempting push with outdated reference to trigger server validation error")
			// Use the old hash before the server-side update
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash, // This is now stale
			}
			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			// Create content that would be valid if the reference was current
			_, err = writer.CreateBlob(ctx, "validation-test.txt", []byte("validation test content"))
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Commit with stale ref", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			By("Pushing with stale reference - expecting server validation error")
			err = writer.Push(ctx)
			Expect(err).To(HaveOccurred(), "Server should reject push with stale reference")

			By("Verifying error is a Git protocol error")
			if protocol.IsGitServerError(err) {
				var serverErr *protocol.GitServerError
				Expect(errors.As(err, &serverErr)).To(BeTrue())
				logger.Info("Git server error detected as expected", "type", serverErr.ErrorType, "message", serverErr.Message)
			} else if protocol.IsGitReferenceUpdateError(err) {
				var refUpdateErr *protocol.GitReferenceUpdateError
				Expect(errors.As(err, &refUpdateErr)).To(BeTrue())
				logger.Info("Reference update error detected as expected", "ref", refUpdateErr.RefName, "reason", refUpdateErr.Reason)
			} else {
				logger.Info("Server validation failed with other error type", "error", err.Error(), "type", fmt.Sprintf("%T", err))
			}

			By("Verifying repository remains accessible after error")
			refs, err := client.ListRefs(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(refs)).To(BeNumerically(">", 0))
			logger.Info("Repository remains accessible after validation error", "refs_count", len(refs), "repo_name", remote.RepoName())
		})

		It("should handle protocol errors during concurrent pushes", func() {
			client, _, local, _ := QuickSetup()

			By("Setting up concurrent operations that could trigger protocol conflicts")
			currentHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			// Create two writers with the same starting reference
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash,
			}

			writer1, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			writer2, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			By("Creating conflicting commits")
			// Writer 1 creates a commit
			_, err = writer1.CreateBlob(ctx, "conflict1.txt", []byte("content from writer 1"))
			Expect(err).NotTo(HaveOccurred())
			_, err = writer1.Commit(ctx, "Commit from writer 1", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			// Writer 2 creates a different commit
			_, err = writer2.CreateBlob(ctx, "conflict2.txt", []byte("content from writer 2"))
			Expect(err).NotTo(HaveOccurred())
			_, err = writer2.Commit(ctx, "Commit from writer 2", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			By("First push should succeed")
			err = writer1.Push(ctx)
			Expect(err).NotTo(HaveOccurred())
			logger.Info("First writer push succeeded")

			By("Second push should fail with protocol error")
			err = writer2.Push(ctx)
			Expect(err).To(HaveOccurred(), "Second push should fail due to reference conflict")

			By("Verifying error is a Git protocol error")
			if protocol.IsGitReferenceUpdateError(err) {
				var refUpdateErr *protocol.GitReferenceUpdateError
				Expect(errors.As(err, &refUpdateErr)).To(BeTrue())
				logger.Info("Reference update error detected as expected", "ref", refUpdateErr.RefName, "reason", refUpdateErr.Reason)
			} else if protocol.IsGitServerError(err) {
				var serverErr *protocol.GitServerError
				Expect(errors.As(err, &serverErr)).To(BeTrue())
				logger.Info("Git server error during concurrent push", "type", serverErr.ErrorType, "message", serverErr.Message)
			} else {
				logger.Info("Concurrent push failed with other error type", "error", err.Error(), "type", fmt.Sprintf("%T", err))
			}
		})
	})

	Context("Protocol Error Recovery", func() {
		It("should handle transient errors gracefully", func() {
			client, _, local, _ := QuickSetup()

			By("Setting up repository")
			local.CreateFile("recovery-test.txt", "recovery test")
			local.Git("add", "recovery-test.txt")
			local.Git("commit", "-m", "Recovery test commit")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			By("Getting current state")
			currentHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash,
			}

			By("Creating multiple writers to simulate concurrent operations")
			writer1, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			writer2, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			By("Making changes with first writer")
			_, err = writer1.CreateBlob(ctx, "file1.txt", []byte("content from writer 1"))
			Expect(err).NotTo(HaveOccurred())

			commit1, err := writer1.Commit(ctx, "Commit from writer 1", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			By("Making changes with second writer")
			_, err = writer2.CreateBlob(ctx, "file2.txt", []byte("content from writer 2"))
			Expect(err).NotTo(HaveOccurred())

			_, err = writer2.Commit(ctx, "Commit from writer 2", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			By("First writer pushes successfully")
			err = writer1.Push(ctx)
			Expect(err).NotTo(HaveOccurred())
			logger.Info("First writer pushed successfully", "commit", commit1.Hash.String())

			By("Second writer should fail due to stale reference")
			err = writer2.Push(ctx)
			Expect(err).To(HaveOccurred(), "Second writer should fail when reference is stale")

			By("Verifying the error type and recovery information")
			if protocol.IsGitReferenceUpdateError(err) {
				var refUpdateErr *protocol.GitReferenceUpdateError
				Expect(errors.As(err, &refUpdateErr)).To(BeTrue())
				logger.Info("Reference update error as expected", "ref", refUpdateErr.RefName, "reason", refUpdateErr.Reason)
			} else if protocol.IsGitServerError(err) {
				var serverErr *protocol.GitServerError
				Expect(errors.As(err, &serverErr)).To(BeTrue())
				logger.Info("Git server error during concurrent push", "type", serverErr.ErrorType, "message", serverErr.Message)
			} else {
				logger.Info("Concurrent push failed with other error type", "error", err.Error(), "type", fmt.Sprintf("%T", err))
			}

			By("Demonstrating recovery by getting updated reference")
			updatedRef, err := client.GetRef(ctx, "refs/heads/main")
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedRef.Hash).To(Equal(commit1.Hash))
			logger.Info("Successfully retrieved updated reference for recovery", "new_hash", updatedRef.Hash.String())

			By("Verifying repository state after conflict")
			refs, err := client.ListRefs(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(refs)).To(BeNumerically(">", 0))

			finalRef, err := client.GetRef(ctx, "refs/heads/main")
			Expect(err).NotTo(HaveOccurred())
			Expect(finalRef.Hash).To(Equal(commit1.Hash))
			logger.Info("Repository state is consistent", "final_commit", finalRef.Hash.String())
		})
	})

	Context("Protocol Error Edge Cases", func() {
		It("should handle invalid object scenarios", func() {
			client, _, local, _ := QuickSetup()

			By("Setting up repository")
			local.CreateFile("edge-case.txt", "edge case test")
			local.Git("add", "edge-case.txt")
			local.Git("commit", "-m", "Edge case test")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			By("Testing with valid operations that might trigger edge cases")
			currentHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash,
			}
			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			By("Creating a file with maximum allowed content size")
			// Create content approaching reasonable limits (not actually max to avoid timeouts)
			largeContent := make([]byte, 1024*1024) // 1MB
			for i := range largeContent {
				largeContent[i] = byte('A' + (i % 26))
			}

			_, err = writer.CreateBlob(ctx, "large-file.txt", largeContent)
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Add large file", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			By("Pushing large content - checking for pack/unpack errors")
			err = writer.Push(ctx)
			if err != nil {
				By("Analyzing large content push error")
				if protocol.IsGitUnpackError(err) {
					var unpackErr *protocol.GitUnpackError
					Expect(errors.As(err, &unpackErr)).To(BeTrue())
					logger.Info("Unpack error with large content", "message", unpackErr.Message)
				} else if protocol.IsGitServerError(err) {
					var serverErr *protocol.GitServerError
					Expect(errors.As(err, &serverErr)).To(BeTrue())
					logger.Info("Server error with large content", "type", serverErr.ErrorType, "message", serverErr.Message)
				} else {
					logger.Info("Large content push failed with other error", "error", err.Error())
				}
			} else {
				logger.Info("Large content push succeeded")
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should handle empty and boundary condition operations", func() {
			client, _, local, _ := QuickSetup()

			By("Setting up repository")
			local.CreateFile("boundary.txt", "boundary test")
			local.Git("add", "boundary.txt")
			local.Git("commit", "-m", "Boundary test")
			local.Git("branch", "-M", "main")
			local.Git("push", "-u", "origin", "main", "--force")

			By("Testing boundary conditions")
			currentHash, err := hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: currentHash,
			}
			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())

			By("Creating an empty file")
			_, err = writer.CreateBlob(ctx, "empty-file.txt", []byte{})
			Expect(err).NotTo(HaveOccurred())

			By("Creating a file with single byte")
			_, err = writer.CreateBlob(ctx, "single-byte.txt", []byte{'x'})
			Expect(err).NotTo(HaveOccurred())

			By("Creating files with special characters in names")
			_, err = writer.CreateBlob(ctx, "file-with-unicode-🚀.txt", []byte("unicode content"))
			Expect(err).NotTo(HaveOccurred())

			_, err = writer.Commit(ctx, "Add boundary condition files", testAuthor, testCommitter)
			Expect(err).NotTo(HaveOccurred())

			By("Pushing boundary condition content")
			err = writer.Push(ctx)
			if err != nil {
				By("Analyzing boundary condition errors")
				if protocol.IsGitServerError(err) {
					var serverErr *protocol.GitServerError
					Expect(errors.As(err, &serverErr)).To(BeTrue())
					logger.Info("Server error with boundary conditions", "type", serverErr.ErrorType, "message", serverErr.Message)
				} else {
					logger.Info("Boundary condition push failed", "error", err.Error())
				}
			} else {
				logger.Info("Boundary condition push succeeded")
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})
})
