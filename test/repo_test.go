package integration_test

import (
	"context"

	"github.com/grafana/nanogit"
)

// TestClient_RepoExists tests repository existence functionality
func (s *IntegrationTestSuite) TestRepoExists() {
	client, remote, _ := s.TestRepo()

	s.Run("existing repository", func() {
		exists, err := client.RepoExists(context.Background())
		s.NoError(err)
		s.True(exists)
	})

	s.Run("non-existent repository", func() {
		nonExistentClient, err := nanogit.NewHTTPClient(remote.URL()+"/nonexistent", nanogit.WithBasicAuth(remote.User.Username, remote.User.Password))
		s.NoError(err)

		exists, err := nonExistentClient.RepoExists(context.Background())
		s.NoError(err)
		s.False(exists)
	})

	s.Run("unauthorized access", func() {
		unauthorizedClient, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth("wronguser", "wrongpass"))
		s.NoError(err)

		exists, err := unauthorizedClient.RepoExists(context.Background())
		s.Error(err)
		s.Contains(err.Error(), "401 Unauthorized")
		s.False(exists)
	})
}
