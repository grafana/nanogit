package main

import (
	"context"
	"fmt"
	"os"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/options"
)

// setupClient resolves authentication from flags/env, constructs a nanogit
// Client, and returns a context that carries the CLI logger so library
// internals can emit Info/Debug that the user can see with -v or
// NANOGIT_TRACE=1. Callers may pass extra nanogit options (e.g.
// options.WithReceivePackCapabilities) that are appended after the default
// auth / suffix options.
func setupClient(ctx context.Context, repoURL string, extra ...options.Option) (context.Context, nanogit.Client, error) {
	token := globalToken
	if token == "" {
		token = os.Getenv("NANOGIT_TOKEN")
	}

	username := globalUsername
	if username == "" {
		username = os.Getenv("NANOGIT_USERNAME")
	}
	if username == "" {
		username = "git"
	}

	ctx = log.ToContext(ctx, newCLILogger(globalVerbose))

	opts := make([]options.Option, 0, 3+len(extra))
	if token != "" {
		opts = append(opts, options.WithBasicAuth(username, token))
	}
	opts = append(opts, options.WithoutGitSuffix())
	if l, ok := limitsFromGlobalFlags(); ok {
		opts = append(opts, options.WithLimits(l))
	}
	opts = append(opts, extra...)

	client, err := nanogit.NewHTTPClient(repoURL, opts...)
	if err != nil {
		return ctx, nil, fmt.Errorf("failed to create client: %w", err)
	}
	return ctx, client, nil
}

// limitsFromGlobalFlags returns the options.Limits derived from the CLI
// --max-bytes-* flags, plus a flag indicating whether any limit was set.
// When all four flags are zero (the default), the bool is false so callers
// can skip applying WithLimits and preserve the library's default behavior.
func limitsFromGlobalFlags() (options.Limits, bool) {
	l := options.Limits{
		SingleObjectFetch:   globalMaxBytesSingleObject,
		MultiObjectFetch:    globalMaxBytesMultiObject,
		RefsMetadata:        globalMaxBytesRefs,
		ReceivePackResponse: globalMaxBytesReceivePack,
	}
	any := globalMaxBytesSingleObject != 0 ||
		globalMaxBytesMultiObject != 0 ||
		globalMaxBytesRefs != 0 ||
		globalMaxBytesReceivePack != 0
	return l, any
}
