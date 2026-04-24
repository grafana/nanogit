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
