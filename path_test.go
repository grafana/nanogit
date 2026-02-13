package nanogit

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
		errType error
	}{
		{
			name:  "simple file path",
			input: "file.txt",
			want:  "file.txt",
		},
		{
			name:  "nested path",
			input: "dir/subdir/file.txt",
			want:  "dir/subdir/file.txt",
		},
		{
			name:  "trailing slash",
			input: "file.txt/",
			want:  "file.txt",
		},
		{
			name:  "leading slash",
			input: "/file.txt",
			want:  "file.txt",
		},
		{
			name:  "multiple trailing slashes",
			input: "file.txt///",
			want:  "file.txt",
		},
		{
			name:  "multiple leading slashes",
			input: "///file.txt",
			want:  "file.txt",
		},
		{
			name:  "multiple consecutive slashes in middle",
			input: "dir//subdir///file.txt",
			want:  "dir/subdir/file.txt",
		},
		{
			name:  "whitespace trimming",
			input: "  file.txt  ",
			want:  "file.txt",
		},
		{
			name:  "whitespace with slashes",
			input: "  /dir/file.txt/  ",
			want:  "dir/file.txt",
		},
		{
			name:  "empty path",
			input: "",
			want:  "",
		},
		{
			name:  "only slashes",
			input: "///",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   ",
			want:  "",
		},
		{
			name:    "parent reference at start",
			input:   "../outside.txt",
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:    "parent reference in middle",
			input:   "dir/../outside.txt",
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:    "parent reference at end",
			input:   "dir/..",
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:    "multiple parent references",
			input:   "../../outside.txt",
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:  "current directory reference",
			input: "./file.txt",
			want:  "file.txt",
		},
		{
			name:  "multiple current directory references",
			input: "./dir/./file.txt",
			want:  "dir/file.txt",
		},
		{
			name:  "dot alone",
			input: ".",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizePath(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					assert.True(t, errors.Is(err, tt.errType), "expected error type %v, got %v", tt.errType, err)
				}
				var invalidPathErr *InvalidPathError
				assert.True(t, errors.As(err, &invalidPathErr), "expected InvalidPathError")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateBlobPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
		errType error
	}{
		{
			name:  "simple file path",
			input: "file.txt",
			want:  "file.txt",
		},
		{
			name:  "nested path",
			input: "dir/subdir/file.txt",
			want:  "dir/subdir/file.txt",
		},
		{
			name:    "trailing slash rejected (file not directory)",
			input:   "file.txt/",
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:  "multiple slashes normalized",
			input: "dir//file.txt",
			want:  "dir/file.txt",
		},
		{
			name:    "empty path",
			input:   "",
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:    "only slashes",
			input:   "///",
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:    "parent reference",
			input:   "../outside.txt",
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:  "whitespace trimmed",
			input: "  file.txt  ",
			want:  "file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateBlobPath(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					assert.True(t, errors.Is(err, tt.errType), "expected error type %v, got %v", tt.errType, err)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateTreePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
		errType error
	}{
		{
			name:  "simple directory path",
			input: "dir",
			want:  "dir",
		},
		{
			name:  "nested path",
			input: "dir/subdir",
			want:  "dir/subdir",
		},
		{
			name:  "trailing slash removed",
			input: "dir/",
			want:  "dir",
		},
		{
			name:  "multiple slashes normalized",
			input: "dir//subdir",
			want:  "dir/subdir",
		},
		{
			name:  "empty path (root tree)",
			input: "",
			want:  "",
		},
		{
			name:  "only slashes (root tree)",
			input: "///",
			want:  "",
		},
		{
			name:    "parent reference",
			input:   "../outside",
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:  "whitespace trimmed",
			input: "  dir  ",
			want:  "dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateTreePath(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					assert.True(t, errors.Is(err, tt.errType), "expected error type %v, got %v", tt.errType, err)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
