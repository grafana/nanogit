package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/spf13/cobra"
)

// repoEnv is the environment variable fallback for the repository URL so that
// users can avoid repeating the URL on every command.
const repoEnv = "NANOGIT_REPO"

// resolveRepoURL returns the repository URL and the remaining positional args
// after stripping the optional repo positional. `required` is the total number
// of positional args expected when the repo URL is passed explicitly. When
// len(args) equals required-1, the URL is read from NANOGIT_REPO instead.
// Callers must pair this with repoArgs so argument counts are already
// validated by the time this runs.
func resolveRepoURL(args []string, required int) (string, []string) {
	if len(args) == required {
		return args[0], args[1:]
	}
	return os.Getenv(repoEnv), args
}

// repoArgs builds a cobra positional-arg validator that accepts either
// `required` args (explicit repo URL as the first positional) or `required-1`
// args (repo URL from NANOGIT_REPO). Commands with a variable-length tail
// (clone, put-file) have bespoke validators and don't use this.
func repoArgs(required int) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == required {
			return nil
		}
		if len(args) == required-1 && os.Getenv(repoEnv) != "" {
			return nil
		}
		return fmt.Errorf("accepts %d arg(s), received %d (set %s to omit the <repository> argument)", required, len(args), repoEnv)
	}
}

// looksLikeRepoURL is a coarse heuristic used by `clone` to disambiguate a
// single positional between a repo URL and a destination path when
// NANOGIT_REPO is set. A scheme separator (`://`) is enough to decide; any
// string without one is treated as a path.
func looksLikeRepoURL(s string) bool {
	return strings.Contains(s, "://")
}

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
