package log_test

import (
	"context"
	"testing"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/log/mocks"
	"github.com/stretchr/testify/require"
)

func TestWithContextLogger(t *testing.T) {
	t.Run("adds logger to context", func(t *testing.T) {
		customLogger := &mocks.FakeLogger{}
		ctx := context.Background()
		newCtx := log.WithContextLogger(ctx, customLogger)

		// Verify logger was added to context
		logger := log.GetContextLogger(newCtx)
		require.Equal(t, customLogger, logger, "context should contain provided logger")

		// Verify original context was not modified
		originalLogger := log.GetContextLogger(ctx)
		require.NotEqual(t, customLogger, originalLogger, "original context should not be modified")
	})

	t.Run("returns nil logger if no logger in context", func(t *testing.T) {
		ctx := context.Background()
		logger := log.GetContextLogger(ctx)
		require.Nil(t, logger, "should return nil logger")
	})
}
