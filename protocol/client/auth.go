package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// WithBasicAuth sets the HTTP Basic Auth options.
// This is not a particularly secure method of authentication, so you probably want to recommend or require WithTokenAuth instead.
func WithBasicAuth(username, password string) Option {
	// NOTE: basic auth is defined as a valid authentication method by the http-protocol spec.
	// See: https://git-scm.com/docs/http-protocol#_authentication
	return func(c *rawClient) error {
		if username == "" {
			return errors.New("username cannot be empty")
		}
		if c.tokenAuth != nil {
			return errors.New("cannot use both basic auth and token auth")
		}
		c.basicAuth = &struct{ Username, Password string }{username, password}
		return nil
	}
}

// WithTokenAuth sets the Authorization header to the given token.
// We will not modify it for you. As such, if it needs a "Bearer" or "token" prefix, you must add that yourself.
func WithTokenAuth(token string) Option {
	// NOTE: auth beyond basic is defined as a valid authentication method by the http-protocol spec, if the server wants to implement it.
	// See: https://git-scm.com/docs/http-protocol#_authentication
	return func(c *rawClient) error {
		if token == "" {
			return errors.New("token cannot be empty")
		}
		if c.basicAuth != nil {
			return errors.New("cannot use both basic auth and token auth")
		}
		c.tokenAuth = &token
		return nil
	}
}

// IsAuthorized checks if the client can successfully communicate with the Git server.
// It performs a basic connectivity test by attempting to fetch the server's capabilities
// through the git-upload-pack service.
//
// Returns:
//   - true if the server is reachable and the client is authorized
//   - false if the server returns a 401 Unauthorized response
//   - error if there are any other connection or protocol issues
func (c *rawClient) IsAuthorized(ctx context.Context) (bool, error) {
	// First get the initial capability advertisement
	_, err := c.SmartInfo(ctx, "git-upload-pack")
	if err != nil {
		if strings.Contains(err.Error(), "401 Unauthorized") {
			return false, nil
		}
		return false, fmt.Errorf("get repository info: %w", err)
	}

	return true, nil
}
