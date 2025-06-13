package nanogit

import (
	"net/http"

	"github.com/grafana/nanogit/protocol/client"
)

// Option is a function that configures a Client during creation.
// Options allow customization of the HTTP client, authentication, logging, and other settings.
type Option client.Option

// WithBasicAuth sets the HTTP Basic Auth options.
// This is not a particularly secure method of authentication, so you probably want to recommend or require WithTokenAuth instead.
func WithBasicAuth(username, password string) Option {
	return func(c *client.RawClient) error {
		return client.WithBasicAuth(username, password)(c)
	}
}

// WithTokenAuth sets the Authorization header to the given token.
// We will not modify it for you. As such, if it needs a "Bearer" or "token" prefix, you must add that yourself.
func WithTokenAuth(token string) Option {
	return func(c *client.RawClient) error {
		return client.WithTokenAuth(token)(c)
	}
}

// WithHTTPClient sets a custom HTTP client to use for requests.
// This allows customization of timeouts, transport, and other HTTP client settings.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *client.RawClient) error {
		return client.WithHTTPClient(httpClient)(c)
	}
}
