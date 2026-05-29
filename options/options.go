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

type BasicAuth struct {
	Username string
	Password string
}

type Options struct {
	HTTPClient    *http.Client
	UserAgent     string
	BasicAuth     *BasicAuth
	AuthToken     *string
	SkipGitSuffix bool
	// ReceivePackCapabilities, when non-empty, overrides the capabilities
	// advertised on receive-pack ref update commands. When nil or empty,
	// protocol.DefaultReceivePackCapabilities() is used.
	ReceivePackCapabilities []protocol.Capability
	// Limits caps the bytes nanogit will read from the server per HTTP
	// response, classified by operation. The zero value disables every cap
	// so embedders that don't opt in keep today's unbounded behavior.
	Limits Limits
	// NegotiateCapabilities, when true, makes the client fetch the server's
	// receive-pack capability advertisement once per client lifetime and
	// advertise the intersection with its desired set on subsequent ref
	// updates. Default is false (no behavior change).
	NegotiateCapabilities bool
}

// Limits caps the total bytes nanogit will read from the server in a single
// HTTP response, broken down by operation class. A zero value for any field
// means "no limit" so the zero Limits preserves nanogit's historic behavior.
//
// Negative values are rejected at construction time (see WithLimits).
// All three read-side caps govern the git-upload-pack endpoint (which
// carries both the fetch and ls-refs commands in protocol v2); they are
// split by operation rather than by endpoint because their expected
// response sizes differ by orders of magnitude. ReceivePackResponseMaxBytes
// is the lone write-side cap and governs git-receive-pack.
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
}

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
