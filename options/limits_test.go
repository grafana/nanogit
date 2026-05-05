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
			SingleObjectFetch:   1 << 20,
			MultiObjectFetch:    1 << 30,
			RefsMetadata:        1 << 16,
			ReceivePackResponse: 1 << 16,
		}
		resolved, err := Resolve(WithLimits(want))
		require.NoError(t, err)
		assert.Equal(t, want, resolved.Limits)
	})

	t.Run("negative SingleObjectFetch rejected", func(t *testing.T) {
		_, err := Resolve(WithLimits(Limits{SingleObjectFetch: -1}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SingleObjectFetch")
	})

	t.Run("negative MultiObjectFetch rejected", func(t *testing.T) {
		_, err := Resolve(WithLimits(Limits{MultiObjectFetch: -1}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MultiObjectFetch")
	})

	t.Run("negative RefsMetadata rejected", func(t *testing.T) {
		_, err := Resolve(WithLimits(Limits{RefsMetadata: -1}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "RefsMetadata")
	})

	t.Run("negative ReceivePackResponse rejected", func(t *testing.T) {
		_, err := Resolve(WithLimits(Limits{ReceivePackResponse: -1}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ReceivePackResponse")
	})
}
