package integration_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/options"
	"github.com/stretchr/testify/require"
)

// TestProvidersPreserveSubmodulesOnWrite is the real-provider counterpart to
// the Gitea integration tests in writer_submodule_integration_test.go.
//
// It exercises grafana/grafana#123891: when a Git Sync repository contains a
// submodule, every commit pushed by nanogit's StagedWriter would silently drop
// the submodule from any tree it had to rebuild (root, or a directory sharing
// the submodule's parent).
//
// Skipped (like the other TestProviders* tests) when TEST_REPO / TEST_TOKEN /
// TEST_USER are unset; also skipped when the configured TEST_REPO has no
// submodules on main, so a generic provider matrix run does not regress to a
// failure for repos that simply don't exercise this code path.
func TestProvidersPreserveSubmodulesOnWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testproviders suite in short mode")
		return
	}

	repoURL := os.Getenv("TEST_REPO")
	user := os.Getenv("TEST_USER")
	token := os.Getenv("TEST_TOKEN")
	if repoURL == "" || user == "" || token == "" {
		t.Skip("Skipping testproviders suite: TEST_REPO or TEST_TOKEN or TEST_USER not set")
		return
	}

	ctx := log.ToContext(context.Background(), gittest.NewStructuredLogger(gittest.NewTestLogger(t)))
	client, err := nanogit.NewHTTPClient(
		repoURL,
		options.WithBasicAuth(user, token),
	)
	require.NoError(t, err)

	// Sanity: read access works.
	auth, err := client.IsAuthorized(ctx)
	require.NoError(t, err)
	require.True(t, auth)

	// We need to read the *raw* root tree of main to find any submodule
	// (gitlink, mode 0o160000) entries — nanogit's public Get*Tree APIs
	// strip those entries on purpose. So we set up a throwaway local clone
	// over the same authenticated URL and use git ls-tree directly. The
	// local repo is also reused at the end to verify the just-pushed branch.
	authURL, err := injectBasicAuth(repoURL, user, token)
	require.NoError(t, err, "failed to build authenticated remote URL")

	local, err := gittest.NewLocalRepo(ctx, gittest.WithRepoLogger(gittest.NewTestLogger(t)))
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := local.Cleanup(); err != nil {
			t.Logf("cleanup: failed to remove local scratch repo: %v", err)
		}
	})
	_, err = local.Git("remote", "add", "origin", authURL)
	require.NoError(t, err)
	// Shallow fetch is enough — we only inspect the tip's root tree.
	_, err = local.Git("fetch", "--depth=1", "origin", "main")
	require.NoError(t, err)

	mainTree, err := local.Git("ls-tree", "-r", "--full-tree", "FETCH_HEAD")
	require.NoError(t, err)
	submodulePath, expectedGitlink := findFirstSubmodule(mainTree)
	if submodulePath == "" {
		t.Skipf("TEST_REPO main branch contains no submodule (gitlink) entries; "+
			"this test requires a repo with at least one submodule to exercise grafana/grafana#123891. "+
			"Inspected tree:\n%s", mainTree)
		return
	}
	t.Logf("found submodule on main: path=%s gitlink=%s", submodulePath, expectedGitlink)

	mainRef, err := client.GetRef(ctx, "refs/heads/main")
	require.NoError(t, err)

	branchName := fmt.Sprintf("test-submodule-preserve-%d", time.Now().UnixNano())
	fullRef := "refs/heads/" + branchName
	require.NoError(t, client.CreateRef(ctx, nanogit.Ref{
		Name: fullRef,
		Hash: mainRef.Hash,
	}))
	t.Cleanup(func() {
		if err := client.DeleteRef(ctx, fullRef); err != nil {
			t.Logf("cleanup: failed to delete %s: %v", fullRef, err)
		}
	})

	branchRef, err := client.GetRef(ctx, fullRef)
	require.NoError(t, err)

	writer, err := client.NewStagedWriter(ctx, branchRef)
	require.NoError(t, err)

	author := nanogit.Author{
		Name:  "Submodule Preserve Test",
		Email: "submodule-preserve-test@example.com",
		Time:  time.Now(),
	}
	committer := nanogit.Committer{
		Name:  "Submodule Preserve Test",
		Email: "submodule-preserve-test@example.com",
		Time:  time.Now(),
	}

	// Force a *root* tree rebuild by creating a top-level file with a name
	// that won't already exist in the repo. addMissingOrStaleTreeEntries
	// always marks the root dirty for any modification, but using a
	// repository-root path makes the dependency on root rebuild explicit
	// and reproduces the grafana/grafana#123891 shape regardless of where
	// the submodule sits in the tree.
	probeFile := fmt.Sprintf("nanogit-issue-123891-%d.txt", time.Now().UnixNano())
	_, err = writer.CreateBlob(ctx, probeFile, []byte("probe for submodule preservation"))
	require.NoError(t, err)

	commit, err := writer.Commit(ctx, "test: probe submodule preservation across nanogit push", author, committer)
	require.NoError(t, err)
	require.NoError(t, writer.Push(ctx))

	// Fetch the freshly pushed branch into the same local scratch repo and
	// inspect its tree directly — this is the only way to *prove* the
	// gitlink survived (nanogit's Get*Tree APIs would mask a regression by
	// silently filtering submodules either way).
	_, err = local.Git("fetch", "--depth=1", "origin", branchName)
	require.NoError(t, err)
	pushedTree, err := local.Git("ls-tree", "-r", "--full-tree", "FETCH_HEAD")
	require.NoError(t, err)

	gotPath, gotGitlink := findSubmoduleAt(pushedTree, submodulePath)
	require.Equalf(t, submodulePath, gotPath,
		"submodule entry at %q must still exist in the new commit %s; "+
			"nanogit dropped it (grafana/grafana#123891).\nFull tree:\n%s",
		submodulePath, commit.Hash, pushedTree)
	require.Equalf(t, expectedGitlink, gotGitlink,
		"submodule gitlink hash at %q must be unchanged across the nanogit-authored commit; "+
			"got %s, want %s",
		submodulePath, gotGitlink, expectedGitlink)

	// Belt-and-braces: the probe file we wrote also has to be there, so we
	// know we actually exercised the write path (and aren't trivially
	// passing because the push was a no-op).
	require.Contains(t, pushedTree, "\t"+probeFile,
		"probe file %q must exist in the pushed commit; the test did not actually exercise a write",
		probeFile)
}

