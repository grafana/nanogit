package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Option func(*RawClient) error

type RawClient struct {
	// Base URL of the Git repository
	base *url.URL
	// HTTP client used for making requests
	client *http.Client
	// User-Agent header value for requests
	userAgent string
	// Basic authentication credentials (username/password)
	basicAuth *struct{ Username, Password string }
	// Token-based authentication header
	tokenAuth *string
}

// NewRawClient creates a new Git client for the specified repository URL.
// The client implements the Git Smart Protocol version 2 over HTTP/HTTPS transport.
// It supports both HTTP and HTTPS URLs and can be configured with various options
// for authentication, logging, and HTTP client customization.
//
// Parameters:
//   - repo: Repository URL (must be HTTP or HTTPS)
//   - options: Configuration options for authentication, logging, etc.
//
// Returns:
//   - Client: Configured Git client interface
//   - error: Error if URL is invalid or configuration fails
//
// Example:
//
//	// Create client with basic authentication
//	client, err := nanogit.NewRawClient(
//	    "https://github.com/user/repo",
//	    nanogit.WithBasicAuth("username", "password"),
//	    nanogit.WithLogger(logger),
//	)
//	if err != nil {
//	    return err
//	}
func NewRawClient(repo string, options ...Option) (*RawClient, error) {
	if repo == "" {
		return nil, errors.New("repository URL cannot be empty")
	}

	u, err := url.Parse(repo)
	if err != nil {
		return nil, fmt.Errorf("parsing url: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("only HTTP and HTTPS URLs are supported")
	}

	u.Path = strings.TrimRight(u.Path, "/")

	c := &RawClient{
		base:   u,
		client: &http.Client{},
	}

	for _, option := range options {
		if option == nil { // allow for easy optional options
			continue
		}
		if err := option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// addDefaultHeaders adds the default headers to the request.
func (c *RawClient) addDefaultHeaders(req *http.Request) {
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
	return func(c *RawClient) error {
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
	return func(c *RawClient) error {
		if client == nil {
			return errors.New("httpClient is nil")
		}

		c.client = client
		return nil
	}
}
