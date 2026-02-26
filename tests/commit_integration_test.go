package integration_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Commits", func() {
	Context("GetCommit operations", func() {
		var (
			client               nanogit.Client
			local                *gittest.LocalRepo
			user                 *gittest.User
			initialCommitHash    hash.Hash
			modifyFileCommitHash hash.Hash
			renameCommitHash     hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, user = QuickSetup()

			By("Getting initial commit hash")
			output, err := local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			initialCommitHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())

			By("Creating modify file commit")
			err = local.UpdateFile("test.txt", "modified content")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("add", "test.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Modify file")
			Expect(err).NotTo(HaveOccurred())
			output, err = local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			modifyFileCommitHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())

			By("Creating rename file commit")
			_, err = local.Git("mv", "test.txt", "renamed.txt")
			Expect(err).NotTo(HaveOccurred())
			err = local.CreateFile("renamed.txt", "modified content")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Rename and add files")
			Expect(err).NotTo(HaveOccurred())
			output, err = local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			renameCommitHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())

			By("Pushing commits")
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get initial commit details", func() {
			commit, err := client.GetCommit(ctx, initialCommitHash)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying commit details")
			Expect(commit.Hash).To(Equal(initialCommitHash))
			Expect(commit.Parent).To(Equal(hash.Zero)) // First commit has no parent
			Expect(commit.Author.Name).To(Equal(user.Username))
			Expect(commit.Author.Email).To(Equal(user.Email))
			Expect(commit.Author.Time).NotTo(BeZero())

			By("Checking that commit times are recent")
			now := time.Now()
			Expect(commit.Committer.Time.Unix()).To(BeNumerically("~", now.Unix(), 50))
			Expect(commit.Author.Time.Unix()).To(BeNumerically("~", now.Unix(), 50))
		})

		It("should get modify file commit details", func() {
			commit, err := client.GetCommit(ctx, modifyFileCommitHash)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying commit details")
			Expect(commit.Hash).To(Equal(modifyFileCommitHash))
			Expect(commit.Parent).To(Equal(initialCommitHash))
			Expect(commit.Author.Name).To(Equal(user.Username))
			Expect(commit.Author.Email).To(Equal(user.Email))
			Expect(commit.Author.Time).NotTo(BeZero())
			Expect(commit.Committer.Name).To(Equal(user.Username))
			Expect(commit.Committer.Email).To(Equal(user.Email))
			Expect(commit.Committer.Time).NotTo(BeZero())
			Expect(commit.Message).To(Equal("Modify file"))

			By("Checking that commit times are recent")
			now := time.Now()
			Expect(now.Sub(commit.Committer.Time)).To(BeNumerically("<", 10*time.Second))
			Expect(now.Sub(commit.Author.Time)).To(BeNumerically("<", 10*time.Second))
		})

		It("should get rename file commit details", func() {
			commit, err := client.GetCommit(ctx, renameCommitHash)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying commit details")
			Expect(commit.Parent).To(Equal(modifyFileCommitHash))
			Expect(commit.Author.Name).To(Equal(user.Username))
			Expect(commit.Author.Email).To(Equal(user.Email))
			Expect(commit.Author.Time).NotTo(BeZero())
			Expect(commit.Committer.Name).To(Equal(user.Username))
			Expect(commit.Committer.Email).To(Equal(user.Email))
			Expect(commit.Committer.Time).NotTo(BeZero())
			Expect(commit.Message).To(Equal("Rename and add files"))

			By("Checking that commit times are recent")
			now := time.Now()
			Expect(now.Sub(commit.Committer.Time)).To(BeNumerically("<", 10*time.Second))
			Expect(now.Sub(commit.Author.Time)).To(BeNumerically("<", 10*time.Second))
		})
		It("should fail with Object not found error if commit does not exist", func() {
			nonExistentHash, err := hash.FromHex("deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
			Expect(err).NotTo(HaveOccurred())

			_, err = client.GetCommit(ctx, nonExistentHash)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrObjectNotFound)).To(BeTrue())
		})
	})

	Context("CompareCommits operations", func() {
		var (
			client             nanogit.Client
			local              *gittest.LocalRepo
			initialCommitHash  hash.Hash
			modifiedCommitHash hash.Hash
			renamedCommitHash  hash.Hash
			initialFileHash    hash.Hash
			modifiedFileHash   hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, _ = QuickSetup()

			By("Creating initial commit with a file")
			err := local.CreateFile("test.txt", "initial content")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("add", "test.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Initial commit")
			Expect(err).NotTo(HaveOccurred())
			output, err := local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			initialCommitHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())

			By("Creating second commit that modifies the file")
			err = local.CreateFile("test.txt", "modified content")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("add", "test.txt")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Modify file")
			Expect(err).NotTo(HaveOccurred())
			output, err = local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			modifiedCommitHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())

			By("Creating third commit that renames and adds files")
			_, err = local.Git("mv", "test.txt", "renamed.txt")
			Expect(err).NotTo(HaveOccurred())
			err = local.CreateFile("new.txt", "modified content")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("commit", "-m", "Rename and add files")
			Expect(err).NotTo(HaveOccurred())
			output, err = local.Git("rev-parse", "HEAD")
			Expect(err).NotTo(HaveOccurred())
			renamedCommitHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())

			By("Pushing all commits")
			_, err = local.Git("push", "origin", "main", "--force")
			Expect(err).NotTo(HaveOccurred())

			By("Getting the file hashes for verification")
			output, err = local.Git("rev-parse", initialCommitHash.String()+":test.txt")
			Expect(err).NotTo(HaveOccurred())
			initialFileHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())
			output, err = local.Git("rev-parse", modifiedCommitHash.String()+":test.txt")
			Expect(err).NotTo(HaveOccurred())
			modifiedFileHash, err = hash.FromHex(output)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should compare initial and modified commits", func() {
			changes, err := client.CompareCommits(ctx, initialCommitHash, modifiedCommitHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Path).To(Equal("test.txt"))
			Expect(changes[0].Status).To(Equal(protocol.FileStatusModified))
			Expect(changes[0].OldHash).To(Equal(initialFileHash))
			Expect(changes[0].Hash).To(Equal(modifiedFileHash))
		})

		It("should compare modified and renamed commits", func() {
			changes, err := client.CompareCommits(ctx, modifiedCommitHash, renamedCommitHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(changes).To(HaveLen(3))

			Expect(changes[0].Path).To(Equal("new.txt"))
			Expect(changes[0].Status).To(Equal(protocol.FileStatusAdded))

			Expect(changes[1].Path).To(Equal("renamed.txt"))
			Expect(changes[1].Status).To(Equal(protocol.FileStatusAdded))

			Expect(changes[2].Path).To(Equal("test.txt"))
			Expect(changes[2].Status).To(Equal(protocol.FileStatusDeleted))
		})

		It("should compare renamed and modified commits in inverted direction", func() {
			changes, err := client.CompareCommits(ctx, renamedCommitHash, modifiedCommitHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(changes).To(HaveLen(3))

			Expect(changes[0].Path).To(Equal("new.txt"))
			Expect(changes[0].Status).To(Equal(protocol.FileStatusDeleted))

			Expect(changes[1].Path).To(Equal("renamed.txt"))
			Expect(changes[1].Status).To(Equal(protocol.FileStatusDeleted))

			Expect(changes[2].Path).To(Equal("test.txt"))
			Expect(changes[2].Status).To(Equal(protocol.FileStatusAdded))
		})

		It("should compare modified and initial commits in inverted direction", func() {
			changes, err := client.CompareCommits(ctx, modifiedCommitHash, initialCommitHash)
			Expect(err).NotTo(HaveOccurred())
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Path).To(Equal("test.txt"))
			Expect(changes[0].Status).To(Equal(protocol.FileStatusModified))
			Expect(changes[0].OldHash).To(Equal(modifiedFileHash))
			Expect(changes[0].Hash).To(Equal(initialFileHash))
		})
		It("should return object not found error if base commit does not exist", func() {
			nonExistentHash, err := hash.FromHex("deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
			Expect(err).NotTo(HaveOccurred())

			_, err = client.CompareCommits(ctx, nonExistentHash, modifiedCommitHash)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrObjectNotFound)).To(BeTrue())
		})

		It("should return object not found error if head commit does not exist", func() {
			nonExistentHash, err := hash.FromHex("cafebabecafebabecafebabecafebabecafebabe")
			Expect(err).NotTo(HaveOccurred())

			_, err = client.CompareCommits(ctx, initialCommitHash, nonExistentHash)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrObjectNotFound)).To(BeTrue())
		})
	})

	Context("ListCommits operations", func() {
		Context("basic functionality", func() {
			var (
				client   nanogit.Client
				local    *gittest.LocalRepo
				headHash hash.Hash
			)

			BeforeEach(func() {
				By("Setting up test repository")
				client, _, local, _ = QuickSetup()

				By("Creating several commits to test with")
				err := local.CreateFile("file1.txt", "content 1")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("add", "file1.txt")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("commit", "-m", "Add file1")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("push", "-u", "origin", "main", "--force")
				Expect(err).NotTo(HaveOccurred())

				err = local.CreateFile("file2.txt", "content 2")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("add", "file2.txt")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("commit", "-m", "Add file2")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("push", "origin", "main")
				Expect(err).NotTo(HaveOccurred())

				err = local.CreateFile("file3.txt", "content 3")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("add", "file3.txt")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("commit", "-m", "Add file3")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("push", "origin", "main")
				Expect(err).NotTo(HaveOccurred())

				By("Getting the current HEAD commit")
				output, err := local.Git("rev-parse", "HEAD")
				Expect(err).NotTo(HaveOccurred())
				headHash, err = hash.FromHex(output)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should list commits without options", func() {
				commits, err := client.ListCommits(ctx, headHash, nanogit.ListCommitsOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(commits)).To(BeNumerically(">=", 3), "Should have at least 3 commits")

				By("Verifying commits are in reverse chronological order")
				for i := 1; i < len(commits); i++ {
					Expect(commits[i-1].Time().After(commits[i].Time()) || commits[i-1].Time().Equal(commits[i].Time())).To(BeTrue(),
						"Commits should be in reverse chronological order")
				}

				By("Verifying the latest commit message")
				Expect(commits[0].Message).To(Equal("Add file3"))
			})
		})

		Context("pagination", func() {
			var (
				client   nanogit.Client
				local    *gittest.LocalRepo
				headHash hash.Hash
			)

			BeforeEach(func() {
				By("Setting up test repository")
				client, _, local, _ = QuickSetup()

				By("Creating multiple commits")
				for i := 1; i <= 120; i++ {
					err := local.CreateFile(fmt.Sprintf("file%d.txt", i), fmt.Sprintf("content %d", i))
					Expect(err).NotTo(HaveOccurred())
					_, err = local.Git("add", fmt.Sprintf("file%d.txt", i))
					Expect(err).NotTo(HaveOccurred())
					_, err = local.Git("commit", "-m", fmt.Sprintf("Add file%d", i))
					Expect(err).NotTo(HaveOccurred())
				}

				_, err := local.Git("push", "-u", "origin", "main", "--force")
				Expect(err).NotTo(HaveOccurred())

				output, err := local.Git("rev-parse", "HEAD")
				Expect(err).NotTo(HaveOccurred())
				headHash, err = hash.FromHex(output)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should support first page with 2 items per page", func() {
				options := nanogit.ListCommitsOptions{
					PerPage: 2,
					Page:    1,
				}

				commits, err := client.ListCommits(ctx, headHash, options)
				Expect(err).NotTo(HaveOccurred())
				Expect(commits).To(HaveLen(2))
				Expect(commits[0].Message).To(Equal("Add file120"))
				Expect(commits[1].Message).To(Equal("Add file119"))
			})
			It("should default to page 1 if page is zero", func() {
				options := nanogit.ListCommitsOptions{
					PerPage: 2,
					Page:    0,
				}

				commits, err := client.ListCommits(ctx, headHash, options)
				Expect(err).NotTo(HaveOccurred())
				Expect(commits).To(HaveLen(2))
				Expect(commits[0].Message).To(Equal("Add file120"))
				Expect(commits[1].Message).To(Equal("Add file119"))
			})
			It("should return no error and an empty slice if requesting a page that does not exist", func() {
				options := nanogit.ListCommitsOptions{
					PerPage: 100,
					Page:    100, // Deliberately high page number, should be out of range
				}

				commits, err := client.ListCommits(ctx, headHash, options)
				Expect(err).NotTo(HaveOccurred())
				Expect(commits).To(BeEmpty(), "Should return an empty slice for non-existent page")
			})

			It("should support second page", func() {
				options := nanogit.ListCommitsOptions{
					PerPage: 2,
					Page:    2,
				}

				commits, err := client.ListCommits(ctx, headHash, options)
				Expect(err).NotTo(HaveOccurred())
				Expect(commits).To(HaveLen(2))
				Expect(commits[0].Message).To(Equal("Add file118"))
				Expect(commits[1].Message).To(Equal("Add file117"))
			})

			It("should return all pages until the end", func() {
				const perPage = 50
				var (
					allCommits []nanogit.Commit
					lastBatch  []nanogit.Commit
					page       = 1
				)

				for {
					options := nanogit.ListCommitsOptions{
						PerPage: perPage,
						Page:    page,
					}
					commits, err := client.ListCommits(ctx, headHash, options)
					Expect(err).NotTo(HaveOccurred())
					allCommits = append(allCommits, commits...)

					if len(commits) < perPage {
						lastBatch = commits
						break // last page reached
					}
					page++
				}

				Expect(len(allCommits)).To(Equal(121))
				Expect(page).To(Equal(3))
				Expect(lastBatch).To(HaveLen(21))
			})

			It("should default to 100 items per page if more than 100 is requested", func() {
				options := nanogit.ListCommitsOptions{
					PerPage: 101,
					Page:    1,
				}

				commits, err := client.ListCommits(ctx, headHash, options)
				Expect(err).NotTo(HaveOccurred())
				Expect(commits).To(HaveLen(100))
			})
		})

		Context("path filtering", func() {
			var (
				client   nanogit.Client
				local    *gittest.LocalRepo
				headHash hash.Hash
			)

			BeforeEach(func() {
				By("Setting up test repository")
				client, _, local, _ = QuickSetup()

				By("Creating commits affecting different paths")
				err := local.CreateDirPath("docs")
				Expect(err).NotTo(HaveOccurred())
				err = local.CreateFile("docs/readme.md", "readme content")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("add", "docs/readme.md")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("commit", "-m", "Add docs")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("branch", "-M", "main")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("push", "-u", "origin", "main", "--force")
				Expect(err).NotTo(HaveOccurred())

				err = local.CreateDirPath("src")
				Expect(err).NotTo(HaveOccurred())
				err = local.CreateFile("src/main.go", "main content")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("add", "src/main.go")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("commit", "-m", "Add main")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("push", "origin", "main")
				Expect(err).NotTo(HaveOccurred())

				err = local.CreateFile("docs/guide.md", "guide content")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("add", "docs/guide.md")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("commit", "-m", "Add guide")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("push", "origin", "main")
				Expect(err).NotTo(HaveOccurred())

				output, err := local.Git("rev-parse", "HEAD")
				Expect(err).NotTo(HaveOccurred())
				headHash, err = hash.FromHex(output)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should filter commits affecting docs directory", func() {
				options := nanogit.ListCommitsOptions{
					Path: "docs",
				}
				commits, err := client.ListCommits(ctx, headHash, options)
				Expect(err).NotTo(HaveOccurred())

				By("Finding commits that affect docs directory")
				found := 0
				for _, commit := range commits {
					if commit.Message == "Add docs" || commit.Message == "Add guide" {
						found++
					}
				}
				Expect(found).To(BeNumerically(">=", 2), "Should find commits affecting docs directory")
			})

			It("should fail when path contains empty components", func() {
				options := nanogit.ListCommitsOptions{
					Path: "//docs/guide.md", // Invalid path with empty component
				}
				_, err := client.ListCommits(ctx, headHash, options)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("path component is empty"))
			})
			It("should fail when path ends empty component", func() {
				options := nanogit.ListCommitsOptions{
					Path: "docs/  ", // Invalid path with empty component
				}
				_, err := client.ListCommits(ctx, headHash, options)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("path component is empty"))
			})

			It("should filter commits affecting specific file", func() {
				options := nanogit.ListCommitsOptions{
					Path: "src/main.go",
				}
				commits, err := client.ListCommits(ctx, headHash, options)
				Expect(err).NotTo(HaveOccurred())
				Expect(commits).To(HaveLen(1))

				By("Finding the commit that added main.go")
				found := 0
				for _, commit := range commits {
					if commit.Message == "Add main" {
						found++
					}
				}
				Expect(found).To(BeNumerically(">=", 1), "Should find commit affecting src/main.go")
			})

		})

		Context("time filtering", func() {
			var (
				client   nanogit.Client
				local    *gittest.LocalRepo
				headHash hash.Hash
				midTime  time.Time
			)

			BeforeEach(func() {
				By("Setting up test repository")
				client, _, local, _ = QuickSetup()

				By("Creating first commit")
				err := local.CreateFile("file1.txt", "content 1")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("add", "file1.txt")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("commit", "-m", "Old commit")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("branch", "-M", "main")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("push", "-u", "origin", "main", "--force")
				Expect(err).NotTo(HaveOccurred())

				By("Waiting and recording mid time")
				time.Sleep(2 * time.Second)
				midTime = time.Now()
				time.Sleep(2 * time.Second)

				By("Creating second commit")
				err = local.CreateFile("file2.txt", "content 2")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("add", "file2.txt")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("commit", "-m", "New commit")
				Expect(err).NotTo(HaveOccurred())
				_, err = local.Git("push", "origin", "main")
				Expect(err).NotTo(HaveOccurred())

				output, err := local.Git("rev-parse", "HEAD")
				Expect(err).NotTo(HaveOccurred())
				headHash, err = hash.FromHex(output)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should filter commits since midTime", func() {
				options := nanogit.ListCommitsOptions{
					Since: midTime,
				}
				commits, err := client.ListCommits(ctx, headHash, options)
				Expect(err).NotTo(HaveOccurred())

				By("Finding at least the new commit")
				found := false
				for _, commit := range commits {
					if commit.Message == "New commit" {
						found = true
						Expect(commit.Time().After(midTime)).To(BeTrue(), "Commit should be after the since time")
					}
				}
				Expect(found).To(BeTrue(), "Should find the new commit")
			})
		})
	})
})
