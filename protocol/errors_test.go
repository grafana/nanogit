package protocol

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStrError(t *testing.T) {
	tests := []struct {
		name     string
		err      strError
		expected string
	}{
		{
			name:     "simple error message",
			err:      strError("test error"),
			expected: "test error",
		},
		{
			name:     "empty error message",
			err:      strError(""),
			expected: "",
		},
		{
			name:     "error with special characters",
			err:      strError("error: %s\n\tat line 42"),
			expected: "error: %s\n\tat line 42",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.err.Error()
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestStrError_TypeAssertion(t *testing.T) {
	// Test that we can type assert to strError
	var err error = strError("test error")

	// Test type assertion using require.ErrorAs
	var se strError
	require.ErrorAs(t, err, &se, "should be able to get strError using ErrorAs")
	require.Equal(t, "test error", se.Error())
}

func TestEOFIsUnexpected(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected error
	}{
		{
			name:     "io.EOF becomes io.ErrUnexpectedEOF",
			input:    io.EOF,
			expected: io.ErrUnexpectedEOF,
		},
		{
			name:     "wrapped io.EOF becomes io.ErrUnexpectedEOF",
			input:    fmt.Errorf("wrapped: %w", io.EOF),
			expected: io.ErrUnexpectedEOF,
		},
		{
			name:     "other error remains unchanged",
			input:    errors.New("some other error"),
			expected: errors.New("some other error"),
		},
		{
			name:     "nil error remains nil",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := eofIsUnexpected(tt.input)
			if tt.expected == nil {
				require.NoError(t, got)
			} else {
				require.Equal(t, tt.expected.Error(), got.Error())
			}
		})
	}
}

func TestEOFIsUnexpected_ErrorIs(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		check    error
		expected bool
	}{
		{
			name:     "io.EOF becomes io.ErrUnexpectedEOF",
			input:    io.EOF,
			check:    io.ErrUnexpectedEOF,
			expected: true,
		},
		{
			name:     "wrapped io.EOF becomes io.ErrUnexpectedEOF",
			input:    fmt.Errorf("wrapped: %w", io.EOF),
			check:    io.ErrUnexpectedEOF,
			expected: true,
		},
		{
			name:     "other error is not io.ErrUnexpectedEOF",
			input:    errors.New("some other error"),
			check:    io.ErrUnexpectedEOF,
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := eofIsUnexpected(tt.input)
			require.Equal(t, tt.expected, errors.Is(err, tt.check))
		})
	}
}

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
