package nanogit

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
)

func TestDropUnsafePaths(t *testing.T) {
	t.Parallel()

	safe := []string{"README.md", "a/b/c.txt", ".github/workflows/ci.yml", "a/.gitignore", "x..y"}
	unsafe := []string{
		"",
		".",
		"a/..",
		"../escape",
		"sub/../../escape",
		"/abs/escape",
		".git",
		".git/hooks/pre-commit",
		".GIT/config",
		"sub/.Git/config",
	}

	tree := &FlatTree{}
	for _, p := range append(append([]string{}, safe...), unsafe...) {
		tree.Entries = append(tree.Entries, FlatTreeEntry{Path: p, Type: protocol.ObjectTypeBlob})
	}

	got := dropUnsafePaths(tree, &log.NoopLogger{})

	var paths []string
	for _, e := range got.Entries {
		paths = append(paths, e.Path)
	}
	require.Equal(t, safe, paths)
	require.Len(t, tree.Entries, len(safe)+len(unsafe), "input tree must not be modified")
}
