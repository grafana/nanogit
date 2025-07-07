package protocol

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFetchResponse(t *testing.T) {
	// Create a minimal valid packfile
	validPackfile := []byte("PACK" + // signature
		"\x00\x00\x00\x02" + // version 2
		"\x00\x00\x00\x00") // 0 objects

	tests := []struct {
		name    string
		lines   [][]byte
		want    *FetchResponse
		wantErr error
		// Custom assertions to run after basic error checking
		assert func(t *testing.T, got *FetchResponse)
	}{
		{
			name:    "empty response",
			lines:   [][]byte{},
			want:    &FetchResponse{},
			wantErr: nil,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.NotNil(t, got)
				assert.Nil(t, got.Packfile)
				assert.False(t, got.Acks.Nack)
				assert.Nil(t, got.Acks.Acks)
			},
		},
		{
			name: "acknowledgements section with NAK",
			lines: [][]byte{
				[]byte("acknowledgements"),
				[]byte("NAK"),
			},
			want: &FetchResponse{
				Acks: Acknowledgements{
					// TODO: implement
					Nack: false,
					Acks: nil,
				},
			},
			wantErr: nil,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.NotNil(t, got)
				assert.Nil(t, got.Packfile)
				// TODO: implement NAK parsing
				// assert.True(t, got.Acks.Nack)
				// assert.Nil(t, got.Acks.Acks)
			},
		},
		{
			name: "acknowledgements section with multiple ACKs",
			lines: [][]byte{
				[]byte("acknowledgements"),
				[]byte("ACK abc123"),
				[]byte("ACK def456"),
			},
			want: &FetchResponse{
				Acks: Acknowledgements{
					Nack: false,
					// TODO: implement
					Acks: nil,
					// Acks: []string{"abc123", "def456"},
				},
			},
			wantErr: nil,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.NotNil(t, got)
				assert.Nil(t, got.Packfile)
				assert.False(t, got.Acks.Nack)
				// TODO: implement ACK parsing
				// assert.Equal(t, []string{"abc123", "def456"}, got.Acks.Acks)
			},
		},
		{
			name: "packfile section with valid data",
			lines: [][]byte{
				[]byte("packfile"),
				append([]byte{1}, validPackfile...),
			},
			want: &FetchResponse{
				Packfile: &PackfileReader{}, // Will be populated by ParsePackfile
			},
			wantErr: nil,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.NotNil(t, got)
				assert.NotNil(t, got.Packfile)
				assert.False(t, got.Acks.Nack)
				assert.Nil(t, got.Acks.Acks)
			},
		},
		{
			name: "packfile section with progress message",
			lines: [][]byte{
				[]byte("packfile"),
				append([]byte{2}, []byte("progress message")...),
				append([]byte{1}, validPackfile...),
			},
			want: &FetchResponse{
				// Progress messages should be ignored, so no packfile
			},
			wantErr: nil,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.NotNil(t, got)
				assert.NotNil(t, got.Packfile)
				assert.False(t, got.Acks.Nack)
				assert.Nil(t, got.Acks.Acks)
			},
		},
		{
			name: "packfile section with multiple messages",
			lines: [][]byte{
				[]byte("packfile"),
				append([]byte{2}, []byte("progress message")...),
				append([]byte{1}, validPackfile...),
			},
			want: &FetchResponse{
				Packfile: &PackfileReader{}, // Will be populated by ParsePackfile
			},
			wantErr: nil,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.NotNil(t, got)
				assert.NotNil(t, got.Packfile)
				assert.False(t, got.Acks.Nack)
				assert.Nil(t, got.Acks.Acks)
			},
		},
		{
			name: "packfile section with fatal error",
			lines: [][]byte{
				[]byte("packfile"),
				append([]byte{3}, []byte("fatal error")...),
			},
			want:    nil,
			wantErr: FatalFetchError("fatal error"),
			assert: func(t *testing.T, got *FetchResponse) {
				assert.Nil(t, got)
			},
		},
		{
			name: "packfile section with invalid status",
			lines: [][]byte{
				[]byte("packfile"),
				append([]byte{4}, []byte("invalid status")...),
			},
			want:    nil,
			wantErr: ErrInvalidFetchStatus,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.Nil(t, got)
			},
		},
		{
			name: "line too long to be a section header",
			lines: [][]byte{
				[]byte("this is a very long line that exceeds 30 characters and should not be treated as a section header"),
				[]byte("packfile"),
				append([]byte{1}, validPackfile...),
			},
			want: &FetchResponse{
				Packfile: &PackfileReader{}, // Will be populated by ParsePackfile
			},
			wantErr: nil,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.NotNil(t, got)
				assert.NotNil(t, got.Packfile)
				assert.False(t, got.Acks.Nack)
				assert.Nil(t, got.Acks.Acks)
			},
		},
		{
			name: "invalid PACK",
			lines: [][]byte{
				[]byte("packfile"),
				append([]byte{1}, []byte("invalid packfile data")...),
			},
			want:    nil,
			wantErr: ErrNoPackfileSignature,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.Nil(t, got)
			},
		},
		{
			name: "ignored sections",
			lines: [][]byte{
				[]byte("shallow-info"),
				[]byte("wanted-refs"),
			},
			want:    &FetchResponse{},
			wantErr: nil,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.NotNil(t, got)
				assert.Nil(t, got.Packfile)
				assert.False(t, got.Acks.Nack)
				assert.Nil(t, got.Acks.Acks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert lines to proper Git packet format
			var packetData []byte
			for _, line := range tt.lines {
				pkt, err := PackLine(line).Marshal()
				require.NoError(t, err)
				packetData = append(packetData, pkt...)
			}

			parser := NewParser(io.NopCloser(bytes.NewReader(packetData)))
			got, err := ParseFetchResponse(context.Background(), parser)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)

			// Run custom assertions for this test case
			tt.assert(t, got)
		})
	}
}

func TestFatalFetchError(t *testing.T) {
	err := FatalFetchError("test error")
	assert.Equal(t, "test error", err.Error())
}
