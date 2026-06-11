package protocol_test

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
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
				entry, err := pr.ReadObject(context.Background())
				require.NoError(t, err)

				require.NotNil(t, entry.Object)
				require.Nil(t, entry.Trailer)

				require.Equal(t, obj, entry.Object.Type)

			}

			// There should be a trailer here.
			entry, err := pr.ReadObject(context.Background())
			require.NoError(t, err)
			require.Nil(t, entry.Object)
			require.NotNil(t, entry.Trailer)

			_, err = pr.ReadObject(context.Background())
			require.ErrorIs(t, err, io.EOF)
		})
	}
}

func TestReadObject_RefDeltaShortRead(t *testing.T) {
	t.Parallel()

	baseData := []byte("some base object data here")
	baseHash, err := protocol.Object(crypto.SHA1, protocol.ObjectTypeBlob, baseData)
	require.NoError(t, err)

	deltaPayload := []byte{
		byte(len(baseData)), // 1a: source size 26
		5,                   // 05: target size 5
		5,                   // 05: insert the next 5 bytes
		'h', 'e', 'l', 'l', 'o',
	}

	var pack bytes.Buffer
	pack.WriteString("PACK")                                                 // 50 41 43 4b
	require.NoError(t, binary.Write(&pack, binary.BigEndian, uint32(2)))     // 00 00 00 02: version 2
	require.NoError(t, binary.Write(&pack, binary.BigEndian, uint32(2)))     // 00 00 00 02: 2 objects
	pack.Write(objectHeader(protocol.ObjectTypeBlob, len(baseData)))         // ba 01: blob, size 26
	pack.Write(zlibCompress(t, baseData))                                    // zlib stream
	pack.Write(objectHeader(protocol.ObjectTypeRefDelta, len(deltaPayload))) // 78: ref-delta, size 8
	pack.Write(baseHash[:])                                                  // 20-byte base object hash
	pack.Write(zlibCompress(t, deltaPayload))                                // zlib stream
	pack.Write(make([]byte, 20))                                             // trailer checksum

	pr, err := protocol.ParsePackfile(context.Background(), iotest.OneByteReader(bytes.NewReader(pack.Bytes())))
	require.NoError(t, err)

	entry, err := pr.ReadObject(context.Background())
	require.NoError(t, err)
	require.NotNil(t, entry.Object)
	require.Equal(t, protocol.ObjectTypeBlob, entry.Object.Type)
	require.Equal(t, baseHash, entry.Object.Hash)

	entry, err = pr.ReadObject(context.Background())
	require.NoError(t, err)
	require.NotNil(t, entry.Object)
	require.Equal(t, protocol.ObjectTypeRefDelta, entry.Object.Type)
	require.NotNil(t, entry.Object.Delta)
	require.Equal(t, baseHash.String(), entry.Object.Delta.Parent)

	resolved, err := protocol.ApplyDelta(baseData, entry.Object.Delta)
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), resolved)
}

