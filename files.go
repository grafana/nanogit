package nanogit

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

type File struct {
	Name    string
	Mode    uint32
	Hash    hash.Hash
	Path    string
	Content []byte
}

// GetFile retrieves a file from the repository at the given path
func (c *clientImpl) GetFile(ctx context.Context, hash hash.Hash, path string) (*File, error) {
	tree, err := c.GetTree(ctx, hash)
	if err != nil {
		return nil, err
	}

	// TODO: Is there a way to do this without iterating over all entries?
	for _, entry := range tree.Entries {
		if entry.Path == path {
			content, err := c.GetBlob(ctx, entry.Hash)
			if err != nil {
				return nil, err
			}

			return &File{
				Name:    entry.Name,
				Mode:    entry.Mode,
				Hash:    entry.Hash,
				Path:    entry.Path,
				Content: content,
			}, nil
		}
	}

	return nil, errors.New("file not found")
}

// CreateFile creates a new file in the specified branch.
// It creates a new commit with the file content and updates the branch reference.
func (c *clientImpl) CreateFile(ctx context.Context, ref Ref, path string, content []byte, author Author, committer Committer, message string) error {
	// Create a packfile writer
	writer := protocol.NewPackfileWriter(crypto.SHA1)

	// Create the blob for the file content
	blobHash, err := writer.AddBlob(content)
	if err != nil {
		return fmt.Errorf("creating blob: %w", err)
	}

	parentHash, err := hash.FromHex(ref.Hash)
	if err != nil {
		return fmt.Errorf("parsing parent hash: %w", err)
	}
	currentTree, err := c.GetTree(ctx, parentHash)
	if err != nil {
		return fmt.Errorf("getting current tree: %w", err)
	}

	// get the tree entries and not blobs
	entries := make(map[string]*TreeEntry)
	if currentTree != nil {
		for _, entry := range currentTree.Entries {
			if entry.Type == protocol.ObjectTypeTree {
				entries[entry.Path] = &entry
			}
		}
	}

	if entries[path] != nil {
		return errors.New("file already exists")
	}

	// Create the root tree
	rootTreeHash, err := c.addMissingOrStaleTreeEntries(ctx, path, currentTree.Hash, blobHash, entries, writer)
	if err != nil {
		return fmt.Errorf("creating root tree: %w", err)
	}

	// Create the commit
	authorIdentity := protocol.Identity{
		Name:      author.Name,
		Email:     author.Email,
		Timestamp: author.Time.Unix(),
		Timezone:  author.Time.Format("-0700"),
	}

	committerIdentity := protocol.Identity{
		Name:      committer.Name,
		Email:     committer.Email,
		Timestamp: committer.Time.Unix(),
		Timezone:  committer.Time.Format("-0700"),
	}

	_, err = writer.AddCommit(rootTreeHash, parentHash, &authorIdentity, &committerIdentity, message)
	if err != nil {
		return fmt.Errorf("creating commit: %w", err)
	}

	// Write the packfile
	packfile, err := writer.WritePackfile()
	if err != nil {
		return fmt.Errorf("writing packfile: %w", err)
	}

	// Send the packfile to the server
	_, err = c.receivePack(ctx, packfile)
	if err != nil {
		return fmt.Errorf("sending packfile: %w", err)
	}

	return nil
}

// updateTree updates the tree for the given path.
// It returns the new tree hash
func (c *clientImpl) addMissingOrStaleTreeEntries(ctx context.Context, path string, treeHash hash.Hash, blobHash hash.Hash, entries map[string]*TreeEntry, writer *protocol.PackfileWriter) (hash.Hash, error) {
	// Split the path into parts
	pathParts := strings.Split(path, "/")
	if len(pathParts) == 0 {
		return nil, errors.New("empty path")
	}

	// Get the file name and directory parts
	fileName := pathParts[len(pathParts)-1]
	dirParts := pathParts[:len(pathParts)-1]

	current := protocol.PackfileTreeEntry{
		FileMode: 0o100644,
		FileName: fileName,
		Hash:     blobHash.String(),
	}
	// Iterate bottom up checking if the existing tree.
	// if it does not exist or hash is different, create it with the previous tree entry.
	// if it exists and hash is the same, continue.
	// Add the file to the tree
	for i := len(dirParts) - 1; i >= 0; i-- {
		currentPath := strings.Join(dirParts[:i+1], "/")
		// Check if we already have this tree
		existingEntry, exists := entries[currentPath]
		// build the tree object to compare
		newObj, err := protocol.BuildTreeObject(crypto.SHA1, []protocol.PackfileTreeEntry{current})
		if err != nil {
			return nil, fmt.Errorf("building tree object: %w", err)
		}

		if exists && (existingEntry.Hash.Is(newObj.Hash) || existingEntry.Type != protocol.ObjectTypeTree) {
			return nil, errors.New("this should not be the case")
		}

		// Create new tree
		if !exists {
			// Create new tree
			treeHash, err := writer.AddTree([]protocol.PackfileTreeEntry{current})
			if err != nil {
				return nil, fmt.Errorf("creating tree for %s: %w", currentPath, err)
			}

			c.logger.Debug("add new tree object", "path", currentPath, "hash", treeHash.String(), "child", current.Hash, "child_path", current.FileName)

			// Add this tree to the parent's entries
			current = protocol.PackfileTreeEntry{
				FileMode: 0o40000, // Directory mode
				FileName: dirParts[i],
				Hash:     treeHash.String(),
			}
		} else {
			// If tree exists, add our entries to it
			existingTree, err := c.getObject(ctx, existingEntry.Hash)
			if err != nil {
				return nil, fmt.Errorf("getting existing tree %s: %w", currentPath, err)
			}

			treeHash, err := c.updateTreeEntry(existingTree, current, writer)
			if err != nil {
				return nil, fmt.Errorf("updating tree for %s: %w", currentPath, err)
			}

			c.logger.Debug("add updated tree object", "path", currentPath, "hash", treeHash.String(), "children", len(existingTree.Tree)+1)
			current = protocol.PackfileTreeEntry{
				FileMode: 0o40000, // Directory mode
				FileName: dirParts[i],
				Hash:     treeHash.String(),
			}
		}
	}

	// TODO: we have already fetched this once for building the tree
	originalRoot, err := c.getObject(ctx, treeHash)
	if err != nil {
		return nil, fmt.Errorf("getting root tree: %w", err)
	}

	if originalRoot.Type != protocol.ObjectTypeTree {
		return nil, errors.New("root is not a tree")
	}

	if len(originalRoot.Tree) == 0 {
		rootHash, err := writer.AddTree([]protocol.PackfileTreeEntry{current})
		if err != nil {
			return nil, fmt.Errorf("adding root tree: %w", err)
		}

		return rootHash, nil
	}

	newRootHash, err := c.updateTreeEntry(originalRoot, current, writer)
	if err != nil {
		return nil, fmt.Errorf("updating root tree: %w", err)
	}

	return newRootHash, nil
}

func (c *clientImpl) updateTreeEntry(obj *protocol.PackfileObject, current protocol.PackfileTreeEntry, writer *protocol.PackfileWriter) (hash.Hash, error) {
	if obj.Type != protocol.ObjectTypeTree {
		return nil, errors.New("object is not a tree")
	}

	combinedEntries := make([]protocol.PackfileTreeEntry, len(obj.Tree)+1)
	for i, entry := range obj.Tree {
		combinedEntries[i] = entry
	}

	combinedEntries[len(obj.Tree)] = current

	// Create new tree with combined entries
	return writer.AddTree(combinedEntries)
}
