package nanogit

import (
	"context"
	"crypto"
	"testing"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hashOrFail is a tiny helper that turns a hex string into hash.Hash;
// the inputs are static, so a parse failure is always a test bug.
func hashOrFail(t *testing.T, hex string) hash.Hash {
	t.Helper()
	h, err := hash.FromHex(hex)
	require.NoError(t, err)
	return h
}

// newWriterWithState constructs a stagedWriter wired up for in-memory unit
// testing — no networking, no packfile state. Callers seed treeEntries and
// submoduleEntries directly to model whatever post-staging shape they need.
func newWriterWithState(
	t *testing.T,
	tree map[string]*FlatTreeEntry,
	submodules map[string]*FlatTreeEntry,
) *stagedWriter {
	t.Helper()
	if tree == nil {
		tree = map[string]*FlatTreeEntry{}
	}
	if submodules == nil {
		submodules = map[string]*FlatTreeEntry{}
	}
	return &stagedWriter{
		client:           &httpClient{},
		ref:              Ref{Name: "refs/heads/main", Hash: hash.Zero},
		writer:           protocol.NewPackfileWriter(crypto.SHA1, protocol.PackfileStorageMemory),
		objStorage:       storage.NewInMemoryStorage(context.Background()),
		treeEntries:      tree,
		submoduleEntries: submodules,
		dirtyPaths:       map[string]bool{},
		storageMode:      protocol.PackfileStorageMemory,
	}
}

// directChildPaths is a small helper that pulls out the FileName (i.e.,
// the last path component) of each direct child returned by
// collectDirectChildren so assertions can ignore mode/hash plumbing and
// focus on which entries got merged in.
func directChildPaths(entries []protocol.PackfileTreeEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.FileName
	}
	return out
}

