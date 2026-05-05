package integration_test

import (
	"context"
	"errors"
	"os"
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

	// Resolve a known commit hash with an unconstrained client so the
	// cap-triggering subtests have a stable target to call GetCommit
	// against without depending on the provider's exact ref shape.
	baseClient, err := nanogit.NewHTTPClient(repoURL, authOpt)
	require.NoError(t, err)
	mainRef, err := baseClient.GetRef(ctx, "refs/heads/main")
	require.NoError(t, err)

	t.Run("RefsMetadata cap surfaces ErrResponseTooLarge from ListRefs", func(t *testing.T) {
		client, err := nanogit.NewHTTPClient(repoURL,
			authOpt,
			options.WithLimits(options.Limits{RefsMetadata: 64}),
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

	t.Run("SingleObjectFetch cap surfaces ErrResponseTooLarge from GetCommit", func(t *testing.T) {
		c, err := nanogit.NewHTTPClient(repoURL,
			authOpt,
			options.WithLimits(options.Limits{SingleObjectFetch: 128}),
		)
		require.NoError(t, err)

		_, err = c.GetCommit(ctx, mainRef.Hash)
		require.Error(t, err)
		var tooLarge *clientpkg.ErrResponseTooLarge
		require.True(t, errors.As(err, &tooLarge),
			"expected *client.ErrResponseTooLarge, got %T: %v", err, err)
		require.Equal(t, "fetch", tooLarge.Op)
		require.Equal(t, int64(128), tooLarge.Limit)
	})

	t.Run("MultiObjectFetch cap surfaces ErrResponseTooLarge from GetFlatTree", func(t *testing.T) {
		c, err := nanogit.NewHTTPClient(repoURL,
			authOpt,
			options.WithLimits(options.Limits{MultiObjectFetch: 256}),
		)
		require.NoError(t, err)

		_, err = c.GetFlatTree(ctx, mainRef.Hash)
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
				SingleObjectFetch:   1 << 30, // 1 GiB
				MultiObjectFetch:    1 << 30,
				RefsMetadata:        1 << 20, // 1 MiB
				ReceivePackResponse: 1 << 20,
			}),
		)
		require.NoError(t, err)

		// Hitting GetRef and GetCommit through a generously-capped
		// client must NOT regress against the unconfigured baseline.
		ref, err := c.GetRef(ctx, "refs/heads/main")
		require.NoError(t, err)
		require.Equal(t, mainRef.Hash, ref.Hash)

		commit, err := c.GetCommit(ctx, mainRef.Hash)
		require.NoError(t, err)
		require.Equal(t, mainRef.Hash, commit.Hash)
	})
}

