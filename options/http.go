package options

import (
	"errors"
	"net/http"

	"github.com/grafana/nanogit/protocol"
)

// WithHTTPClient sets a custom HTTP client for making Git protocol requests.
// This allows customization of transport, timeouts, and other HTTP client settings.
// If nil is provided, an error will be returned.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(o *Options) error {
		if httpClient == nil {
			return errors.New("httpClient is nil")
		}
		o.HTTPClient = httpClient
		return nil
	}
}

// WithUserAgent sets a custom User-Agent header for Git protocol requests.
// This can be useful for identifying your application in server logs and metrics.
// If empty, a default User-Agent will be used.
func WithUserAgent(userAgent string) Option {
	return func(o *Options) error {
		o.UserAgent = userAgent
		return nil
	}
}

// WithoutGitSuffix disables the automatic appending of ".git" to the repository URL path.
// By default, nanogit appends ".git" to URLs that don't already end with it.
// Some Git hosting providers (e.g., Azure DevOps) treat ".git" as a literal part of
// the repository name rather than a suffix to strip, causing 404 errors.
// Use this option when connecting to such providers.
func WithoutGitSuffix() Option {
	return func(o *Options) error {
		o.SkipGitSuffix = true
		return nil
	}
}

// WithReceivePackCapabilities overrides the capabilities nanogit advertises on
// receive-pack ref update commands. When the caller passes any capabilities
// (even a single one), the given set replaces the default entirely — nanogit
// does not merge with protocol.DefaultReceivePackCapabilities().
//
// Use this when the default set is not appropriate for the target server.
// A common case is omitting protocol.CapSideBand64k to work around servers
// that wrap report-status in side-band channel 1; see
// WithoutReceivePackSideBand for that ergonomic shortcut.
func WithReceivePackCapabilities(caps ...protocol.Capability) Option {
	return func(o *Options) error {
		// Copy so subsequent mutations by the caller don't leak into Options.
		o.ReceivePackCapabilities = append([]protocol.Capability(nil), caps...)
		return nil
	}
}

// WithoutReceivePackSideBand is a convenience wrapper around
// WithReceivePackCapabilities that advertises nanogit's default receive-pack
// capabilities minus protocol.CapSideBand64k. Without side-band, the server
// sends report-status packets directly (no channel prefix), which makes
// failure detection reliable on servers that otherwise wrap them in side-band
// channel 1 (notably some GitLab configurations — see the package docs).
func WithoutReceivePackSideBand() Option {
	return func(o *Options) error {
		defaults := protocol.DefaultReceivePackCapabilities()
		out := defaults[:0]
		for _, c := range defaults {
			if c != protocol.CapSideBand64k {
				out = append(out, c)
			}
		}
		o.ReceivePackCapabilities = out
		return nil
	}
}
