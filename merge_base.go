package nanogit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/client"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/storage"
)

// MergeBase finds the best common ancestor of two commits, equivalent to
// `git merge-base a b`. This is the commit to diff against for three-dot
// (`a...b`) comparison semantics, such as listing the changes introduced
// by a branch relative to the point where it forked from its base.
//
// The server computes the ancestry cut: a single fetch negotiation
// requesting b while declaring a as already present returns exactly the
// commits reachable from b but not from a. The merge base is the newest
// parent of that range that falls outside it, so the operation completes
// in one round trip and its cost is proportional to how far b has
// diverged from a, not to repository history.
//
// Parameters:
//   - ctx: Context for the operation
//   - a: Hash of the first commit
//   - b: Hash of the second commit
//
// Returns:
//   - hash.Hash: Hash of the merge base commit
//   - error: ErrNoMergeBase if the commits share no common ancestor,
//     or an error if a commit cannot be fetched
//
// Example:
//
//	base, err := client.MergeBase(ctx, mainHash, branchHash)
//	if err != nil {
//	    return err
//	}
//	changes, err := client.CompareCommits(ctx, base, branchHash)
func (c *httpClient) MergeBase(ctx context.Context, a, b hash.Hash) (hash.Hash, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Merge base",
		"hash_a", a.String(),
		"hash_b", b.String())

	ctx, allObjects := storage.FromContextOrInMemory(ctx)

	objects, err := c.Fetch(ctx, client.FetchOptions{
		NoProgress:       true,
		NoBlobFilter:     true,
		NoCache:          true,
		Want:             []hash.Hash{b},
		Have:             []hash.Hash{a},
		Done:             true,
		MaxResponseBytes: c.limits.MultiObjectFetchMaxBytes,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not our ref") {
			return hash.Zero, NewObjectNotFoundError(b)
		}
		return hash.Zero, fmt.Errorf("fetch commit range %s..%s: %w", a.String(), b.String(), err)
	}

	inRange := make(map[string]bool, len(objects))
	for _, obj := range objects {
		if obj.Type == protocol.ObjectTypeCommit {
			inRange[obj.Hash.String()] = true
		}
	}

	if len(inRange) == 0 {
		logger.Debug("Empty commit range, commit is reachable from the other",
			"merge_base", b.String())
		return b, nil
	}

	candidates := make(map[string]hash.Hash)
	for _, obj := range objects {
		if obj.Type != protocol.ObjectTypeCommit {
			continue
		}
		for _, parent := range obj.Commit.Parents {
			if !inRange[parent.String()] {
				candidates[parent.String()] = parent
			}
		}
	}

	if len(candidates) == 0 {
		if _, err := c.getCommit(ctx, a, true); err != nil {
			return hash.Zero, err
		}
		return hash.Zero, fmt.Errorf("merge base for %s and %s: %w", a.String(), b.String(), ErrNoMergeBase)
	}

	var mergeBase hash.Hash
	if len(candidates) == 1 {
		for _, candidate := range candidates {
			mergeBase = candidate
		}
	} else {
		mergeBase, err = c.newestCommit(ctx, candidates, allObjects)
		if err != nil {
			return hash.Zero, err
		}
	}

	logger.Debug("Merge base found",
		"hash_a", a.String(),
		"hash_b", b.String(),
		"merge_base", mergeBase.String(),
		"range_size", len(inRange),
		"candidates", len(candidates))
	return mergeBase, nil
}

// newestCommit returns the candidate with the latest committer time, breaking
// ties by hash for deterministic results.
func (c *httpClient) newestCommit(ctx context.Context, candidates map[string]hash.Hash, allObjects storage.PackfileStorage) (hash.Hash, error) {
	var best hash.Hash
	var bestTime time.Time

	for key, candidate := range candidates {
		obj, err := c.fetchCommitObject(ctx, candidate, 1, allObjects)
		if err != nil {
			return hash.Zero, fmt.Errorf("fetch commit %s: %w", key, err)
		}

		commitTime, err := obj.Commit.Committer.Time()
		if err != nil {
			return hash.Zero, fmt.Errorf("parse committer time for %s: %w", key, err)
		}

		if best.Is(hash.Zero) || commitTime.After(bestTime) ||
			(commitTime.Equal(bestTime) && key > best.String()) {
			best = candidate
			bestTime = commitTime
		}
	}

	return best, nil
}
