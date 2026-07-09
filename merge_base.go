package nanogit

import (
	"container/heap"
	"context"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/client"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/storage"
)

// Reachability flags used while painting the commit graph during MergeBase.
const (
	flagReachableFromA uint8 = 1 << iota // commit is an ancestor of (or equal to) commitA
	flagReachableFromB                   // commit is an ancestor of (or equal to) commitB
	flagStale                            // commit is a common ancestor (or ancestor of one); no longer interesting
	flagResult                           // commit has been recorded as a merge-base candidate
)

const (
	// defaultMergeBaseMaxCommits bounds how many distinct commits MergeBase will
	// examine before giving up. It prevents an unbounded walk (and unbounded
	// network fetches) on pathological or unrelated histories.
	defaultMergeBaseMaxCommits = 2000

	// mergeBaseFetchDepth is how deep each commit fetch reaches so ancestors are
	// warmed into storage in batches, reducing round-trips during the walk.
	mergeBaseFetchDepth = 100
)

// MergeBaseOptions configures the behavior of MergeBase.
type MergeBaseOptions struct {
	// MaxCommits bounds the number of distinct commits examined during the walk.
	// When the walk exceeds this budget without finding a common ancestor,
	// MergeBase returns ErrMergeBaseLimitExceeded. If <= 0, a default is used.
	MaxCommits int
}

// MergeBaseOption configures MergeBase behavior.
type MergeBaseOption func(*MergeBaseOptions)

// WithMergeBaseMaxCommits overrides the maximum number of commits MergeBase will
// examine before returning ErrMergeBaseLimitExceeded.
func WithMergeBaseMaxCommits(max int) MergeBaseOption {
	return func(opts *MergeBaseOptions) {
		opts.MaxCommits = max
	}
}

func defaultMergeBaseOptions() *MergeBaseOptions {
	return &MergeBaseOptions{MaxCommits: defaultMergeBaseMaxCommits}
}

// MergeBase finds the best common ancestor (merge base) of two commits, i.e. the
// commit where their histories diverged. This is the commit Git uses as the base
// for a three-dot diff (a...b): comparing the merge base against a ref yields only
// the changes that ref actually introduced, excluding commits that landed on the
// other branch after the fork point.
//
// The graph is walked from both commits following all parents (merge commits
// included), ordered by committer time so the walk terminates as soon as a common
// ancestor is reached rather than descending to the repository root.
//
// Parameters:
//   - ctx: Context for the operation
//   - commitA: Hash of the first commit
//   - commitB: Hash of the second commit
//   - opts: Optional configuration (e.g. WithMergeBaseMaxCommits)
//
// Returns:
//   - hash.Hash: Hash of the merge base commit
//   - error: ErrNoMergeBase if the commits share no common ancestor,
//     ErrMergeBaseLimitExceeded if the walk exceeds its commit budget, or a
//     fetch/parse error if a commit cannot be retrieved.
//
// Limitation: in the presence of criss-cross merges a pair of commits can have
// more than one merge base. MergeBase returns a single one (the most recent by
// committer time), which is sufficient for computing a three-dot file diff.
//
// Example:
//
//	base, err := client.MergeBase(ctx, mainTip, prBranchTip)
//	if err != nil {
//	    return err
//	}
//	changes, err := client.CompareCommits(ctx, base, prBranchTip) // three-dot diff
func (c *httpClient) MergeBase(ctx context.Context, commitA, commitB hash.Hash, opts ...MergeBaseOption) (hash.Hash, error) {
	options := defaultMergeBaseOptions()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(options)
	}
	if options.MaxCommits <= 0 {
		options.MaxCommits = defaultMergeBaseMaxCommits
	}

	logger := log.FromContext(ctx)
	logger.Debug("Merge base",
		"commit_a", commitA.String(),
		"commit_b", commitB.String(),
		"max_commits", options.MaxCommits)

	// A commit is its own merge base with itself.
	if commitA.Is(commitB) {
		return commitA, nil
	}

	ctx, store := storage.FromContextOrInMemory(ctx)
	walk := &mergeBaseWalk{
		client: c,
		store:  store,
		flags:  make(map[string]uint8),
		pq:     &commitHeap{},
		max:    options.MaxCommits,
	}
	heap.Init(walk.pq)

	if err := walk.seed(ctx, commitA, flagReachableFromA); err != nil {
		return hash.Zero, err
	}
	if err := walk.seed(ctx, commitB, flagReachableFromB); err != nil {
		return hash.Zero, err
	}

	results, err := walk.run(ctx)
	if err != nil {
		return hash.Zero, err
	}
	if len(results) == 0 {
		return hash.Zero, ErrNoMergeBase
	}

	best := results[0]
	for _, candidate := range results[1:] {
		if mergeBaseMoreRecent(candidate, best) {
			best = candidate
		}
	}

	logger.Debug("Merge base found",
		"commit_a", commitA.String(),
		"commit_b", commitB.String(),
		"merge_base", best.Hash.String(),
		"candidates", len(results),
		"examined", walk.distinct)
	return best.Hash, nil
}

// mergeBaseWalk holds the mutable state of a single merge-base graph walk.
type mergeBaseWalk struct {
	client   *httpClient
	store    storage.PackfileStorage
	flags    map[string]uint8 // reachability flags keyed by commit hash
	pq       *commitHeap      // frontier ordered by committer time (newest first)
	distinct int              // number of distinct commits introduced into the walk
	max      int              // upper bound on distinct commits before giving up
}

