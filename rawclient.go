package nanogit

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/storage"
)

type rawClient struct {
	// Base URL of the Git repository
	base *url.URL
	// HTTP client used for making requests
	client *http.Client
	// User-Agent header value for requests
	userAgent string
	// Logger for debug and info messages
	logger log.Logger
	// Basic authentication credentials (username/password)
	basicAuth *struct{ Username, Password string }
	// Token-based authentication header
	tokenAuth *string
	// Packfile storage
	packfileStorage storage.PackfileStorage
}

// addDefaultHeaders adds the default headers to the request.
func (c *rawClient) addDefaultHeaders(req *http.Request) {
	req.Header.Add("Git-Protocol", "version=2")
	if c.userAgent == "" {
		c.userAgent = "nanogit/0"
	}
	req.Header.Add("User-Agent", c.userAgent)

	if c.basicAuth != nil {
		req.SetBasicAuth(c.basicAuth.Username, c.basicAuth.Password)
	} else if c.tokenAuth != nil {
		req.Header.Set("Authorization", *c.tokenAuth)
	}
}

// WithUserAgent configures a custom User-Agent header for HTTP requests.
// If not specified, a default User-Agent will be used.
//
// Parameters:
//   - agent: Custom User-Agent string to use in requests
//
// Returns:
//   - Option: Configuration function for the client
func WithUserAgent(agent string) Option {
	return func(c *rawClient) error {
		c.userAgent = agent
		return nil
	}
}

// WithHTTPClient configures a custom HTTP client for making requests.
// This allows customization of timeouts, transport settings, proxies, and other HTTP behavior.
// The provided client must not be nil.
//
// Parameters:
//   - client: Custom HTTP client to use for requests
//
// Returns:
//   - Option: Configuration function for the client
//   - error: Error if the provided HTTP client is nil
func WithHTTPClient(client *http.Client) Option {
	return func(c *rawClient) error {
		if client == nil {
			return errors.New("httpClient is nil")
		}

		c.client = client
		return nil
	}
}
