package integration_test

import (
	"context"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol/signing/testsigning"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Shared test infrastructure
var (
	gitServer *gittest.Server
	logger    interface {
		gittest.Logger
		Debug(msg string, keysAndValues ...any)
		Info(msg string, keysAndValues ...any)
		Warn(msg string, keysAndValues ...any)
		Error(msg string, keysAndValues ...any)
		Success(msg string, keysAndValues ...any)
	}
	ctx context.Context

	// quickSetupExtraOpts is prepended to the per-call extraOpts in QuickSetup.
	// Outer Describe/Context blocks set this in BeforeEach to parameterize an
	// entire tree of specs (e.g. once with WithCapabilityNegotiation and once
	// without) without having to thread the option through every call site.
	quickSetupExtraOpts []options.Option
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

	ssh := testsigning.LoadSSH(GinkgoT())

	var err error
	gitServer, err = gittest.NewServer(context.Background(),
		gittest.WithLogger(baseLogger),
		gittest.WithTrustedSSHKeys(string(ssh.PublicLine)),
	)
	Expect(err).NotTo(HaveOccurred())

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
func QuickSetup(extraOpts ...options.Option) (nanogit.Client, *gittest.RemoteRepository, *gittest.LocalRepo, *gittest.User) {
	user, err := gitServer.CreateUser(ctx)
	Expect(err).NotTo(HaveOccurred())

	repo, err := gitServer.CreateRepo(ctx, gittest.RandomRepoName(), user)
	Expect(err).NotTo(HaveOccurred())

	local, err := gittest.NewLocalRepo(ctx,
		gittest.WithRepoLogger(logger),
		gittest.WithGitTrace(),
	)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		Expect(local.Cleanup()).To(Succeed())
	})

	// Initialize with remote and get connection info
	connInfo, err := local.InitWithRemote(user, repo)
	Expect(err).NotTo(HaveOccurred())

	// Create nanogit client from connection info. The package-level
	// quickSetupExtraOpts is applied first so per-call extraOpts can still
	// override (e.g. an outer Describe sets WithCapabilityNegotiation while
	// an inner spec adds WithReceivePackCapabilities).
	clientOpts := []options.Option{
		options.WithBasicAuth(connInfo.Username, connInfo.Password),
	}
	clientOpts = append(clientOpts, quickSetupExtraOpts...)
	clientOpts = append(clientOpts, extraOpts...)
	client, err := nanogit.NewHTTPClient(connInfo.URL, clientOpts...)
	Expect(err).NotTo(HaveOccurred())

	return client, repo, local, user
}
