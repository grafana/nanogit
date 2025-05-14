package object

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestType_String(t *testing.T) {
	tests := []struct {
		name     string
		objType  Type
		expected string
	}{
		{
			name:     "invalid type",
			objType:  TypeInvalid,
			expected: "OBJ_INVALID",
		},
		{
			name:     "commit type",
			objType:  TypeCommit,
			expected: "OBJ_COMMIT",
		},
		{
			name:     "tree type",
			objType:  TypeTree,
			expected: "OBJ_TREE",
		},
		{
			name:     "blob type",
			objType:  TypeBlob,
			expected: "OBJ_BLOB",
		},
		{
			name:     "tag type",
			objType:  TypeTag,
			expected: "OBJ_TAG",
		},
		{
			name:     "reserved type",
			objType:  TypeReserved,
			expected: "OBJ_RESERVED",
		},
		{
			name:     "offset delta type",
			objType:  TypeOfsDelta,
			expected: "OBJ_OFS_DELTA",
		},
		{
			name:     "ref delta type",
			objType:  TypeRefDelta,
			expected: "OBJ_REF_DELTA",
		},
		{
			name:     "unknown type",
			objType:  Type(255),
			expected: "object.Type(255)",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.objType.String()
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestType_Bytes(t *testing.T) {
	tests := []struct {
		name     string
		objType  Type
		expected []byte
	}{
		{
			name:     "commit type",
			objType:  TypeCommit,
			expected: []byte("commit"),
		},
		{
			name:     "tree type",
			objType:  TypeTree,
			expected: []byte("tree"),
		},
		{
			name:     "blob type",
			objType:  TypeBlob,
			expected: []byte("blob"),
		},
		{
			name:     "tag type",
			objType:  TypeTag,
			expected: []byte("tag"),
		},
		{
			name:     "offset delta type",
			objType:  TypeOfsDelta,
			expected: []byte("ofs-delta"),
		},
		{
			name:     "ref delta type",
			objType:  TypeRefDelta,
			expected: []byte("ref-delta"),
		},
		{
			name:     "invalid type",
			objType:  TypeInvalid,
			expected: []byte("unknown"),
		},
		{
			name:     "reserved type",
			objType:  TypeReserved,
			expected: []byte("unknown"),
		},
		{
			name:     "unknown type",
			objType:  Type(255),
			expected: []byte("unknown"),
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.objType.Bytes()
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestType_Constants(t *testing.T) {
	// Test that the constants have the expected values
	require.Equal(t, TypeInvalid, Type(0))
	require.Equal(t, TypeCommit, Type(1))
	require.Equal(t, TypeTree, Type(2))
	require.Equal(t, TypeBlob, Type(3))
	require.Equal(t, TypeTag, Type(4))
	require.Equal(t, TypeReserved, Type(5))
	require.Equal(t, TypeOfsDelta, Type(6))
	require.Equal(t, TypeRefDelta, Type(7))
}
