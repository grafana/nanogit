package integration_test

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/test/helpers"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite provides a shared Git server for integration tests
type IntegrationTestSuite struct {
	suite.Suite
	GitServer *helpers.GitServer
	Logger    *helpers.TestLogger
}

// SetupSuite runs once before all tests in the suite
func (s *IntegrationTestSuite) SetupSuite() {
	s.Logger = helpers.NewSuiteLogger(func() *testing.T { return s.T() })
	s.Logger.Info("🚀 Setting up integration test suite with shared Git server")
	s.GitServer = helpers.NewGitServer(func() *testing.T { return s.T() }, s.Logger)
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
}

// CreateTestRepo creates a fresh repository for a test with a unique name
func (s *IntegrationTestSuite) CreateTestRepo() (*helpers.RemoteRepo, *helpers.User) {
	user := s.GitServer.CreateUser()

	// Generate unique repo name
	var suffix uint32
	err := binary.Read(rand.Reader, binary.LittleEndian, &suffix)
	s.Require().NoError(err)
	suffix = suffix % 10000

	repoName := fmt.Sprintf("testrepo-%d", suffix)
	remote := s.GitServer.CreateRepo(repoName, user)

	return remote, user
}

// TestRepo is a convenience method that creates a test repo and returns client, remote, and local
func (s *IntegrationTestSuite) TestRepo() (nanogit.Client, *helpers.RemoteRepo, *helpers.LocalGitRepo) {
	return s.GitServer.TestRepo()
}

// Require returns a require.Assertions instance for the current test
func (s *IntegrationTestSuite) Require() *require.Assertions {
	return require.New(s.T())
}

// NoError asserts that the provided error is nil
func (s *IntegrationTestSuite) NoError(err error, msgAndArgs ...interface{}) {
	s.Require().NoError(err, msgAndArgs...)
}

// Error asserts that the provided error is not nil
func (s *IntegrationTestSuite) Error(err error, msgAndArgs ...interface{}) {
	s.Require().Error(err, msgAndArgs...)
}

// Equal asserts that two objects are equal
func (s *IntegrationTestSuite) Equal(expected, actual interface{}, msgAndArgs ...interface{}) {
	s.Require().Equal(expected, actual, msgAndArgs...)
}

// NotEqual asserts that two objects are not equal
func (s *IntegrationTestSuite) NotEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	s.Require().NotEqual(expected, actual, msgAndArgs...)
}

// True asserts that the specified value is true
func (s *IntegrationTestSuite) True(value bool, msgAndArgs ...interface{}) {
	s.Require().True(value, msgAndArgs...)
}

// False asserts that the specified value is false
func (s *IntegrationTestSuite) False(value bool, msgAndArgs ...interface{}) {
	s.Require().False(value, msgAndArgs...)
}

// NotNil asserts that the specified object is not nil
func (s *IntegrationTestSuite) NotNil(object interface{}, msgAndArgs ...interface{}) {
	s.Require().NotNil(object, msgAndArgs...)
}

// Nil asserts that the specified object is nil
func (s *IntegrationTestSuite) Nil(object interface{}, msgAndArgs ...interface{}) {
	s.Require().Nil(object, msgAndArgs...)
}

// Contains asserts that the specified string, list, or slice contains the specified substring
func (s *IntegrationTestSuite) Contains(s1, s2 interface{}, msgAndArgs ...interface{}) {
	s.Require().Contains(s1, s2, msgAndArgs...)
}

// NotEmpty asserts that the specified object is not empty
func (s *IntegrationTestSuite) NotEmpty(object interface{}, msgAndArgs ...interface{}) {
	s.Require().NotEmpty(object, msgAndArgs...)
}

// Empty asserts that the specified object is empty
func (s *IntegrationTestSuite) Empty(object interface{}, msgAndArgs ...interface{}) {
	s.Require().Empty(object, msgAndArgs...)
}

// Len asserts that the specified object has the specific length
func (s *IntegrationTestSuite) Len(object interface{}, length int, msgAndArgs ...interface{}) {
	s.Require().Len(object, length, msgAndArgs...)
}

// ErrorAs asserts that at least one of the errors in err's chain matches target
func (s *IntegrationTestSuite) ErrorAs(err error, target interface{}, msgAndArgs ...interface{}) {
	s.Require().ErrorAs(err, target, msgAndArgs...)
}

// ErrorIs asserts that at least one of the errors in err's chain matches target
func (s *IntegrationTestSuite) ErrorIs(err, target error, msgAndArgs ...interface{}) {
	s.Require().ErrorIs(err, target, msgAndArgs...)
}

// T returns the current test instance (for cases where you still need direct access)
func (s *IntegrationTestSuite) T() *testing.T {
	return s.Suite.T()
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
