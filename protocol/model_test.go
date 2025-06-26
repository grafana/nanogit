package protocol

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFetchResponseStream(t *testing.T) {
	// Create a minimal valid packfile
	validPackfile := []byte("PACK" + // signature
		"\x00\x00\x00\x02" + // version 2
		"\x00\x00\x00\x00") // 0 objects

	tests := []struct {
		name    string
		input   []byte // Raw protocol stream data
		want    *FetchResponse
		wantErr error
		// Custom assertions to run after basic error checking
		assert func(t *testing.T, got *FetchResponse)
	}{
		{
			name:    "empty response",
			input:   []byte{},
			want:    &FetchResponse{},
			wantErr: nil,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.NotNil(t, got)
				assert.NotNil(t, got.Packfile) // Streaming always creates a packfile reader
				assert.False(t, got.Acks.Nack)
				assert.Nil(t, got.Acks.Acks)
			},
		},
		{
			name: "acknowledgements section",
			input: func() []byte {
				ack1, _ := PackLine("acknowledgements").Marshal()
				ack2, _ := PackLine("NAK").Marshal()
				flush := []byte("0000")
				return append(append(ack1, ack2...), flush...)
			}(),
			want: &FetchResponse{
				Acks: Acknowledgements{
					Nack: false, // TODO: implement acknowledgements parsing
					Acks: nil,
				},
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
			name: "packfile section with valid data",
			input: func() []byte {
				header, _ := PackLine("packfile").Marshal()
				packData := append([]byte{1}, validPackfile...)
				packPkt, _ := PackLine(string(packData)).Marshal()
				flush := []byte("0000")
				return append(append(header, packPkt...), flush...)
			}(),
			want:    &FetchResponse{},
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
			input: func() []byte {
				header, _ := PackLine("packfile").Marshal()
				progData := append([]byte{2}, []byte("progress message")...)
				progPkt, _ := PackLine(string(progData)).Marshal()
				flush := []byte("0000")
				return append(append(header, progPkt...), flush...)
			}(),
			want:    &FetchResponse{},
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
			input: func() []byte {
				header, _ := PackLine("packfile").Marshal()
				errorData := append([]byte{3}, []byte("fatal error")...)
				errorPkt, _ := PackLine(string(errorData)).Marshal()
				return append(header, errorPkt...)
			}(),
			want:    &FetchResponse{},
			wantErr: nil,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.NotNil(t, got)
				assert.NotNil(t, got.Packfile)
				// Error will be returned when trying to read objects
				_, err := got.Packfile.ReadObject()
				assert.ErrorIs(t, err, FatalFetchError("fatal error"))
			},
		},
		{
			name: "packfile section with invalid status",
			input: func() []byte {
				header, _ := PackLine("packfile").Marshal()
				invalidData := append([]byte{4}, []byte("invalid status")...)
				invalidPkt, _ := PackLine(string(invalidData)).Marshal()
				return append(header, invalidPkt...)
			}(),
			want:    &FetchResponse{},
			wantErr: nil,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.NotNil(t, got)
				assert.NotNil(t, got.Packfile)
				// Error will be returned when trying to read objects
				_, err := got.Packfile.ReadObject()
				assert.ErrorIs(t, err, ErrInvalidFetchStatus)
			},
		},
		{
			name: "ignored sections",
			input: func() []byte {
				shallow, _ := PackLine("shallow-info").Marshal()
				wanted, _ := PackLine("wanted-refs").Marshal()
				flush := []byte("0000")
				return append(append(shallow, wanted...), flush...)
			}(),
			want:    &FetchResponse{},
			wantErr: nil,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.NotNil(t, got)
				assert.NotNil(t, got.Packfile)
				assert.False(t, got.Acks.Nack)
				assert.Nil(t, got.Acks.Acks)
			},
		},
		{
			name: "line too long to be section header",
			input: func() []byte {
				longLine, _ := PackLine("this is a very long line that exceeds 30 characters and should not be treated as a section header").Marshal()
				header, _ := PackLine("packfile").Marshal()
				packData := append([]byte{1}, validPackfile...)
				packPkt, _ := PackLine(string(packData)).Marshal()
				flush := []byte("0000")
				return append(append(append(longLine, header...), packPkt...), flush...)
			}(),
			want:    &FetchResponse{},
			wantErr: nil,
			assert: func(t *testing.T, got *FetchResponse) {
				assert.NotNil(t, got)
				assert.NotNil(t, got.Packfile)
				assert.False(t, got.Acks.Nack)
				assert.Nil(t, got.Acks.Acks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := io.NopCloser(bytes.NewReader(tt.input))
			got, err := ParseFetchResponse(reader)

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