// mark records bits on a commit, counting it the first time it is seen.
func (w *mergeBaseWalk) mark(h hash.Hash, bits uint8) {
	key := h.String()
	if _, seen := w.flags[key]; !seen {
		w.distinct++
	}
	w.flags[key] |= bits
}

// seed fetches a starting commit, flags it, and pushes it onto the frontier.
func (w *mergeBaseWalk) seed(ctx context.Context, h hash.Hash, bits uint8) error {
	commit, err := w.client.fetchCommitForMergeBase(ctx, h, w.store)
	if err != nil {
		return err
	}
	w.mark(h, bits)
	heap.Push(w.pq, commit)
	return nil
}

// run paints the graph from the seeded frontier until every queued commit is
// stale, returning the common-ancestor candidates discovered along the way.
func (w *mergeBaseWalk) run(ctx context.Context) ([]*Commit, error) {
	var results []*Commit
	for w.pq.Len() > 0 {
		// Once every queued commit is stale, all remaining work is below merge
		// bases we have already found; stop.
		if w.pq.allStale(w.flags) {
			break
		}

		current := heap.Pop(w.pq).(*Commit)
		key := current.Hash.String()
		fl := w.flags[key] & (flagReachableFromA | flagReachableFromB)

		if fl == (flagReachableFromA | flagReachableFromB) {
			// Reachable from both inputs: a common ancestor. Record it once and
			// propagate as stale so its ancestors are pruned from the search.
			if w.flags[key]&flagResult == 0 {
				w.flags[key] |= flagResult
				results = append(results, current)
			}
			fl |= flagStale
		}

		if err := w.expand(ctx, current, fl); err != nil {
			return nil, err
		}
	}
	return results, nil
}

// expand propagates flags fl to the parents of current, fetching and queueing
// any parent that gains new reachability information.
func (w *mergeBaseWalk) expand(ctx context.Context, current *Commit, fl uint8) error {
	for _, parent := range current.Parents {
		key := parent.String()
		// Skip parents that already carry all of these flags: no new
		// reachability information would be propagated.
		if w.flags[key]&fl == fl {
			continue
		}
		if _, seen := w.flags[key]; !seen && w.distinct >= w.max {
			return ErrMergeBaseLimitExceeded
		}
		w.mark(parent, fl)
		parentCommit, err := w.client.fetchCommitForMergeBase(ctx, parent, w.store)
		if err != nil {
			return err
		}
		heap.Push(w.pq, parentCommit)
	}
	return nil
}

// mergeBaseMoreRecent reports whether candidate should be preferred over best as
// the returned merge base: later committer time wins, with the larger hash as a
// deterministic tie-breaker.
func mergeBaseMoreRecent(candidate, best *Commit) bool {
	ct, bt := candidate.Committer.Time, best.Committer.Time
	if ct.Equal(bt) {
		return candidate.Hash.String() > best.Hash.String()
	}
	return ct.After(bt)
}

// fetchCommitForMergeBase returns a commit, reading it from storage when present
// and otherwise fetching it (warming a batch of ancestors into storage to reduce
// round-trips). It mirrors the fetch strategy used by ListCommits.
func (c *httpClient) fetchCommitForMergeBase(ctx context.Context, commitHash hash.Hash, store storage.PackfileStorage) (*Commit, error) {
	if obj, ok := store.GetByType(commitHash, protocol.ObjectTypeCommit); ok {
		return packfileObjectToCommit(obj)
	}

	objects, err := c.Fetch(ctx, client.FetchOptions{
		NoProgress:       true,
		NoBlobFilter:     true,
		Want:             []hash.Hash{commitHash},
		Deepen:           mergeBaseFetchDepth,
		Done:             true,
		NoExtraObjects:   false, // we want ancestor commits to warm the walk
		MaxResponseBytes: c.limits.MultiObjectFetchMaxBytes,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not our ref") {
			return nil, NewObjectNotFoundError(commitHash)
		}
		return nil, fmt.Errorf("fetch commit %s: %w", commitHash.String(), err)
	}

	obj, ok := objects[commitHash.String()]
	if !ok || obj.Type != protocol.ObjectTypeCommit {
		obj, ok = store.GetByType(commitHash, protocol.ObjectTypeCommit)
		if !ok {
			return nil, NewObjectNotFoundError(commitHash)
		}
	}

	return packfileObjectToCommit(obj)
}

// commitHeap is a max-heap of commits ordered by committer time so that the most
// recent commit is popped first during the merge-base walk.
type commitHeap []*Commit

func (h commitHeap) Len() int { return len(h) }

func (h commitHeap) Less(i, j int) bool {
	// Pop the newest commit first. Break ties by hash for deterministic ordering.
	ti, tj := h[i].Committer.Time, h[j].Committer.Time
	if ti.Equal(tj) {
		return h[i].Hash.String() > h[j].Hash.String()
	}
	return ti.After(tj)
}

func (h commitHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *commitHeap) Push(x any) {
	*h = append(*h, x.(*Commit))
}

func (h *commitHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return item
}

// allStale reports whether every commit currently queued is stale, meaning all
// remaining work lies below merge bases already discovered.
func (h commitHeap) allStale(flags map[string]uint8) bool {
	for _, commit := range h {
		if flags[commit.Hash.String()]&flagStale == 0 {
			return false
		}
	}
	return true
}
