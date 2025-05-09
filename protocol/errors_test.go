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
				require.Nil(t, got)
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
