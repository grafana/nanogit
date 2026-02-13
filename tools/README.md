# Nanogit Tools

Diagnostic and debugging tools for nanogit development and troubleshooting.

## Available Tools

### missing-tree-in-collection

Diagnostic tool for testing and verifying the fix for GitHub issue #116880 - the "tree object not found in collection" bug.

**Purpose**: Validates that the verification fix correctly fetches all tree objects before the flatten operation begins when working with complex repository structures.

**Usage**:
```bash
TEST_GITHUB_TOKEN=token \
TEST_REPO_URL=https://github.com/your/repo.git \
TEST_COMMIT_SHA=abc123... \
./tools/missing-tree-in-collection/test.sh
```

See [`missing-tree-in-collection/README.md`](./missing-tree-in-collection/README.md) for details.

---

## Tool Development Guidelines

When adding new tools to this directory:

1. **Create a dedicated subdirectory** for each tool
2. **Include a README.md** explaining:
   - What problem the tool solves
   - How to use it
   - What it tests/validates
   - Expected output
3. **Make tools self-contained** - include all dependencies or document them clearly
4. **Handle CI environments** - tools should skip or gracefully handle CI detection
5. **Security conscious** - never hardcode credentials, tokens, or sensitive data
6. **Clear error messages** - help users understand what went wrong

## Running Tools

Most tools in this directory are diagnostic or manual testing tools that:
- Are NOT run automatically in CI
- Require manual invocation with specific parameters
- May need external resources (network, credentials, etc.)
- Are designed for debugging specific issues

For regular automated tests, see the `tests/` directory instead.
