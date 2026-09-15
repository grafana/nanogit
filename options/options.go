// Package options configures the nanogit HTTP client. It defines the
// functional options passed to nanogit.NewHTTPClient: authentication
// (WithBasicAuth, WithTokenAuth), user agent, custom HTTP transport,
// response size limits, and receive-pack capability control.
package options

import (
	"net/http"

	"github.com/grafana/nanogit/protocol"
)

// defaultHTTPClient returns the zero-config HTTP client used when callers do
// not provide one. It is seeded into Options before Option callbacks run so
// options like WithHTTPClient(...) can replace it and options that tweak
// default fields (e.g., o.HTTPClient.Timeout) do not dereference nil.
func defaultHTTPClient() *http.Client {
	return &http.Client{}
}

// BasicAuth holds credentials for HTTP basic authentication.
type BasicAuth struct {
	// Username is the basic auth username.
	Username string
	// Password is the basic auth password or personal access token.
	Password string
}

// Options is the resolved client configuration produced by applying Option
// functions (see Resolve). Most callers never build it directly; they pass
// Option values to nanogit.NewHTTPClient instead.
type Options struct {
	// HTTPClient performs the underlying HTTP requests.
	HTTPClient *http.Client
	// UserAgent overrides the User-Agent header sent with each request.
	UserAgent string
	// BasicAuth holds basic authentication credentials, if set.
	BasicAuth *BasicAuth
	// AuthToken is the raw Authorization header value, if set.
	AuthToken *string
	// SkipGitSuffix disables appending ".git" to the repository URL path.
	SkipGitSuffix bool
	// ReceivePackCapabilities, when non-empty, overrides the capabilities
	// advertised on receive-pack ref update commands. When nil or empty,
	// protocol.DefaultReceivePackCapabilities() is used.
	ReceivePackCapabilities []protocol.Capability
	// Limits caps the bytes nanogit will read from the server, classified by
	// operation. The zero value disables the four wire caps (embedders that
	// don't opt in keep today's unbounded per-response behavior), but decoded-
	// object protection stays on: MaxObjectDecodedBytes falls back to nanogit's
	// built-in default rather than becoming unlimited. See Limits for details.
	Limits Limits
	// NegotiateCapabilities, when true, makes the client fetch the server's
	// receive-pack capability advertisement once per client lifetime and
	// advertise the intersection with its desired set on subsequent ref
	// updates. Default is false (no behavior change).
	NegotiateCapabilities bool
}

// Limits caps how much data nanogit will read from the server, broken down by
// operation class. The four *MaxBytes fields cap the bytes read from a single
// HTTP response (wire bytes); a zero value for any of them means "no limit",
// so the zero value of those four preserves nanogit's historic behavior. Three
// of them — SingleObjectFetchMaxBytes, MultiObjectFetchMaxBytes, and
// RefsMetadataMaxBytes — are read-side and govern the git-upload-pack endpoint
// (which carries both the fetch and ls-refs commands in protocol v2); they are
// split by operation rather than by endpoint because their expected response
// sizes differ by orders of magnitude. ReceivePackResponseMaxBytes is the lone
// write-side cap and governs git-receive-pack.
//
// MaxObjectDecodedBytes is the exception: it bounds decoded (post-inflation)
// object size rather than wire bytes, and a zero value does NOT disable it —
// nanogit's built-in default stays in force (see the field docs). This keeps
// decompression-bomb protection always on.
//
// Negative values are rejected at construction time (see WithLimits).
type Limits struct {
	// SingleObjectFetchMaxBytes caps the git-upload-pack response for
	// fetches that target a single object (GetBlob, GetTree, GetCommit, ...).
	SingleObjectFetchMaxBytes int64
	// MultiObjectFetchMaxBytes caps the git-upload-pack response for
	// fetches that may return many objects (GetFlatTree, ListCommits,
	// CompareCommits, Clone).
	MultiObjectFetchMaxBytes int64
	// RefsMetadataMaxBytes caps ref-listing and protocol-detection
	// responses, which also ride git-upload-pack (ls-refs command) and the
	// smart-info / capability advertisement (ListRefs, GetRef).
	RefsMetadataMaxBytes int64
	// ReceivePackResponseMaxBytes caps the git-receive-pack reply to a
	// push (CreateRef, UpdateRef, DeleteRef, staged Push).
	ReceivePackResponseMaxBytes int64
	// MaxObjectDecodedBytes caps the *decoded* (inflated) size of any single
	// object read from a packfile. Unlike the wire caps above (which bound
	// the compressed git-upload-pack response), this bounds post-decompression
	// memory: an object's declared decoded size is checked before it is
	// allocated, so a small, highly compressible payload that would inflate to
	// gigabytes is rejected up front. This is what defeats decompression bombs.
	//
	// Unlike the wire caps, a zero value does NOT disable the check: it leaves
	// nanogit's built-in default (protocol.MaxUnpackedObjectSize) in place, so
	// decoded-size protection is always on. Set a positive value to raise or
	// lower that ceiling; oversized objects surface as *protocol.ObjectTooLargeError
	// (which wraps protocol.ErrObjectTooLarge).
	MaxObjectDecodedBytes int64
}

// Option mutates Options during Resolve. An Option returns an error to
// reject invalid configuration at client construction time.
type Option func(*Options) error

// Resolve applies the given Option functions in order to a fresh Options
// struct seeded with a default HTTPClient so options that read or mutate
// o.HTTPClient in place do not dereference nil. Callers may safely resolve
// the same slice more than once to re-derive the resolved state; nil options
// are ignored and the first option returning an error short-circuits.
func Resolve(opts ...Option) (*Options, error) {
	resolved := &Options{HTTPClient: defaultHTTPClient()}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(resolved); err != nil {
			return nil, err
		}
	}
	return resolved, nil
}
