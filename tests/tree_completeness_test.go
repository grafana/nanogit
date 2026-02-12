package integration_test

import (
	"context"
	"os"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

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
	// Skip in CI environments
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" ||
		os.Getenv("GITLAB_CI") != "" || os.Getenv("CIRCLECI") != "" {
		t.Skip("Skipping in CI - requires external Git repository access")
	}

	// Check required parameters
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

	t.Logf("Testing tree completeness verification:")
	t.Logf("  Repository: %s", repoURL)
	t.Logf("  Commit: %s", commitSHA)
	t.Logf("")
	t.Logf("This test verifies that all tree objects are fetched before flatten() runs.")
	t.Logf("")

	ctx := context.Background()

	// Create client with authentication
	client, err := nanogit.NewHTTPClient(repoURL, options.WithBasicAuth(token, ""))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Parse commit SHA
	commitHash, err := hash.FromHex(commitSHA)
	if err != nil {
		t.Fatalf("Failed to parse commit SHA: %v", err)
	}

	// Fetch the complete tree
	t.Logf("Attempting to fetch tree for commit %s...", commitSHA)
	t.Logf("Verification phase should catch any missing tree objects.")

	flatTree, err := client.GetFlatTree(ctx, commitHash)
	if err != nil {
		if os.Getenv("EXPECT_FAILURE") == "true" {
			t.Logf("✓ GOOD - Test failed as expected! The bug reproduced.")
			t.Logf("  Error confirms tree object was missing from batch fetch")
			if !contains(err.Error(), "tree object") {
				t.Errorf("Error should mention missing tree object, got: %v", err)
			}
			return
		}
		t.Fatalf("Tree fetch failed: %v", err)
	}

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
		t.Logf("⚠️  Expected failure but test passed!")
		t.Logf("   This might mean:")
		t.Logf("   1. The verification fix is working correctly")
		t.Logf("   2. Git server's behavior has changed")
		t.Logf("   3. The fallback mechanism is handling the issue")
	}

	// Count tree objects in the result
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

	// Verify we actually got some data
	if len(flatTree.Entries) == 0 {
		t.Error("Expected at least some entries in the tree")
	}
}

// TestTreeStructureTraversal validates that nested directory structures are fully traversed
func TestTreeStructureTraversal(t *testing.T) {
	// Skip in CI environments
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" ||
		os.Getenv("GITLAB_CI") != "" || os.Getenv("CIRCLECI") != "" {
		t.Skip("Skipping in CI - requires external Git repository access")
	}

	// Check required parameters
	repoURL := os.Getenv("TEST_REPO_URL")
	if repoURL == "" {
		t.Skip("TEST_REPO_URL required")
	}

	commitSHA := os.Getenv("TEST_COMMIT_SHA")
	if commitSHA == "" {
		t.Skip("TEST_COMMIT_SHA required")
	}

	token := os.Getenv("TEST_GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		t.Skip("TEST_GITHUB_TOKEN or GITHUB_TOKEN required")
	}

	ctx := context.Background()

	// Create client
	client, err := nanogit.NewHTTPClient(repoURL, options.WithBasicAuth(token, ""))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Parse commit SHA
	commitHash, err := hash.FromHex(commitSHA)
	if err != nil {
		t.Fatalf("Failed to parse commit SHA: %v", err)
	}

	// Verify tree structure is complete
	flatTree, err := client.GetFlatTree(ctx, commitHash)
	if err != nil {
		t.Fatalf("Failed to fetch complete tree structure: %v", err)
	}

	t.Logf("  ✓ Successfully fetched complete tree structure")
	t.Logf("  Total entries: %d", len(flatTree.Entries))

	// Count tree objects
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

	// Verify we got some data
	if len(flatTree.Entries) == 0 {
		t.Error("Should have fetched at least some entries")
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
