package hash

import (
	"crypto"
	"testing"

	"github.com/grafana/nanogit/protocol/object"
	"github.com/stretchr/testify/require"
)

func TestObject(t *testing.T) {
	tests := []struct {
		name    string
		algo    crypto.Hash
		objType object.Type
		data    []byte
		want    Hash
		wantErr bool
	}{
		{
			name:    "valid sha1 blob",
			algo:    crypto.SHA1,
			objType: object.TypeBlob,
			data:    []byte("test content"),
			// Header: "blob 12\0"
			// Content: "test content"
			// Full object: "blob 12\0test content"
			want:    Hash{0x08, 0xcf, 0x61, 0x01, 0x41, 0x6f, 0x0c, 0xe0, 0xdd, 0xa3, 0xc8, 0x0e, 0x62, 0x7f, 0x33, 0x38, 0x54, 0xc4, 0x08, 0x5c},
			wantErr: false,
		},
		{
			name:    "valid sha1 tree",
			algo:    crypto.SHA1,
			objType: object.TypeTree,
			data:    []byte("100644 test.txt\x00"),
			// Header: "tree 16\0"
			// Content: "100644 test.txt\x00"
			// Full object: "tree 16\0100644 test.txt\x00"
			want:    Hash{0x12, 0x7d, 0xe0, 0x49, 0x11, 0xa6, 0x35, 0xc8, 0x5f, 0xdf, 0x7d, 0xab, 0x6c, 0x78, 0xc6, 0xdd, 0xda, 0xe4, 0x0e, 0xec},
			wantErr: false,
		},
		{
			name:    "valid sha1 commit",
			algo:    crypto.SHA1,
			objType: object.TypeCommit,
			data:    []byte("tree 1234567890abcdef\nparent 0987654321fedcba\nauthor Test <test@example.com>\ncommitter Test <test@example.com>\n\nTest commit\n"),
			// Header: "commit 123\0"
			// Content: "tree 1234567890abcdef\nparent 0987654321fedcba\nauthor Test <test@example.com>\ncommitter Test <test@example.com>\n\nTest commit\n"
			// Full object: "commit 123\0tree 1234567890abcdef\nparent 0987654321fedcba\nauthor Test <test@example.com>\ncommitter Test <test@example.com>\n\nTest commit\n"
			want:    Hash{0x10, 0xe9, 0x0b, 0x93, 0x84, 0x40, 0xae, 0x64, 0x05, 0xbb, 0x30, 0x12, 0xd6, 0x5e, 0xc4, 0x4a, 0x06, 0x6c, 0x2f, 0xef},
			wantErr: false,
		},
		{
			name:    "valid sha256 blob",
			algo:    crypto.SHA256,
			objType: object.TypeBlob,
			data:    []byte("test content"),
			// Header: "blob 12\0"
			// Content: "test content"
			// Full object: "blob 12\0test content"
			want:    Hash{0x18, 0xf2, 0x76, 0x9e, 0xc7, 0x4f, 0xa8, 0x25, 0x6f, 0x4b, 0x1f, 0x8b, 0x9f, 0xa4, 0x6a, 0xe0, 0xb9, 0x8a, 0xf0, 0xef, 0xcb, 0x65, 0x8e, 0x7e, 0x95, 0xe8, 0x00, 0x90, 0xf6, 0x6c, 0x33, 0x8a},
			wantErr: false,
		},
		{
			name:    "unavailable algorithm",
			algo:    99999,
			objType: object.TypeBlob,
			data:    []byte("test content"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty data",
			algo:    crypto.SHA1,
			objType: object.TypeBlob,
			data:    []byte{},
			// Header: "blob 0\0"
			// Content: ""
			// Full object: "blob 0\0"
			want:    Hash{0xe6, 0x9d, 0xe2, 0x9b, 0xb2, 0xd1, 0xd6, 0x43, 0x4b, 0x8b, 0x29, 0xae, 0x77, 0x5a, 0xd8, 0xc2, 0xe4, 0x8c, 0x53, 0x91},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Object(tt.algo, tt.objType, tt.data)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, ErrUnlinkedAlgorithm, err)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				require.Equal(t, tt.want, got, "hash mismatch for %s", tt.name)
			}
		})
	}
}

