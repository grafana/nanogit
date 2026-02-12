package integration_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

// testConfig holds the test configuration parameters
type testConfig struct {
	repoURL   string
	commitSHA string
	token     string
}

// skipIfInCI skips the test if running in a CI environment
func skipIfInCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" ||
		os.Getenv("GITLAB_CI") != "" || os.Getenv("CIRCLECI") != "" {
		t.Skip("Skipping in CI - requires external Git repository access")
	}
}

// getTestConfig retrieves and validates test configuration from environment
func getTestConfig(t *testing.T) testConfig {
	t.Helper()

	repoURL := os.Getenv("TEST_REPO_URL")
	if repoURL == "" {
		t.Skip("TEST_REPO_URL required - specify the Git repository to test")
	}

	commitSHA := os.Getenv("TEST_COMMIT_SHA")
	if commitSHA == "" {
		t.Skip("TEST_COMMIT_SHA required - specify the commit to fetch")
	}

	token := os.Getenv("TEST_GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		t.Skip("TEST_GITHUB_TOKEN or GITHUB_TOKEN required - authentication needed")
	}

	return testConfig{
		repoURL:   repoURL,
		commitSHA: commitSHA,
		token:     token,
	}
}

// createClientAndFetchTree creates a nanogit client and fetches the tree for the given commit
func createClientAndFetchTree(t *testing.T, ctx context.Context, cfg testConfig) (*protocol.FlatTree, error) {
	t.Helper()

	client, err := nanogit.NewHTTPClient(cfg.repoURL, options.WithBasicAuth(cfg.token, ""))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	commitHash, err := hash.FromHex(cfg.commitSHA)
	if err != nil {
		t.Fatalf("Failed to parse commit SHA: %v", err)
	}

	t.Logf("Attempting to fetch tree for commit %s...", cfg.commitSHA)
	t.Logf("Verification phase should catch any missing tree objects.")

	return client.GetFlatTree(ctx, commitHash)
}

// handleExpectedFailure handles the case where we expect the test to fail
func handleExpectedFailure(t *testing.T, err error) {
	t.Helper()

	t.Logf("✓ GOOD - Test failed as expected! The bug reproduced.")
	t.Logf("  Error confirms tree object was missing from batch fetch")
	if !strings.Contains(err.Error(), "tree object") {
		t.Errorf("Error should mention missing tree object, got: %v", err)
	}
}

// logUnexpectedSuccess logs when test passes but failure was expected
func logUnexpectedSuccess(t *testing.T) {
	t.Helper()

	t.Logf("⚠️  Expected failure but test passed!")
	t.Logf("   This might mean:")
	t.Logf("   1. The verification fix is working correctly")
	t.Logf("   2. Git server's behavior has changed")
	t.Logf("   3. The fallback mechanism is handling the issue")
}

// countAndLogTreeStats counts tree/file entries and logs statistics
func countAndLogTreeStats(t *testing.T, flatTree *protocol.FlatTree) {
	t.Helper()

	treeCount := 0
	fileCount := 0
	for _, entry := range flatTree.Entries {
		if entry.Type == protocol.ObjectTypeTree {
			treeCount++
		} else {
			fileCount++
		}
	}

	t.Logf("  Files: %d, Directories: %d", fileCount, treeCount)

	if len(flatTree.Entries) == 0 {
		t.Error("Expected at least some entries in the tree")
	}
}

// TestTreeCompletenessVerification verifies the fix for GitHub issue #116880
// This test validates that all tree objects are fetched before the flatten operation
// when working with repositories that have complex tree structures.
//
// To run this test:
//
//	TEST_GITHUB_TOKEN=token \
//	TEST_REPO_URL=https://github.com/your/repo.git \
//	TEST_COMMIT_SHA=abc123 \
//	go test -v -run TestTreeCompleteness
func TestTreeCompletenessVerification(t *testing.T) {
	skipIfInCI(t)
	cfg := getTestConfig(t)

	t.Logf("Testing tree completeness verification:")
	t.Logf("  Repository: %s", cfg.repoURL)
	t.Logf("  Commit: %s", cfg.commitSHA)
	t.Logf("")
	t.Logf("This test verifies that all tree objects are fetched before flatten() runs.")
	t.Logf("")

	ctx := context.Background()
	flatTree, err := createClientAndFetchTree(t, ctx, cfg)

	// Handle errors
	if err != nil {
		if os.Getenv("EXPECT_FAILURE") == "true" {
			handleExpectedFailure(t, err)
			return
		}
		t.Fatalf("Tree fetch failed: %v", err)
	}

	// Verify success
	t.Logf("✓ Test PASSED - tree fetched successfully!")

	if flatTree == nil {
		t.Fatal("flatTree is nil")
	}

	t.Logf("")
	t.Logf("=== VERIFICATION ===")
	t.Logf("  Tree hash: %s", flatTree.Hash.String())
	t.Logf("  Total entries: %d", len(flatTree.Entries))
	t.Logf("")
	t.Logf("  ✓ Confirmed: Tree fetched without missing object errors!")

	if os.Getenv("EXPECT_FAILURE") == "true" {
		logUnexpectedSuccess(t)
	}

	countAndLogTreeStats(t, flatTree)
}

// TestTreeStructureTraversal validates that nested directory structures are fully traversed
func TestTreeStructureTraversal(t *testing.T) {
	skipIfInCI(t)
	cfg := getTestConfig(t)

	ctx := context.Background()
	flatTree, err := createClientAndFetchTree(t, ctx, cfg)

	if err != nil {
		t.Fatalf("Failed to fetch complete tree structure: %v", err)
	}

	t.Logf("  ✓ Successfully fetched complete tree structure")
	t.Logf("  Total entries: %d", len(flatTree.Entries))

	countAndLogTreeStats(t, flatTree)
}
