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
