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

	opts := make([]options.Option, 0, 2+len(extra))
	if token != "" {
		opts = append(opts, options.WithBasicAuth(username, token))
	}
	opts = append(opts, options.WithoutGitSuffix())
	opts = append(opts, extra...)

	client, err := nanogit.NewHTTPClient(repoURL, opts...)
	if err != nil {
		return ctx, nil, fmt.Errorf("failed to create client: %w", err)
	}
	return ctx, client, nil
}
