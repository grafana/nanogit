package client

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerUnavailableError(t *testing.T) {
	t.Parallel()

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := errors.New("got status code 500: 500 Internal Server Error")
		err := NewServerUnavailableError(500, underlying)

		unwrapped := errors.Unwrap(err)
		require.Equal(t, underlying, unwrapped, "Unwrap should return the underlying error")
	})

	t.Run("Is enables errors.Is compatibility", func(t *testing.T) {
		t.Parallel()
		underlying := errors.New("got status code 500: 500 Internal Server Error")
		err := NewServerUnavailableError(500, underlying)

		require.True(t, errors.Is(err, ErrServerUnavailable), "errors.Is should find ErrServerUnavailable")
		require.False(t, errors.Is(err, errors.New("different error")), "errors.Is should not match different errors")
	})

	t.Run("Unwrap preserves error chain", func(t *testing.T) {
		t.Parallel()
		underlying := fmt.Errorf("got status code 500: %w", errors.New("Internal Server Error"))
		err := NewServerUnavailableError(500, underlying)

		// Unwrap should return the underlying error
		unwrapped := errors.Unwrap(err)
		require.Equal(t, underlying, unwrapped)

		// Should still be able to check for ErrServerUnavailable
		require.True(t, errors.Is(err, ErrServerUnavailable))

		// Should be able to unwrap further to get the original error
		originalErr := errors.Unwrap(unwrapped)
		require.NotNil(t, originalErr)
		require.Contains(t, originalErr.Error(), "Internal Server Error")
	})

	t.Run("Error message includes status code and underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := errors.New("got status code 500: 500 Internal Server Error")
		err := NewServerUnavailableError(500, underlying)

		msg := err.Error()
		require.Contains(t, msg, "server unavailable")
		require.Contains(t, msg, "status code 500")
		require.Contains(t, msg, underlying.Error())
	})

	t.Run("Error message works with nil underlying error", func(t *testing.T) {
		t.Parallel()
		err := NewServerUnavailableError(503, nil)

		msg := err.Error()
		require.Contains(t, msg, "server unavailable")
		require.Contains(t, msg, "status code 503")
	})
}


