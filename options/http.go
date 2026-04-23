package options

import (
	"errors"
	"net/http"
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

// WithoutPushSideBand disables the "side-band-64k" capability advertised on
// push (receive-pack) requests. Without side-band, the server sends
// report-status packets directly, which nanogit's parser can reliably
// inspect for "ng <ref> <reason>" and "unpack <status>" failures.
//
// Some Git servers (notably certain GitLab configurations) wrap report-status
// in side-band channel 1. nanogit's current error detection does not strip
// the side-band channel byte before matching "ng"/"unpack" prefixes, so
// rejections from push rules, pre-receive hooks, or branch protection can
// be silently swallowed and appear as successful pushes that produce empty
// branches (the ref points at its previous value, without the new commit).
//
// Use this option when talking to servers that surface push failures via
// side-band and you observe silent push failures. GitHub does not require
// this workaround.
func WithoutPushSideBand() Option {
	return func(o *Options) error {
		o.DisablePushSideBand = true
		return nil
	}
}
