package metrics_test

import (
	"context"
	"testing"

	"github.com/grafana/nanogit/metrics"
	"github.com/grafana/nanogit/mocks"
	"github.com/stretchr/testify/require"
)

func TestContextRecorder(t *testing.T) {
	t.Run("adds recorder to context", func(t *testing.T) {
		customRecorder := &mocks.FakeRecorder{}
		ctx := context.Background()
		newCtx := metrics.ToContext(ctx, customRecorder)

		// Verify recorder was added to context
		recorder := metrics.FromContext(newCtx)
		require.Equal(t, customRecorder, recorder, "context should contain provided recorder")

		// Verify original context was not modified
		originalRecorder := metrics.FromContext(ctx)
		require.NotEqual(t, customRecorder, originalRecorder, "original context should not be modified")
	})

	t.Run("returns noop recorder if no recorder in context", func(t *testing.T) {
		ctx := context.Background()
		recorder := metrics.FromContext(ctx)
		require.NotNil(t, recorder, "should return noop recorder")
		require.IsType(t, &metrics.NoopRecorder{}, recorder, "should return noop recorder")
	})
}
