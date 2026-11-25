package nanogit

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/storage"
)

func TestFlatten_Paths(t *testing.T) {
	t.Parallel()

	blob := hash.MustFromHex("b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0").String()
	// tree builds a tree object whose hash is derived from id, so each test
	// tree is distinct.
	tree := func(id string, entries ...protocol.PackfileTreeEntry) *protocol.PackfileObject {
		return &protocol.PackfileObject{
			Hash: hash.MustFromHex(strings.Repeat(id, 40)),
			Type: protocol.ObjectTypeTree,
			Tree: entries,
		}
	}
	file := func(name string) protocol.PackfileTreeEntry {
		return protocol.PackfileTreeEntry{FileName: name, FileMode: 0o100644, Hash: blob}
	}
	dir := func(name string, child *protocol.PackfileObject) protocol.PackfileTreeEntry {
		return protocol.PackfileTreeEntry{FileName: name, FileMode: 0o40000, Hash: child.Hash.String()}
	}

	tests := []struct {
		name  string
		trees []*protocol.PackfileObject // trees[0] is the root
		want  []string
	}{
		{
			name: "well-formed tree",
			trees: func() []*protocol.PackfileObject {
				a := tree("1", file("b.txt"))
				return []*protocol.PackfileObject{tree("2", dir("a", a), file("c.txt")), a}
			}(),
			want: []string{"a", "a/b.txt", "c.txt"},
		},
		{
			// The child must not collide with the root README.md.
			name: "empty-named root directory keeps its ancestry",
			trees: func() []*protocol.PackfileObject {
				empty := tree("3", file("README.md"))
				return []*protocol.PackfileObject{tree("4", dir("", empty), file("README.md")), empty}
			}(),
			want: []string{"", "/README.md", "README.md"},
		},
		{
			name: "nested empty-named directories at the root",
			trees: func() []*protocol.PackfileObject {
				inner := tree("5", file("x"))
				outer := tree("6", dir("", inner))
				return []*protocol.PackfileObject{tree("7", dir("", outer)), outer, inner}
			}(),
			want: []string{"", "/", "//x"},
		},
		{
			name: "empty-named directory below the root",
			trees: func() []*protocol.PackfileObject {
				inner := tree("8", file("x"))
				a := tree("9", dir("", inner))
				return []*protocol.PackfileObject{tree("a", dir("a", a)), a, inner}
			}(),
			want: []string{"a", "a/", "a//x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			objects := storage.NewInMemoryStorage(ctx)
			objects.Add(tt.trees...)

			// Every tree is in storage, so flatten never fetches and needs no server.
			flat, _, err := (&httpClient{}).flatten(ctx, tt.trees[0], objects)
			require.NoError(t, err)

			var paths []string
			for _, e := range flat.Entries {
				paths = append(paths, e.Path)
			}
			require.Equal(t, tt.want, paths)
		})
	}
}
