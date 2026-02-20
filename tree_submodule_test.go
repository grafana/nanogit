package nanogit

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/storage"

	"github.com/stretchr/testify/require"
)

// Test constants for submodule tests.
// These are well-formed SHA-1 hashes used as test fixtures.
var (
	testRootTreeHash  = hash.MustFromHex("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	testChildTreeHash = hash.MustFromHex("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	testSubmoduleHash = hash.MustFromHex("cccccccccccccccccccccccccccccccccccccccc")
	testBlobHash      = hash.MustFromHex("dddddddddddddddddddddddddddddddddddddddd")
)

// makeRootTreeWithSubmodule creates a root tree object that contains:
//   - a regular file (mode 0o100644)
//   - a child directory (mode 0o40000)
//   - a submodule / gitlink entry (mode 0o160000)
func makeRootTreeWithSubmodule() *protocol.PackfileObject {
	return &protocol.PackfileObject{
		Hash: testRootTreeHash,
		Type: protocol.ObjectTypeTree,
		Tree: []protocol.PackfileTreeEntry{
			{
				FileName: "README.md",
				FileMode: 0o100644,
				Hash:     testBlobHash.String(),
			},
			{
				FileName: "subdir",
				FileMode: 0o40000,
				Hash:     testChildTreeHash.String(),
			},
			{
				FileName: "external-lib",
				FileMode: 0o160000,
				Hash:     testSubmoduleHash.String(),
			},
		},
	}
}

// makeChildTree creates a simple child tree object (a subdirectory with one file).
func makeChildTree() *protocol.PackfileObject {
	return &protocol.PackfileObject{
		Hash: testChildTreeHash,
		Type: protocol.ObjectTypeTree,
		Tree: []protocol.PackfileTreeEntry{
			{
				FileName: "file.txt",
				FileMode: 0o100644,
				Hash:     testBlobHash.String(),
			},
		},
	}
}

func TestCollectMissingTreeHashes_SubmoduleNotQueued(t *testing.T) {
	t.Parallel()

	client := &httpClient{}
	ctx := context.Background()
	allObjects := storage.NewInMemoryStorage(ctx)

	rootTree := makeRootTreeWithSubmodule()
	allObjects.Add(rootTree)

	objects := map[string]*protocol.PackfileObject{
		rootTree.Hash.String(): rootTree,
	}
	pending := []hash.Hash{}
	processedTrees := make(map[string]bool)
	requestedHashes := make(map[string]bool)

	newPending, err := client.collectMissingTreeHashes(ctx, objects, allObjects, pending, processedTrees, requestedHashes)
	require.NoError(t, err)

	// The child tree hash (subdir) should be in pending because it's a real tree
	// that needs fetching. But the submodule hash should NOT be in pending.
	for _, h := range newPending {
		require.NotEqual(t, testSubmoduleHash.String(), h.String(),
			"submodule hash %s should not be queued for tree fetching; "+
				"it is a gitlink (mode 0o160000) pointing to a commit in another repository",
			testSubmoduleHash.String())
	}

	// The child tree (subdir) should still be detected as missing
	found := false
	for _, h := range newPending {
		if h.String() == testChildTreeHash.String() {
			found = true
			break
		}
	}
	require.True(t, found, "child tree hash %s should be queued for fetching", testChildTreeHash.String())
}

func TestVerifyTreeCompleteness_SubmoduleNotReportedMissing(t *testing.T) {
	t.Parallel()

	client := &httpClient{}
	ctx := context.Background()
	allObjects := storage.NewInMemoryStorage(ctx)

	rootTree := makeRootTreeWithSubmodule()
	childTree := makeChildTree()
	allObjects.Add(rootTree, childTree)

	missing, err := client.verifyTreeCompleteness(ctx, rootTree, allObjects)
	require.NoError(t, err)

	// With all real tree objects present, nothing should be reported as missing.
	// The submodule entry should NOT be treated as a missing tree.
	for _, h := range missing {
		require.NotEqual(t, testSubmoduleHash.String(), h.String(),
			"submodule hash %s should not be reported as a missing tree; "+
				"it is a gitlink (mode 0o160000) pointing to a commit in another repository",
			testSubmoduleHash.String())
	}

	require.Empty(t, missing, "no trees should be missing when all real tree objects are present")
}

func TestFlatten_SkipsSubmoduleEntries(t *testing.T) {
	t.Parallel()

	// Use a real httpClient with a mock server so that if flatten incorrectly
	// tries to fetch the submodule as a tree, it gets a clean error instead of a panic.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/info/refs") {
			w.Write([]byte("001e# service=git-upload-pack\n0000")) //nolint:errcheck
			return
		}
		// Return "not our ref" for any upload-pack request (submodule fetch attempt)
		var response bytes.Buffer
		response.Write([]byte("0000"))
		response.Write([]byte("0008NAK\n"))
		response.Write([]byte("0045ERR not our ref cccccccccccccccccccccccccccccccccccccccc\n"))
		response.Write([]byte("0000"))
		w.WriteHeader(http.StatusOK)
		w.Write(response.Bytes()) //nolint:errcheck
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL)
	require.NoError(t, err)
	httpC := client.(*httpClient)

	ctx := context.Background()
	allObjects := storage.NewInMemoryStorage(ctx)

	rootTree := makeRootTreeWithSubmodule()
	childTree := makeChildTree()
	allObjects.Add(rootTree, childTree)

	flatTree, err := httpC.flatten(ctx, rootTree, allObjects)

	// With the bug present, flatten() treats the submodule (mode 0o160000) as a tree,
	// fails to find it in the collection, and tries to fetch it individually from the
	// server, which returns "not our ref".
	// After the fix, flatten() should succeed and skip the submodule entry entirely.
	require.NoError(t, err, "flatten should succeed when submodule entries are handled correctly")
	require.NotNil(t, flatTree)

	// Submodule entries should be excluded from the flat tree — callers expect
	// only trees and blobs.
	for _, entry := range flatTree.Entries {
		require.NotEqual(t, "external-lib", entry.Name,
			"submodule entry 'external-lib' should not appear in flat tree")
	}

	// We expect only the regular entries: README.md (blob), subdir (tree), subdir/file.txt (blob)
	expectedPaths := []string{"README.md", "subdir", "subdir/file.txt"}
	actualPaths := make([]string, len(flatTree.Entries))
	for i, e := range flatTree.Entries {
		actualPaths[i] = e.Path
	}
	require.ElementsMatch(t, expectedPaths, actualPaths,
		"flat tree should contain only trees and blobs, not submodule entries")
}

