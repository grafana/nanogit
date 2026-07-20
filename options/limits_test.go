package options

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithLimits(t *testing.T) {
	t.Parallel()

	t.Run("zero value preserves unlimited behavior", func(t *testing.T) {
		resolved, err := Resolve()
		require.NoError(t, err)
		assert.Equal(t, Limits{}, resolved.Limits)
	})

	t.Run("round-trips configured values", func(t *testing.T) {
		want := Limits{
			SingleObjectFetchMaxBytes:   1 << 20,
			MultiObjectFetchMaxBytes:    1 << 30,
			RefsMetadataMaxBytes:        1 << 16,
			ReceivePackResponseMaxBytes: 1 << 16,
		}
		resolved, err := Resolve(WithLimits(want))
		require.NoError(t, err)
		assert.Equal(t, want, resolved.Limits)
	})

	t.Run("negative SingleObjectFetchMaxBytes rejected", func(t *testing.T) {
		_, err := Resolve(WithLimits(Limits{SingleObjectFetchMaxBytes: -1}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SingleObjectFetchMaxBytes")
	})

	t.Run("negative MultiObjectFetchMaxBytes rejected", func(t *testing.T) {
		_, err := Resolve(WithLimits(Limits{MultiObjectFetchMaxBytes: -1}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MultiObjectFetchMaxBytes")
	})

	t.Run("negative RefsMetadataMaxBytes rejected", func(t *testing.T) {
		_, err := Resolve(WithLimits(Limits{RefsMetadataMaxBytes: -1}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "RefsMetadataMaxBytes")
	})

	t.Run("negative ReceivePackResponseMaxBytes rejected", func(t *testing.T) {
		_, err := Resolve(WithLimits(Limits{ReceivePackResponseMaxBytes: -1}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ReceivePackResponseMaxBytes")
	})

	t.Run("WithLimits is composable with other options", func(t *testing.T) {
		// Setting limits must not clobber unrelated fields applied by
		// other options. Regression guard: if WithLimits ever started
		// returning a freshly-built Options instead of mutating, this
		// test would catch the lost UserAgent.
		resolved, err := Resolve(
			WithUserAgent("ua/1"),
			WithLimits(Limits{SingleObjectFetchMaxBytes: 1024}),
		)
		require.NoError(t, err)
		assert.Equal(t, "ua/1", resolved.UserAgent)
		assert.Equal(t, int64(1024), resolved.Limits.SingleObjectFetchMaxBytes)
	})

	t.Run("WithLimits called twice keeps the last value", func(t *testing.T) {
		// Successive WithLimits calls must overwrite, not merge —
		// callers that pass two slices of options expect the latest
		// to win, matching the precedent set by other With* helpers.
		resolved, err := Resolve(
			WithLimits(Limits{SingleObjectFetchMaxBytes: 1}),
			WithLimits(Limits{MultiObjectFetchMaxBytes: 2}),
		)
		require.NoError(t, err)
		assert.Equal(t, int64(0), resolved.Limits.SingleObjectFetchMaxBytes,
			"second WithLimits must overwrite the first, not merge")
		assert.Equal(t, int64(2), resolved.Limits.MultiObjectFetchMaxBytes)
	})

	t.Run("rejection short-circuits subsequent options", func(t *testing.T) {
		// If WithLimits fails validation, options applied after it
		// must not run — Resolve's contract is "first error wins".
		called := false
		tail := func(*Options) error { called = true; return nil }

		_, err := Resolve(WithLimits(Limits{SingleObjectFetchMaxBytes: -1}), tail)
		require.Error(t, err)
		assert.False(t, called, "options after a failing one must not run")
	})
}
