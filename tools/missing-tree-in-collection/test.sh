#!/bin/bash
# Missing Tree in Collection - Test Tool
# Part of the fix for GitHub issue #116880
#
# This tool tests the tree fetching fix against real Git repositories to verify
# that all tree objects are correctly collected before the flatten operation.
#
# Usage:
#   TEST_GITHUB_TOKEN=your_token \
#   TEST_REPO_URL=https://github.com/your/repo.git \
#   TEST_COMMIT_SHA=commit_sha \
#   ./tools/missing-tree-in-collection/test.sh
#
# Requirements:
#   - TEST_GITHUB_TOKEN or GITHUB_TOKEN: Authentication token for Git provider
#   - TEST_REPO_URL: Git repository URL to test
#   - TEST_COMMIT_SHA: Commit SHA to fetch
#   - Internet connectivity

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}Real World Bug Test (Issue #116880)${NC}"
echo -e "${BLUE}================================${NC}"
echo ""

# Check if we're in CI
if [ -n "$CI" ] || [ -n "$GITHUB_ACTIONS" ] || [ -n "$GITLAB_CI" ] || [ -n "$CIRCLECI" ]; then
    echo -e "${YELLOW}⚠️  CI environment detected - skipping real world bug test${NC}"
    echo -e "${YELLOW}   This test requires external Git repository access and should be run manually${NC}"
    exit 0
fi

# Check for required parameters
MISSING_PARAMS=()

if [ -z "$GITHUB_TOKEN" ] && [ -z "$TEST_GITHUB_TOKEN" ]; then
    MISSING_PARAMS+=("GITHUB_TOKEN or TEST_GITHUB_TOKEN")
fi

if [ -z "$TEST_REPO_URL" ]; then
    MISSING_PARAMS+=("TEST_REPO_URL")
fi

if [ -z "$TEST_COMMIT_SHA" ]; then
    MISSING_PARAMS+=("TEST_COMMIT_SHA")
fi

if [ ${#MISSING_PARAMS[@]} -gt 0 ]; then
    echo -e "${RED}✗ ERROR: Missing required parameters${NC}"
    echo ""
    for param in "${MISSING_PARAMS[@]}"; do
        echo -e "  - ${YELLOW}${param}${NC}"
    done
    echo ""
    echo -e "${YELLOW}This test requires authentication and repository details.${NC}"
    echo ""
    echo -e "Usage:"
    echo -e "  ${BLUE}TEST_GITHUB_TOKEN=your_token \\"
    echo -e "  TEST_REPO_URL=https://github.com/your/repo.git \\"
    echo -e "  TEST_COMMIT_SHA=commit_sha \\"
    echo -e "  ./scripts/test-real-world-bug.sh${NC}"
    echo ""
    echo -e "Example:"
    echo -e "  ${BLUE}TEST_GITHUB_TOKEN=ghp_xxxxxxxxxxxx \\"
    echo -e "  TEST_REPO_URL=https://github.com/example/myrepo.git \\"
    echo -e "  TEST_COMMIT_SHA=abc123def456 \\"
    echo -e "  ./scripts/test-real-world-bug.sh${NC}"
    echo ""
    echo -e "To create a GitHub token:"
    echo -e "  1. Go to https://github.com/settings/tokens"
    echo -e "  2. Click 'Generate new token (classic)'"
    echo -e "  3. Select scopes: ${YELLOW}repo${NC} (for private repos) or ${YELLOW}public_repo${NC} (for public only)"
    echo -e "  4. Click 'Generate token' and copy it"
    echo ""
    exit 1
fi

# Set TEST_GITHUB_TOKEN from GITHUB_TOKEN if not already set
if [ -n "$GITHUB_TOKEN" ] && [ -z "$TEST_GITHUB_TOKEN" ]; then
    export TEST_GITHUB_TOKEN="$GITHUB_TOKEN"
fi

echo -e "${GREEN}✓ All parameters configured${NC}"
echo ""
echo -e "${BLUE}Test Configuration:${NC}"
echo -e "  Repository: ${YELLOW}${TEST_REPO_URL}${NC}"
echo -e "  Commit:     ${YELLOW}${TEST_COMMIT_SHA}${NC}"
echo -e "  Token:      ${GREEN}✓ Configured${NC}"
echo ""

echo -e "${BLUE}Running tests...${NC}"
echo ""

# Run the test and show output in real-time, also capture it
TEST_OUTPUT_FILE=$(mktemp)
trap "rm -f $TEST_OUTPUT_FILE" EXIT

# Run test with timeout, showing output in real-time
set +e  # Don't exit on error
go test -v ./tests \
    -run TestTreeCompleteness \
    -timeout 5m 2>&1 | tee "$TEST_OUTPUT_FILE"
TEST_EXIT_CODE=${PIPESTATUS[0]}
set -e

echo ""

# Read the captured output
TEST_OUTPUT=$(cat "$TEST_OUTPUT_FILE")

# Check if any tests actually ran (not just skipped)
if echo "$TEST_OUTPUT" | grep -q "no tests to run"; then
    echo -e "${RED}================================${NC}"
    echo -e "${RED}✗ No tests found${NC}"
    echo -e "${RED}================================${NC}"
    echo ""
    echo -e "${YELLOW}No matching tests found. This could mean:${NC}"
    echo -e "${YELLOW}  1. Test file missing or not compiled${NC}"
    echo -e "${YELLOW}  2. Test name/pattern doesn't match${NC}"
    exit 1
fi

# Check if tests were skipped
if echo "$TEST_OUTPUT" | grep -q "SKIP" && ! echo "$TEST_OUTPUT" | grep -q "PASS"; then
    echo -e "${YELLOW}================================${NC}"
    echo -e "${YELLOW}⚠️  Tests were skipped${NC}"
    echo -e "${YELLOW}================================${NC}"
    echo ""
    echo -e "${YELLOW}Check the skip reason above.${NC}"
    echo -e "${YELLOW}Common reasons:${NC}"
    echo -e "${YELLOW}  - Missing required parameters${NC}"
    echo -e "${YELLOW}  - CI environment detected${NC}"
    exit 1
fi

# Check test exit code
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}================================${NC}"
    echo -e "${GREEN}✓ All tests PASSED${NC}"
    echo -e "${GREEN}================================${NC}"
    echo ""
    echo -e "${GREEN}The verification fix successfully fetched the commit!${NC}"
    echo -e "${GREEN}This confirms the fix prevents the 'tree object not found' error.${NC}"
    exit 0
else
    echo -e "${RED}================================${NC}"
    echo -e "${RED}✗ Tests FAILED (exit code: $TEST_EXIT_CODE)${NC}"
    echo -e "${RED}================================${NC}"
    echo ""
    echo -e "${YELLOW}Common failure reasons:${NC}"
    echo -e "${YELLOW}  - Invalid commit SHA (check the commit exists)${NC}"
    echo -e "${YELLOW}  - Network connectivity issues${NC}"
    echo -e "${YELLOW}  - Invalid or expired token${NC}"
    echo -e "${YELLOW}  - Repository access denied${NC}"
    echo -e "${YELLOW}  - Repository not found${NC}"
    exit 1
fi
