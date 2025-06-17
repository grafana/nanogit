package nanogit_test

import (
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
	"github.com/stretchr/testify/require"
)

// TestNewHTTPClient_Success tests that NewHTTPClient returns a valid client for a well-formed repo URL.
func TestNewHTTPClient_Success(t *testing.T) {
	// This is a dummy URL; in real tests, use a test server or mock.
	repoURL := "https://example.com/user/repo.git"
	client, err := nanogit.NewHTTPClient(repoURL)
	require.NoError(t, err)
	require.NotNil(t, client)
}

// TestNewHTTPClient_InvalidURL tests that NewHTTPClient returns an error for an invalid repo URL.
func TestNewHTTPClient_InvalidURL(t *testing.T) {
	invalidURL := "not-a-valid-url"
	client, err := nanogit.NewHTTPClient(invalidURL)
	require.Error(t, err)
	require.Nil(t, client)
}

// TestNewHTTPClient_WithOptions tests that NewHTTPClient accepts options and returns a client.
func TestNewHTTPClient_WithOptions(t *testing.T) {
	repoURL := "https://example.com/user/repo.git"
	client, err := nanogit.NewHTTPClient(repoURL, options.WithUserAgent("nanogit-test"))
	require.NoError(t, err)
	require.NotNil(t, client)
}
