package nanogit_test

import (
	"context"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/internal/testhelpers"
	"github.com/grafana/nanogit/log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Shared test infrastructure
var (
	gitServer *testhelpers.GitServer
	logger    *testhelpers.TestLogger
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

	logger = testhelpers.NewTestLogger(GinkgoWriter.Printf)
	gitServer = testhelpers.NewGitServer(logger)
	logger.Success("🚀 Integration test suite setup complete")
	logger.Info("📋 Git server available", "host", gitServer.Host, "port", gitServer.Port)
	//nolint:fatcontext // we need to pass the logger to the context for the tests to work
	ctx = log.ToContext(context.Background(), logger)
})

var _ = AfterSuite(func() {
	By("Tearing down shared Git server")
	logger.Info("🧹 Tearing down integration test suite")
	logger.Success("✅ Integration test suite teardown complete")
})

// QuickSetup provides a complete test setup with client, remote repo, local repo, and user
func QuickSetup() (nanogit.Client, *testhelpers.RemoteRepo, *testhelpers.LocalGitRepo, *testhelpers.User) {
	client, remote, local := gitServer.TestRepo()
	return client, remote, local, remote.User
}
