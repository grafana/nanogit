package nanogit

import (
	"context"
	"errors"
	"fmt"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

type TreeEntry struct {
	Name string
	Path string
	// Mode is in octal
	Mode uint32
	Hash hash.Hash
	Type protocol.ObjectType
}

type Tree struct {
	Entries []TreeEntry
	Hash    hash.Hash
}

// GetTree retrieves a tree for a given commit hash
func (c *clientImpl) GetTree(ctx context.Context, h hash.Hash) (*Tree, error) {
	obj, err := c.getObject(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("getting object: %w", err)
	}

	var tree *protocol.PackfileObject
	if obj.Type == protocol.ObjectTypeCommit && obj.Hash.Is(h) {
		// Find the commit and tree in the packfile
		// TODO: should we make it work for commit object type?
		treeHash, err := hash.FromHex(obj.Commit.Tree.String())
		if err != nil {
			return nil, fmt.Errorf("parsing tree hash: %w", err)
		}

		treeObj, err := c.getObject(ctx, treeHash)
		if err != nil {
			return nil, fmt.Errorf("getting tree: %w", err)
		}
		tree = treeObj
		h = treeHash
	} else if obj.Type == protocol.ObjectTypeTree && obj.Hash.Is(h) {
		tree = obj
	} else {
		return nil, errors.New("not found")
	}

	return c.processTree(ctx, h, tree)
}

func (c *clientImpl) processTree(ctx context.Context, treeHash hash.Hash, tree *protocol.PackfileObject) (*Tree, error) {
	// Convert PackfileTreeEntry to TreeEntry
	entries := make([]TreeEntry, len(tree.Tree))
	for i, entry := range tree.Tree {
		hash, err := hash.FromHex(entry.Hash)
		if err != nil {
			return nil, fmt.Errorf("parsing hash: %w", err)
		}

		// Determine the type based on the mode
		entryType := protocol.ObjectTypeBlob
		if entry.FileMode&0o40000 != 0 {
			entryType = protocol.ObjectTypeTree
		}

		entries[i] = TreeEntry{
			Name: entry.FileName,
			Path: entry.FileName,
			Mode: uint32(entry.FileMode),
			Hash: hash,
			Type: entryType,
		}
	}

	// Process all entries recursively
	result, err := c.processTreeEntries(ctx, entries, "")
	if err != nil {
		return nil, fmt.Errorf("processing tree entries: %w", err)
	}

	if len(result) == 0 {
		return nil, errors.New("tree not found")
	}

	return &Tree{
		Entries: result,
		Hash:    treeHash,
	}, nil
}

// processTreeEntries recursively processes tree entries and builds a flat list
func (c *clientImpl) processTreeEntries(ctx context.Context, entries []TreeEntry, basePath string) ([]TreeEntry, error) {
	result := make([]TreeEntry, 0, len(entries))
	for _, entry := range entries {
		// Build the full path for the entry
		entryPath := entry.Name
		if basePath != "" {
			entryPath = basePath + "/" + entry.Name
		}

		// Update the path for this entry
		entry.Path = entryPath

		// Always add the entry itself
		result = append(result, entry)

		// If this is a tree, recursively process its entries
		if entry.Type == protocol.ObjectTypeTree {
			// Fetch the nested tree
			// TODO: is there a way to avoid fetching the tree again?
			nestedTree, err := c.GetTree(ctx, entry.Hash)
			if err != nil {
				return nil, fmt.Errorf("fetching nested tree %s: %w", entry.Hash, err)
			}

			// Process nested entries with the updated base path
			nestedEntries, err := c.processTreeEntries(ctx, nestedTree.Entries, entryPath)
			if err != nil {
				return nil, fmt.Errorf("processing nested tree entries: %w", err)
			}
			result = append(result, nestedEntries...)
		}
	}

	return result, nil
}
