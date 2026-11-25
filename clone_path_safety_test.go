package nanogit

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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

func TestWriteBlobToFile_ConfinedToRoot(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	dest := filepath.Join(parent, "dest")
	require.NoError(t, os.MkdirAll(dest, 0o755))

	root, err := os.OpenRoot(dest)
	require.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	c := &httpClient{}
	logger := &log.NoopLogger{}

	t.Run("valid nested path is written", func(t *testing.T) {
		err := c.writeBlobToFile(context.Background(), root, FlatTreeEntry{Path: "a/b/c.txt"}, []byte("hi"), logger)
		require.NoError(t, err)
		got, err := os.ReadFile(filepath.Join(dest, "a", "b", "c.txt"))
		require.NoError(t, err)
		require.Equal(t, "hi", string(got))
	})

	t.Run("dot-dot path is rejected", func(t *testing.T) {
		err := c.writeBlobToFile(context.Background(), root, FlatTreeEntry{Path: "../escape"}, []byte("x"), logger)
		require.Error(t, err)
		require.NoFileExists(t, filepath.Join(parent, "escape"))
	})

	t.Run("pre-existing symlink pointing outside is not followed", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink creation requires privileges on Windows")
		}
		outside := filepath.Join(parent, "outside")
		require.NoError(t, os.MkdirAll(outside, 0o755))
		require.NoError(t, os.Symlink(outside, filepath.Join(dest, "link")))

		err := c.writeBlobToFile(context.Background(), root, FlatTreeEntry{Path: "link/pwn"}, []byte("x"), logger)
		require.Error(t, err)
		require.NoFileExists(t, filepath.Join(outside, "pwn"))
	})
}
