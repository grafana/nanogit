package integration_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Commits", func() {
	Context("GetCommit operations", func() {
		var (
			client               nanogit.Client
			local                *LocalGitRepo
			user                 *User
			initialCommitHash    hash.Hash
			modifyFileCommitHash hash.Hash
			renameCommitHash     hash.Hash
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, _, local, user = QuickSetup()

			By("Getting initial commit hash")
			var err error
			initialCommitHash, err = hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			By("Creating modify file commit")
			local.UpdateFile("test.txt", "modified content")
			local.Git("add", "test.txt")
			local.Git("commit", "-m", "Modify file")
			modifyFileCommitHash, err = hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			By("Creating rename file commit")
			local.Git("mv", "test.txt", "renamed.txt")
			local.CreateFile("renamed.txt", "modified content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Rename and add files")
			renameCommitHash, err = hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			By("Pushing commits")
			local.Git("push", "origin", "main", "--force")
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
			Expect(commit.Committer.Time.Unix()).To(BeNumerically("~", now.Unix(), 5))
			Expect(commit.Author.Time.Unix()).To(BeNumerically("~", now.Unix(), 5))
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
			Expect(commit.Committer.Time.Unix()).To(BeNumerically("~", now.Unix(), 5))
			Expect(commit.Author.Time.Unix()).To(BeNumerically("~", now.Unix(), 5))
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
			local              *LocalGitRepo
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
			local.CreateFile("test.txt", "initial content")
			local.Git("add", "test.txt")
			local.Git("commit", "-m", "Initial commit")
			var err error
			initialCommitHash, err = hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			By("Creating second commit that modifies the file")
			local.CreateFile("test.txt", "modified content")
			local.Git("add", "test.txt")
			local.Git("commit", "-m", "Modify file")
			modifiedCommitHash, err = hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			By("Creating third commit that renames and adds files")
			local.Git("mv", "test.txt", "renamed.txt")
			local.CreateFile("new.txt", "modified content")
			local.Git("add", ".")
			local.Git("commit", "-m", "Rename and add files")
			renamedCommitHash, err = hash.FromHex(local.Git("rev-parse", "HEAD"))
			Expect(err).NotTo(HaveOccurred())

			By("Pushing all commits")
			local.Git("push", "origin", "main", "--force")

			By("Getting the file hashes for verification")
			initialFileHash, err = hash.FromHex(local.Git("rev-parse", initialCommitHash.String()+":test.txt"))
			Expect(err).NotTo(HaveOccurred())
			modifiedFileHash, err = hash.FromHex(local.Git("rev-parse", modifiedCommitHash.String()+":test.txt"))
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
				local    *LocalGitRepo
				headHash hash.Hash
			)

			BeforeEach(func() {
				By("Setting up test repository")
				client, _, local, _ = QuickSetup()

				By("Creating several commits to test with")
				local.CreateFile("file1.txt", "content 1")
				local.Git("add", "file1.txt")
				local.Git("commit", "-m", "Add file1")
				local.Git("push", "-u", "origin", "main", "--force")

				local.CreateFile("file2.txt", "content 2")
				local.Git("add", "file2.txt")
				local.Git("commit", "-m", "Add file2")
				local.Git("push", "origin", "main")

				local.CreateFile("file3.txt", "content 3")
				local.Git("add", "file3.txt")
				local.Git("commit", "-m", "Add file3")
				local.Git("push", "origin", "main")

				By("Getting the current HEAD commit")
				var err error
				headHash, err = hash.FromHex(local.Git("rev-parse", "HEAD"))
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
				local    *LocalGitRepo
				headHash hash.Hash
			)

			BeforeEach(func() {
				By("Setting up test repository")
				client, _, local, _ = QuickSetup()

				By("Creating multiple commits")
				for i := 1; i <= 120; i++ {
					local.CreateFile(fmt.Sprintf("file%d.txt", i), fmt.Sprintf("content %d", i))
					local.Git("add", fmt.Sprintf("file%d.txt", i))
					local.Git("commit", "-m", fmt.Sprintf("Add file%d", i))
				}

				local.Git("push", "-u", "origin", "main", "--force")

				var err error
				headHash, err = hash.FromHex(local.Git("rev-parse", "HEAD"))
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
				local    *LocalGitRepo
				headHash hash.Hash
			)

			BeforeEach(func() {
				By("Setting up test repository")
				client, _, local, _ = QuickSetup()

				By("Creating commits affecting different paths")
				local.CreateDirPath("docs")
				local.CreateFile("docs/readme.md", "readme content")
				local.Git("add", "docs/readme.md")
				local.Git("commit", "-m", "Add docs")
				local.Git("branch", "-M", "main")
				local.Git("push", "-u", "origin", "main", "--force")

				local.CreateDirPath("src")
				local.CreateFile("src/main.go", "main content")
				local.Git("add", "src/main.go")
				local.Git("commit", "-m", "Add main")
				local.Git("push", "origin", "main")

				local.CreateFile("docs/guide.md", "guide content")
				local.Git("add", "docs/guide.md")
				local.Git("commit", "-m", "Add guide")
				local.Git("push", "origin", "main")

				var err error
				headHash, err = hash.FromHex(local.Git("rev-parse", "HEAD"))
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
				local    *LocalGitRepo
				headHash hash.Hash
				midTime  time.Time
			)

			BeforeEach(func() {
				By("Setting up test repository")
				client, _, local, _ = QuickSetup()

				By("Creating first commit")
				local.CreateFile("file1.txt", "content 1")
				local.Git("add", "file1.txt")
				local.Git("commit", "-m", "Old commit")
				local.Git("branch", "-M", "main")
				local.Git("push", "-u", "origin", "main", "--force")

				By("Waiting and recording mid time")
				time.Sleep(2 * time.Second)
				midTime = time.Now()
				time.Sleep(2 * time.Second)

				By("Creating second commit")
				local.CreateFile("file2.txt", "content 2")
				local.Git("add", "file2.txt")
				local.Git("commit", "-m", "New commit")
				local.Git("push", "origin", "main")

				var err error
				headHash, err = hash.FromHex(local.Git("rev-parse", "HEAD"))
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
