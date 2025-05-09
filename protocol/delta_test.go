package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDeltaHeaderSize tests the parsing of delta headers.
func TestDeltaHeaderSize(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		wantSize    uint64
		wantRemain  []byte
		description string
	}{
		{
			name: "single byte header",
			// 0x7F = 127 (max value for single byte)
			input:       []byte{0x7F, 0x01},
			wantSize:    127,
			wantRemain:  []byte{0x01},
			description: "Tests parsing of a single-byte header with maximum value (127)",
		},
		{
			name: "empty input",
			// Empty input should return 0 size and empty remaining bytes
			input:       []byte{},
			wantSize:    0,
			wantRemain:  []byte{},
			description: "Tests behavior with empty input",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tt.description)

			gotSize, gotRemain := deltaHeaderSize(tt.input)
			require.Equal(t, tt.wantSize, gotSize)
			require.Equal(t, tt.wantRemain, gotRemain)
		})
	}
}
