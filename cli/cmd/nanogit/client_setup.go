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
// NANOGIT_TRACE=1.
func setupClient(ctx context.Context, repoURL string) (context.Context, nanogit.Client, error) {
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

	var (
		client nanogit.Client
		err    error
	)
	if token != "" {
		client, err = nanogit.NewHTTPClient(repoURL,
			options.WithBasicAuth(username, token),
			options.WithoutGitSuffix())
	} else {
		client, err = nanogit.NewHTTPClient(repoURL, options.WithoutGitSuffix())
	}
	if err != nil {
		return ctx, nil, fmt.Errorf("failed to create client: %w", err)
	}
	return ctx, client, nil
}
