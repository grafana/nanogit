package nanogit

import (
	"path"
	"strings"
)

// normalizePath normalizes a Git path by removing leading/trailing slashes,
// collapsing multiple slashes, and trimming whitespace.
// It returns an error if the path contains invalid patterns like parent references.
func normalizePath(p string) (string, error) {
	// Trim whitespace
	p = strings.TrimSpace(p)

	// Empty path is valid (represents root)
	if p == "" {
		return "", nil
	}

	// Remove leading slashes (Git paths are always relative)
	for strings.HasPrefix(p, "/") {
		p = strings.TrimPrefix(p, "/")
	}

	// Remove trailing slashes
	for strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
	}

	// After trimming, check for empty result (was only slashes)
	if p == "" {
		return "", nil
	}

	// Collapse multiple consecutive slashes
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}

	// Reject paths containing ".." before cleaning (Git doesn't allow parent references)
	// We need to check both as a component and at boundaries
	if strings.Contains(p, "..") {
		// Check if ".." appears as a path component
		parts := strings.Split(p, "/")
		for _, part := range parts {
			if part == ".." {
				return "", NewInvalidPathError(p, "path contains parent directory references (..)")
			}
		}
	}

	// Clean the path to resolve any . components
	cleaned := path.Clean(p)

	// path.Clean returns "." for empty path, convert back to empty string
	if cleaned == "." {
		return "", nil
	}

	return cleaned, nil
}

// validateBlobPath validates and normalizes a path for blob operations.
// Blob paths cannot be empty (empty path represents the root tree, not a blob).
// Blob paths cannot end with trailing slashes (files are not directories).
func validateBlobPath(p string) (string, error) {
	// Check for trailing slash before normalization (files shouldn't look like directories)
	trimmed := strings.TrimSpace(p)
	if trimmed != "" && strings.HasSuffix(trimmed, "/") {
		return "", NewInvalidPathError(p, "blob path cannot end with trailing slash (files are not directories)")
	}

	normalized, err := normalizePath(p)
	if err != nil {
		return "", err
	}

	if normalized == "" {
		return "", NewInvalidPathError(p, "blob path cannot be empty")
	}

	return normalized, nil
}

// validateTreePath validates and normalizes a path for tree operations.
// Tree paths can be empty (empty path represents the root tree).
func validateTreePath(p string) (string, error) {
	return normalizePath(p)
}
