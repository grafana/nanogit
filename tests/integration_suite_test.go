package integration_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Shared test infrastructure
var (
	gitServer *testutil.Server
	logger    testutil.Logger
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

	logger = testutil.NewWriterLogger(GinkgoWriter)
	structuredLogger := newGinkgoStructuredLogger(logger)

	var err error
	gitServer, err = testutil.NewServer(context.Background(),
		testutil.WithLogger(logger),
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

	localRepo, err := testutil.NewLocalRepo(ctx, testutil.WithRepoLogger(logger))
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		Expect(localRepo.Cleanup()).To(Succeed())
	})

	// Wrap local repo for Ginkgo-friendly error handling
	local := &LocalGitRepo{LocalRepo: localRepo}

	client, _, err := localRepo.QuickInit(user, repo.AuthURL)
	Expect(err).NotTo(HaveOccurred())

	// Wrap repo for backward compatibility
	remoteRepo := &RemoteRepo{Repo: repo}

	return client, remoteRepo, local, user
}

// generateRepoName generates a unique repository name
func generateRepoName() string {
	return fmt.Sprintf("testrepo-%d", GinkgoRandomSeed())
}

// ginkgoStructuredLogger wraps testutil.Logger to provide structured logging methods
type ginkgoStructuredLogger struct {
	logger testutil.Logger
}

func newGinkgoStructuredLogger(logger testutil.Logger) *ginkgoStructuredLogger {
	return &ginkgoStructuredLogger{logger: logger}
}

func (l *ginkgoStructuredLogger) Logf(format string, args ...any) {
	l.logger.Logf(format, args...)
}

func (l *ginkgoStructuredLogger) Debug(msg string, keysAndValues ...any) {
	l.log("DEBUG", msg, keysAndValues)
}

func (l *ginkgoStructuredLogger) Info(msg string, keysAndValues ...any) {
	l.log("INFO", msg, keysAndValues)
}

func (l *ginkgoStructuredLogger) Warn(msg string, keysAndValues ...any) {
	l.log("WARN", msg, keysAndValues)
}

func (l *ginkgoStructuredLogger) Error(msg string, keysAndValues ...any) {
	l.log("ERROR", msg, keysAndValues)
}

func (l *ginkgoStructuredLogger) Success(msg string, keysAndValues ...any) {
	l.log("SUCCESS", msg, keysAndValues)
}

func (l *ginkgoStructuredLogger) log(level, msg string, args []any) {
	formattedMsg := msg
	if len(args) > 0 {
		var pairs []string
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				pairs = append(pairs, fmt.Sprintf("%s=%v", args[i], args[i+1]))
			}
		}
		if len(pairs) > 0 {
			formattedMsg = msg + " (" + strings.Join(pairs, ", ") + ")"
		}
	}

	var emoji string
	switch level {
	case "DEBUG":
		emoji = "🔍"
	case "INFO":
		emoji = "ℹ️ "
	case "WARN":
		emoji = "⚠️ "
	case "ERROR":
		emoji = "❌"
	case "SUCCESS":
		emoji = "✅"
	}

	l.logger.Logf("%s [%s] %s", emoji, level, formattedMsg)
}
