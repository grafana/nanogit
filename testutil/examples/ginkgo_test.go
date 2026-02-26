package examples

import (
	"context"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestGinkgoIntegration runs the Ginkgo test suite.
func TestGinkgoIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testutil Ginkgo Integration Suite")
}

var _ = Describe("Testutil with Ginkgo", func() {
	var (
		ctx     context.Context
		client  nanogit.Client
		repo    *testutil.Repo
		local   *testutil.LocalRepo
		user    *testutil.User
		cleanup func()
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Use QuickSetup with Ginkgo-friendly logger
		var err error
		client, repo, local, user, cleanup, err = testutil.QuickSetup(ctx,
			testutil.WithQuickSetupLogger(testutil.NewWriterLogger(GinkgoWriter)),
		)
		Expect(err).NotTo(HaveOccurred(), "failed to set up test environment")
	})

	AfterEach(func() {
		if cleanup != nil {
			cleanup()
		}
	})

	Context("when working with a test repository", func() {
		It("should have valid test instances", func() {
			Expect(client).NotTo(BeNil())
			Expect(repo).NotTo(BeNil())
			Expect(local).NotTo(BeNil())
			Expect(user).NotTo(BeNil())

			GinkgoWriter.Printf("Test environment ready:\n")
			GinkgoWriter.Printf("  Server: %s\n", repo.URL)
			GinkgoWriter.Printf("  User: %s\n", user.Username)
			GinkgoWriter.Printf("  Repo: %s\n", repo.Name)
		})

		It("should create and push files", func() {
			// Create a new file
			err := local.CreateFile("feature.txt", "New feature")
			Expect(err).NotTo(HaveOccurred())

			// Add and commit
			_, err = local.Git("add", "feature.txt")
			Expect(err).NotTo(HaveOccurred())

			_, err = local.Git("commit", "-m", "Add feature")
			Expect(err).NotTo(HaveOccurred())

			_, err = local.Git("push", "origin", "main")
			Expect(err).NotTo(HaveOccurred())

			// Verify with client
			ref, err := client.GetRef(ctx, "refs/heads/main")
			Expect(err).NotTo(HaveOccurred())
			Expect(ref).NotTo(BeNil())
		})

		It("should handle file updates", func() {
			// Update existing file
			err := local.UpdateFile("test.txt", "Updated content")
			Expect(err).NotTo(HaveOccurred())

			// Commit and push
			_, err = local.Git("add", "test.txt")
			Expect(err).NotTo(HaveOccurred())

			_, err = local.Git("commit", "-m", "Update test.txt")
			Expect(err).NotTo(HaveOccurred())

			_, err = local.Git("push", "origin", "main")
			Expect(err).NotTo(HaveOccurred())

			// Verify update
			ref, err := client.GetRef(ctx, "refs/heads/main")
			Expect(err).NotTo(HaveOccurred())
			Expect(ref).NotTo(BeNil())
		})

		It("should create directory structures", func() {
			// Create nested directories
			err := local.CreateDirPath("docs/api/v1")
			Expect(err).NotTo(HaveOccurred())

			err = local.CreateDirPath("docs/guides")
			Expect(err).NotTo(HaveOccurred())

			// Add files in directories
			err = local.CreateFile("docs/api/v1/endpoints.md", "# API Endpoints\n")
			Expect(err).NotTo(HaveOccurred())

			err = local.CreateFile("docs/guides/quickstart.md", "# Quick Start\n")
			Expect(err).NotTo(HaveOccurred())

			// Commit and push
			_, err = local.Git("add", ".")
			Expect(err).NotTo(HaveOccurred())

			_, err = local.Git("commit", "-m", "Add documentation")
			Expect(err).NotTo(HaveOccurred())

			_, err = local.Git("push", "origin", "main")
			Expect(err).NotTo(HaveOccurred())

			// Verify
			ref, err := client.GetRef(ctx, "refs/heads/main")
			Expect(err).NotTo(HaveOccurred())
			Expect(ref).NotTo(BeNil())

			// Log contents for debugging
			local.LogContents()
		})
	})

	Context("when using manual setup", func() {
		var (
			server        *testutil.Server
			serverCleanup func()
		)

		BeforeEach(func() {
			var err error
			server, serverCleanup, err = testutil.QuickServer(ctx,
				testutil.WithLogger(testutil.NewWriterLogger(GinkgoWriter)),
			)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if serverCleanup != nil {
				serverCleanup()
			}
		})

		It("should create multiple users", func() {
			// Create first user
			user1, err := server.CreateUser(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(user1.Username).NotTo(BeEmpty())

			// Create second user
			user2, err := server.CreateUser(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(user2.Username).NotTo(BeEmpty())

			// Users should be different
			Expect(user1.Username).NotTo(Equal(user2.Username))
			Expect(user1.Email).NotTo(Equal(user2.Email))

			GinkgoWriter.Printf("Created users: %s, %s\n", user1.Username, user2.Username)
		})

		It("should create multiple repositories", func() {
			// Create user
			user, err := server.CreateUser(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Create multiple repos
			repo1, err := server.CreateRepo(ctx, "repo1", user)
			Expect(err).NotTo(HaveOccurred())
			Expect(repo1.Name).To(Equal("repo1"))

			repo2, err := server.CreateRepo(ctx, "repo2", user)
			Expect(err).NotTo(HaveOccurred())
			Expect(repo2.Name).To(Equal("repo2"))

			// URLs should be different
			Expect(repo1.URL).NotTo(Equal(repo2.URL))

			GinkgoWriter.Printf("Created repos: %s, %s\n", repo1.Name, repo2.Name)
		})

		It("should generate user tokens", func() {
			// Create user
			user, err := server.CreateUser(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Generate token
			token, err := server.GenerateUserToken(ctx, user.Username)
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())
			Expect(token).To(HavePrefix("token "))

			GinkgoWriter.Printf("Generated token for %s: %s\n", user.Username, token)
		})
	})

	Context("with colored logger", func() {
		var (
			coloredClient  nanogit.Client
			coloredRepo    *testutil.Repo
			coloredLocal   *testutil.LocalRepo
			coloredCleanup func()
		)

		BeforeEach(func() {
			// Use colored logger for nice output
			var err error
			coloredClient, coloredRepo, coloredLocal, _, coloredCleanup, err = testutil.QuickSetup(ctx,
				testutil.WithQuickSetupLogger(testutil.NewColoredLogger(GinkgoWriter)),
			)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if coloredCleanup != nil {
				coloredCleanup()
			}
		})

		It("should show colored output", func() {
			// This test demonstrates colored output
			// The logs will have colors and emojis if your terminal supports it

			err := coloredLocal.CreateFile("colors.txt", "Rainbow! 🌈")
			Expect(err).NotTo(HaveOccurred())

			_, err = coloredLocal.Git("add", "colors.txt")
			Expect(err).NotTo(HaveOccurred())

			_, err = coloredLocal.Git("commit", "-m", "Add colorful file")
			Expect(err).NotTo(HaveOccurred())

			_, err = coloredLocal.Git("push", "origin", "main")
			Expect(err).NotTo(HaveOccurred())

			ref, err := coloredClient.GetRef(ctx, "refs/heads/main")
			Expect(err).NotTo(HaveOccurred())
			Expect(ref).NotTo(BeNil())

			GinkgoWriter.Println("✨ Colored logging demonstration complete!")
		})
	})
})