func TestNewHasher(t *testing.T) {
	tests := []struct {
		name    string
		algo    crypto.Hash
		objType object.Type
		size    int64
		wantErr bool
	}{
		{
			name:    "valid sha1 hasher",
			algo:    crypto.SHA1,
			objType: object.TypeBlob,
			size:    123,
			// Will write header: "blob 123\0"
			wantErr: false,
		},
		{
			name:    "valid sha256 hasher",
			algo:    crypto.SHA256,
			objType: object.TypeTree,
			size:    456,
			// Will write header: "tree 456\0"
			wantErr: false,
		},
		{
			name:    "unavailable algorithm",
			algo:    99999,
			objType: object.TypeBlob,
			size:    123,
			wantErr: true,
		},
		{
			name:    "zero size",
			algo:    crypto.SHA1,
			objType: object.TypeBlob,
			size:    0,
			// Will write header: "blob 0\0"
			wantErr: false,
		},
		{
			name:    "negative size",
			algo:    crypto.SHA1,
			objType: object.TypeBlob,
			size:    -1,
			// Will write header: "blob -1\0"
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewHasher(tt.algo, tt.objType, tt.size)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, ErrUnlinkedAlgorithm, err)
				require.Empty(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got.Hash)
			}
		})
	}
}

func TestHasher_Write(t *testing.T) {
	tests := []struct {
		name    string
		algo    crypto.Hash
		objType object.Type
		size    int64
		data    []byte
		want    Hash
	}{
		{
			name:    "write to sha1 hasher",
			algo:    crypto.SHA1,
			objType: object.TypeBlob,
			size:    12,
			data:    []byte("test content"),
			// Header: "blob 12\0"
			// Content: "test content"
			// Full object: "blob 12\0test content"
			want: Hash{0x08, 0xcf, 0x61, 0x01, 0x41, 0x6f, 0x0c, 0xe0, 0xdd, 0xa3, 0xc8, 0x0e, 0x62, 0x7f, 0x33, 0x38, 0x54, 0xc4, 0x08, 0x5c},
		},
		{
			name:    "write to sha256 hasher",
			algo:    crypto.SHA256,
			objType: object.TypeBlob,
			size:    12,
			data:    []byte("test content"),
			// Header: "blob 12\0"
			// Content: "test content"
			// Full object: "blob 12\0test content"
			want: Hash{0x18, 0xf2, 0x76, 0x9e, 0xc7, 0x4f, 0xa8, 0x25, 0x6f, 0x4b, 0x1f, 0x8b, 0x9f, 0xa4, 0x6a, 0xe0, 0xb9, 0x8a, 0xf0, 0xef, 0xcb, 0x65, 0x8e, 0x7e, 0x95, 0xe8, 0x00, 0x90, 0xf6, 0x6c, 0x33, 0x8a},
		},
		{
			name:    "write empty data",
			algo:    crypto.SHA1,
			objType: object.TypeBlob,
			size:    0,
			data:    []byte{},
			// Header: "blob 0\0"
			// Content: ""
			// Full object: "blob 0\0"
			want: Hash{0xe6, 0x9d, 0xe2, 0x9b, 0xb2, 0xd1, 0xd6, 0x43, 0x4b, 0x8b, 0x29, 0xae, 0x77, 0x5a, 0xd8, 0xc2, 0xe4, 0x8c, 0x53, 0x91},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h, err := NewHasher(tt.algo, tt.objType, tt.size)
			require.NoError(t, err)

			n, err := h.Write(tt.data)
			require.NoError(t, err)
			require.Equal(t, len(tt.data), n)

			// Verify the hash matches the expected value
			sum := h.Sum(nil)
			got := Hash(sum)
			require.Equal(t, tt.want, got, "hash mismatch for %s", tt.name)
		})
	}
}
