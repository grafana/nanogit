package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

type Client interface {
	// TODO(mem): this is probably not the right interface.
	SendCommands(ctx context.Context, data []byte) ([]byte, error)
	SmartInfoRequest(ctx context.Context) ([]byte, error)
}

type clientImpl struct {
	base      *url.URL
	client    *http.Client
	userAgent string

	basicAuth *struct{ Username, Password string }
	tokenAuth *string
}

func (ci *clientImpl) addDefaultHeaders(req *http.Request) {
	req.Header.Add("Git-Protocol", "version=2")
	if ci.userAgent == "" {
		ci.userAgent = "nanogit/0"
	}
	req.Header.Add("User-Agent", ci.userAgent)

	if ci.basicAuth != nil {
		req.SetBasicAuth(ci.basicAuth.Username, ci.basicAuth.Password)
	} else if ci.tokenAuth != nil {
		req.Header.Set("Authorization", *ci.tokenAuth)
	}
}

func (ci *clientImpl) SendCommands(ctx context.Context, data []byte) ([]byte, error) {
	body := bytes.NewReader(data)

	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/git-upload-pack.
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := ci.base.JoinPath("git-upload-pack").String()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return nil, err
	}

	ci.addDefaultHeaders(req)

	res, err := ci.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	return io.ReadAll(res.Body)
}

func (ci *clientImpl) SmartInfoRequest(ctx context.Context) ([]byte, error) {
	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/info/refs.
	// The ?service=git-upload-pack is documented in the protocol-v2 spec. It also implies elsewhere that ?svc is also valid.
	// See: https://git-scm.com/docs/http-protocol#_smart_clients
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := ci.base.JoinPath("info/refs")

	query := make(url.Values)
	query.Set("service", "git-upload-pack")
	u.RawQuery = query.Encode()

	slog.Info("SmartInfoRequest", "url", u.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	ci.addDefaultHeaders(req)

	res, err := ci.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	return io.ReadAll(res.Body)
}

type Option = func(*clientImpl) error

// New returns a new Client for the given repository.
func New(repo string, options ...Option) (*clientImpl, error) {
	u, err := url.Parse(repo)
	if err != nil {
		return nil, fmt.Errorf("parsing url: %w", err)
	}

	client := &clientImpl{
		base:   u,
		client: &http.Client{},
	}
	for _, option := range options {
		if option == nil { // allow for easy optional options
			continue
		}
		if err := option(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// WithBasicAuth sets the HTTP Basic Auth options.
// This is not a particularly secure method of authentication, so you probably want to recommend or require WithTokenAuth instead.
func WithBasicAuth(username, password string) Option {
	// NOTE: basic auth is defined as a valid authentication method by the http-protocol spec.
	// See: https://git-scm.com/docs/http-protocol#_authentication
	return func(ci *clientImpl) error {
		ci.basicAuth = &struct{ Username, Password string }{username, password}
		ci.tokenAuth = nil
		return nil
	}
}

// WithTokenAuth sets the Authorization header to the given token.
// We will not modify it for you. As such, if it needs a "Bearer" or "token" prefix, you must add that yourself.
func WithTokenAuth(token string) Option {
	// NOTE: auth beyond basic is defined as a valid authentication method by the http-protocol spec, if the server wants to implement it.
	// See: https://git-scm.com/docs/http-protocol#_authentication
	return func(ci *clientImpl) error {
		ci.basicAuth = nil
		ci.tokenAuth = &token
		return nil
	}
}

// WithUserAgent overrides the default User-Agent header.
func WithUserAgent(agent string) Option {
	return func(ci *clientImpl) error {
		ci.userAgent = agent
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
	return func(ci *clientImpl) error {
		if ci.tokenAuth != nil && !strings.HasPrefix(*ci.tokenAuth, "token ") {
			fixed := "token " + *ci.tokenAuth
			ci.tokenAuth = &fixed
		}
		ci.base.Path = strings.TrimRight(ci.base.Path, "/")
		ci.base.Path = strings.TrimSuffix(ci.base.Path, ".git")
		return nil
	}
}
