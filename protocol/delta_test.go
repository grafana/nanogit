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

func TestParseDelta(t *testing.T) {
	tests := []struct {
		name        string
		parent      string
		payload     []byte
		want        *Delta
		wantErr     error
		description string
	}{
		{
			name:   "payload too short",
			parent: "parent123",
			payload: []byte{
				0x01, // Source size (1 byte)
				0x02, // Target size (1 byte)
			},
			wantErr:     strError("payload too short"),
			description: "Tests error when payload is shorter than minimum required size",
		},
		{
			name:   "reserved cmd 0x0",
			parent: "parent123",
			payload: []byte{
				0x01, // Source size (1 byte)
				0x02, // Target size (1 byte)
				0x00, // Reserved command byte
				0x00, // Extra byte to meet minDeltaSize
			},
			wantErr:     strError("payload included a cmd 0x0 (reserved) instruction"),
			description: "Tests error when command byte is 0x0 (reserved)",
		},
		// TODO: add more cases once we understand the logic better and we have integration test coverage.
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tt.description)

			got, err := parseDelta(tt.parent, tt.payload)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
