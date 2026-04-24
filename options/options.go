package options

import (
	"net/http"

	"github.com/grafana/nanogit/protocol"
)

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
// struct. Options are pure in nanogit (they do not observe prior state), so
// callers may safely resolve the same slice more than once to re-derive the
// resolved state — notably, this lets layers above protocol.client re-extract
// their own fields without coupling to a shared default-application step.
// Nil options are ignored; the first option returning an error short-circuits.
func Resolve(opts ...Option) (*Options, error) {
	resolved := &Options{}
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
