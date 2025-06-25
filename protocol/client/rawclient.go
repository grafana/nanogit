package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
)

// RawClient is a client that can be used to make raw Git protocol requests.
// It is used to implement the Git Smart Protocol version 2 over HTTP/HTTPS transport.
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o ../../mocks/raw_client.go . RawClient
type RawClient interface {
	IsAuthorized(ctx context.Context) (bool, error)
	SmartInfo(ctx context.Context, service string) (io.ReadCloser, error)
	UploadPack(ctx context.Context, data io.Reader) (io.ReadCloser, error)
	ReceivePack(ctx context.Context, data io.Reader) (io.ReadCloser, error)
	Fetch(ctx context.Context, opts FetchOptions) (map[string]*protocol.PackfileObject, error)
	LsRefs(ctx context.Context, opts LsRefsOptions) ([]protocol.RefLine, error)
}

type rawClient struct {
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
//	client, err := client.NewHTTPClient(
//	    "https://github.com/user/repo",
//	    options.WithBasicAuth("username", "password"),
//	    options.WithLogger(logger),
//	)
//	if err != nil {
//	    return err
//	}
func NewRawClient(repo string, opts ...options.Option) (*rawClient, error) {
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

	options := &options.Options{
		HTTPClient: &http.Client{},
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if err := opt(options); err != nil {
			return nil, err
		}
	}

	var basicAuth *struct{ Username, Password string }
	if options.BasicAuth != nil {
		basicAuth = &struct {
			Username string
			Password string
		}{
			Username: options.BasicAuth.Username,
			Password: options.BasicAuth.Password,
		}
	}

	c := &rawClient{
		base:      u,
		client:    options.HTTPClient,
		userAgent: options.UserAgent,
		basicAuth: basicAuth,
		tokenAuth: options.AuthToken,
	}

	return c, nil
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
