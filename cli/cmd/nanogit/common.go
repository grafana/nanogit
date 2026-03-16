package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
)

// resolveRef resolves a ref name to a commit hash.
// It tries multiple strategies:
// 1. Try as full ref name (refs/heads/main, refs/tags/v1.0.0)
// 2. Try as branch name (main -> refs/heads/main)
// 3. Try as tag name (v1.0.0 -> refs/tags/v1.0.0)
// 4. Try as commit hash directly
func resolveRef(ctx context.Context, client nanogit.Client, ref string) (hash.Hash, error) {
	// Try as-is first (might already be a full ref or commit hash)
	if strings.HasPrefix(ref, "refs/") {
		refObj, err := client.GetRef(ctx, ref)
		if err == nil {
			return refObj.Hash, nil
		}
	}

	// Try as branch name
	refObj, err := client.GetRef(ctx, "refs/heads/"+ref)
	if err == nil {
		return refObj.Hash, nil
	}

	// Try as tag name
	refObj, err = client.GetRef(ctx, "refs/tags/"+ref)
	if err == nil {
		return refObj.Hash, nil
	}

	// Try as commit hash directly
	h, err := hash.FromHex(ref)
	if err == nil {
		// Verify the commit exists
		_, err = client.GetCommit(ctx, h)
		if err == nil {
			return h, nil
		}
	}

	return hash.Hash{}, fmt.Errorf("reference not found: %s", ref)
}