func TestReadObject_TooLarge(t *testing.T) {
	t.Parallel()

	var pack bytes.Buffer
	pack.WriteString("PACK")
	require.NoError(t, binary.Write(&pack, binary.BigEndian, uint32(2)))
	require.NoError(t, binary.Write(&pack, binary.BigEndian, uint32(1)))
	pack.Write(objectHeader(protocol.ObjectTypeBlob, protocol.MaxUnpackedObjectSize+1))

	pr, err := protocol.ParsePackfile(context.Background(), &pack)
	require.NoError(t, err)

	_, err = pr.ReadObject(context.Background())
	require.ErrorIs(t, err, protocol.ErrObjectTooLarge)
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

// TestBuildTreeObject_GitlinkSortsAsFile guards against a class of regression
// where gitlink (submodule, mode 0o160000) entries get sorted with a trailing
// "/" because their mode happens to share the 0o40000 bit with directories.
// Git's canonical tree ordering treats only real directory trees as if they
// had a trailing slash; gitlinks sort as files. A tree built with the wrong
// sort key passes nanogit's local hashing but is rejected by server-side
// fsck on a real Git server (and silently mis-orders entries against any
// other client that follows the spec).
//
// Concrete shape: with names "submod" (gitlink) and "submod-x" (file), the
// correct order is ["submod", "submod-x"] because raw-name comparison places
// the shorter prefix first. The buggy order — appending "/" to the gitlink
// for sorting — would yield "submod/" > "submod-x" because '/' (0x2F) >
// '-' (0x2D), flipping the order to ["submod-x", "submod"].
func TestBuildTreeObject_GitlinkSortsAsFile(t *testing.T) {
	t.Parallel()

	entries := []protocol.PackfileTreeEntry{
		{
			FileName: "submod-x",
			FileMode: 0o100644, // file
			Hash:     "111111dd22095088e7a0ecfcd02b817755f43180",
		},
		{
			FileName: "submod",
			FileMode: 0o160000, // gitlink / submodule
			Hash:     "222222dd22095088e7a0ecfcd02b817755f43181",
		},
		{
			FileName: "submod-dir",
			FileMode: 0o40000, // real directory
			Hash:     "333333dd22095088e7a0ecfcd02b817755f43182",
		},
	}

	treeObj, err := protocol.BuildTreeObject(crypto.SHA1, entries)
	require.NoError(t, err)

	actualOrder := make([]string, len(treeObj.Tree))
	for i, e := range treeObj.Tree {
		actualOrder[i] = e.FileName
	}

	// Expected canonical order:
	//   "submod"      — gitlink, sorted as raw name
	//   "submod-dir/" — real directory, implicit trailing "/" → after
	//                   "submod" because the shorter prefix wins, but
	//                   before "submod-x" because '-' (0x2D) < 'x' (0x78)
	//                   at position 6.
	//   "submod-x"    — file, sorted as raw name
	// The buggy variant (gitlink sorted as "submod/") would push the
	// gitlink after "submod-x" because '/' (0x2F) > '-' (0x2D),
	// producing ["submod-dir", "submod-x", "submod"].
	require.Equal(t,
		[]string{"submod", "submod-dir", "submod-x"},
		actualOrder,
		"gitlink (mode 0o160000) must sort as a file, not as a directory")
}

func TestParseCommit_Signature(t *testing.T) {
	t.Parallel()

	ident := &protocol.Identity{Name: "A", Email: "a@b", Timestamp: 1234567890, Timezone: "+0000"}

	t.Run("round-trips a multi-line gpgsig", func(t *testing.T) {
		t.Parallel()
		signature := "-----BEGIN PGP SIGNATURE-----\n\nwsBcBAABCAAQBQ\nABCDEFGH123456\n-----END PGP SIGNATURE-----"
		c := &protocol.PackfileCommit{Tree: hash.Zero, Parent: hash.Zero, Author: ident, Committer: ident, Message: "signed commit\n", Signature: signature}
		obj := &protocol.PackfileObject{Type: protocol.ObjectTypeCommit, Data: c.Build()}
		require.NoError(t, obj.Parse())
		require.Equal(t, signature, obj.Commit.Signature)
		require.Equal(t, "signed commit\n", obj.Commit.Message)
		require.Equal(t, "a@b", obj.Commit.Author.Email)
		require.Empty(t, obj.Commit.Fields)
	})

	t.Run("unsigned commit has empty signature", func(t *testing.T) {
		t.Parallel()
		c := &protocol.PackfileCommit{Tree: hash.Zero, Parent: hash.Zero, Author: ident, Committer: ident, Message: "no sig\n"}
		obj := &protocol.PackfileObject{Type: protocol.ObjectTypeCommit, Data: c.Build()}
		require.NoError(t, obj.Parse())
		require.Empty(t, obj.Commit.Signature)
		require.Equal(t, "no sig\n", obj.Commit.Message)
	})

	t.Run("gpgsig coexists with other headers", func(t *testing.T) {
		t.Parallel()
		raw := "tree " + hash.Zero.String() + "\n" +
			"author " + ident.String() + "\n" +
			"committer " + ident.String() + "\n" +
			"encoding UTF-8\n" +
			"gpgsig -----BEGIN SSH SIGNATURE-----\n" +
			" AAAAlinetwo\n" +
			" -----END SSH SIGNATURE-----\n" +
			"\n" +
			"body\n"
		obj := &protocol.PackfileObject{Type: protocol.ObjectTypeCommit, Data: []byte(raw)}
		require.NoError(t, obj.Parse())
		require.Equal(t, "-----BEGIN SSH SIGNATURE-----\nAAAAlinetwo\n-----END SSH SIGNATURE-----", obj.Commit.Signature)
		require.Equal(t, []byte("UTF-8"), obj.Commit.Fields["encoding"])
		require.Equal(t, "body\n", obj.Commit.Message)
	})
}

func TestPackfileCommit_Build(t *testing.T) {
	t.Parallel()

	ident := &protocol.Identity{Name: "A", Email: "a@b", Timestamp: 1234567890, Timezone: "+0000"}

	t.Run("omits parent when zero", func(t *testing.T) {
		t.Parallel()
		c := &protocol.PackfileCommit{Tree: hash.Zero, Parent: hash.Zero, Author: ident, Committer: ident, Message: "m\n"}
		require.NotContains(t, string(c.Build()), "parent ")
	})

	t.Run("includes parent when set", func(t *testing.T) {
		t.Parallel()
		parent := hash.MustFromHex("1234567890123456789012345678901234567890")
		c := &protocol.PackfileCommit{Tree: hash.Zero, Parent: parent, Author: ident, Committer: ident, Message: "m\n"}
		require.Contains(t, string(c.Build()), "parent "+parent.String()+"\n")
	})

	t.Run("folds multi-line gpgsig with leading space", func(t *testing.T) {
		t.Parallel()
		c := &protocol.PackfileCommit{Tree: hash.Zero, Parent: hash.Zero, Author: ident, Committer: ident, Message: "m\n", Signature: "line1\nline2"}
		require.Contains(t, string(c.Build()), "gpgsig line1\n line2\n")
	})

	t.Run("BuildUnsigned drops gpgsig", func(t *testing.T) {
		t.Parallel()
		c := &protocol.PackfileCommit{Tree: hash.Zero, Parent: hash.Zero, Author: ident, Committer: ident, Message: "m\n", Signature: "sig"}
		require.NotContains(t, string(c.BuildUnsigned()), "gpgsig")
		require.Contains(t, string(c.Build()), "gpgsig")
	})
}

func TestAddCommit_Signing(t *testing.T) {
	t.Parallel()

	ident := &protocol.Identity{Name: "A", Email: "a@b", Timestamp: 1234567890, Timezone: "+0000"}

	t.Run("nil signer leaves commit unsigned", func(t *testing.T) {
		t.Parallel()
		w := protocol.NewPackfileWriter(crypto.SHA1, protocol.PackfileStorageMemory)
		h, err := w.AddCommit(hash.Zero, hash.Zero, ident, ident, "m\n", nil)
		require.NoError(t, err)

		want := &protocol.PackfileCommit{Tree: hash.Zero, Parent: hash.Zero, Author: ident, Committer: ident, Message: "m\n"}
		wantHash, err := protocol.Object(crypto.SHA1, protocol.ObjectTypeCommit, want.Build())
		require.NoError(t, err)
		require.Equal(t, wantHash, h)
	})

	t.Run("signer embeds signature in hash", func(t *testing.T) {
		t.Parallel()
		w := protocol.NewPackfileWriter(crypto.SHA1, protocol.PackfileStorageMemory)
		h, err := w.AddCommit(hash.Zero, hash.Zero, ident, ident, "m\n", fakeSigner{sig: "fake-sig"})
		require.NoError(t, err)

		want := &protocol.PackfileCommit{Tree: hash.Zero, Parent: hash.Zero, Author: ident, Committer: ident, Message: "m\n", Signature: "fake-sig"}
		wantHash, err := protocol.Object(crypto.SHA1, protocol.ObjectTypeCommit, want.Build())
		require.NoError(t, err)
		require.Equal(t, wantHash, h)
	})

	t.Run("propagates signer error", func(t *testing.T) {
		t.Parallel()
		w := protocol.NewPackfileWriter(crypto.SHA1, protocol.PackfileStorageMemory)
		_, err := w.AddCommit(hash.Zero, hash.Zero, ident, ident, "m\n", fakeSigner{err: errFakeSign})
		require.ErrorIs(t, err, errFakeSign)
	})
}

var errFakeSign = errors.New("fake sign error")

type fakeSigner struct {
	sig string
	err error
}

func (f fakeSigner) Sign([]byte) (string, error) { return f.sig, f.err }

func objectHeader(objType protocol.ObjectType, size int) []byte {
	b := byte(objType)<<4 | byte(size&0xF)
	size >>= 4
	var out []byte
	for size > 0 {
		out = append(out, b|0x80)
		b = byte(size & 0x7F)
		size >>= 7
	}
	return append(out, b)
}

func zlibCompress(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	_, err := zw.Write(data)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}
