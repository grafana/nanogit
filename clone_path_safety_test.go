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

func TestWriteFilesToDisk_RejectsEscapingPaths(t *testing.T) {
	t.Parallel()

	for _, p := range []string{
		"../escape",
		"sub/../../escape",
		"/abs/escape",
		"",
	} {
		t.Run(p, func(t *testing.T) {
			t.Parallel()
			parent := t.TempDir()
			dest := filepath.Join(parent, "dest")

			c := &httpClient{}
			tree := &FlatTree{Entries: []FlatTreeEntry{{
				Path: p,
				Type: protocol.ObjectTypeBlob,
			}}}

			// The check must fail before any blob is fetched, so a
			// zero-value client (no transport) is sufficient.
			err := c.writeFilesToDisk(context.Background(), dest, tree, 0, 0)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrInvalidTreeEntryName)
			require.NoFileExists(t, filepath.Join(parent, "escape"))
		})
	}
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
