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

func (c *clientImpl) NewRefWriter(ctx context.Context, ref Ref) (RefWriter, error) {
	hash, err := hash.FromHex(ref.Hash)
	if err != nil {
		return nil, fmt.Errorf("parsing ref hash: %w", err)
	}

	commit, err := c.GetCommit(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("getting root tree: %w", err)
	}

	treeObj, err := c.getObject(ctx, commit.Tree)
	if err != nil {
		return nil, fmt.Errorf("getting tree object: %w", err)
	}

	if treeObj.Type != protocol.ObjectTypeTree {
		return nil, errors.New("root is not a tree")
	}

	cache := make(map[string]*protocol.PackfileObject)
	cache[treeObj.Hash.String()] = treeObj

	// TODO: pass function to cache
	currentTree, err := c.processTree(ctx, commit.Tree, treeObj)
	if err != nil {
		return nil, fmt.Errorf("getting current tree: %w", err)
	}

	entries := make(map[string]*TreeEntry, len(currentTree.Entries))
	for _, entry := range currentTree.Entries {
		entries[entry.Path] = &entry
	}

	// Create a packfile writer
	writer := protocol.NewPackfileWriter(crypto.SHA1)
	return &refWriter{
		clientImpl:   c,
		ref:          ref,
		writer:       writer,
		lastCommit:   commit,
		lastTreeHash: commit.Tree,
	}, nil
}

type refWriter struct {
	*clientImpl
	ref          Ref
	writer       *protocol.PackfileWriter
	lastCommit   *Commit
	lastTreeHash hash.Hash
	treeCache    map[string]*protocol.PackfileObject
}

// CreateBlob creates a new blob in the specified path.
func (w *refWriter) CreateBlob(ctx context.Context, path string, content []byte) (hash.Hash, error) {
	// Create the blob for the file content
	blobHash, err := w.writer.AddBlob(content)
	if err != nil {
		return nil, fmt.Errorf("creating blob: %w", err)
	}
	w.logger.Debug("created blob", "hash", blobHash.String())

	// TODO: what should we do with the the in-memory tree?
	// We may not need the entire tree but only this path.
	currentTree, err := w.GetTree(ctx, w.lastCommit.Tree)
	if err != nil {
		return nil, fmt.Errorf("getting current tree: %w", err)
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
		return nil, errors.New("file already exists")
	}

	if err := w.addMissingOrStaleTreeEntries(ctx, path, blobHash, entries); err != nil {
		return nil, fmt.Errorf("creating root tree: %w", err)
	}

	return blobHash, nil
}

func (w *refWriter) UpdateBlob(ctx context.Context, path string, content []byte) (hash.Hash, error) {
	return nil, nil
}

func (w *refWriter) DeleteBlob(ctx context.Context, path string) (hash.Hash, error) {
	return nil, nil
}

func (w *refWriter) Commit(ctx context.Context, message string, author Author, committer Committer) (*Commit, error) {
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

	commitHash, err := w.writer.AddCommit(w.lastTreeHash, w.lastCommit.Hash, &authorIdentity, &committerIdentity, message)
	if err != nil {
		return nil, fmt.Errorf("creating commit: %w", err)
	}

	w.lastCommit = &Commit{
		Hash:      commitHash,
		Tree:      w.lastTreeHash,
		Parent:    w.lastCommit.Hash,
		Author:    author,
		Committer: committer,
		Message:   message,
	}

	return w.lastCommit, nil
}

func (w *refWriter) Push(ctx context.Context) error {
	// TODO: write in chunks and not having all bytes in memory
	// Write the packfile
	packfile, err := w.writer.WritePackfile()
	if err != nil {
		return fmt.Errorf("writing packfile: %w", err)
	}

	// Send the packfile to the server
	if _, err := w.receivePack(ctx, packfile); err != nil {
		return fmt.Errorf("sending packfile: %w", err)
	}

	// Reset things to accumulate things for next push
	w.writer = protocol.NewPackfileWriter(crypto.SHA1)

	return nil
}

