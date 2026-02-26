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
	gitServer *GitServer
	logger    interface {
		gittest.Logger
		Debug(msg string, keysAndValues ...any)
		Info(msg string, keysAndValues ...any)
		Warn(msg string, keysAndValues ...any)
		Error(msg string, keysAndValues ...any)
		Success(msg string, keysAndValues ...any)
	}
	ctx context.Context
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

	baseLogger := gittest.NewWriterLogger(GinkgoWriter)
	logger = gittest.NewStructuredLogger(baseLogger)

	var err error
	server, err := gittest.NewServer(context.Background(),
		gittest.WithLogger(baseLogger),
	)
	Expect(err).NotTo(HaveOccurred())

	// Wrap server for backward compatibility
	gitServer = &GitServer{Server: server}

	logger.Success("🚀 Integration test suite setup complete")
	logger.Info("📋 Git server available", "host", gitServer.Host, "port", gitServer.Port)
	//nolint:fatcontext // we need to pass the logger to the context for the tests to work
	ctx = log.ToContext(context.Background(), logger)
})

var _ = AfterSuite(func() {
	By("Tearing down shared Git server")
	if gitServer != nil {
		Expect(gitServer.Cleanup()).To(Succeed())
	}
})

// QuickSetup provides a complete test setup with client, remote repo, local repo, and user
func QuickSetup() (nanogit.Client, *RemoteRepo, *LocalRepository, *User) {
	user, err := gitServer.CreateUser(ctx)
	Expect(err).NotTo(HaveOccurred())

	repo, err := gitServer.CreateRepo(ctx, gittest.RandomRepoName(), user)
	Expect(err).NotTo(HaveOccurred())

	localRepo, err := gittest.NewLocalRepo(ctx, gittest.WithRepoLogger(logger))
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		Expect(localRepo.Cleanup()).To(Succeed())
	})

	// Wrap local repo for Ginkgo-friendly error handling
	local := &LocalRepository{LocalRepo: localRepo}

	remote := repo
	client := local.InitWithRemote(user, remote)

	// Wrap repo for backward compatibility
	remoteRepo := &RemoteRepo{RemoteRepository: repo}

	return client, remoteRepo, local, user
}