func TestFlatten_SubmoduleNotRecursed(t *testing.T) {
	t.Parallel()

	// Use a real httpClient with a mock server (same reasoning as above)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/info/refs") {
			w.Write([]byte("001e# service=git-upload-pack\n0000")) //nolint:errcheck
			return
		}
		var response bytes.Buffer
		response.Write([]byte("0000"))
		response.Write([]byte("0008NAK\n"))
		response.Write([]byte("0045ERR not our ref cccccccccccccccccccccccccccccccccccccccc\n"))
		response.Write([]byte("0000"))
		w.WriteHeader(http.StatusOK)
		w.Write(response.Bytes()) //nolint:errcheck
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL)
	require.NoError(t, err)
	httpC := client.(*httpClient)

	ctx := context.Background()
	allObjects := storage.NewInMemoryStorage(ctx)

	rootTree := makeRootTreeWithSubmodule()
	childTree := makeChildTree()
	allObjects.Add(rootTree, childTree)

	flatTree, err := httpC.flatten(ctx, rootTree, allObjects)
	require.NoError(t, err, "flatten should succeed without trying to recurse into submodules")

	// The submodule is entirely skipped: neither it nor its contents appear in the flat tree.
	for _, entry := range flatTree.Entries {
		require.False(t, strings.HasPrefix(entry.Path, "external-lib/"),
			"flat tree should not contain entries inside submodule path 'external-lib/', "+
				"found: %s", entry.Path)
	}

	// We expect only regular entries — submodule is skipped:
	//   README.md (blob)
	//   subdir (tree)
	//   subdir/file.txt (blob)
	expectedPaths := []string{"README.md", "subdir", "subdir/file.txt"}
	actualPaths := make([]string, len(flatTree.Entries))
	for i, e := range flatTree.Entries {
		actualPaths[i] = e.Path
	}
	require.ElementsMatch(t, expectedPaths, actualPaths,
		"flat tree should contain exactly the expected entries")
}

