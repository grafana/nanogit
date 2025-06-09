package helpers

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite provides a shared Git server for integration tests
type IntegrationTestSuite struct {
	suite.Suite
	GitServer *GitServer
	Logger    *TestLogger
}

// SetupSuite runs once before all tests in the suite
func (s *IntegrationTestSuite) SetupSuite() {
	s.Logger = NewSuiteLogger(func() *testing.T { return s.T() })
	s.Logger.Info("🚀 Setting up integration test suite with shared Git server")
	s.GitServer = NewGitServer(s.T(), s.Logger)
	s.Logger.Success("✅ Integration test suite setup complete")
}

// TearDownSuite runs once after all tests in the suite
func (s *IntegrationTestSuite) TearDownSuite() {
	s.Logger.Info("🧹 Tearing down integration test suite")
	// The GitServer cleanup is handled by the testcontainer lifecycle
	s.Logger.Success("✅ Integration test suite teardown complete")
}

// SetupTest runs before each test method
func (s *IntegrationTestSuite) SetupTest() {
	// Update logger for the current test
	s.Logger.ForSubtest(s.T())
}

// CreateTestRepo creates a fresh repository for a test with a unique name
func (s *IntegrationTestSuite) CreateTestRepo() (*RemoteRepo, *User) {
	user := s.GitServer.CreateUser(s.T())

	// Generate unique repo name
	var suffix uint32
	err := binary.Read(rand.Reader, binary.LittleEndian, &suffix)
	require.NoError(s.T(), err)
	suffix = suffix % 10000

	repoName := fmt.Sprintf("testrepo-%d", suffix)
	remote := s.GitServer.CreateRepo(s.T(), repoName, user)

	return remote, user
}

// QuickSetup provides the common setup pattern used by many tests
func (s *IntegrationTestSuite) QuickSetup() (*LocalGitRepo, nanogit.Client, string, *RemoteRepo, *User) {
	s.Logger.Info("🔧 Setting up test with fresh repository")
	remote, user := s.CreateTestRepo()
	local := NewLocalGitRepo(s.T(), s.Logger)
	client, initCommitFile := local.QuickInit(s.T(), user, remote.AuthURL())

	s.Logger.Success("✅ Test setup complete")
	return local, client, initCommitFile, remote, user
}

// QuickSetupWithContext provides common setup with a timeout context
func (s *IntegrationTestSuite) QuickSetupWithContext(timeout time.Duration) (context.Context, context.CancelFunc, *LocalGitRepo, nanogit.Client, string, *RemoteRepo, *User) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	local, client, initCommitFile, remote, user := s.QuickSetup()
	return ctx, cancel, local, client, initCommitFile, remote, user
}

// CreateTestRepoWithCommits creates a repository with some initial commits for testing
func (s *IntegrationTestSuite) CreateTestRepoWithCommits(numCommits int) (*LocalGitRepo, nanogit.Client, *RemoteRepo, *User) {
	remote, user := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create initial commits
	for i := 0; i < numCommits; i++ {
		filename := fmt.Sprintf("file%d.txt", i+1)
		content := fmt.Sprintf("Content for file %d", i+1)
		local.CreateFile(s.T(), filename, content)
		local.Git(s.T(), "add", filename)
		local.Git(s.T(), "commit", "-m", fmt.Sprintf("Add %s", filename))
	}

	local.Git(s.T(), "push")

	client := remote.Client(s.T())
	return local, client, remote, user
}

// TestRepo is a convenience method that creates a test repo and returns client, remote, and local
func (s *IntegrationTestSuite) TestRepo() (nanogit.Client, *RemoteRepo, *LocalGitRepo) {
	return s.GitServer.TestRepo(s.T())
}

// CreateContext creates a context with timeout for tests
func (s *IntegrationTestSuite) CreateContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	s.T().Cleanup(cancel)
	return ctx, cancel
}

// StandardTimeout returns a standard timeout duration for integration tests
func (s *IntegrationTestSuite) StandardTimeout() time.Duration {
	return 30 * time.Second
}

// Convenience methods to avoid having to use s.T() everywhere

// Require returns a require.Assertions instance for the current test
func (s *IntegrationTestSuite) Require() *require.Assertions {
	return require.New(s.T())
}

// NoError asserts that the provided error is nil
func (s *IntegrationTestSuite) NoError(err error, msgAndArgs ...interface{}) {
	require.NoError(s.T(), err, msgAndArgs...)
}

// Error asserts that the provided error is not nil
func (s *IntegrationTestSuite) Error(err error, msgAndArgs ...interface{}) {
	require.Error(s.T(), err, msgAndArgs...)
}

// Equal asserts that two objects are equal
func (s *IntegrationTestSuite) Equal(expected, actual interface{}, msgAndArgs ...interface{}) {
	require.Equal(s.T(), expected, actual, msgAndArgs...)
}

// NotEqual asserts that two objects are not equal
func (s *IntegrationTestSuite) NotEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	require.NotEqual(s.T(), expected, actual, msgAndArgs...)
}

// True asserts that the specified value is true
func (s *IntegrationTestSuite) True(value bool, msgAndArgs ...interface{}) {
	require.True(s.T(), value, msgAndArgs...)
}

// False asserts that the specified value is false
func (s *IntegrationTestSuite) False(value bool, msgAndArgs ...interface{}) {
	require.False(s.T(), value, msgAndArgs...)
}

// NotNil asserts that the specified object is not nil
func (s *IntegrationTestSuite) NotNil(object interface{}, msgAndArgs ...interface{}) {
	require.NotNil(s.T(), object, msgAndArgs...)
}

// Nil asserts that the specified object is nil
func (s *IntegrationTestSuite) Nil(object interface{}, msgAndArgs ...interface{}) {
	require.Nil(s.T(), object, msgAndArgs...)
}

// Contains asserts that the specified string, list, or slice contains the specified substring
func (s *IntegrationTestSuite) Contains(s1, s2 interface{}, msgAndArgs ...interface{}) {
	require.Contains(s.T(), s1, s2, msgAndArgs...)
}

// NotEmpty asserts that the specified object is not empty
func (s *IntegrationTestSuite) NotEmpty(object interface{}, msgAndArgs ...interface{}) {
	require.NotEmpty(s.T(), object, msgAndArgs...)
}

// Empty asserts that the specified object is empty
func (s *IntegrationTestSuite) Empty(object interface{}, msgAndArgs ...interface{}) {
	require.Empty(s.T(), object, msgAndArgs...)
}

// Len asserts that the specified object has the specific length
func (s *IntegrationTestSuite) Len(object interface{}, length int, msgAndArgs ...interface{}) {
	require.Len(s.T(), object, length, msgAndArgs...)
}

// ErrorAs asserts that at least one of the errors in err's chain matches target
func (s *IntegrationTestSuite) ErrorAs(err error, target interface{}, msgAndArgs ...interface{}) {
	require.ErrorAs(s.T(), err, target, msgAndArgs...)
}

// T returns the current test instance (for cases where you still need direct access)
func (s *IntegrationTestSuite) T() *testing.T {
	return s.Suite.T()
}
