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

	"github.com/grafana/nanogit/protocol"
)

// Client defines the interface for interacting with a Git repository.
type Client interface {
	// TODO(mem): this is probably not the right interface.
	SendCommands(ctx context.Context, data []byte) ([]byte, error)
	SmartInfoRequest(ctx context.Context) ([]byte, error)
	ListRefs(ctx context.Context) ([]Ref, error)
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

// SendCommands sends a POST request to the git-upload-pack endpoint.
func (c *clientImpl) SendCommands(ctx context.Context, data []byte) ([]byte, error) {
	body := bytes.NewReader(data)

	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/git-upload-pack.
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := c.base.JoinPath("git-upload-pack").String()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
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

	return io.ReadAll(res.Body)
}

// ListRefs sends a request to list all references in the repository.
// It returns a map of reference names to their commit hashes.
func (c *clientImpl) ListRefs(ctx context.Context) ([]Ref, error) {
	// First get the initial capability advertisement
	_, err := c.SmartInfoRequest(ctx)
	if err != nil {
		return nil, fmt.Errorf("get repository info: %w", err)
	}

	// Now send the ls-refs command
	pkt, err := protocol.FormatPacks(
		protocol.PackLine("command=ls-refs\n"),
		protocol.PackLine("object-format=sha1\n"),
		protocol.FlushPacket,
	)
	if err != nil {
		return nil, fmt.Errorf("format ls-refs command: %w", err)
	}

	refsData, err := c.SendCommands(ctx, pkt)
	if err != nil {
		return nil, fmt.Errorf("send ls-refs command: %w", err)
	}

	refs := make([]Ref, 0)
	lines, _, err := protocol.ParsePack(refsData)
	if err != nil {
		return nil, fmt.Errorf("parse refs response: %w", err)
	}

	for _, line := range lines {
		ref, hash, err := parseRefLine(line)
		if err != nil {
			return nil, fmt.Errorf("parse ref line: %w", err)
		}
		if ref != "" {
			refs = append(refs, Ref{Name: ref, Hash: hash})
		}
	}

	return refs, nil
}

// parseRefLine parses a single reference line from the git response.
// Returns the reference name, hash, and any error encountered.
func parseRefLine(line []byte) (ref, hash string, err error) {
	// Skip empty lines and pkt-line flush markers
	if len(line) == 0 || bytes.Equal(line, []byte("0000")) {
		return "", "", nil
	}

	// Skip capability lines (they start with =)
	if len(line) > 0 && line[0] == '=' {
		return "", "", nil
	}

	// Split into hash and rest
	parts := bytes.SplitN(line, []byte(" "), 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid ref format: %s", line)
	}

	// Ensure we have a full 40-character SHA-1 hash
	hash = string(parts[0])
	if len(hash) != 40 {
		return "", "", fmt.Errorf("invalid hash length: got %d, want 40", len(hash))
	}

	refName := string(parts[1])

	// Handle HEAD reference with capabilities
	if refName == "HEAD" {
		// Extract the symref value if present
		if idx := bytes.Index(line, []byte("symref=HEAD:")); idx > 0 {
			// Find the next space or end of line
			end := bytes.IndexByte(line[idx:], ' ')
			if end == -1 {
				end = len(line[idx:])
			}
			refName = string(line[idx+12 : idx+end])
		}
		return refName, hash, nil
	}

	// Remove capability suffix if present
	if idx := bytes.IndexByte(parts[1], '\x00'); idx > 0 {
		refName = string(parts[1][:idx])
	}

	return strings.TrimSpace(refName), strings.TrimSpace(hash), nil
}

// SmartInfoRequest sends a GET request to the info/refs endpoint.
func (c *clientImpl) SmartInfoRequest(ctx context.Context) ([]byte, error) {
	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/info/refs.
	// The ?service=git-upload-pack is documented in the protocol-v2 spec. It also implies elsewhere that ?svc is also valid.
	// See: https://git-scm.com/docs/http-protocol#_smart_clients
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := c.base.JoinPath("info/refs")

	query := make(url.Values)
	query.Set("service", "git-upload-pack")
	u.RawQuery = query.Encode()

	slog.Info("SmartInfoRequest", "url", u.String())

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

	return io.ReadAll(res.Body)
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
