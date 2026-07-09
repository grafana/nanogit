package nanogit

import (
	"container/heap"
	"testing"
	"time"

	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/require"
)

func commitAt(hexByte byte, unix int64) *Commit {
	var h hash.Hash
	for i := range h {
		h[i] = hexByte
	}
	return &Commit{
		Hash:      h,
		Committer: Committer{Time: time.Unix(unix, 0)},
	}
}

func TestCommitHeap_PopsNewestFirst(t *testing.T) {
	t.Parallel()

	older := commitAt(0x01, 1000)
	newest := commitAt(0x02, 3000)
	middle := commitAt(0x03, 2000)

	h := &commitHeap{}
	heap.Init(h)
	heap.Push(h, older)
	heap.Push(h, newest)
	heap.Push(h, middle)

	require.Equal(t, newest.Hash, heap.Pop(h).(*Commit).Hash)
	require.Equal(t, middle.Hash, heap.Pop(h).(*Commit).Hash)
	require.Equal(t, older.Hash, heap.Pop(h).(*Commit).Hash)
}

func TestCommitHeap_TieBreaksByHash(t *testing.T) {
	t.Parallel()

	// Equal timestamps must resolve deterministically (larger hash first) so the
	// walk is reproducible regardless of insertion order.
	low := commitAt(0x0a, 1000)
	high := commitAt(0x0b, 1000)

	h := &commitHeap{}
	heap.Init(h)
	heap.Push(h, low)
	heap.Push(h, high)

	require.Equal(t, high.Hash, heap.Pop(h).(*Commit).Hash)
	require.Equal(t, low.Hash, heap.Pop(h).(*Commit).Hash)
}

func TestCommitHeap_AllStale(t *testing.T) {
	t.Parallel()

	a := commitAt(0x01, 1000)
	b := commitAt(0x02, 2000)
	h := commitHeap{a, b}

	flags := map[string]uint8{a.Hash.String(): flagStale}
	require.False(t, h.allStale(flags), "b is not stale yet")

	flags[b.Hash.String()] = flagStale
	require.True(t, h.allStale(flags))
}

func TestMergeBaseMoreRecent(t *testing.T) {
	t.Parallel()

	older := commitAt(0x01, 1000)
	newer := commitAt(0x02, 2000)
	require.True(t, mergeBaseMoreRecent(newer, older))
	require.False(t, mergeBaseMoreRecent(older, newer))

	// Same time: larger hash wins.
	low := commitAt(0x0a, 1000)
	high := commitAt(0x0b, 1000)
	require.True(t, mergeBaseMoreRecent(high, low))
	require.False(t, mergeBaseMoreRecent(low, high))
}
