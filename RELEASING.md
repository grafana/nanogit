# Release Process

This document describes the automated release process for nanogit.

## Overview

nanogit uses an automated release pipeline powered by [semantic-release](https://semantic-release.gitbook.io/) and GitHub Actions. Releases are triggered automatically when changes are merged to the `main` branch, with version bumps determined by [Conventional Commit](https://www.conventionalcommits.org/) messages.

## How It Works

### Automatic Versioning

The release system analyzes commit messages to determine the version bump:

| Commit Type | Version Bump | Example |
|-------------|--------------|---------|
| `fix:` | Patch | v1.4.0 → v1.4.1 |
| `feat:` | Minor | v1.4.0 → v1.5.0 |
| `feat!:` or `BREAKING CHANGE:` | Major | v1.4.0 → v2.0.0 |
| `perf:` | Patch | v1.4.0 → v1.4.1 |
| `docs:`, `chore:`, `ci:`, etc. | No release | - |

### Release Workflow

When a PR is merged to `main`:

1. **CI Checks Run**: All tests, linting, and security checks must pass
2. **Semantic Release Activates**: The release workflow analyzes commits since the last release
3. **Version Determined**: Based on commit message types
4. **Release Published**: semantic-release creates the root tag (e.g. `v1.5.0`) and publishes the GitHub release with auto-generated notes — these notes are the single source of truth for the changelog
5. **Module Tags Created**: The workflow then tags the nested public modules at the same commit, so all three tags are synchronized:
   - `v1.5.0` - Main nanogit module
   - `gittest/v1.5.0` - Test utilities module (public)
   - `cli/v1.5.0` - CLI module (`go install`-able)
   - Note: `tests/` and `perf/` modules are internal only (no tag needed)
6. **CLI Binaries Uploaded**: GoReleaser attaches platform binaries to the release (module tags are created first, so the release notes' `go install ...@v1.5.0` command always resolves)
7. **Docs Rebuild**: The release workflow dispatches the Documentation workflow (`gh workflow run docs.yml`), which regenerates the changelog page from the GitHub Releases API and deploys it to GitHub Pages
8. **pkg.go.dev Updated**: Go module proxy automatically indexes the public modules

**Note**: There is no `CHANGELOG.md` file in the repository. Release notes live only on the [GitHub Releases](https://github.com/grafana/nanogit/releases) page, and the docs site's [Changelog](https://grafana.github.io/nanogit/changelog) page is generated from them at build time.

### Multiple Commits in a PR

When a PR contains multiple commits, the **highest version bump wins**:

```
fix: bug 1        → Patch
feat: feature 1   → Minor  (wins)
fix: bug 2        → Patch
→ Result: Minor release
```

```
feat: feature 1   → Minor
feat!: breaking   → Major  (wins)
fix: bug 1        → Patch
→ Result: Major release
```

## Release Configuration

### Files

- **`.releaserc.json`**: Semantic-release configuration
- **`.github/workflows/release.yml`**: Release automation workflow
- **`.github/workflows/docs.yml`**: Documentation build/deploy; dispatched by the release workflow (via `workflow_dispatch`) so the changelog page reflects new releases
- **`scripts/prepare-docs.sh`**: Generates `docs/changelog.md` from the GitHub Releases API at docs-build time

### Workflow Permissions

The release workflow requires only:
- `contents: write` - To create tags, push commits, publish the GitHub release, and upload GoReleaser assets
- `actions: write` - To dispatch the Documentation workflow so the changelog page refreshes after a release

`issues: write` and `pull-requests: write` are intentionally **not** granted: `@semantic-release/github` has comments, labels, and fail-issues disabled in `.releaserc.json`, so it only publishes the release.

## For Maintainers

### Merging PRs

When reviewing and merging PRs:

1. **Check Commit Messages**: Ensure they follow conventional commit format
2. **Verify Type**: Confirm the commit type matches the actual change
   - `feat:` for new features
   - `fix:` for bug fixes
   - Breaking changes properly marked with `!` or `BREAKING CHANGE:`
3. **Squash and Merge**: Use squash merge to create a clean commit history
4. **Edit Commit Message**: GitHub allows editing the squashed commit message before merging

### Expected Behavior

After merging to `main`:

1. CI completes (~5-10 minutes)
2. Release workflow runs (~2-3 minutes)
3. New release appears in [Releases](https://github.com/grafana/nanogit/releases)
4. Documentation workflow rebuilds and redeploys the changelog page from the new release (~2-3 minutes)
5. pkg.go.dev indexes the new version (5-15 minutes)

### When No Release Occurs

A release won't be created if:
- All commits are non-release types (`docs:`, `chore:`, `ci:`, etc.)
- Commit message contains `[skip ci]` or `[skip release]`
- Commits don't follow conventional commit format
- Release workflow fails (CI must pass first)

### Troubleshooting

#### Release Didn't Trigger

1. Check the [Actions tab](https://github.com/grafana/nanogit/actions/workflows/release.yml)
2. Verify commit messages follow conventional commits
3. Check if CI jobs passed
4. Look for `[skip ci]` in commit messages

#### Wrong Version Bump

1. Review the merged commit messages
2. Verify commit types match the changes
3. Check for `!` or `BREAKING CHANGE:` in commits

#### Release Failed

1. Check workflow logs in Actions tab
2. Common issues:
   - CI checks failed
   - Network issues with npm packages
   - GitHub token permissions

#### Changelog Page Not Updated

**Problem**: A release was published but the docs site's changelog page is stale

**Solutions**:
1. **Check the Documentation workflow**: The release workflow's "Refresh documentation changelog" step should have dispatched it — verify it ran and deployed
2. **Re-run manually**: Trigger the Documentation workflow via `workflow_dispatch` (`gh workflow run docs.yml --ref main`)
3. **Verify the release is not a draft**: `prepare-docs.sh` skips draft releases

## Manual Release (Emergency)

In rare cases where automatic release fails, you can trigger a manual release:

### Option 1: Fix and Re-trigger

1. Fix the issue (e.g., correct commit message)
2. Create a new commit to `main`
3. Workflow will run again

### Option 2: Manual Tag (Not Recommended)

Only use this as a last resort:

```bash
# Determine next version
git fetch --tags
git describe --tags --abbrev=0  # Shows last tag

# Create and push tag
git tag v0.1.1
git push origin v0.1.1

# Manually create GitHub release
gh release create v0.1.1 --generate-notes
```

**Note**: `--generate-notes` produces basic notes from PR titles rather than the conventional-commit sections semantic-release creates. Prefer fixing the automated process.

## Best Practices

### For Contributors

1. **Write Clear Commit Messages**: Follow conventional commit format
2. **One Logical Change Per Commit**: Makes version bumps predictable
3. **Document Breaking Changes**: Always include `BREAKING CHANGE:` in footer
4. **Test Before Merging**: Ensure CI passes

### For Maintainers

1. **Review Commit History**: Check that version bump will be appropriate
2. **Edit Squash Commits**: Clean up commit messages when squashing
3. **Coordinate Major Releases**: Discuss breaking changes with the team
4. **Monitor Releases**: Check that releases complete successfully
5. **Update Documentation**: Ensure docs reflect new versions

## Multi-Module Repository

nanogit uses a multi-module architecture with synchronized versioning:

### Module Structure

```
/                  - github.com/grafana/nanogit (main module)
├── /gittest       - github.com/grafana/nanogit/gittest (test utilities, tagged gittest/v*)
├── /cli           - github.com/grafana/nanogit/cli (CLI, tagged cli/v*)
├── /tests         - github.com/grafana/nanogit/tests (integration tests, internal)
└── /perf          - github.com/grafana/nanogit/perf (performance tests, internal)
```

### Synchronized Versioning

Public modules share the same version number. When releasing v1.5.0:
- Main: `v1.5.0` (runtime library)
- gittest: `gittest/v1.5.0` (public test utility)
- cli: `cli/v1.5.0` (CLI)

Internal modules (tests, perf) are not tagged as they're only used via the workspace.

### Usage

**Main module:**
```bash
go get github.com/grafana/nanogit@v1.5.0
```

**Test utilities (optional):**
```bash
go get github.com/grafana/nanogit/gittest@gittest/v1.5.0
```

**CLI:**
```bash
go install github.com/grafana/nanogit/cli/cmd/nanogit@v1.5.0   # or @latest
```

**Note:** The main module does NOT depend on gittest, cli, or tests, ensuring users only download what they need.

### CLI module versioning

The `cli/` module's `go.mod` pins the **latest released** nanogit version — never
`v0.0.0`, a pseudo-version, or a `replace` directive (any of those would break
`go install`). Because the pin is a released version, a `cli/vX.Y.Z` tag
created at release time necessarily pins the *previous* nanogit release. This
is by design and safe: the `cli-install-check` CI job builds `cli/` with
`GOWORK=off` on every PR, guaranteeing that main's CLI always compiles against
its pinned released nanogit — so every `cli/v*` tag is installable.

The practical rule this enforces: **a PR may not add a root-module API and
consume it from `cli/` at the same time.** Land the library change, let the
release ship, then bump `cli/go.mod` in a follow-up PR:

```bash
cd cli && go get github.com/grafana/nanogit@v1.5.0 && go mod tidy
```

(This is the same pattern used for `gittest/go.mod`, e.g. PR #367. Renovate
deliberately ignores `github.com/grafana/nanogit*` self-requires, so this bump
is always a manual step.)

## Versioning Strategy

nanogit is **post-1.0 (production-stable)**:

- **v1.x.x**: Production-stable releases
- Breaking changes require a major bump (`feat!:` / `BREAKING CHANGE:` → v2.0.0)
- API stability guaranteed within a major version
- Deprecation warnings before breaking changes

**History note**: the project released as v0.x through v0.18.x before
stabilizing at v1. The `v0.0.0` tag was a semantic-release baseline, not a
usable release — it is [retracted](https://go.dev/ref/mod#go-mod-file-retract)
in `go.mod` and must never be recreated or referenced from module files.

## Resources

- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [semantic-release Documentation](https://semantic-release.gitbook.io/)
- [Keep a Changelog](https://keepachangelog.com/)

## Questions?

If you have questions about the release process:
1. Check this document
2. Review the [CONTRIBUTING.md](CONTRIBUTING.md)
3. Open an issue for clarification
4. Contact the maintainers

---

**Last Updated**: 2026-07-20
**Maintained By**: Grafana Labs