func TestPackfileObjectToTree_SubmoduleSkipped(t *testing.T) {
	t.Parallel()

	rootTree := makeRootTreeWithSubmodule()

	tree, err := packfileObjectToTree(context.Background(), rootTree)
	require.NoError(t, err)

	// Submodule entries should be excluded from the tree.
	for _, entry := range tree.Entries {
		require.NotEqual(t, "external-lib", entry.Name,
			"submodule entry 'external-lib' should not appear in tree")
		require.NotEqual(t, protocol.ObjectTypeCommit, entry.Type,
			"no entry should have type ObjectTypeCommit")
	}

	// Only the blob and child tree should remain.
	require.Len(t, tree.Entries, 2)
	names := []string{tree.Entries[0].Name, tree.Entries[1].Name}
	require.ElementsMatch(t, []string{"README.md", "subdir"}, names)
}

func TestProcessSingleBatch_NotOurRefHandling(t *testing.T) {
	t.Parallel()

	// Build a proper git protocol v2 response with a fatal error inside the packfile section.
	// This is what a real git server sends when it encounters "not our ref".
	// Format: "packfile" section header, then multiplexed error packet (status byte 3).
	errorMsg := "upload-pack: not our ref cccccccccccccccccccccccccccccccccccccccc"
	buildNotOurRefResponse := func() []byte {
		var buf bytes.Buffer

		// "packfile" section header as a pkt-line
		sectionHeader := protocol.PackLine([]byte("packfile"))
		pkt, _ := sectionHeader.Marshal()
		buf.Write(pkt)

		// Fatal error multiplexed packet: status byte 3 + error message
		errorPacket := protocol.PackLine(append([]byte{3}, []byte(errorMsg)...))
		pkt, _ = errorPacket.Marshal()
		buf.Write(pkt)

		return buf.Bytes()
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/info/refs") {
			w.Write([]byte("001e# service=git-upload-pack\n0000")) //nolint:errcheck
			return
		}
		if r.URL.Path == "/git-upload-pack" {
			w.WriteHeader(http.StatusOK)
			w.Write(buildNotOurRefResponse()) //nolint:errcheck
			return
		}
		t.Errorf("unexpected request path: %s", r.URL.Path)
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL)
	require.NoError(t, err)

	httpC, ok := client.(*httpClient)
	require.True(t, ok)

	ctx := context.Background()
	ctx, allObjects := storage.FromContextOrInMemory(ctx)

	batch := []hash.Hash{testSubmoduleHash}
	retries := []hash.Hash{}
	retryCount := make(map[string]int)
	requestedHashes := make(map[string]bool)
	processedTrees := make(map[string]bool)
	pending := []hash.Hash{}
	metrics := &fetchMetrics{totalRequests: 0}

	err = httpC.processSingleBatch(ctx, batch, &retries, retryCount, requestedHashes, processedTrees, allObjects, &pending, metrics, 3)

	// The error should be recognized as an ObjectNotFoundError, not a raw
	// "fetch tree batch: parsing fetch response stream: ..." error. This ensures
	// callers can use errors.Is(err, ErrObjectNotFound) to handle this gracefully.
	require.Error(t, err)
	require.ErrorIs(t, err, ErrObjectNotFound,
		"'not our ref' errors in batch fetching should be converted to ObjectNotFoundError "+
			"for consistent error handling; got: %v", err)
}
