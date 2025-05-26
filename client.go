package nanogit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/grafana/nanogit/protocol/hash"
)

// RefWriter provides a transactional interface for writing changes to Git objects.
// It allows staging multiple changes (file writes, updates, deletes) before committing them together.
// Changes are staged in memory and only sent to the server when Push() is called.
// This can be used to write to any Git object: commits, tags, branches, or other references.
type RefWriter interface {
	// CreateBlob stages a new file to be written at the given path.
	// Returns the hash of the created blob.
	CreateBlob(ctx context.Context, path string, content []byte) (hash.Hash, error)

	// UpdateBlob stages an update to an existing file at the given path.
	// Returns the hash of the updated blob.
	UpdateBlob(ctx context.Context, path string, content []byte) (hash.Hash, error)

	// DeleteBlob stages the deletion of a file at the given path.
	// Returns the hash of the tree after deletion.
	DeleteBlob(ctx context.Context, path string) (hash.Hash, error)

	// Commit creates a new commit with all staged changes.
	// Returns the hash of the created commit.
	Commit(ctx context.Context, message string, author Author, committer Committer) (*Commit, error)

	// Push sends all committed changes to the remote repository.
	// This is the final step that makes changes visible to others.
	// It will update the reference to point to the last commit.
	Push(ctx context.Context) error
}

// Client defines the interface for interacting with a Git repository.
type Client interface {
	// IsAuthorized checks if the client can successfully communicate with the Git server.
	IsAuthorized(ctx context.Context) (bool, error)
	// RepoExists checks if the repository exists on the server.
	RepoExists(ctx context.Context) (bool, error)
	// Ref operations
	ListRefs(ctx context.Context) ([]Ref, error)
	GetRef(ctx context.Context, refName string) (Ref, error)
	CreateRef(ctx context.Context, ref Ref) error
	UpdateRef(ctx context.Context, ref Ref) error
	DeleteRef(ctx context.Context, refName string) error
	NewRefWriter(ctx context.Context, ref Ref) (RefWriter, error)
	// Blob operations
	GetBlob(ctx context.Context, hash hash.Hash) ([]byte, error)
	GetBlobByPath(ctx context.Context, rootHash hash.Hash, path string) (*Blob, error)
	// Tree operations
	GetFlatTree(ctx context.Context, hash hash.Hash) (*FlatTree, error)
	GetTree(ctx context.Context, hash hash.Hash) (*Tree, error)
	GetTreeByPath(ctx context.Context, rootHash hash.Hash, path string) (*Tree, error)
	// File operations
	// Commit operations
	GetCommit(ctx context.Context, hash hash.Hash) (*Commit, error)
	CompareCommits(ctx context.Context, baseCommit, headCommit hash.Hash) ([]CommitFile, error)
}

// Option is a function that configures a Client.
type Option func(*clientImpl) error

// clientImpl is the private implementation of the Client interface.
type clientImpl struct {
	base      *url.URL
	client    *http.Client
	userAgent string
	logger    Logger

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

// uploadPack sends a POST request to the git-upload-pack endpoint.
// This endpoint is used to fetch objects and refs from the remote repository.
func (c *clientImpl) uploadPack(ctx context.Context, data []byte) ([]byte, error) {
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

// receivePack sends a POST request to the git-receive-pack endpoint.
// This endpoint is used to send objects to the remote repository.
func (c *clientImpl) receivePack(ctx context.Context, data []byte) ([]byte, error) {
	body := bytes.NewReader(data)

	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/git-receive-pack.
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := c.base.JoinPath("git-receive-pack")
	c.logger.Info("GitReceivePack", "url", u.String())

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

	c.logger.Info("ReceivePack", "status", res.StatusCode, "statusText", res.Status, "responseBody", string(responseBody), "requestBody", string(data), "url", u.String())

	return responseBody, nil
}

// smartInfo sends a GET request to the info/refs endpoint.
func (c *clientImpl) smartInfo(ctx context.Context, service string) ([]byte, error) {
	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/info/refs.
	// The ?service=git-upload-pack is documented in the protocol-v2 spec. It also implies elsewhere that ?svc is also valid.
	// See: https://git-scm.com/docs/http-protocol#_smart_clients
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := c.base.JoinPath("info/refs")

	query := make(url.Values)
	query.Set("service", service)
	u.RawQuery = query.Encode()

	c.logger.Info("SmartInfo", "url", u.String())

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

	c.logger.Info("SmartInfo", "status", res.StatusCode, "statusText", res.Status, "body", string(body))

	return body, nil
}

// NewClient returns a new Client for the given repository.
func NewClient(repo string, options ...Option) (Client, error) {
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
	u.Path = strings.TrimSuffix(u.Path, ".git")

	c := &clientImpl{
		base:   u,
		client: &http.Client{},
		logger: &noopLogger{}, // No-op logger by default
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
