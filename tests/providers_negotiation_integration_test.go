package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/options"
	"github.com/stretchr/testify/require"
)

// TestProvidersCapabilityNegotiation exercises the full
// WithCapabilityNegotiation() path end-to-end against a real provider
// (GitHub / GitLab / Bitbucket via the make test-providers matrix). Unlike
// the Gitea Ginkgo spec, which only proves the wiring works against a
// container we control, this test exercises the parser against each
// provider's actual receive-pack info/refs response shape and verifies the
// negotiated capability set is sufficient for a real push.
//
// Skipped (like the other TestProviders* tests) when the TEST_REPO /
// TEST_TOKEN / TEST_USER env vars are unset.
func TestProvidersCapabilityNegotiation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testproviders suite in short mode")
		return
	}

	if os.Getenv("TEST_REPO") == "" || os.Getenv("TEST_TOKEN") == "" || os.Getenv("TEST_USER") == "" {
		t.Skip("Skipping testproviders suite: TEST_REPO or TEST_TOKEN or TEST_USER not set")
		return
	}

	ctx := log.ToContext(context.Background(), gittest.NewStructuredLogger(gittest.NewTestLogger(t)))
	client, err := nanogit.NewHTTPClient(
		os.Getenv("TEST_REPO"),
		options.WithBasicAuth(os.Getenv("TEST_USER"), os.Getenv("TEST_TOKEN")),
		options.WithCapabilityNegotiation(),
	)
	require.NoError(t, err)

	// Sanity: the same client we'll push with must report read access.
	// This is the first call that consumes the negotiated set indirectly
	// (LsRefs goes over upload-pack; receive-pack negotiation fires later
	// when we touch ref-update endpoints).
	auth, err := client.IsAuthorized(ctx)
	require.NoError(t, err)
	require.True(t, auth)

	branchName := fmt.Sprintf("test-negotiation-%d", time.Now().Unix())
	mainRef, err := client.GetRef(ctx, "refs/heads/main")
	require.NoError(t, err)

	// CreateRef triggers the lazy negotiation fetch (GET info/refs?service=git-receive-pack)
	// against the real provider. A parse failure here would mean the provider's
	// receive-pack discovery shape is something the parser can't handle.
	err = client.CreateRef(ctx, nanogit.Ref{
		Name: "refs/heads/" + branchName,
		Hash: mainRef.Hash,
	})
	require.NoError(t, err, "CreateRef must succeed when capability negotiation is enabled")

	t.Cleanup(func() {
		// Cleanup uses the cached negotiated set — no extra round-trip.
		err := client.DeleteRef(ctx, "refs/heads/"+branchName)
		require.NoError(t, err)
	})

	// Full push flow. NewStagedWriter goes through the same
	// effectiveReceivePackCapabilities path (already cached at this point)
	// and threads the negotiated set into the packfile writer.
	branchRef, err := client.GetRef(ctx, "refs/heads/"+branchName)
	require.NoError(t, err)

	writer, err := client.NewStagedWriter(ctx, branchRef)
	require.NoError(t, err)

	author := nanogit.Author{
		Name:  "Negotiation Test",
		Email: "negotiation-test@example.com",
		Time:  time.Now(),
	}
	committer := nanogit.Committer{
		Name:  "Negotiation Test",
		Email: "negotiation-test@example.com",
		Time:  time.Now(),
	}

	_, err = writer.CreateBlob(ctx, "negotiation.txt", []byte("pushed via capability negotiation"))
	require.NoError(t, err)

	commit, err := writer.Commit(ctx, "Add file via negotiated capabilities", author, committer)
	require.NoError(t, err)

	// The push itself: a real receive-pack POST advertising only the
	// intersected capabilities. If the provider rejects the negotiated
	// set or our parser dropped a capability the provider requires, this
	// is where it surfaces.
	err = writer.Push(ctx)
	require.NoError(t, err, "Push must succeed using the negotiated capability set")

	// Read-after-write: confirm the ref actually moved to our commit.
	branchRef, err = client.GetRef(ctx, "refs/heads/"+branchName)
	require.NoError(t, err)
	require.Equal(t, commit.Hash, branchRef.Hash, "branch should point at the negotiated push's commit")

	// Second push on the same client. This exercises the cache: the
	// post-push writer reset (writer.go) and any subsequent ref op should
	// reuse the negotiated set without re-fetching info/refs. We can't
	// observe the round-trip count from outside, but if caching were
	// broken in a way that produced a wire-incompatible cap set, this
	// second push would fail.
	_, err = writer.UpdateBlob(ctx, "negotiation.txt", []byte("updated via cached negotiated caps"))
	require.NoError(t, err)
	updateCommit, err := writer.Commit(ctx, "Update file (cached negotiated caps)", author, committer)
	require.NoError(t, err)
	require.NoError(t, writer.Push(ctx), "second push must reuse cached negotiated caps")

	branchRef, err = client.GetRef(ctx, "refs/heads/"+branchName)
	require.NoError(t, err)
	require.Equal(t, updateCommit.Hash, branchRef.Hash, "branch should point at the second commit")

	t.Logf("Capability negotiation succeeded end-to-end against the configured provider")
}
