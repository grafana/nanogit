package integration_test

import (
	"context"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/gittest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Shared test infrastructure
var (
	gitServer *gittest.Server
	logger    gittest.Logger
	ctx       context.Context
)

func TestIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	By("Setting up shared Git server for integration tests")

	logger = gittest.NewWriterLogger(GinkgoWriter)
	structuredLogger := gittest.NewStructuredLogger(logger)

	var err error
	gitServer, err = gittest.NewServer(context.Background(),
		gittest.WithLogger(logger),
	)
	Expect(err).NotTo(HaveOccurred())

	structuredLogger.Success("🚀 Integration test suite setup complete")
	structuredLogger.Info("📋 Git server available", "host", gitServer.Host, "port", gitServer.Port)
	//nolint:fatcontext // we need to pass the logger to the context for the tests to work
	ctx = log.ToContext(context.Background(), structuredLogger)
})

var _ = AfterSuite(func() {
	By("Tearing down shared Git server")
	if gitServer != nil {
		Expect(gitServer.Cleanup()).To(Succeed())
	}
})

// QuickSetup provides a complete test setup with client, remote repo, local repo, and user
func QuickSetup() (nanogit.Client, *RemoteRepo, *LocalGitRepo, *User) {
	user, err := gitServer.CreateUser(ctx)
	Expect(err).NotTo(HaveOccurred())

	repo, err := gitServer.CreateRepo(ctx, generateRepoName(), user)
	Expect(err).NotTo(HaveOccurred())

	localRepo, err := gittest.NewLocalRepo(ctx, gittest.WithRepoLogger(logger))
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		Expect(localRepo.Cleanup()).To(Succeed())
	})

	// Wrap local repo for Ginkgo-friendly error handling
	local := &LocalGitRepo{LocalRepo: localRepo}

	client := local.QuickInit(user, repo.AuthURL)

	// Wrap repo for backward compatibility
	remoteRepo := &RemoteRepo{Repo: repo}

	return client, remoteRepo, local, user
}

// generateRepoName generates a unique repository name
func generateRepoName() string {
	return gittest.RandomRepoName()
}

