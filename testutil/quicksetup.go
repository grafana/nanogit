package testutil

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/grafana/nanogit"
)

// QuickSetupOption configures a QuickSetup call.
type QuickSetupOption func(*quickSetupConfig)

type quickSetupConfig struct {
	logger   Logger
	repoName string
}

// WithQuickSetupLogger sets the logger for quick setup operations.
func WithQuickSetupLogger(logger Logger) QuickSetupOption {
	return func(c *quickSetupConfig) {
		c.logger = logger
	}
}

// WithRepoName sets a custom repository name (default is auto-generated).
func WithRepoName(name string) QuickSetupOption {
	return func(c *quickSetupConfig) {
		c.repoName = name
	}
}

// QuickSetup provides complete test setup in one call:
// - Starts a Gitea server
// - Creates a test user
// - Creates a repository
// - Initializes a local repository with initial commit
// - Returns a configured nanogit client, repo info, local repo, and user
//
// Returns a cleanup function that should be called when done (typically via defer).
//
// Example:
//
//	client, repo, local, user, cleanup, err := testutil.QuickSetup(ctx)
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer cleanup()
//
//	// Now use client, repo, local, and user in your tests
func QuickSetup(ctx context.Context, opts ...QuickSetupOption) (
	client nanogit.Client,
	repo *Repo,
	local *LocalRepo,
	user *User,
	cleanup func(),
	err error,
) {
	cfg := &quickSetupConfig{
		logger:   NoopLogger(),
		repoName: "",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Generate repo name if not provided
	if cfg.repoName == "" {
		now := time.Now().UnixNano()
		var randomBytes [4]byte
		if _, err := rand.Read(randomBytes[:]); err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("failed to generate random bytes: %w", err)
		}
		cfg.repoName = fmt.Sprintf("testrepo-%d%x", now, randomBytes)
	}

	var cleanupFuncs []func()
	cleanup = func() {
		// Run cleanups in reverse order
		for i := len(cleanupFuncs) - 1; i >= 0; i-- {
			cleanupFuncs[i]()
		}
	}

	// Create server
	server, err := NewServer(ctx, WithLogger(cfg.logger))
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to create server: %w", err)
	}
	cleanupFuncs = append(cleanupFuncs, func() { _ = server.Cleanup() })

	// Create user
	user, err = server.CreateUser(ctx)
	if err != nil {
		cleanup()
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create repo
	repo, err = server.CreateRepo(ctx, cfg.repoName, user)
	if err != nil {
		cleanup()
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to create repo: %w", err)
	}

	// Create local repo
	local, err = NewLocalRepo(ctx, WithRepoLogger(cfg.logger))
	if err != nil {
		cleanup()
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to create local repo: %w", err)
	}
	cleanupFuncs = append(cleanupFuncs, func() { _ = local.Cleanup() })

	// Quick init
	client, _, err = local.QuickInit(user, repo.AuthURL)
	if err != nil {
		cleanup()
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to quick init: %w", err)
	}

	return client, repo, local, user, cleanup, nil
}

// MustQuickSetup is like QuickSetup but panics on error.
// Useful for test setup where failure should halt execution.
//
// Example:
//
//	client, repo, local, user, cleanup := testutil.MustQuickSetup(ctx)
//	defer cleanup()
func MustQuickSetup(ctx context.Context, opts ...QuickSetupOption) (
	client nanogit.Client,
	repo *Repo,
	local *LocalRepo,
	user *User,
	cleanup func(),
) {
	client, repo, local, user, cleanup, err := QuickSetup(ctx, opts...)
	if err != nil {
		panic(fmt.Sprintf("failed to quick setup: %v", err))
	}
	return client, repo, local, user, cleanup
}
