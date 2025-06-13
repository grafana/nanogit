package nanogit

import (
	"testing"

	"github.com/grafana/nanogit/log/mocks"
	"github.com/stretchr/testify/require"
)

func TestWithLogger(t *testing.T) {
	t.Run("sets custom logger", func(t *testing.T) {
		logger := &mocks.FakeLogger{}
		client, err := NewHTTPClient("https://github.com/owner/repo", WithLogger(logger))
		require.NoError(t, err)

		c, ok := client.(*httpClient)
		require.True(t, ok, "client should be of type *httpClient")
		require.Equal(t, logger, c.logger, "logger should be set to the provided logger")

		// Trigger a log event
		c.logger.Info("test message", "key", "value")
		require.Equal(t, 1, logger.InfoCallCount())
		msg, args := logger.InfoArgsForCall(0)
		require.Equal(t, "test message", msg)
		require.Equal(t, []any{"key", "value"}, args)
	})

	t.Run("returns error if logger is nil", func(t *testing.T) {
		client, err := NewHTTPClient("https://github.com/owner/repo", WithLogger(nil))
		require.Error(t, err)
		require.Nil(t, client)
		require.Equal(t, "logger cannot be nil", err.Error())
	})
}
