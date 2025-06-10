package integration_test

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/test/helpers"
	. "github.com/onsi/ginkgo/v2"
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

	// Create logger and Git server using our new Ginkgo-compatible helpers
	logger, gitServer = helpers.NewGitServerWithLogger()

	logger.Success("🚀 Integration test suite setup complete")
	logger.Info("📋 Git server available", "host", gitServer.Host, "port", gitServer.Port)
})

var _ = AfterSuite(func() {
	By("Tearing down shared Git server")
	if logger != nil {
		logger.Info("🧹 Tearing down integration test suite")
		logger.Success("✅ Integration test suite teardown complete")
	}
})

// Helper functions for common test patterns

// CreateTestRepo creates a fresh repository for a test with a unique name
func CreateTestRepo() (*helpers.RemoteRepo, *helpers.User) {
	user := gitServer.CreateUser()

	// Generate unique repo name
	var suffix uint32
	err := binary.Read(rand.Reader, binary.LittleEndian, &suffix)
	Expect(err).NotTo(HaveOccurred())
	suffix = suffix % 10000

	repoName := fmt.Sprintf("testrepo-%d", suffix)
	remote := gitServer.CreateRepo(repoName, user)

	return remote, user
}

// CreateTestRepoWithClient is a convenience function that creates a test repo and returns client, remote, and local
func CreateTestRepoWithClient() (nanogit.Client, *helpers.RemoteRepo, *helpers.LocalGitRepo) {
	return gitServer.TestRepo()
}

// QuickSetup provides a complete test setup with client, remote repo, local repo, and user
func QuickSetup() (nanogit.Client, *helpers.RemoteRepo, *helpers.LocalGitRepo, *helpers.User) {
	client, remote, local := gitServer.TestRepo()
	return client, remote, local, remote.User
}

// QuickSetupWithContext provides the same as QuickSetup but returns logger for additional context
func QuickSetupWithContext() (nanogit.Client, *helpers.RemoteRepo, *helpers.LocalGitRepo, *helpers.User, *helpers.TestLogger) {
	client, remote, local, user := QuickSetup()
	return client, remote, local, user, logger
}
