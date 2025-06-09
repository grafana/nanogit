//go:build integration

package integration_test

import (
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/test/helpers"
	"github.com/stretchr/testify/suite"
)

// AuthTestSuite contains tests for client authorization functionality
type AuthTestSuite struct {
	helpers.IntegrationTestSuite
}

// TestClient_IsAuthorized tests the authorization functionality
func (s *AuthTestSuite) TestClient_IsAuthorized() {
	s.Logger.Info("Setting up test repositories using shared Git server")
	_, remote, _ := s.TestRepo()
	user := remote.User

	s.Run("successful authorization", func() {
		s.T().Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(s.Logger))
		s.NoError(err)
		auth, err := client.IsAuthorized(ctx)
		s.NoError(err)
		s.True(auth)
	})

	s.Run("unauthorized access with wrong credentials", func() {
		s.T().Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		unauthorizedClient, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth("wronguser", "wrongpass"))
		s.NoError(err)
		auth, err := unauthorizedClient.IsAuthorized(ctx)
		s.NoError(err)
		s.False(auth)
	})

	s.Run("successful authorization with access token", func() {
		t := s.T()
		t.Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		token := s.GitServer.GenerateUserToken(t, user.Username, user.Password)
		client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithTokenAuth(token), nanogit.WithLogger(s.Logger))
		s.NoError(err)
		auth, err := client.IsAuthorized(ctx)
		s.NoError(err)
		s.True(auth)
	})

	s.Run("unauthorized access with invalid token", func() {
		s.T().Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		invalidToken := "token invalid-token"
		client, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithTokenAuth(invalidToken), nanogit.WithLogger(s.Logger))
		s.NoError(err)
		auth, err := client.IsAuthorized(ctx)
		s.NoError(err)
		s.False(auth)
	})
}

// TestAuthSuite runs the auth test suite
func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}
