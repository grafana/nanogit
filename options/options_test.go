package options

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	t.Run("seeds HTTPClient so in-place option mutations do not nil-deref", func(t *testing.T) {
		// Regression: Resolve must hand each Option a non-nil HTTPClient so
		// options that tune the default (e.g., setting a timeout) work even
		// when the caller has not supplied WithHTTPClient.
		setTimeout := func(o *Options) error {
			o.HTTPClient.Timeout = 7 * time.Second
			return nil
		}
		resolved, err := Resolve(setTimeout)
		require.NoError(t, err)
		require.NotNil(t, resolved.HTTPClient)
		assert.Equal(t, 7*time.Second, resolved.HTTPClient.Timeout)
	})

	t.Run("callers can replace HTTPClient entirely", func(t *testing.T) {
		custom := &http.Client{Timeout: 11 * time.Second}
		resolved, err := Resolve(WithHTTPClient(custom))
		require.NoError(t, err)
		assert.Same(t, custom, resolved.HTTPClient)
	})

	t.Run("nil options are skipped", func(t *testing.T) {
		resolved, err := Resolve(nil, WithUserAgent("ua"), nil)
		require.NoError(t, err)
		assert.Equal(t, "ua", resolved.UserAgent)
	})

	t.Run("first error short-circuits the remaining options", func(t *testing.T) {
		boom := errors.New("boom")
		called := false
		failing := func(*Options) error { return boom }
		tail := func(*Options) error { called = true; return nil }

		resolved, err := Resolve(failing, tail)
		require.ErrorIs(t, err, boom)
		assert.Nil(t, resolved)
		assert.False(t, called, "options after a failing one must not run")
	})
}
