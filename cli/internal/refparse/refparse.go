package refparse

import (
	"context"
	"fmt"
	"regexp"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
)

// hexHashPattern matches full 40-character hex strings (commit hashes)
var hexHashPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// ResolveRefOrHash resolves a reference name or commit hash to a commit hash.
// It supports:
// - Full reference names (refs/heads/main, refs/tags/v1.0.0)
// - Short reference names (main, v1.0.0) - tries refs/heads/* and refs/tags/*
// - Commit hashes (40 hex characters)
//
// For short names, it tries in this order:
// 1. refs/heads/<name>
// 2. refs/tags/<name>
// 3. Exact ref name as provided
//
// For commit hashes, it validates the format and returns directly.
func ResolveRefOrHash(ctx context.Context, client nanogit.Client, refOrHash string) (hash.Hash, error) {
	// Check if it looks like a commit hash (40 hex characters)
	if hexHashPattern.MatchString(refOrHash) {
		commitHash, err := hash.FromHex(refOrHash)
		if err == nil {
			// Successfully parsed as hash, verify it exists by trying to get the commit
			_, err = client.GetCommit(ctx, commitHash)
			if err != nil {
				return hash.Hash{}, fmt.Errorf("commit not found: %s: %w", refOrHash, err)
			}
			return commitHash, nil
		}
	}

	// Try as a reference name
	// If it starts with refs/, use it as-is
	if len(refOrHash) >= 5 && refOrHash[:5] == "refs/" {
		ref, err := client.GetRef(ctx, refOrHash)
		if err != nil {
			return hash.Hash{}, err
		}
		return ref.Hash, nil
	}

	// Try common ref patterns for short names
	patterns := []string{
		"refs/heads/" + refOrHash,
		"refs/tags/" + refOrHash,
		refOrHash, // Try exact name as fallback
	}

	var lastErr error
	for _, pattern := range patterns {
		ref, err := client.GetRef(ctx, pattern)
		if err == nil {
			return ref.Hash, nil
		}
		lastErr = err
	}

	return hash.Hash{}, fmt.Errorf("reference or commit not found: %s: %w", refOrHash, lastErr)
}
