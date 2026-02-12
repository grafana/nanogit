# Missing Tree in Collection Tool

Diagnostic tool for testing and verifying the fix for **GitHub issue #116880** - the "tree object not found in collection" bug.

## Purpose

This tool helps diagnose and verify the tree fetching behavior when working with Git repositories that have complex tree structures. It validates that the verification fix correctly fetches all tree objects before the flatten operation begins.

## The Problem

When fetching large repositories with deeply nested directory structures, the batch tree fetching logic may not collect all descendant tree objects. This causes errors like:

```
tree object <sha> not found in collection and individual fetch failed:
requested tree <sha> not in response
```

## The Fix

The verification fix adds two phases after batch processing:
1. **Verification Phase**: Recursively checks that ALL tree objects are present
2. **Completion Phase**: Fetches any missing trees in larger batches (50 per request)

This ensures the collection is complete before `flatten()` starts traversing.

## Usage

### Prerequisites

You need:
1. **Git Repository URL**: The repository to test
2. **Commit SHA**: A commit with a complex tree structure (many files/folders)
3. **Authentication Token**: Git provider token (GitHub, GitLab, etc.)

### Running the Tool

```bash
cd /path/to/nanogit

TEST_GITHUB_TOKEN="your_token" \
TEST_REPO_URL="https://github.com/your/repo.git" \
TEST_COMMIT_SHA="abc123..." \
./tools/missing-tree-in-collection/test.sh
```

### Manual Test Execution

```bash
cd /path/to/nanogit

export TEST_GITHUB_TOKEN="your_token"
export TEST_REPO_URL="https://github.com/your/repo.git"
export TEST_COMMIT_SHA="abc123..."

go test -v ./tests -run TestTreeCompleteness -timeout 5m
```

### Verbose Output

The test automatically provides detailed logging:
```bash
TEST_GITHUB_TOKEN="your_token" \
TEST_REPO_URL="https://github.com/your/repo.git" \
TEST_COMMIT_SHA="abc123..." \
go test -v ./tests -run TestTreeCompleteness -timeout 5m
```

## What It Tests

1. **Full Commit Fetch Test**
   - Connects to the Git repository
   - Fetches the specified commit
   - Calls `GetFlatTree()` to trigger full tree traversal
   - Verifies all tree objects are fetched
   - Validates tree hash and entry counts

2. **Individual Tree Fetch Test**
   - Attempts to fetch specific tree objects
   - Confirms trees exist in the repository
   - Validates individual fetch capability

## Expected Output

### ✅ Success (With Verification Fix)
```
✓ Test PASSED - tree fetched successfully!

=== VERIFICATION ===
  Expected tree: <hash>
  Actual tree:   <hash>
  Match: true

  Total entries: XXX
  ✓ Confirmed: Fetched the correct tree!
  Files: YYY, Directories: ZZZ
```

### ❌ Failure (Without Fix)
```
✗ Test FAILED with error: tree object <sha> not found in collection
```

## Use Cases

1. **Verify the fix works** - Test that repositories with complex trees are handled correctly
2. **Reproduce issues** - Debug tree fetching problems with real repositories
3. **Regression testing** - Ensure fix continues to work as code evolves
4. **Performance testing** - Measure fetch performance with large repositories

## CI Integration

This tool is **automatically skipped in CI environments** because it:
- Requires external network access
- Needs authentication credentials
- Depends on specific repository access
- Is slower than unit tests

Detects: `CI`, `GITHUB_ACTIONS`, `GITLAB_CI`, `CIRCLECI` environment variables

## Files

- `test.sh` - Test runner script with parameter validation
- `README.md` - This file
- `../../tests/tree_completeness_test.go` - Go test implementation

## Related

- **GitHub Issue**: #116880 - Tree object not found in collection
- **Code**: `tree.go` - `verifyTreeCompleteness()` and `completeMissingTrees()`
- **Test**: `tests/tree_completeness_test.go`

## Security Notes

- ⚠️  Never commit tokens or credentials
- ⚠️  Be careful with repository URLs (private repos)
- ✅ All parameters must be explicitly provided
- ✅ No default repository values to prevent accidental exposure