// TestCollectDirectChildren_MergesSubmodules covers the core fix: a
// rebuilt parent tree must include any submodule that lives directly
// under it, alongside the regular tree/blob entries from treeEntries.
// Without the merge, every dirty parent rebuild silently drops the
// gitlink (grafana/grafana#123891).
func TestCollectDirectChildren_MergesSubmodules(t *testing.T) {
	t.Parallel()

	gitlink := hashOrFail(t, "1111111111111111111111111111111111111111")
	dashboards := hashOrFail(t, "2222222222222222222222222222222222222222")
	rootFile := hashOrFail(t, "3333333333333333333333333333333333333333")

	w := newWriterWithState(t,
		map[string]*FlatTreeEntry{
			"dashboards":    {Path: "dashboards", Hash: dashboards, Type: protocol.ObjectTypeTree, Mode: 0o40000},
			"rootfile.txt":  {Path: "rootfile.txt", Hash: rootFile, Type: protocol.ObjectTypeBlob, Mode: 0o100644},
		},
		map[string]*FlatTreeEntry{
			"thirdparty": {Path: "thirdparty", Hash: gitlink, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
		},
	)

	got := w.collectDirectChildren("")
	assert.ElementsMatch(t,
		[]string{"dashboards", "rootfile.txt", "thirdparty"},
		directChildPaths(got),
		"root rebuild must include the gitlink alongside the regular entries")

	// And the merged gitlink must keep its 0o160000 mode — feeding a
	// 0o40000-bit-set entry that isn't a real tree into BuildTreeObject
	// is what produced non-canonical sort orders before this PR.
	for _, e := range got {
		if e.FileName == "thirdparty" {
			assert.Equal(t, uint32(0o160000), e.FileMode,
				"merged submodule must keep its gitlink mode; "+
					"a sibling tree mode here would mis-sort the rebuilt tree")
		}
	}
}

// TestCollectDirectChildren_ShadowSkipsSubmodule verifies the contract
// that lets the writer survive "user replaces a submodule with a regular
// blob" without emitting both entries: when treeEntries has an entry at
// the same path as a submodule, the submodule is skipped.
func TestCollectDirectChildren_ShadowSkipsSubmodule(t *testing.T) {
	t.Parallel()

	gitlink := hashOrFail(t, "1111111111111111111111111111111111111111")
	shadow := hashOrFail(t, "4444444444444444444444444444444444444444")

	w := newWriterWithState(t,
		map[string]*FlatTreeEntry{
			"thirdparty": {Path: "thirdparty", Hash: shadow, Type: protocol.ObjectTypeBlob, Mode: 0o100644},
		},
		map[string]*FlatTreeEntry{
			"thirdparty": {Path: "thirdparty", Hash: gitlink, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
		},
	)

	got := w.collectDirectChildren("")
	require.Len(t, got, 1, "shadowed submodule must not produce a duplicate entry")
	assert.Equal(t, "thirdparty", got[0].FileName)
	assert.Equal(t, uint32(0o100644), got[0].FileMode,
		"the staged blob must win over the cached submodule on shadow")
	assert.Equal(t, shadow.String(), got[0].Hash)
}

// TestCollectDirectChildren_OnlyDirectChildren guards against a regression
// where the submodule merge accidentally emits indirect descendants. The
// rebuilt parent tree must contain exactly its direct children — anything
// deeper belongs to a deeper rebuild.
func TestCollectDirectChildren_OnlyDirectChildren(t *testing.T) {
	t.Parallel()

	siblingTree := hashOrFail(t, "5555555555555555555555555555555555555555")
	deepLink := hashOrFail(t, "6666666666666666666666666666666666666666")
	directLink := hashOrFail(t, "7777777777777777777777777777777777777777")

	w := newWriterWithState(t,
		map[string]*FlatTreeEntry{
			"dashboards":         {Path: "dashboards", Hash: siblingTree, Type: protocol.ObjectTypeTree, Mode: 0o40000},
			"dashboards/nested":  {Path: "dashboards/nested", Hash: siblingTree, Type: protocol.ObjectTypeTree, Mode: 0o40000},
		},
		map[string]*FlatTreeEntry{
			// Direct child of "dashboards" — must appear in the rebuild.
			"dashboards/lib": {Path: "dashboards/lib", Hash: directLink, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
			// Two levels deep — must NOT appear in this rebuild.
			"dashboards/nested/deep": {Path: "dashboards/nested/deep", Hash: deepLink, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
		},
	)

	got := w.collectDirectChildren("dashboards")
	assert.ElementsMatch(t,
		[]string{"nested", "lib"},
		directChildPaths(got),
		"only direct children of dashboards/ should be returned")
}

// TestPruneSubmoduleEntriesAfterPush_ShadowDropsEntry covers the
// "shadow then delete" resurrection vector: a commit replaces a submodule
// path with a regular blob, so on the next rebuild the gitlink must NOT
// come back from the cache.
func TestPruneSubmoduleEntriesAfterPush_ShadowDropsEntry(t *testing.T) {
	t.Parallel()

	gitlink := hashOrFail(t, "1111111111111111111111111111111111111111")
	shadow := hashOrFail(t, "4444444444444444444444444444444444444444")

	w := newWriterWithState(t,
		map[string]*FlatTreeEntry{
			"thirdparty": {Path: "thirdparty", Hash: shadow, Type: protocol.ObjectTypeBlob, Mode: 0o100644},
		},
		map[string]*FlatTreeEntry{
			"thirdparty": {Path: "thirdparty", Hash: gitlink, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
		},
	)

	w.pruneSubmoduleEntriesAfterPush()
	assert.NotContains(t, w.submoduleEntries, "thirdparty",
		"a shadowed submodule must be dropped so a later DeleteBlob doesn't resurrect it")
}

// TestPruneSubmoduleEntriesAfterPush_AncestorMissingDropsEntry covers
// the "DeleteTree(parent) then write something else there" resurrection
// path. The submodule's parent dir is gone from treeEntries, so the
// submodule path is unreachable in the pushed commit and must be dropped.
func TestPruneSubmoduleEntriesAfterPush_AncestorMissingDropsEntry(t *testing.T) {
	t.Parallel()

	gitlink := hashOrFail(t, "1111111111111111111111111111111111111111")
	w := newWriterWithState(t,
		nil, // dashboards/* removed by an earlier DeleteTree
		map[string]*FlatTreeEntry{
			"dashboards/lib": {Path: "dashboards/lib", Hash: gitlink, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
		},
	)

	w.pruneSubmoduleEntriesAfterPush()
	assert.NotContains(t, w.submoduleEntries, "dashboards/lib",
		"submodule with a missing ancestor in treeEntries must be dropped")
}

// TestPruneSubmoduleEntriesAfterPush_AncestorAsBlobDropsEntry covers the
// "DeleteTree(a) + CreateBlob(a, ...)" sequence: the ancestor key is
// back in treeEntries, but as a blob, not a tree — the submodule below
// it is unreachable, so a key-presence-only check would wrongly keep it.
func TestPruneSubmoduleEntriesAfterPush_AncestorAsBlobDropsEntry(t *testing.T) {
	t.Parallel()

	gitlink := hashOrFail(t, "1111111111111111111111111111111111111111")
	blob := hashOrFail(t, "8888888888888888888888888888888888888888")
	w := newWriterWithState(t,
		map[string]*FlatTreeEntry{
			"dashboards": {Path: "dashboards", Hash: blob, Type: protocol.ObjectTypeBlob, Mode: 0o100644},
		},
		map[string]*FlatTreeEntry{
			"dashboards/lib": {Path: "dashboards/lib", Hash: gitlink, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
		},
	)

	w.pruneSubmoduleEntriesAfterPush()
	assert.NotContains(t, w.submoduleEntries, "dashboards/lib",
		"submodule whose ancestor is now a blob must be dropped — "+
			"key presence isn't enough, the ancestor must still be a tree")
}

// TestPruneSubmoduleEntriesAfterPush_KeepsValidSubmodule guards against
// over-pruning: when every ancestor is a tree (or the implicit root) and
// the submodule's own path is unshadowed, the entry must survive.
func TestPruneSubmoduleEntriesAfterPush_KeepsValidSubmodule(t *testing.T) {
	t.Parallel()

	gitlink := hashOrFail(t, "1111111111111111111111111111111111111111")
	dashTree := hashOrFail(t, "2222222222222222222222222222222222222222")
	w := newWriterWithState(t,
		map[string]*FlatTreeEntry{
			"dashboards": {Path: "dashboards", Hash: dashTree, Type: protocol.ObjectTypeTree, Mode: 0o40000},
		},
		map[string]*FlatTreeEntry{
			"dashboards/lib":  {Path: "dashboards/lib", Hash: gitlink, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
			"thirdparty":      {Path: "thirdparty", Hash: gitlink, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
		},
	)

	w.pruneSubmoduleEntriesAfterPush()
	assert.Contains(t, w.submoduleEntries, "dashboards/lib",
		"submodule with a tree ancestor must be kept")
	assert.Contains(t, w.submoduleEntries, "thirdparty",
		"root-level submodule has no parent to check and must always be kept when unshadowed")
}

// TestReparentSubmodules covers MoveTree's responsibility to translate
// every cached gitlink under srcPath onto destPath. Without it, the
// moved subtree's rebuild would drop the submodule and the old srcPath
// keys would dangle as resurrection bait.
func TestReparentSubmodules(t *testing.T) {
	t.Parallel()

	gitlinkExact := hashOrFail(t, "1111111111111111111111111111111111111111")
	gitlinkNested := hashOrFail(t, "2222222222222222222222222222222222222222")
	gitlinkOutside := hashOrFail(t, "3333333333333333333333333333333333333333")

	w := newWriterWithState(t, nil, map[string]*FlatTreeEntry{
		// Exact match — srcPath itself is a submodule path.
		"vendor": {Path: "vendor", Hash: gitlinkExact, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
		// Prefix match — under srcPath.
		"vendor/lib/sub": {Path: "vendor/lib/sub", Hash: gitlinkNested, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
		// No match — outside srcPath; must be untouched.
		"unrelated/sub": {Path: "unrelated/sub", Hash: gitlinkOutside, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
	})

	w.reparentSubmodules("vendor", "third_party")

	assert.NotContains(t, w.submoduleEntries, "vendor")
	assert.NotContains(t, w.submoduleEntries, "vendor/lib/sub")

	moved, ok := w.submoduleEntries["third_party"]
	require.True(t, ok, "exact-match submodule should reappear under the new path")
	assert.Equal(t, "third_party", moved.Path)
	assert.Equal(t, gitlinkExact, moved.Hash)

	movedNested, ok := w.submoduleEntries["third_party/lib/sub"]
	require.True(t, ok, "nested submodule should reappear under the new path")
	assert.Equal(t, "third_party/lib/sub", movedNested.Path)
	assert.Equal(t, gitlinkNested, movedNested.Hash)

	untouched, ok := w.submoduleEntries["unrelated/sub"]
	require.True(t, ok, "submodule outside srcPath must be untouched")
	assert.Equal(t, gitlinkOutside, untouched.Hash)
}

// TestReparentSubmodules_NoMatch ensures the helper is a no-op when no
// cached submodule lives under srcPath — a common case for repos with
// only top-level submodules.
func TestReparentSubmodules_NoMatch(t *testing.T) {
	t.Parallel()

	gitlink := hashOrFail(t, "1111111111111111111111111111111111111111")
	w := newWriterWithState(t, nil, map[string]*FlatTreeEntry{
		"thirdparty": {Path: "thirdparty", Hash: gitlink, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
	})

	w.reparentSubmodules("dashboards", "dash2")
	require.Len(t, w.submoduleEntries, 1)
	assert.Contains(t, w.submoduleEntries, "thirdparty",
		"submodule outside the moved subtree must remain in place")
}

// TestCleanup_ResetsSubmoduleEntries covers the contract that a Cleanup
// drops the cache so a long-lived stagedWriter can't pin large
// submodule-list slices for the rest of the program's lifetime.
func TestCleanup_ResetsSubmoduleEntries(t *testing.T) {
	t.Parallel()

	gitlink := hashOrFail(t, "1111111111111111111111111111111111111111")
	w := newWriterWithState(t, nil, map[string]*FlatTreeEntry{
		"thirdparty": {Path: "thirdparty", Hash: gitlink, Type: protocol.ObjectTypeCommit, Mode: 0o160000},
	})

	require.NoError(t, w.Cleanup(context.Background()))
	assert.Empty(t, w.submoduleEntries,
		"Cleanup must drop submoduleEntries for GC; otherwise long-lived "+
			"writers retain references to the repo's initial submodule list")
}
