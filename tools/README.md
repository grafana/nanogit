# Nanogit Tools

Diagnostic and debugging tools for nanogit development and troubleshooting. The directory is currently empty; the guidelines below apply when adding new tools.

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