// updateTree updates the tree for the given path.
// It returns the new tree hash
func (w *refWriter) addMissingOrStaleTreeEntries(ctx context.Context, path string, blobHash hash.Hash, entries map[string]*TreeEntry) error {
	// Split the path into parts
	pathParts := strings.Split(path, "/")
	if len(pathParts) == 0 {
		return errors.New("empty path")
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
			return fmt.Errorf("building tree object: %w", err)
		}

		if exists && (existingEntry.Hash.Is(newObj.Hash) || existingEntry.Type != protocol.ObjectTypeTree) {
			return errors.New("this should not be the case")
		}

		// Create new tree
		if !exists {
			// Create new tree
			treeHash, err := w.writer.AddTree([]protocol.PackfileTreeEntry{current})
			if err != nil {
				return fmt.Errorf("creating tree for %s: %w", currentPath, err)
			}

			w.logger.Debug("add new tree object", "path", currentPath, "hash", treeHash.String(), "child", current.Hash, "child_path", current.FileName)

			// Add this tree to the parent's entries
			current = protocol.PackfileTreeEntry{
				FileMode: 0o40000, // Directory mode
				FileName: dirParts[i],
				Hash:     treeHash.String(),
			}
		} else {
			// If tree exists, add our entries to it
			existingTree, ok := w.treeCache[existingEntry.Hash.String()]
			if !ok {
				existingTree, err = w.getObject(ctx, existingEntry.Hash)
				if err != nil {
					return fmt.Errorf("getting existing tree %s: %w", currentPath, err)
				}
				w.treeCache[existingEntry.Hash.String()] = existingTree
			}

			treeHash, err := w.updateTreeEntry(existingTree, current)
			if err != nil {
				return fmt.Errorf("updating tree for %s: %w", currentPath, err)
			}

			w.logger.Debug("add updated tree object", "path", currentPath, "hash", treeHash.String(), "children", len(existingTree.Tree)+1)
			current = protocol.PackfileTreeEntry{
				FileMode: 0o40000, // Directory mode
				FileName: dirParts[i],
				Hash:     treeHash.String(),
			}
		}
	}

	// TODO: no need to have separate hash. We can keep the entry
	lastTree, ok := w.treeCache[w.lastTreeHash.String()]
	if !ok {
		return errors.New("root tree not found in cache")
	}

	if len(lastTree.Tree) == 0 {
		rootHash, err := w.writer.AddTree([]protocol.PackfileTreeEntry{current})
		if err != nil {
			return fmt.Errorf("adding root tree: %w", err)
		}

		w.lastTreeHash = rootHash
		w.treeCache[rootHash.String()] = &protocol.PackfileObject{
			Hash: rootHash,
			Type: protocol.ObjectTypeTree,
			Tree: []protocol.PackfileTreeEntry{current},
		}

		return nil
	}

	newRootHash, err := w.updateTreeEntry(lastTree, current)
	if err != nil {
		return fmt.Errorf("updating root tree: %w", err)
	}

	w.treeCache[newRootHash.String()] = &protocol.PackfileObject{
		Hash: newRootHash,
		Type: protocol.ObjectTypeTree,
		Tree: []protocol.PackfileTreeEntry{current},
	}

	w.lastTreeHash = newRootHash

	return nil
}

func (w *refWriter) updateTreeEntry(obj *protocol.PackfileObject, current protocol.PackfileTreeEntry) (hash.Hash, error) {
	if obj.Type != protocol.ObjectTypeTree {
		return nil, errors.New("object is not a tree")
	}

	combinedEntries := make([]protocol.PackfileTreeEntry, 0, len(obj.Tree)+1)
	combinedEntries = append(combinedEntries, obj.Tree...)
	combinedEntries = append(combinedEntries, current)

	// Create new tree with combined entries
	return w.writer.AddTree(combinedEntries)
}
