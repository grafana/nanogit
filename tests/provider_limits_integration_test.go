package integration_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/options"
	clientpkg "github.com/grafana/nanogit/protocol/client"
	"github.com/stretchr/testify/require"
)

// TestProvidersByteLimits verifies the DoS-protection caps surface
// *client.ErrResponseTooLarge end-to-end against a real Git provider
// (GitHub / GitLab / Bitbucket / Gitea). Triggering each cap is
// provider-agnostic: a tight enough limit (smaller than the smallest
// realistic response for the operation) will trip even against a
// minimally-populated repository, so we don't need to seed any data.
//
// Run via: TEST_REPO=... TEST_TOKEN=... TEST_USER=... make test-providers
func TestProvidersByteLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testproviders suite in short mode")
		return
	}
	if os.Getenv("TEST_REPO") == "" || os.Getenv("TEST_TOKEN") == "" || os.Getenv("TEST_USER") == "" {
		t.Skip("Skipping testproviders suite: TEST_REPO or TEST_TOKEN or TEST_USER not set")
		return
	}

	ctx := log.ToContext(context.Background(), gittest.NewStructuredLogger(gittest.NewTestLogger(t)))
	authOpt := options.WithBasicAuth(os.Getenv("TEST_USER"), os.Getenv("TEST_TOKEN"))
	repoURL := os.Getenv("TEST_REPO")

	// Resolve any default branch with an unconstrained client so the
	// cap-triggering subtests have a stable commit hash to call
	// GetCommit against. We don't assume "main" — providers and repos
	// vary (master, develop, trunk, …) — so we pick the first head we
	// find. This keeps the test repo-agnostic, matching the file's
	// header contract that no specific seeding is required.
	baseClient, err := nanogit.NewHTTPClient(repoURL, authOpt)
	require.NoError(t, err)
	headRef, err := resolveAnyDefaultBranch(ctx, baseClient)
	require.NoError(t, err)

	t.Run("RefsMetadataMaxBytes cap surfaces ErrResponseTooLarge from ListRefs", func(t *testing.T) {
		client, err := nanogit.NewHTTPClient(repoURL,
			authOpt,
			options.WithLimits(options.Limits{RefsMetadataMaxBytes: 64}),
		)
		require.NoError(t, err)

		_, err = client.ListRefs(ctx)
		require.Error(t, err)
		var tooLarge *clientpkg.ErrResponseTooLarge
		require.True(t, errors.As(err, &tooLarge),
			"expected *client.ErrResponseTooLarge, got %T: %v", err, err)
		require.Equal(t, "ls-refs", tooLarge.Op)
		require.Equal(t, int64(64), tooLarge.Limit)
	})

	t.Run("SingleObjectFetchMaxBytes cap surfaces ErrResponseTooLarge from GetCommit", func(t *testing.T) {
		c, err := nanogit.NewHTTPClient(repoURL,
			authOpt,
			options.WithLimits(options.Limits{SingleObjectFetchMaxBytes: 128}),
		)
		require.NoError(t, err)

		_, err = c.GetCommit(ctx, headRef.Hash)
		require.Error(t, err)
		var tooLarge *clientpkg.ErrResponseTooLarge
		require.True(t, errors.As(err, &tooLarge),
			"expected *client.ErrResponseTooLarge, got %T: %v", err, err)
		require.Equal(t, "fetch", tooLarge.Op)
		require.Equal(t, int64(128), tooLarge.Limit)
	})

	t.Run("MultiObjectFetchMaxBytes cap surfaces ErrResponseTooLarge from GetFlatTree", func(t *testing.T) {
		c, err := nanogit.NewHTTPClient(repoURL,
			authOpt,
			options.WithLimits(options.Limits{MultiObjectFetchMaxBytes: 256}),
		)
		require.NoError(t, err)

		_, err = c.GetFlatTree(ctx, headRef.Hash)
		require.Error(t, err)
		var tooLarge *clientpkg.ErrResponseTooLarge
		require.True(t, errors.As(err, &tooLarge),
			"expected *client.ErrResponseTooLarge, got %T: %v", err, err)
		require.Equal(t, "fetch", tooLarge.Op)
		require.Equal(t, int64(256), tooLarge.Limit)
	})

	t.Run("Generous caps do not interfere with normal operation", func(t *testing.T) {
		c, err := nanogit.NewHTTPClient(repoURL,
			authOpt,
			options.WithLimits(options.Limits{
				SingleObjectFetchMaxBytes:   1 << 30, // 1 GiB
				MultiObjectFetchMaxBytes:    1 << 30,
				RefsMetadataMaxBytes:        1 << 20, // 1 MiB
				ReceivePackResponseMaxBytes: 1 << 20,
			}),
		)
		require.NoError(t, err)

		// Hitting GetRef and GetCommit through a generously-capped
		// client must NOT regress against the unconfigured baseline.
		ref, err := c.GetRef(ctx, headRef.Name)
		require.NoError(t, err)
		require.Equal(t, headRef.Hash, ref.Hash)

		commit, err := c.GetCommit(ctx, headRef.Hash)
		require.NoError(t, err)
		require.Equal(t, headRef.Hash, commit.Hash)
	})
}

// resolveAnyDefaultBranch returns a Ref pointing at a branch that exists
// on the remote, trying common defaults (main, master) first and falling
// back to the first refs/heads/* the server advertises. Used to keep
// provider tests repo-agnostic: the test header promises the test does
// not need a specific default branch, so hardcoding "refs/heads/main"
// would silently break repos using a different convention.
func resolveAnyDefaultBranch(ctx context.Context, c nanogit.Client) (nanogit.Ref, error) {
	for _, candidate := range []string{"refs/heads/main", "refs/heads/master"} {
		ref, err := c.GetRef(ctx, candidate)
		if err == nil {
			return ref, nil
		}
	}
	refs, err := c.ListRefs(ctx)
	if err != nil {
		return nanogit.Ref{}, err
	}
	for _, ref := range refs {
		if strings.HasPrefix(ref.Name, "refs/heads/") {
			return ref, nil
		}
	}
	return nanogit.Ref{}, errors.New("no refs/heads/* branch found on remote")
}

