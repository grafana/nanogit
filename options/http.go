package options

import (
	"errors"
	"fmt"
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
// does not merge with protocol.DefaultReceivePackCapabilities(). Callers that
// want a subset of the defaults must build the desired list themselves.
//
// Use this when the default set is not appropriate for the target server.
// A common case is omitting protocol.CapSideBand64k to work around servers
// that wrap report-status in side-band channel 1 (notably some GitLab
// configurations — see the package docs).
//
// Each capability is validated against protocol.Capability.Validate; an
// invalid token (empty, containing whitespace, NUL, or other control chars)
// is rejected at client construction time rather than producing a malformed
// pkt-line at push time.
func WithReceivePackCapabilities(caps ...protocol.Capability) Option {
	return func(o *Options) error {
		for i, c := range caps {
			if err := c.Validate(); err != nil {
				return fmt.Errorf("WithReceivePackCapabilities[%d]: %w", i, err)
			}
		}
		// Copy so subsequent mutations by the caller don't leak into Options.
		o.ReceivePackCapabilities = append([]protocol.Capability(nil), caps...)
		return nil
	}
}

// WithCapabilityNegotiation enables opt-in receive-pack capability
// negotiation. When set, the first call that needs receive-pack
// capabilities — creating a staged writer or any ref update (CreateRef,
// UpdateRef, DeleteRef) — fetches the server's advertised capabilities via
// GET info/refs?service=git-receive-pack and intersects them with the
// client's desired set. The result is cached for the lifetime of the
// client (a single extra round-trip per client, not per operation), so
// subsequent ref ops and writer resets across Push/Cleanup do not
// re-negotiate. Failed first attempts (network blip, transient 5xx) are
// not cached; the next call retries from scratch.
//
// Use this when you want defensive correctness against strict servers that
// reject unknown capabilities, without having to enumerate the safe subset
// by hand the way WithReceivePackCapabilities requires. The two options
// compose: WithReceivePackCapabilities provides the desired set, and
// WithCapabilityNegotiation filters that set against what the server
// advertises. report-status-v2 and agent= are always retained on the client
// side (and re-injected with nanogit's defaults if the user-supplied set
// stripped them) because dropping them would either break the response
// parser or strip the client identifier.
//
// On error during negotiation (network failure, 4xx/5xx, parse error) the
// caller's operation aborts. Silent fallback to the static set would hide
// server misconfiguration and contradicts the explicit opt-in.
func WithCapabilityNegotiation() Option {
	return func(o *Options) error {
		o.NegotiateCapabilities = true
		return nil
	}
}
