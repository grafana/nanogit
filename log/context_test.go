package log_test

import (
	"context"
	"testing"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/log/mocks"
	"github.com/stretchr/testify/require"
)

func TestContextLogger(t *testing.T) {
	t.Run("adds logger to context", func(t *testing.T) {
		customLogger := &mocks.FakeLogger{}
		ctx := context.Background()
		newCtx := log.ToContext(ctx, customLogger)

		// Verify logger was added to context
		logger := log.FromContext(newCtx)
		require.Equal(t, customLogger, logger, "context should contain provided logger")

		// Verify original context was not modified
		originalLogger := log.FromContext(ctx)
		require.NotEqual(t, customLogger, originalLogger, "original context should not be modified")
	})

	t.Run("returns noop logger if no logger in context", func(t *testing.T) {
		ctx := context.Background()
		logger := log.FromContext(ctx)
		require.NotNil(t, logger, "should return noop logger")
		require.IsType(t, &log.NoopLogger{}, logger, "should return noop logger")
	})
}
