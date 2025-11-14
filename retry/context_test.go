package retry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextHelpers(t *testing.T) {
	t.Parallel()

	t.Run("ToContext and FromContext", func(t *testing.T) {
		ctx := context.Background()
		retrier := NewExponentialBackoffRetrier()

		ctx = ToContext(ctx, retrier)
		retrieved := FromContext(ctx)

		require.Equal(t, retrier, retrieved)
	})

	t.Run("FromContext returns nil when no retrier", func(t *testing.T) {
		ctx := context.Background()
		retrieved := FromContext(ctx)

		require.Nil(t, retrieved)
	})

	t.Run("FromContextOrNoop returns retrier from context", func(t *testing.T) {
		ctx := context.Background()
		retrier := NewExponentialBackoffRetrier()
		ctx = ToContext(ctx, retrier)

		retrieved := FromContextOrNoop(ctx)
		require.Equal(t, retrier, retrieved)
	})

	t.Run("FromContextOrNoop returns NoopRetrier when none set", func(t *testing.T) {
		ctx := context.Background()
		retrieved := FromContextOrNoop(ctx)

		require.IsType(t, &NoopRetrier{}, retrieved)
	})

	t.Run("context isolation", func(t *testing.T) {
		ctx1 := context.Background()
		ctx2 := context.Background()

		retrier1 := NewExponentialBackoffRetrier().WithMaxAttempts(1)
		retrier2 := NewExponentialBackoffRetrier().WithMaxAttempts(2)

		ctx1 = ToContext(ctx1, retrier1)
		ctx2 = ToContext(ctx2, retrier2)

		require.Equal(t, retrier1, FromContext(ctx1))
		require.Equal(t, retrier2, FromContext(ctx2))
		require.NotEqual(t, FromContext(ctx1), FromContext(ctx2))
	})
}

