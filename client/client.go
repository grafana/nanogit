package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// Client defines the interface for interacting with a Git repository.
type Client interface {
	// TODO: this is probably not the right interface, we may have to separate the client
	// Raw protocol requests
	// UploadPack sends a POST request to the git-upload-pack endpoint.
	UploadPack(ctx context.Context, data []byte) ([]byte, error)
	// ReceivePack sends a POST request to the git-receive-pack endpoint.
	ReceivePack(ctx context.Context, data []byte) ([]byte, error)
	// SmartInfo sends a GET request to the info/refs endpoint.
	SmartInfo(ctx context.Context, service string) ([]byte, error)

	// TODO: is this a good signature?
	// Ref operations
	ListRefs(ctx context.Context) ([]Ref, error)
	GetRef(ctx context.Context, refName string) (Ref, error)
	CreateRef(ctx context.Context, ref Ref) error
	UpdateRef(ctx context.Context, ref Ref) error
	DeleteRef(ctx context.Context, refName string) error
}

// Option is a function that configures a Client.
type Option func(*clientImpl) error

// clientImpl is the private implementation of the Client interface.
type clientImpl struct {
	base      *url.URL
	client    *http.Client
	userAgent string

	basicAuth *struct{ Username, Password string }
	tokenAuth *string
}

// addDefaultHeaders adds the default headers to the request.
func (c *clientImpl) addDefaultHeaders(req *http.Request) {
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

// UploadPack sends a POST request to the git-upload-pack endpoint.
// This endpoint is used to fetch objects and refs from the remote repository.
func (c *clientImpl) UploadPack(ctx context.Context, data []byte) ([]byte, error) {
	body := bytes.NewReader(data)

	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/git-upload-pack.
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := c.base.JoinPath("git-upload-pack").String()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-git-upload-pack-request")
	c.addDefaultHeaders(req)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	return io.ReadAll(res.Body)
}

// ReceivePack sends a POST request to the git-receive-pack endpoint.
// This endpoint is used to send objects to the remote repository.
func (c *clientImpl) ReceivePack(ctx context.Context, data []byte) ([]byte, error) {
	body := bytes.NewReader(data)

	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/git-receive-pack.
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := c.base.JoinPath("git-receive-pack")
	slog.Info("GitReceivePack", "url", u.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), body)
	if err != nil {
		return nil, err
	}

	c.addDefaultHeaders(req)
	req.Header.Add("Content-Type", "application/x-git-receive-pack-request")
	req.Header.Add("Accept", "application/x-git-receive-pack-result")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	slog.Info("ReceivePack", "status", res.StatusCode, "statusText", res.Status, "responseBody", string(responseBody), "requestBody", string(data), "url", u.String())

	return responseBody, nil
}

// SmartInfo sends a GET request to the info/refs endpoint.
func (c *clientImpl) SmartInfo(ctx context.Context, service string) ([]byte, error) {
	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/info/refs.
	// The ?service=git-upload-pack is documented in the protocol-v2 spec. It also implies elsewhere that ?svc is also valid.
	// See: https://git-scm.com/docs/http-protocol#_smart_clients
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := c.base.JoinPath("info/refs")

	query := make(url.Values)
	query.Set("service", service)
	u.RawQuery = query.Encode()

	slog.Info("SmartInfo", "url", u.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	c.addDefaultHeaders(req)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	slog.Info("SmartInfo", "status", res.StatusCode, "statusText", res.Status, "body", string(body))

	return body, nil
}

// New returns a new Client for the given repository.
func New(repo string, options ...Option) (Client, error) {
	u, err := url.Parse(repo)
	if err != nil {
		return nil, fmt.Errorf("parsing url: %w", err)
	}

	c := &clientImpl{
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

// WithBasicAuth sets the HTTP Basic Auth options.
// This is not a particularly secure method of authentication, so you probably want to recommend or require WithTokenAuth instead.
func WithBasicAuth(username, password string) Option {
	// NOTE: basic auth is defined as a valid authentication method by the http-protocol spec.
	// See: https://git-scm.com/docs/http-protocol#_authentication
	return func(c *clientImpl) error {
		c.basicAuth = &struct{ Username, Password string }{username, password}
		c.tokenAuth = nil
		return nil
	}
}

// WithTokenAuth sets the Authorization header to the given token.
// We will not modify it for you. As such, if it needs a "Bearer" or "token" prefix, you must add that yourself.
func WithTokenAuth(token string) Option {
	// NOTE: auth beyond basic is defined as a valid authentication method by the http-protocol spec, if the server wants to implement it.
	// See: https://git-scm.com/docs/http-protocol#_authentication
	return func(c *clientImpl) error {
		c.basicAuth = nil
		c.tokenAuth = &token
		return nil
	}
}

// WithUserAgent overrides the default User-Agent header.
func WithUserAgent(agent string) Option {
	return func(c *clientImpl) error {
		c.userAgent = agent
		return nil
	}
}

// WithHTTPClient overrides the default http.Client.
// It will return an error if the provided http.Client is nil.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *clientImpl) error {
		if httpClient == nil {
			return errors.New("httpClient is nil")
		}

		c.client = httpClient
		return nil
	}
}

// WithGitHub verifies the other options are valid, and modifies those that aren't.
// This must be the final option, if you wish to apply it.
// WithGitHub is valid with both GitHub.com and a GitHub Enterprise Server.
//
// What does that entail?:
//   - Check that the token auth header has a "token" prefix, if it is used.
//   - Check that the base URL has no ".git" suffix, or trailing slashes.
func WithGitHub() Option {
	return func(c *clientImpl) error {
		if c.tokenAuth != nil && !strings.HasPrefix(*c.tokenAuth, "token ") {
			fixed := "token " + *c.tokenAuth
			c.tokenAuth = &fixed
		}
		c.base.Path = strings.TrimRight(c.base.Path, "/")
		c.base.Path = strings.TrimSuffix(c.base.Path, ".git")
		return nil
	}
}
