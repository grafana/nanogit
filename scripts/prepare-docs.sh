#!/usr/bin/env bash
# Prepare documentation by copying source files to docs/ directory
# This prevents duplication in git while maintaining proper structure for MkDocs

set -e

echo "Preparing documentation..."

# Copy CHANGELOG.md from root
echo "Copying changelog..."
cp CHANGELOG.md docs/changelog.md

echo "Documentation prepared successfully!"
