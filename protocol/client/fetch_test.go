package client

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/grafana/nanogit/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllWantedObjectsCollected(t *testing.T) {
	t.Parallel()

	t.Run("nil wanted map returns false", func(t *testing.T) {
		// The caller's "early termination" branches MUST stay
		// disabled when NoExtraObjects was not set on the fetch —
		// otherwise we'd silently break out of any fetch the moment
		// a transport-level problem appeared, hiding real errors.
		assert.False(t, allWantedObjectsCollected(nil, map[string]*protocol.PackfileObject{}))
	})

	t.Run("empty wanted map returns false", func(t *testing.T) {
		assert.False(t, allWantedObjectsCollected(map[string]bool{}, map[string]*protocol.PackfileObject{}))
	})

	t.Run("missing object returns false", func(t *testing.T) {
		wanted := map[string]bool{"a": true, "b": true}
		got := map[string]*protocol.PackfileObject{"a": {}}
		assert.False(t, allWantedObjectsCollected(wanted, got))
	})

	t.Run("all present returns true", func(t *testing.T) {
		wanted := map[string]bool{"a": true, "b": true}
		got := map[string]*protocol.PackfileObject{"a": {}, "b": {}}
		assert.True(t, allWantedObjectsCollected(wanted, got))
	})

	t.Run("extra collected entries do not affect the result", func(t *testing.T) {
		wanted := map[string]bool{"a": true}
		got := map[string]*protocol.PackfileObject{"a": {}, "b": {}, "c": {}}
		assert.True(t, allWantedObjectsCollected(wanted, got))
	})
}

func TestClassifyReadObjectErr(t *testing.T) {
	t.Parallel()

	t.Run("non-cap error is tolerated as natural EOF", func(t *testing.T) {
		// zlib / unexpected-EOF / malformed-delta errors keep the
		// pre-existing tolerance: returning nil here lets the caller
		// break out of the read loop and finalize whatever was
		// already collected. Tightening this is a separate refactor.
		err := classifyReadObjectErr(io.ErrUnexpectedEOF, nil, nil)
		assert.NoError(t, err)
	})

	t.Run("cap error propagates when wanted objects are missing", func(t *testing.T) {
		// The DoS-protection contract: under the configured cap a
		// truncation must look distinct from a successful partial
		// fetch. With wanted objects still pending, the cap error
		// has to bubble up so the caller can tell a too-tight cap
		// apart from a real ObjectNotFound.
		capErr := &ErrResponseTooLarge{Limit: 100, Op: "fetch"}
		wanted := map[string]bool{"a": true}
		got := map[string]*protocol.PackfileObject{}

		err := classifyReadObjectErr(capErr, wanted, got)
		require.Error(t, err)

		var tooLarge *ErrResponseTooLarge
		require.True(t, errors.As(err, &tooLarge))
		assert.Equal(t, "fetch", tooLarge.Op)
		assert.Equal(t, int64(100), tooLarge.Limit)
	})

	t.Run("cap error is swallowed when every wanted object is already collected", func(t *testing.T) {
		// The exception that motivates this helper: a NoExtraObjects
		// fetch that has every requested object in hand only hits
		// the cap on bytes the caller doesn't need. Returning an
		// error there would turn successful single-object lookups
		// into false negatives under servers that over-send.
		capErr := &ErrResponseTooLarge{Limit: 100, Op: "fetch"}
		wanted := map[string]bool{"a": true, "b": true}
		got := map[string]*protocol.PackfileObject{"a": {}, "b": {}}

		err := classifyReadObjectErr(capErr, wanted, got)
		assert.NoError(t, err)
	})

	t.Run("wrapped cap error is still recognized", func(t *testing.T) {
		// The packfile reader may wrap with fmt.Errorf("...: %w", err);
		// classifyReadObjectErr must use errors.As so the swallow /
		// propagate decision is robust to the wrapping layer.
		capErr := &ErrResponseTooLarge{Limit: 50, Op: "fetch"}
		wrapped := fmt.Errorf("read object: %w", capErr)

		// Wanted hash present → no error
		wanted := map[string]bool{"x": true}
		got := map[string]*protocol.PackfileObject{"x": {}}
		assert.NoError(t, classifyReadObjectErr(wrapped, wanted, got))

		// Wanted hash absent → propagate
		err := classifyReadObjectErr(wrapped, wanted, map[string]*protocol.PackfileObject{})
		require.Error(t, err)
		var tooLarge *ErrResponseTooLarge
		assert.True(t, errors.As(err, &tooLarge))
	})
}
