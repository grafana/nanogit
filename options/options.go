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
	// Limits caps how much data nanogit reads from the server. Its zero value
	// leaves the wire caps disabled but keeps decoded-object protection on; see
	// Limits.
	Limits Limits
	// NegotiateCapabilities, when true, makes the client fetch the server's
	// receive-pack capability advertisement once per client lifetime and
	// advertise the intersection with its desired set on subsequent ref
	// updates. Default is false (no behavior change).
	NegotiateCapabilities bool
}

// Limits caps how much data nanogit reads from the server. The four *MaxBytes
// fields cap a single HTTP response (wire bytes) and default to no limit;
// MaxObjectDecodedBytes caps a single object's decoded size and stays on by
// default. Negative values are rejected by WithLimits.
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
	// MaxObjectDecodedBytes caps the decoded (inflated) size of any single
	// object read from a packfile, checked against its declared size before
	// allocation — this is what defeats decompression bombs. Unlike the wire
	// caps, a zero value does not disable it: it keeps the built-in default
	// (protocol.MaxUnpackedObjectSize). Oversized objects surface as
	// *protocol.ObjectTooLargeError.
	//
	// That default reproduces nanogit's historical, previously-hardcoded
	// unpacked-object limit, so embedders that leave this unset see no change in
	// behavior; setting a positive value only raises or lowers that ceiling.
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
