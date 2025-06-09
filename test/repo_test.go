package integration_test

import (
	"github.com/grafana/nanogit"
)

// TestClient_RepoExists tests repository existence functionality
func (s *IntegrationTestSuite) TestClient_RepoExists() {
	s.Logger.Info("Setting up test repositories using shared Git server")
	client, remote, _ := s.TestRepo()

	s.Run("existing repository", func() {
		s.T().Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		exists, err := client.RepoExists(ctx)
		s.NoError(err)
		s.True(exists)
	})

	s.Run("non-existent repository", func() {
		s.T().Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		nonExistentClient, err := nanogit.NewHTTPClient(remote.URL()+"/nonexistent", nanogit.WithBasicAuth(remote.User.Username, remote.User.Password))
		s.NoError(err)

		exists, err := nonExistentClient.RepoExists(ctx)
		s.NoError(err)
		s.False(exists)
	})

	s.Run("unauthorized access", func() {
		s.T().Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		unauthorizedClient, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth("wronguser", "wrongpass"))
		s.NoError(err)

		exists, err := unauthorizedClient.RepoExists(ctx)
		s.Error(err)
		s.Contains(err.Error(), "401 Unauthorized")
		s.False(exists)
	})
}
