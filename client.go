package nanogit

import (
	"context"
	"fmt"
	"sync"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/client"
	"github.com/grafana/nanogit/protocol/hash"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -header internal/tools/fake_header.txt -o mocks/staged_writer.go . StagedWriter

// StagedWriter provides a transactional interface for writing changes to Git objects.
// It allows staging multiple changes (file writes, updates, deletes) before committing them together.
// Changes are staged in memory and only sent to the server when Push() is called.
// This can be used to write to any Git object: commits, tags, branches, or other references.
type StagedWriter interface {
	// BlobExists checks if a blob exists at the given path.
	BlobExists(ctx context.Context, path string) (bool, error)

	// CreateBlob stages a new file to be written at the given path.
	// Returns the hash of the created blob.
	CreateBlob(ctx context.Context, path string, content []byte) (hash.Hash, error)

	// UpdateBlob stages an update to an existing file at the given path.
	// Returns the hash of the updated blob.
	UpdateBlob(ctx context.Context, path string, content []byte) (hash.Hash, error)

	// DeleteBlob stages the deletion of a file at the given path.
	// Returns the hash of the tree after deletion.
	DeleteBlob(ctx context.Context, path string) (hash.Hash, error)

	// MoveBlob stages the move of a file from srcPath to destPath.
	// This is equivalent to copying the file to the new location and deleting the original.
	// Returns the hash of the moved blob.
	MoveBlob(ctx context.Context, srcPath, destPath string) (hash.Hash, error)

	// GetTree gets the tree object at the given path.
	GetTree(ctx context.Context, path string) (*Tree, error)

	// DeleteTree stages the deletion of a directory and all its contents at the given path.
	// Returns the hash of the deleted tree.
	DeleteTree(ctx context.Context, path string) (hash.Hash, error)

	// MoveTree stages the move of a directory and all its contents from srcPath to destPath.
	// This recursively moves all files and subdirectories within the specified path.
	// Returns the hash of the moved tree.
	MoveTree(ctx context.Context, srcPath, destPath string) (hash.Hash, error)

	// Commit creates a new commit with all staged changes.
	// Returns the hash of the created commit.
	Commit(ctx context.Context, message string, author Author, committer Committer) (*Commit, error)

	// Push sends all committed changes to the remote repository.
	// This is the final step that makes changes visible to others.
	// It will update the reference to point to the last commit.
	Push(ctx context.Context) error

	// Cleanup releases any resources held by the writer and clears all staged changes.
	// This should be called when the writer is no longer needed or to cancel all pending changes.
	// After calling Cleanup, the writer should not be used for further operations.
	Cleanup(ctx context.Context) error
}

// Client defines the interface for interacting with a Git repository.
// It provides methods for repository operations, reference management,
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -header internal/tools/fake_header.txt -o mocks/client.go . Client
type Client interface {
	// Repo operations
	CanRead(ctx context.Context) (bool, error)
	CanWrite(ctx context.Context) (bool, error)
	IsAuthorized(ctx context.Context) (bool, error) // Deprecated: Use CanRead instead
	RepoExists(ctx context.Context) (bool, error)
	IsServerCompatible(ctx context.Context) (bool, error)
	// Ref operations
	ListRefs(ctx context.Context) ([]Ref, error)
	GetRef(ctx context.Context, refName string) (Ref, error)
	CreateRef(ctx context.Context, ref Ref) error
	UpdateRef(ctx context.Context, ref Ref) error
	DeleteRef(ctx context.Context, refName string) error
	// Blob operations
	GetBlob(ctx context.Context, hash hash.Hash) (*Blob, error)
	GetBlobByPath(ctx context.Context, rootHash hash.Hash, path string) (*Blob, error)
	// Tree operations
	GetFlatTree(ctx context.Context, hash hash.Hash) (*FlatTree, error)
	GetTree(ctx context.Context, hash hash.Hash) (*Tree, error)
	GetTreeByPath(ctx context.Context, rootHash hash.Hash, path string) (*Tree, error)
	// Commit operations
	GetCommit(ctx context.Context, hash hash.Hash) (*Commit, error)
	CompareCommits(ctx context.Context, baseCommit, headCommit hash.Hash, opts ...CompareCommitsOption) ([]CommitFile, error)
	ListCommits(ctx context.Context, startCommit hash.Hash, options ListCommitsOptions) ([]Commit, error)
	// Clone operations
	Clone(ctx context.Context, opts CloneOptions) (*CloneResult, error)
	// Write operations
	NewStagedWriter(ctx context.Context, ref Ref, options ...WriterOption) (StagedWriter, error)
}

// httpClient is the private implementation of the Client interface.
// It implements the Git Smart Protocol version 2 over HTTP/HTTPS transport.
type httpClient struct {
	client.RawClient
	// receivePackCapabilities is advertised on receive-pack ref update commands.
	// When nil or empty, protocol.DefaultReceivePackCapabilities() is used.
	receivePackCapabilities []protocol.Capability
	// negotiateCaps gates capability negotiation. See
	// options.WithCapabilityNegotiation.
	negotiateCaps bool
	// negotiateMu serializes the lazy fetch+intersect so concurrent first
	// callers don't race the network call. Once a successful negotiation
	// has populated negotiatedCaps, the fast path under negotiateMu's read
	// lock returns the cached value without extra round-trips. Failures
	// are NOT cached: a transient first-call error must not poison the
	// client for its lifetime.
	negotiateMu sync.RWMutex
	// negotiated is true only after a successful negotiation has populated
	// negotiatedCaps. It guards the fast path and ensures retry semantics
	// after a failed first call.
	negotiated bool
	// negotiatedCaps is the result of intersecting the desired client set
	// with the server's advertised set. Only safe to read while holding
	// negotiateMu (read or write) and only meaningful when negotiated.
	negotiatedCaps []protocol.Capability
}

// NewHTTPClient creates a new Git client for the specified repository URL.
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
//	client, err := nanogit.NewHTTPClient(
//	    "https://github.com/user/repo",
//	    options.WithBasicAuth("username", "password"),
//	    options.WithLogger(logger),
//	)
//	if err != nil {
//	    return err
//	}
func NewHTTPClient(repo string, opts ...options.Option) (Client, error) {
	// Resolve options once so both the raw transport and the higher-level
	// httpClient fields come from the same application of each Option.
	// Applying options twice would be safe for the pure options shipped with
	// nanogit but could misbehave for user-supplied options that observe or
	// mutate prior state.
	resolved, err := options.Resolve(opts...)
	if err != nil {
		return nil, err
	}

	rawClient, err := client.NewRawClientFromOptions(repo, resolved)
	if err != nil {
		return nil, err
	}

	return &httpClient{
		RawClient:               rawClient,
		receivePackCapabilities: resolved.ReceivePackCapabilities,
		negotiateCaps:           resolved.NegotiateCapabilities,
	}, nil
}

// effectiveReceivePackCapabilities returns the capabilities to advertise on
// receive-pack ref update commands. When negotiation is disabled this is just
// c.receivePackCapabilities, so the existing nil-slice → DefaultReceivePackCapabilities
// fallback in the protocol layer keeps working unchanged. When negotiation
// is enabled, the first successful call performs a single GET info/refs
// fetch and intersects the desired client set with the server's advertised
// set; the result is cached under c.negotiateMu so subsequent ref ops and
// writer resets reuse it without extra round-trips.
//
// Failures are not cached: a transient first-call error (network blip,
// context deadline) must not poison the client for its entire lifetime, so
// the next call retries the fetch from scratch. Once negotiation has
// succeeded the cached value is returned forever.
//
// Returns the negotiation error verbatim (wrapped) so the caller can
// surface it instead of silently falling back to the static set — silent
// fallback would hide server misconfiguration and contradict the explicit
// opt-in.
func (c *httpClient) effectiveReceivePackCapabilities(ctx context.Context) ([]protocol.Capability, error) {
	if !c.negotiateCaps {
		return c.receivePackCapabilities, nil
	}

	// Fast path: a previous call already negotiated successfully.
	c.negotiateMu.RLock()
	if c.negotiated {
		caps := c.negotiatedCaps
		c.negotiateMu.RUnlock()
		return caps, nil
	}
	c.negotiateMu.RUnlock()

	// Slow path: take the write lock and re-check, in case a concurrent
	// caller negotiated while we were waiting.
	c.negotiateMu.Lock()
	defer c.negotiateMu.Unlock()
	if c.negotiated {
		return c.negotiatedCaps, nil
	}

	logger := log.FromContext(ctx)
	logger.Debug("Negotiating receive-pack capabilities")

	serverCaps, err := c.FetchReceivePackCapabilities(ctx)
	if err != nil {
		// Do not cache failure: leave c.negotiated false so the next
		// caller retries the fetch instead of inheriting our error.
		return nil, fmt.Errorf("negotiate receive-pack capabilities: %w", err)
	}

	// The desired client set is whatever the user configured (via
	// WithReceivePackCapabilities) or the library defaults if they did
	// not. Resolve it once here so the intersection has a concrete list
	// to filter rather than carrying the nil-fallback through.
	desired := c.receivePackCapabilities
	if len(desired) == 0 {
		desired = protocol.DefaultReceivePackCapabilities()
	}

	c.negotiatedCaps = protocol.IntersectCapabilities(desired, serverCaps)
	c.negotiated = true
	logger.Debug("Receive-pack capabilities negotiated",
		"server_count", len(serverCaps),
		"client_count", len(desired),
		"intersected_count", len(c.negotiatedCaps))
	return c.negotiatedCaps, nil
}
