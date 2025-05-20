package nanogit

import (
	"context"
	"errors"
	"fmt"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/protocol/object"
)

type TreeEntry struct {
	Name string
	Path string
	// Mode is in octal
	Mode uint32
	Hash hash.Hash
	Type object.Type
}

type Tree struct {
	Entries []TreeEntry
	Hash    hash.Hash
}

// GetTree retrieves a tree for a given commit hash
func (c *clientImpl) GetTree(ctx context.Context, commitHash hash.Hash) (*Tree, error) {
	// Format the fetch request
	pkt, err := protocol.FormatPacks(
		protocol.PackLine("command=fetch\n"),
		protocol.PackLine("object-format=sha1\n"),
		protocol.SpecialPack(protocol.DelimeterPacket),
		protocol.PackLine("no-progress\n"),
		protocol.PackLine(fmt.Sprintf("want %s\n", commitHash.String())),
		protocol.PackLine("done\n"),
	)
	if err != nil {
		return nil, fmt.Errorf("formatting packets: %w", err)
	}

	// Send the request
	out, err := c.UploadPack(ctx, pkt)
	if err != nil {
		return nil, fmt.Errorf("sending commands: %w", err)
	}

	// Parse the response
	lines, _, err := protocol.ParsePack(out)
	if err != nil {
		return nil, fmt.Errorf("parsing packet: %w", err)
	}

	response, err := protocol.ParseFetchResponse(lines)
	if err != nil {
		return nil, fmt.Errorf("parsing fetch response: %w", err)
	}

	// Find the commit and tree in the packfile
	var tree *protocol.PackfileObject
	var commit *protocol.PackfileObject
	for {
		obj, err := response.Packfile.ReadObject()
		if err != nil {
			return nil, fmt.Errorf("reading object: %w", err)
		}
		if obj.Object == nil {
			break
		}

		// If we find a commit with our hash, store it
		if obj.Object.Type == object.TypeCommit && obj.Object.Hash.Is(commitHash) {
			commit = obj.Object
			// Request the tree from the commit
			treePkt, err := protocol.FormatPacks(
				protocol.PackLine("command=fetch\n"),
				protocol.PackLine("object-format=sha1\n"),
				protocol.SpecialPack(protocol.DelimeterPacket),
				protocol.PackLine("no-progress\n"),
				protocol.PackLine(fmt.Sprintf("want %s\n", commit.Commit.Tree.String())),
				protocol.PackLine("done\n"),
			)
			if err != nil {
				return nil, fmt.Errorf("formatting tree request: %w", err)
			}

			// Send the tree request
			treeOut, err := c.UploadPack(ctx, treePkt)
			if err != nil {
				return nil, fmt.Errorf("sending tree request: %w", err)
			}

			// Parse the tree response
			treeLines, _, err := protocol.ParsePack(treeOut)
			if err != nil {
				return nil, fmt.Errorf("parsing tree response: %w", err)
			}

			treeResponse, err := protocol.ParseFetchResponse(treeLines)
			if err != nil {
				return nil, fmt.Errorf("parsing tree fetch response: %w", err)
			}

			// Find the tree in the response
			for {
				treeObj, err := treeResponse.Packfile.ReadObject()
				if err != nil {
					return nil, fmt.Errorf("reading tree object: %w", err)
				}
				if treeObj.Object == nil {
					break
				}

				if treeObj.Object.Type == object.TypeTree && treeObj.Object.Hash.Is(commit.Commit.Tree) {
					tree = treeObj.Object
					break
				}
			}
		}

		// If we find a tree with our hash, store it
		if obj.Object.Type == object.TypeTree && obj.Object.Hash.Is(commitHash) {
			tree = obj.Object
		}
	}

	if tree == nil {
		return nil, errors.New("not found")
	}

	// Convert PackfileTreeEntry to TreeEntry
	entries := make([]TreeEntry, len(tree.Tree))
	for i, entry := range tree.Tree {
		hash, err := hash.FromHex(entry.Hash)
		if err != nil {
			return nil, fmt.Errorf("parsing hash: %w", err)
		}

		// Determine the type based on the mode
		entryType := object.TypeBlob
		if entry.FileMode&0o40000 != 0 {
			entryType = object.TypeTree
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
		Hash:    commitHash,
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
		if entry.Type == object.TypeTree {
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
