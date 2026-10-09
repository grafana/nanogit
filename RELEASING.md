# Release Process

This document describes the automated release process for nanogit.

## Overview

nanogit uses an automated release pipeline powered by [semantic-release](https://semantic-release.gitbook.io/) and GitHub Actions. Releases are triggered automatically when changes are merged to the `main` branch, with version bumps determined by [Conventional Commit](https://www.conventionalcommits.org/) messages.

## How It Works

### Automatic Versioning

The release system analyzes commit messages to determine the version bump:

| Commit Type | Version Bump | Example |
|-------------|--------------|---------|
| `fix:` | Patch | v1.7.0 → v1.7.1 |
| `feat:` | Minor | v1.7.0 → v1.8.0 |
| `feat!:` or `BREAKING CHANGE:` | Major | v1.7.0 → v2.0.0 |
| `perf:` | Patch | v1.7.0 → v1.7.1 |
| `docs:`, `chore:`, `ci:`, etc. | No release | - |

### Release Workflow

When a PR is merged to `main`:

1. **CI Checks Run**: All tests, linting, and security checks must pass
2. **Semantic Release Activates**: The release workflow analyzes commits since the last release
3. **Version Determined**: Based on commit message types
4. **Release Published**: semantic-release creates the root tag (e.g. `v1.8.0`) and publishes the GitHub release with auto-generated notes — these notes are the single source of truth for the changelog
5. **CLI Binaries Uploaded**: GoReleaser attaches platform binaries to the release
6. **Module Tag Created**: The workflow tags the gittest module at the same commit:
   - `v1.8.0` - Main nanogit module
   - `gittest/v1.8.0` - Test utilities module (public)
   - Note: `cli/`, `tests/` and `perf/` are not tagged (see [CLI module versioning](#cli-module-versioning))
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

Only use this as a last resort. Note that pushing a tag does **not** build
binaries or refresh the docs site — neither is tag-triggered — so the last
two steps dispatch them explicitly:

```bash
# Determine next version
git fetch --tags
git describe --tags --abbrev=0 --match 'v[0-9]*'  # Shows last release tag

# Create and push the release tags
git tag v1.4.1
git push origin v1.4.1
git tag gittest/v1.4.1 v1.4.1^{}
git push origin gittest/v1.4.1

# Manually create the GitHub release
gh release create v1.4.1 --generate-notes

# Build and upload the CLI binaries for the new tag
gh workflow run goreleaser.yml --ref main -f tag=v1.4.1

# Rebuild the docs site so the changelog page reflects the new release
gh workflow run docs.yml --ref main
```

**Note**: `--generate-notes` produces basic notes from PR titles rather than the conventional-commit sections semantic-release creates. Prefer fixing the automated process.

## Recovering a Failed Release

If `release.yml` fails **after** semantic-release created the tag and GitHub
release (e.g. the GoReleaser step errored), do not re-run the workflow
expecting a rebuild: semantic-release will find no new commits, report
`new_tag == last_tag`, and skip every post-release step.

Instead:

1. **Missing binaries**: dispatch the "Rebuild Release Binaries" workflow with
   the existing tag — `gh workflow run goreleaser.yml --ref main -f tag=v1.4.1`.
   Re-uploads are safe (`replace_existing_artifacts` is enabled).
2. **Missing gittest tag**: create it manually at the release commit —
   `git tag gittest/v1.4.1 v1.4.1^{} && git push origin gittest/v1.4.1`.
   The automated step only creates the tag for the version it is currently
   releasing — it does not scan for or backfill tags missed by earlier
   releases, so a later release will **not** self-heal this for you.
3. **Stale changelog page**: dispatch the docs workflow —
   `gh workflow run docs.yml --ref main`.

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
├── /cli           - github.com/grafana/nanogit/cli (CLI, intentionally untagged)
├── /tests         - github.com/grafana/nanogit/tests (integration tests, internal)
└── /perf          - github.com/grafana/nanogit/perf (performance tests, internal)
```

### Synchronized Versioning

Tagged modules share the same version number. When releasing v1.8.0:
- Main: `v1.8.0` (runtime library)
- gittest: `gittest/v1.8.0` (public test utility)

The `cli`, `tests` and `perf` modules are not tagged.

### Usage

**Main module:**
```bash
go get github.com/grafana/nanogit@v1.8.0
```

**Test utilities (optional):**
```bash
go get github.com/grafana/nanogit/gittest@gittest/v1.8.0
```

**CLI:**
```bash
go install github.com/grafana/nanogit/cli/cmd/nanogit@latest
```

**Note:** The main module does NOT depend on gittest, cli or tests, ensuring users only download what they need.

### CLI module versioning

`cli/` is a separate module so that the library's dependency graph stays free
of Cobra. That has one consequence worth writing down: a nested module can
only be installed with `go install` if its own `go.mod` resolves without the
workspace, which rules out `v0.0.0`, pseudo-versions and `replace` directives
for its `github.com/grafana/nanogit` requirement. So `cli/go.mod` pins a
**released** nanogit version.

The module is deliberately left untagged:

- `go install .../cli/cmd/nanogit@latest` builds the tip of `main`, and the
  `cli-install-check` CI job (which builds `cli/` with `GOWORK=off`, exactly
  as `go install` does) keeps that working on every PR.
- Need an exact version? Download the release binary. Those are built by
  GoReleaser through the workspace, so they contain the released library code.

Tagging `cli/vX.Y.Z` per release was considered and rejected: the tagged tree
would still link against whatever older release `cli/go.mod` pins, so making a
pinned install truthful would require a `cli/go.mod` bump commit after every
release.

**This needs no per-release maintenance.** The pin is bumped only when the CLI
starts using a root API newer than the pin — `cli-install-check` fails the PR
and says so. Land the library change, wait for its release, then:

```bash
cd cli && go get github.com/grafana/nanogit@latest && go mod tidy
```

Local development and the rest of CI are unaffected by the pin: `go.work`
always builds the CLI against the working tree.

## Versioning Strategy

nanogit is post-1.0 and production-stable:

- **v1.x.x**: Production-stable releases
- Breaking changes require a major bump (`feat!:` / `BREAKING CHANGE:` → v2.0.0)
- API stability guaranteed within a major version
- Deprecation warnings before breaking changes

**History note**: the project released as v0.1.0 through v0.18.1 before
stabilizing at v1.0.0. The `v0.0.0` tag was a semantic-release baseline, not a
usable release — it is
[retracted](https://go.dev/ref/mod#go-mod-file-retract) in `go.mod` and must
never be recreated or referenced from a `go.mod` file.

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

**Last Updated**: 2026-10-09
**Maintained By**: Grafana Labs
