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

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -header internal/tools/fake_header.txt -o mocks/client.go . Client

// Client is the interface for interacting with a single remote Git repository
// over the Smart HTTP protocol v2. It covers repository probes, reference
// management, object reads (blobs, trees, commits), history comparison,
// path-filtered cloning, and creating a StagedWriter for transactional writes.
//
// Instances are created with NewHTTPClient. All operations are stateless HTTP
// exchanges: nothing is written to the local filesystem unless explicitly
// requested (Clone), and no per-repository state outlives the client.
type Client interface {
	// CanRead reports whether the client can read from the repository, by
	// probing the git-upload-pack service with the configured credentials.
	CanRead(ctx context.Context) (bool, error)

	// CanWrite reports whether the client has repository-level write access,
	// by probing the git-receive-pack service. It cannot see branch-level
	// restrictions (e.g. protected branches); a push may still be rejected.
	CanWrite(ctx context.Context) (bool, error)

	// IsAuthorized reports whether the client can communicate with the
	// repository using the configured credentials.
	//
	// Deprecated: Use CanRead instead.
	IsAuthorized(ctx context.Context) (bool, error)

	// RepoExists reports whether the repository exists on the server, by
	// attempting to fetch its refs. It returns false (without error) when the
	// server answers with 404.
	RepoExists(ctx context.Context) (bool, error)

	// IsServerCompatible reports whether the server supports Git protocol v2,
	// which nanogit requires. It returns false for servers that only speak
	// protocol v1 (for example Azure DevOps).
	IsServerCompatible(ctx context.Context) (bool, error)

	// ListRefs retrieves all references (branches, tags, and others)
	// advertised by the remote repository, without downloading object data.
	ListRefs(ctx context.Context) ([]Ref, error)

	// GetRef retrieves a single reference by its fully qualified name, such
	// as "refs/heads/main" or "refs/tags/v1.0.0". It returns a
	// RefNotFoundError if the reference does not exist; short names like
	// "main" are not resolved.
	GetRef(ctx context.Context, refName string) (Ref, error)

	// CreateRef creates a new reference pointing at ref.Hash. The reference
	// must not already exist.
	CreateRef(ctx context.Context, ref Ref) error

	// UpdateRef moves an existing reference to point at ref.Hash. The
	// reference must already exist.
	UpdateRef(ctx context.Context, ref Ref) error

	// DeleteRef removes a reference from the remote repository. Only the
	// reference is removed, not the objects it pointed to.
	DeleteRef(ctx context.Context, refName string) error

	// GetBlob retrieves a blob (file content) by its object hash.
	GetBlob(ctx context.Context, hash hash.Hash) (*Blob, error)

	// GetBlobByPath retrieves a file by walking the tree hierarchy from
	// rootHash to the given slash-separated path, e.g. "docs/README.md".
	// Tip: pass Commit.Tree as the root.
	GetBlobByPath(ctx context.Context, rootHash hash.Hash, path string) (*Blob, error)

	// GetFlatTree retrieves a recursive listing of every file and directory
	// reachable from the given commit or tree hash, with each entry carrying
	// its full path from the repository root.
	GetFlatTree(ctx context.Context, hash hash.Hash) (*FlatTree, error)

	// GetTree retrieves a single tree object (one directory level) by its
	// hash.
	GetTree(ctx context.Context, hash hash.Hash) (*Tree, error)

	// GetTreeByPath retrieves the tree object for a directory at the given
	// slash-separated path, walking down from rootHash. The path "" or "."
	// returns the root tree itself.
	GetTreeByPath(ctx context.Context, rootHash hash.Hash, path string) (*Tree, error)

	// GetCommit retrieves a single commit object, including its author,
	// committer, message, parent hashes, and root tree hash.
	GetCommit(ctx context.Context, hash hash.Hash) (*Commit, error)

	// CompareCommits returns the file-level differences between two commits:
	// files added, modified, or deleted between baseCommit and headCommit,
	// sorted by path. Rename detection is enabled with WithRenameDetection.
	CompareCommits(ctx context.Context, baseCommit, headCommit hash.Hash, opts ...CompareCommitsOption) ([]CommitFile, error)

	// ListCommits walks the history backwards from startCommit and returns
	// the matching commits. ListCommitsOptions provides pagination (Page,
	// PerPage) and filtering (Path, Since, Until).
	ListCommits(ctx context.Context, startCommit hash.Hash, options ListCommitsOptions) ([]Commit, error)

	// Clone writes a snapshot of the repository at CloneOptions.Hash to a
	// local directory, optionally filtered to specific paths with glob
	// patterns. It fetches only the objects the filtered snapshot needs; it
	// does not create a .git directory or a working clone.
	Clone(ctx context.Context, opts CloneOptions) (*CloneResult, error)

	// NewStagedWriter creates a StagedWriter that stages changes on top of
	// the commit currently referenced by ref, to be committed and pushed as
	// one atomic update. WriterOption values choose where staged objects are
	// buffered (memory, disk, or automatic).
	NewStagedWriter(ctx context.Context, ref Ref, options ...WriterOption) (StagedWriter, error)
}

// httpClient is the private implementation of the Client interface.
// It implements the Git Smart Protocol version 2 over HTTP/HTTPS transport.
type httpClient struct {
	client.RawClient
	// receivePackCapabilities is advertised on receive-pack ref update commands.
	// When nil or empty, protocol.DefaultReceivePackCapabilities() is used.
	receivePackCapabilities []protocol.Capability
	// limits caps response bytes per operation class. The high-level
	// methods read these to populate FetchOptions.MaxResponseBytes per
	// call so the right cap (single-object vs multi-object) is applied.
	limits options.Limits
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
// The client implements the Git Smart Protocol version 2 over HTTP/HTTPS
// transport and is configured with functional options from the
// [github.com/grafana/nanogit/options] package (authentication, response
// limits, receive-pack capabilities, HTTP client customization).
//
// It returns an error if the URL is not a valid HTTP or HTTPS repository URL
// or if an option fails to apply.
//
// Example:
//
//	// Create a client with basic authentication.
//	client, err := nanogit.NewHTTPClient(
//	    "https://github.com/user/repo",
//	    options.WithBasicAuth("username", "password"),
//	)
//	if err != nil {
//	    return err
//	}
//
// Logging and retries are configured per call through the context; see
// [github.com/grafana/nanogit/log.ToContext] and
// [github.com/grafana/nanogit/retry.ToContext].
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
		limits:                  resolved.Limits,
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
