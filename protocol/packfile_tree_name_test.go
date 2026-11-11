package protocol_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol"
)

// rawTree encodes tree entries in the canonical on-wire format:
// "<octal mode> <name>\x00<20-byte hash>" repeated.
func rawTree(entries ...string) []byte {
	var buf bytes.Buffer
	for _, name := range entries {
		fmt.Fprintf(&buf, "100644 %s", name)
		buf.WriteByte(0)
		buf.Write(bytes.Repeat([]byte{0xab}, 20))
	}
	return buf.Bytes()
}

func TestParseTree_RejectsInvalidEntryNames(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"",
		".",
		"..",
		"a/b",
		"../x",
		"/abs",
		"trailing/",
		".git",
		".GIT",
		".Git",
	} {
		t.Run(fmt.Sprintf("%q", name), func(t *testing.T) {
			t.Parallel()
			obj := &protocol.PackfileObject{
				Type: protocol.ObjectTypeTree,
				Data: rawTree("ok.txt", name),
			}
			err := obj.Parse()
			require.Error(t, err)
			require.ErrorIs(t, err, protocol.ErrInvalidTreeEntryName)
		})
	}
}

func TestParseTree_AcceptsValidEntryNames(t *testing.T) {
	t.Parallel()

	names := []string{
		"file.txt",
		"..foo",
		"foo..",
		"...",
		".gitignore",
		".github",
		".gitmodules",
		"git",
		`a\b`,
		"with space",
		"ünïcödé",
	}
	obj := &protocol.PackfileObject{
		Type: protocol.ObjectTypeTree,
		Data: rawTree(names...),
	}
	require.NoError(t, obj.Parse())
	require.Len(t, obj.Tree, len(names))
	for i, name := range names {
		require.Equal(t, name, obj.Tree[i].FileName)
	}
}
