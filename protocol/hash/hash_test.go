package hash

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromHex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Hash
		wantErr bool
	}{
		{
			name:    "valid hex string",
			input:   "0123456789abcdef0123456789abcdef01234567",
			want:    Hash{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67},
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    Zero,
			wantErr: false,
		},
		{
			name:    "invalid hex string",
			input:   "invalid",
			want:    Zero,
			wantErr: true,
		},
		{
			name:    "wrong length hex string",
			input:   "0123456789abcdef",
			want:    Zero,
			wantErr: true,
		},
		{
			name:    "non-hex characters",
			input:   "ghijklmn",
			want:    Zero,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := FromHex(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, Zero, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestHash_String(t *testing.T) {
	tests := []struct {
		name string
		h    Hash
		want string
	}{
		{
			name: "valid hash",
			h:    Hash{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67},
			want: "0123456789abcdef0123456789abcdef01234567",
		},
		{
			name: "zero hash",
			h:    Zero,
			want: "0000000000000000000000000000000000000000",
		},
		{
			name: "single byte set",
			h:    Hash{0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			want: "ff00000000000000000000000000000000000000",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.h.String()
			require.Equal(t, tt.want, got)
		})
	}
}

func TestHash_Is(t *testing.T) {
	tests := []struct {
		name string
		h1   Hash
		h2   Hash
		want bool
	}{
		{
			name: "equal hashes",
			h1:   Hash{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67},
			h2:   Hash{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67},
			want: true,
		},
		{
			name: "different hashes",
			h1:   Hash{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67},
			h2:   Hash{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x68},
			want: false,
		},
		{
			name: "zero hashes",
			h1:   Hash{},
			h2:   Hash{},
			want: true,
		},
		{
			name: "zero hash comparison",
			h1:   Zero,
			h2:   Zero,
			want: true,
		},
		{
			name: "zero hash with empty hash",
			h1:   Zero,
			h2:   Hash{},
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.h1.Is(tt.h2)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMustFromHex(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      Hash
		wantPanic bool
	}{
		{
			name:      "valid hex string",
			input:     "0123456789abcdef0123456789abcdef01234567",
			want:      Hash{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67},
			wantPanic: false,
		},
		{
			name:      "invalid hex string",
			input:     "invalid",
			want:      Zero,
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantPanic {
				require.Panics(t, func() {
					MustFromHex(tt.input)
				})
				return
			}
			got := MustFromHex(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}
