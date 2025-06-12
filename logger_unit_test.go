package nanogit

import (
	"context"
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

func TestWithContextLogger(t *testing.T) {
	t.Run("adds logger to context", func(t *testing.T) {
		customLogger := &testLogger{}
		ctx := context.Background()
		newCtx := WithContextLogger(ctx, customLogger)

		// Verify logger was added to context
		logger := getContextLogger(newCtx)
		require.Equal(t, customLogger, logger, "context should contain provided logger")

		// Verify original context was not modified
		originalLogger := getContextLogger(ctx)
		require.NotEqual(t, customLogger, originalLogger, "original context should not be modified")
	})

	t.Run("returns nil logger if no logger in context", func(t *testing.T) {
		ctx := context.Background()
		logger := getContextLogger(ctx)
		require.Nil(t, logger, "should return nil logger")
	})
}
