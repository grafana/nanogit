package nanogit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithLogger(t *testing.T) {
	t.Run("sets custom logger", func(t *testing.T) {
		logger := &testLogger{}
		client, err := NewHTTPClient("https://github.com/owner/repo", WithLogger(logger))
		require.NoError(t, err)

		c, ok := client.(*httpClient)
		require.True(t, ok, "client should be of type *httpClient")
		require.Equal(t, logger, c.logger, "logger should be set to the provided logger")

		// Trigger a log event
		c.logger.Info("test message", "key", "value")
		require.Len(t, logger.entries, 1)
		require.Equal(t, "Info", logger.entries[0].level)
		require.Equal(t, "test message", logger.entries[0].msg)
		require.Equal(t, []any{"key", "value"}, logger.entries[0].args)
	})

	t.Run("returns error if logger is nil", func(t *testing.T) {
		client, err := NewHTTPClient("https://github.com/owner/repo", WithLogger(nil))
		require.Error(t, err)
		require.Nil(t, client)
		require.Equal(t, "logger cannot be nil", err.Error())
	})
}
