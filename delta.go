package nanogit

import (
	"context"
	"fmt"
	"sort"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/protocol/object"
)

// CommitFile represents a file change in a commit
type CommitFile struct {
	Path    string              // Path of the file
	OldPath string              // Original path (for renamed/copied files)
	Mode    uint32              // File mode
	OldMode uint32              // Original mode (for modified files)
	Hash    hash.Hash           // File hash
	OldHash hash.Hash           // Original hash (for modified files)
	Status  protocol.FileStatus // Status of the file
}

// CompareCommits compares two commits and returns the differences
func (c *clientImpl) CompareCommits(ctx context.Context, baseCommit, headCommit hash.Hash) ([]CommitFile, error) {
	// Get both commits
	base, err := c.GetObject(ctx, baseCommit)
	if err != nil {
		return nil, fmt.Errorf("getting base commit: %w", err)
	}
	head, err := c.GetObject(ctx, headCommit)
	if err != nil {
		return nil, fmt.Errorf("getting head commit: %w", err)
	}

	// Get both trees
	baseTree, err := c.GetTree(ctx, base.Commit.Tree)
	if err != nil {
		return nil, fmt.Errorf("getting base tree: %w", err)
	}
	headTree, err := c.GetTree(ctx, head.Commit.Tree)
	if err != nil {
		return nil, fmt.Errorf("getting head tree: %w", err)
	}

	// Compare trees recursively
	var changes []CommitFile
	changes, err = c.compareTrees(baseTree, headTree)
	if err != nil {
		return nil, fmt.Errorf("comparing trees: %w", err)
	}

	return changes, nil
}

// compareTrees recursively compares two trees and collects changes
func (c *clientImpl) compareTrees(base, head *Tree) ([]CommitFile, error) {
	changes := make([]CommitFile, 0)

	inHead := make(map[string]TreeEntry)
	for _, entry := range head.Entries {
		inHead[entry.Path] = entry
	}

	inBase := make(map[string]TreeEntry)
	for _, entry := range base.Entries {
		inBase[entry.Path] = entry
	}

	for _, entry := range head.Entries {
		if _, ok := inBase[entry.Path]; !ok {
			changes = append(changes, CommitFile{
				Path:   entry.Path,
				Status: protocol.FileStatusAdded,
				Mode:   entry.Mode,
				Hash:   entry.Hash,
			})
		} else if inBase[entry.Path].Hash.String() != entry.Hash.String() && entry.Type != object.TypeTree {
			changes = append(changes, CommitFile{
				Path:    entry.Path,
				Status:  protocol.FileStatusModified,
				Mode:    entry.Mode,
				Hash:    entry.Hash,
				OldHash: inBase[entry.Path].Hash,
				OldMode: inBase[entry.Path].Mode,
			})
		}
	}

	for _, entry := range base.Entries {
		if _, ok := inHead[entry.Path]; !ok {
			changes = append(changes, CommitFile{
				Path:   entry.Path,
				Status: protocol.FileStatusDeleted,
				Mode:   entry.Mode,
				Hash:   entry.Hash,
			})
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Path < changes[j].Path
	})

	return changes, nil
}
