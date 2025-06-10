package integration_test

import (
	"context"

	"github.com/grafana/nanogit"
)

// TestClient_IsAuthorized tests the authorization functionality
func (s *IntegrationTestSuite) TestIsAuthorized() {
	s.Logger.Info("Setting up test repositories using shared Git server")
	_, remote, _ := s.TestRepo()
	user := remote.User

	s.Run("successful authorization", func() {
		s.T().Parallel()

		client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(s.Logger))
		s.NoError(err)
		auth, err := client.IsAuthorized(context.Background())
		s.NoError(err)
		s.True(auth)
	})

	s.Run("unauthorized access with wrong credentials", func() {
		s.T().Parallel()

		unauthorizedClient, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth("wronguser", "wrongpass"))
		s.NoError(err)
		auth, err := unauthorizedClient.IsAuthorized(context.Background())
		s.NoError(err)
		s.False(auth)
	})

	s.Run("successful authorization with access token", func() {
		s.T().Parallel()

		token := s.GitServer.GenerateUserToken(user.Username, user.Password)
		client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithTokenAuth(token), nanogit.WithLogger(s.Logger))
		s.NoError(err)
		auth, err := client.IsAuthorized(context.Background())
		s.NoError(err)
		s.True(auth)
	})

	s.Run("unauthorized access with invalid token", func() {
		s.T().Parallel()

		invalidToken := "token invalid-token"
		client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithTokenAuth(invalidToken), nanogit.WithLogger(s.Logger))
		s.NoError(err)
		auth, err := client.IsAuthorized(context.Background())
		s.NoError(err)
		s.False(auth)
	})
}
