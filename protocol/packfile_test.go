package protocol_test

import (
	"bytes"
	"context"
	"crypto"
	"io"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol"
)

func TestParsePackfile(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		input         []byte
		expectedError error
	}{
		"empty": {
			input:         []byte{},
			expectedError: protocol.ErrNoPackfileSignature,
		},
		"no signature": {
			input:         []byte("HELO"),
			expectedError: protocol.ErrNoPackfileSignature,
		},
		"truncated": {
			input:         []byte("PA"),
			expectedError: protocol.ErrNoPackfileSignature,
		},
		"empty version 2": {
			input: []byte("PACK" +
				"\x00\x00\x00\x02" +
				"\x00\x00\x00\x00"),
		},
		"empty version 3": {
			input: []byte("PACK" +
				"\x00\x00\x00\x03" +
				"\x00\x00\x00\x00"),
		},
		"invalid version": {
			input: []byte("PACK" +
				"\x00\x00\x00\x04" +
				"\x00\x00\x00\x00"),
			expectedError: protocol.ErrUnsupportedPackfileVersion,
		},
		"valid": {
			input: []byte("PACK" +
				"\x00\x00\x00\x02" +
				"\x00\x00\x00\x01"),
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := protocol.ParsePackfile(context.Background(), bytes.NewReader(tc.input))
			require.ErrorIs(t, err, tc.expectedError)

			// We don't really have a way to validate that the
			// number of objects field was read correctly.
		})
	}
}

func TestGolden(t *testing.T) {
	testcases := map[string]struct {
		expectedObjects []protocol.ObjectType
	}{
		"simple.dat": {
			expectedObjects: []protocol.ObjectType{
				protocol.ObjectTypeTree,
				protocol.ObjectTypeCommit,
			},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data := loadGolden(t, name)
			pr, err := protocol.ParsePackfile(context.Background(), bytes.NewReader(data))
			require.NoError(t, err)

			for _, obj := range tc.expectedObjects {
				entry, err := pr.ReadObject()
				require.NoError(t, err)

				require.NotNil(t, entry.Object)
				require.Nil(t, entry.Trailer)

				require.Equal(t, obj, entry.Object.Type)

			}

			// There should be a trailer here.
			entry, err := pr.ReadObject()
			require.NoError(t, err)
			require.Nil(t, entry.Object)
			require.NotNil(t, entry.Trailer)

			_, err = pr.ReadObject()
			require.ErrorIs(t, err, io.EOF)
		})
	}
}

func loadGolden(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(path.Join("testdata", name))
	require.NoError(t, err)

	return data
}

func TestBuildTreeObject_DirectoryFileSorting(t *testing.T) {
	t.Parallel()

	// Test case based on the problematic tree structure where "robertoonboarding" directory
	// should be sorted correctly among other entries
	testCases := []struct {
		name          string
		entries       []protocol.PackfileTreeEntry
		expectedOrder []string
	}{
		{
			name: "directory_and_file_with_similar_names",
			entries: []protocol.PackfileTreeEntry{
				{
					FileName: "robertoonboarding",
					FileMode: 0o40000, // directory
					Hash:     "68cff12dd22095088e7a0ecfcd02b817755f4318",
				},
				{
					FileName: "repofolder",
					FileMode: 0o40000, // directory
					Hash:     "abcdef0d22095088e7a0ecfcd02b817755f43181",
				},
				{
					FileName: "README.md",
					FileMode: 0o100644, // file
					Hash:     "123456dd22095088e7a0ecfcd02b817755f43182",
				},
			},
			expectedOrder: []string{"README.md", "repofolder", "robertoonboarding"},
		},
		{
			name: "complex_directory_structure",
			entries: []protocol.PackfileTreeEntry{
				{
					FileName: "another-one.json",
					FileMode: 0o100644, // file
					Hash:     "aaa111dd22095088e7a0ecfcd02b817755f43180",
				},
				{
					FileName: "dir1",
					FileMode: 0o40000, // directory
					Hash:     "bbb222dd22095088e7a0ecfcd02b817755f43181",
				},
				{
					FileName: "example.json",
					FileMode: 0o100644, // file
					Hash:     "ccc333dd22095088e7a0ecfcd02b817755f43182",
				},
				{
					FileName: "finaltest",
					FileMode: 0o40000, // directory
					Hash:     "ddd444dd22095088e7a0ecfcd02b817755f43183",
				},
				{
					FileName: "grafana",
					FileMode: 0o40000, // directory
					Hash:     "eee555dd22095088e7a0ecfcd02b817755f43184",
				},
				{
					FileName: "legacy",
					FileMode: 0o40000, // directory
					Hash:     "fff666dd22095088e7a0ecfcd02b817755f43185",
				},
				{
					FileName: "legacy-dashboard.json",
					FileMode: 0o100644, // file
					Hash:     "777777dd22095088e7a0ecfcd02b817755f43186",
				},
				{
					FileName: "robertoonboarding",
					FileMode: 0o40000, // directory
					Hash:     "888888dd22095088e7a0ecfcd02b817755f43187",
				},
				{
					FileName: "whatever",
					FileMode: 0o40000, // directory
					Hash:     "999999dd22095088e7a0ecfcd02b817755f43188",
				},
			},
			// Expected order: alphabetical, but directories treated as if they have trailing /
			expectedOrder: []string{
				"another-one.json",
				"dir1",
				"example.json",
				"finaltest",
				"grafana",
				"legacy-dashboard.json",
				"legacy",
				"robertoonboarding",
				"whatever",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Build the tree object
			treeObj, err := protocol.BuildTreeObject(crypto.SHA1, tc.entries)
			require.NoError(t, err)

			// Extract the order of filenames from the built tree
			actualOrder := make([]string, len(treeObj.Tree))
			for i, entry := range treeObj.Tree {
				actualOrder[i] = entry.FileName
			}

			// Verify the order matches expected
			require.Equal(t, tc.expectedOrder, actualOrder,
				"Tree entries should be sorted according to Git specification")

			// Verify the tree object was built successfully with a valid hash
			require.NotEqual(t, "", treeObj.Hash.String())
			require.Equal(t, protocol.ObjectTypeTree, treeObj.Type)
		})
	}
}
