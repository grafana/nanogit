package client

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol/hash"
)

func TestAllWantedObjectsCollected(t *testing.T) {
	t.Parallel()

	t.Run("nil wanted map returns false", func(t *testing.T) {
		// The caller's "early termination" branches MUST stay
		// disabled when NoExtraObjects was not set on the fetch —
		// otherwise we'd silently break out of any fetch the moment
		// a transport-level problem appeared, hiding real errors.
		assert.False(t, allWantedObjectsCollected(nil))
	})

	t.Run("non-nil empty map returns true", func(t *testing.T) {
		// The read loop tracks pending hashes by deletion, so an
		// empty (but non-nil) map is the precise signal that every
		// wanted hash has been seen.
		assert.True(t, allWantedObjectsCollected(map[string]bool{}))
	})

	t.Run("map with pending entries returns false", func(t *testing.T) {
		assert.False(t, allWantedObjectsCollected(map[string]bool{"a": true}))
	})
}

func TestClassifyReadObjectErr(t *testing.T) {
	t.Parallel()

	t.Run("non-cap error propagates with object position", func(t *testing.T) {
		// Corrupt packfiles, short reads, and malformed delta data
		// all surface as hard errors now (matching the strict
		// packfile-parser behavior). The 1-based position points at
		// the offending object.
		err := classifyReadObjectErr(io.ErrUnexpectedEOF, nil, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading packfile object 1")
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})

	t.Run("object position is objectsRead+1", func(t *testing.T) {
		zlibErr := errors.New("zlib: invalid header")
		err := classifyReadObjectErr(zlibErr, nil, 4)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading packfile object 5")
	})

	t.Run("cap error propagates when wanted objects are missing", func(t *testing.T) {
		// The DoS-protection contract: under the configured cap a
		// truncation must look distinct from a successful partial
		// fetch. With wanted objects still pending, the cap error
		// has to bubble up so the caller can tell a too-tight cap
		// apart from a real ObjectNotFound.
		capErr := &ErrResponseTooLarge{Limit: 100, Op: "fetch"}
		wanted := map[string]bool{"a": true}

		err := classifyReadObjectErr(capErr, wanted, 0)
		require.Error(t, err)

		var tooLarge *ErrResponseTooLarge
		require.True(t, errors.As(err, &tooLarge))
		assert.Equal(t, "fetch", tooLarge.Op)
		assert.Equal(t, int64(100), tooLarge.Limit)
	})

	t.Run("cap error is swallowed when every wanted object is already collected", func(t *testing.T) {
		// The exception that motivates this helper: a NoExtraObjects
		// fetch that has every requested object in hand (i.e. the
		// pending set has been drained to empty) only hits the cap
		// on bytes the caller doesn't need. Returning an error there
		// would turn successful single-object lookups into false
		// negatives under servers that over-send.
		capErr := &ErrResponseTooLarge{Limit: 100, Op: "fetch"}
		empty := map[string]bool{}

		err := classifyReadObjectErr(capErr, empty, 3)
		assert.NoError(t, err)
	})

	t.Run("non-cap error still propagates even when wanted set is empty", func(t *testing.T) {
		// The swallow exception is scoped to *ErrResponseTooLarge.
		// A corrupt packfile must surface as an error regardless of
		// how many wanted objects were collected first.
		zlibErr := errors.New("zlib: invalid header")
		err := classifyReadObjectErr(zlibErr, map[string]bool{}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading packfile object 2")
	})

	t.Run("wrapped cap error is still recognized", func(t *testing.T) {
		// The packfile reader may wrap with fmt.Errorf("...: %w", err);
		// classifyReadObjectErr must use errors.As so the swallow /
		// propagate decision is robust to the wrapping layer.
		capErr := &ErrResponseTooLarge{Limit: 50, Op: "fetch"}
		wrapped := fmt.Errorf("read object: %w", capErr)

		// Pending set empty → swallow
		assert.NoError(t, classifyReadObjectErr(wrapped, map[string]bool{}, 0))

		// Pending set non-empty → propagate
		err := classifyReadObjectErr(wrapped, map[string]bool{"x": true}, 0)
		require.Error(t, err)
		var tooLarge *ErrResponseTooLarge
		assert.True(t, errors.As(err, &tooLarge))
	})
}

func TestShouldTerminateEarly(t *testing.T) {
	t.Parallel()

	t.Run("nil wanted set never terminates", func(t *testing.T) {
		// Fetches without NoExtraObjects don't pre-build a wanted
		// map; this branch must therefore stay disabled — otherwise
		// any matching hash would prematurely cut a normal multi-
		// object fetch short.
		assert.False(t, shouldTerminateEarly(nil, "abc"))
	})

	t.Run("non-wanted hash leaves the set alone", func(t *testing.T) {
		wanted := map[string]bool{"want": true}
		assert.False(t, shouldTerminateEarly(wanted, "other"))
		// Pending set unchanged.
		assert.Equal(t, 1, len(wanted))
		assert.True(t, wanted["want"])
	})

	t.Run("wanted hash drains the set and signals when empty", func(t *testing.T) {
		wanted := map[string]bool{"a": true, "b": true}

		assert.False(t, shouldTerminateEarly(wanted, "a"),
			"first wanted hash should not yet terminate")
		assert.Equal(t, 1, len(wanted), "set should shrink as hashes are seen")
		assert.False(t, wanted["a"], "seen hash should be removed")

		assert.True(t, shouldTerminateEarly(wanted, "b"),
			"final wanted hash should terminate")
		assert.Equal(t, 0, len(wanted))
	})

	t.Run("duplicate wanted hash does not double-count", func(t *testing.T) {
		// Regression guard: a malicious or buggy server that sends
		// the same wanted object more than once must NOT trigger
		// early termination before every DISTINCT wanted hash has
		// been collected. Deletion-based tracking inherently
		// handles this — once a hash is removed from the pending
		// set, seeing it again is a no-op.
		wanted := map[string]bool{"a": true, "b": true}

		assert.False(t, shouldTerminateEarly(wanted, "a"))
		assert.False(t, shouldTerminateEarly(wanted, "a"),
			"duplicate of an already-collected hash must not drain the set further")
		assert.Equal(t, 1, len(wanted),
			"only the distinct hash should have been removed")

		// 'b' still pending until actually seen.
		assert.True(t, shouldTerminateEarly(wanted, "b"))
	})
}

func TestFetch_CorruptPackfile(t *testing.T) {
	t.Parallel()

	var body bytes.Buffer
	writePkt := func(b []byte) {
		fmt.Fprintf(&body, "%04x", len(b)+4)
		body.Write(b)
	}
	writePkt([]byte("packfile\n"))
	pack := []byte("PACK" +
		"\x00\x00\x00\x02" + // version 2
		"\x00\x00\x00\x01" + // 1 object
		"\x33" + // blob, size 3
		"\xff\xff") // invalid zlib stream
	writePkt(append([]byte{1}, pack...))
	body.WriteString("0000")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(body.Bytes()); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewRawClient(server.URL + "/repo")
	require.NoError(t, err)

	want, err := hash.FromHex("0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)

	_, err = client.Fetch(t.Context(), FetchOptions{Want: []hash.Hash{want}, Done: true})
	require.ErrorContains(t, err, "reading packfile object 1")
	require.ErrorContains(t, err, "zlib: invalid header")
}
