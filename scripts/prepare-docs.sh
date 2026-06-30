#!/usr/bin/env bash
# Prepare documentation by generating the changelog page from GitHub Releases.
#
# GitHub Releases are the source of truth for release notes (produced by
# semantic-release). This script renders them into docs/changelog.md at build
# time, so the docs site never carries a separate, hand-maintained changelog.
#
# The repository is public, so the Releases API is read anonymously — no token
# is required. If GITHUB_TOKEN happens to be set, it is used to raise the API
# rate limit, but it is entirely optional.
#
# The release body has a goreleaser-appended "## Installation" footer; we strip
# it so the changelog page shows only the notes.

set -euo pipefail

echo "Preparing documentation..."

REPO="${GITHUB_REPOSITORY:-grafana/nanogit}"
CHANGELOG_OUT="docs/changelog.md"
API="https://api.github.com/repos/${REPO}/releases"

for cmd in curl jq; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Error: '$cmd' is required to generate the changelog." >&2
    exit 1
  fi
done

# Optional auth: use a token only if one is already in the environment.
auth=()
if [ -n "${GITHUB_TOKEN:-}" ]; then
  auth=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
fi

echo "Generating changelog from GitHub Releases (${REPO})..."

{
  cat <<'EOF'
# Changelog

This page is generated from [GitHub Releases](https://github.com/grafana/nanogit/releases),
which are the source of truth for all release notes.

EOF

  # Releases are returned newest-first. Page through them (100 per page) and,
  # for each non-draft release, drop the goreleaser "## Installation" footer,
  # strip carriage returns, and trim trailing newlines so entries are separated
  # by a single blank line.
  page=1
  while :; do
    resp=$(curl -fsSL "${auth[@]}" \
      -H "Accept: application/vnd.github+json" \
      -H "X-GitHub-Api-Version: 2022-11-28" \
      "${API}?per_page=100&page=${page}")

    [ "$(jq 'length' <<<"$resp")" -eq 0 ] && break

    jq -r '.[] | select(.draft | not) | (.body | split("## Installation")[0] | gsub("\r"; "") | gsub("[\n]+$"; "")) + "\n"' <<<"$resp"

    page=$((page + 1))
  done
} > "${CHANGELOG_OUT}"

echo "Documentation prepared successfully!"