// injectBasicAuth returns the given URL with userinfo replaced by basic
// auth credentials. It does not log the resulting URL — callers should
// avoid emitting it to the test log.
func injectBasicAuth(rawURL, user, token string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse repo URL: %w", err)
	}
	u.User = url.UserPassword(user, token)
	return u.String(), nil
}

// findFirstSubmodule scans `git ls-tree -r --full-tree` output and returns the
// path and gitlink hash of the first submodule (mode 160000) entry found, or
// ("", "") if none.
func findFirstSubmodule(lsTreeOutput string) (path, gitlink string) {
	for line := range strings.SplitSeq(strings.TrimSpace(lsTreeOutput), "\n") {
		if line == "" {
			continue
		}
		// Format: "<mode> SP <type> SP <hash> TAB <path>"
		mode, rest, ok := strings.Cut(line, " ")
		if !ok || mode != "160000" {
			continue
		}
		_, rest, ok = strings.Cut(rest, " ")
		if !ok {
			continue
		}
		hashStr, p, ok := strings.Cut(rest, "\t")
		if !ok {
			continue
		}
		return p, hashStr
	}
	return "", ""
}

// findSubmoduleAt returns the (path, gitlink) for the entry at wantPath if it
// is a submodule, or empty strings otherwise.
func findSubmoduleAt(lsTreeOutput, wantPath string) (path, gitlink string) {
	for line := range strings.SplitSeq(strings.TrimSpace(lsTreeOutput), "\n") {
		if line == "" {
			continue
		}
		mode, rest, ok := strings.Cut(line, " ")
		if !ok || mode != "160000" {
			continue
		}
		_, rest, ok = strings.Cut(rest, " ")
		if !ok {
			continue
		}
		hashStr, p, ok := strings.Cut(rest, "\t")
		if !ok || p != wantPath {
			continue
		}
		return p, hashStr
	}
	return "", ""
}
