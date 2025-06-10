package integration_test

import (
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/test/helpers"

	//nolint:stylecheck // specifically ignore ST1001 (dot-imports)
	. "github.com/onsi/ginkgo/v2"
	//nolint:stylecheck // specifically ignore ST1001 (dot-imports)
	. "github.com/onsi/gomega"
)

// Shared test infrastructure
var (
	gitServer *helpers.GitServer
	logger    *helpers.TestLogger
)

func TestIntegrationSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	By("Setting up shared Git server for integration tests")

	logger = helpers.NewTestLogger()
	gitServer = helpers.NewGitServer(logger)
	logger.Success("🚀 Integration test suite setup complete")
	logger.Info("📋 Git server available", "host", gitServer.Host, "port", gitServer.Port)
})

var _ = AfterSuite(func() {
	By("Tearing down shared Git server")
	logger.Info("🧹 Tearing down integration test suite")
	logger.Success("✅ Integration test suite teardown complete")
})

// QuickSetup provides a complete test setup with client, remote repo, local repo, and user
func QuickSetup() (nanogit.Client, *helpers.RemoteRepo, *helpers.LocalGitRepo, *helpers.User) {
	client, remote, local := gitServer.TestRepo()
	return client, remote, local, remote.User
}
