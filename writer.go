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
	commit, err := c.GetCommit(ctx, ref.Hash)
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
		clientImpl: c,
		ref:        ref,
		writer:     writer,
		lastCommit: commit,
		lastTree:   treeObj,
		// TODO: I think we only need one
		treeCache:   cache,
		treeEntries: entries,
	}, nil
}

type refWriter struct {
	*clientImpl
	ref         Ref
	writer      *protocol.PackfileWriter
	lastCommit  *Commit
	lastTree    *protocol.PackfileObject
	treeCache   map[string]*protocol.PackfileObject
	treeEntries map[string]*TreeEntry
}

// CreateBlob creates a new blob in the specified path.
func (w *refWriter) CreateBlob(ctx context.Context, path string, content []byte) (hash.Hash, error) {
	if w.treeEntries[path] != nil {
		return nil, errors.New("blob at that path already exists")
	}

	// Create the blob for the file content
	blobHash, err := w.writer.AddBlob(content)
	if err != nil {
		return nil, fmt.Errorf("creating blob: %w", err)
	}

	w.logger.Debug("created blob", "hash", blobHash.String())

	if err := w.addMissingOrStaleTreeEntries(ctx, path, blobHash); err != nil {
		return nil, fmt.Errorf("creating root tree: %w", err)
	}

	return blobHash, nil
}

func (w *refWriter) UpdateBlob(ctx context.Context, path string, content []byte) (hash.Hash, error) {
	if w.treeEntries[path] == nil {
		return nil, errors.New("blob at that path does not exist")
	}

	// Create the blob for the file content
	blobHash, err := w.writer.AddBlob(content)
	if err != nil {
		return nil, fmt.Errorf("creating blob: %w", err)
	}

	w.logger.Debug("created blob", "hash", blobHash.String())
	w.treeEntries[path] = &TreeEntry{
		Path: path,
		Hash: blobHash,
		Type: protocol.ObjectTypeBlob,
	}

	// Add the new entry
	if err := w.addMissingOrStaleTreeEntries(ctx, path, blobHash); err != nil {
		return nil, fmt.Errorf("updating tree: %w", err)
	}

	return blobHash, nil
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

	commitHash, err := w.writer.AddCommit(w.lastTree.Hash, w.lastCommit.Hash, &authorIdentity, &committerIdentity, message)
	if err != nil {
		return nil, fmt.Errorf("creating commit: %w", err)
	}

	w.lastCommit = &Commit{
		Hash:      commitHash,
		Tree:      w.lastTree.Hash,
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
func (w *refWriter) addMissingOrStaleTreeEntries(ctx context.Context, path string, blobHash hash.Hash) error {
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
		// Check if not a tree
		existingObj, exists := w.treeEntries[currentPath]
		if exists && existingObj.Type != protocol.ObjectTypeTree {
			return errors.New("existing tree entry is not a tree")
		}

		// Create new tree
		if !exists {
			// Create new tree
			treeObj, err := protocol.BuildTreeObject(crypto.SHA1, []protocol.PackfileTreeEntry{current})
			if err != nil {
				return fmt.Errorf("creating tree for %s: %w", currentPath, err)
			}

			w.writer.AddObject(treeObj)
			w.logger.Debug("add new tree object", "path", currentPath, "hash", treeObj.Hash.String(), "child", current.Hash, "child_path", current.FileName)

			// Add this tree to the parent's entries
			current = protocol.PackfileTreeEntry{
				FileMode: 0o40000, // Directory mode
				FileName: dirParts[i],
				Hash:     treeObj.Hash.String(),
			}

			w.treeCache[treeObj.Hash.String()] = &treeObj
			w.treeEntries[currentPath] = &TreeEntry{
				Path: currentPath,
				Hash: treeObj.Hash,
				Type: protocol.ObjectTypeTree,
			}
		} else {
			// If tree exists, add our entries to it
			existingTree, ok := w.treeCache[existingObj.Hash.String()]
			if !ok {
				existingTree, err := w.getObject(ctx, existingObj.Hash)
				if err != nil {
					return fmt.Errorf("getting existing tree %s: %w", currentPath, err)
				}
				w.treeCache[existingObj.Hash.String()] = existingTree
			}

			newObj, err := w.updateTreeEntry(existingTree, current)
			if err != nil {
				return fmt.Errorf("updating tree for %s: %w", currentPath, err)
			}

			w.logger.Debug("add updated tree object", "path", currentPath, "hash", newObj.Hash.String(), "children", len(existingTree.Tree)+1)
			current = protocol.PackfileTreeEntry{
				FileMode: 0o40000, // Directory mode
				FileName: dirParts[i],
				Hash:     newObj.Hash.String(),
			}

			w.treeCache[newObj.Hash.String()] = newObj
			w.treeEntries[currentPath] = &TreeEntry{
				Path: currentPath,
				Hash: newObj.Hash,
				Type: protocol.ObjectTypeTree,
			}
		}
	}

	if len(w.lastTree.Tree) == 0 {
		newRoot, err := protocol.BuildTreeObject(crypto.SHA1, []protocol.PackfileTreeEntry{current})
		if err != nil {
			return fmt.Errorf("building new root tree: %w", err)
		}

		w.writer.AddObject(newRoot)
		w.lastTree = &newRoot
		w.treeCache[newRoot.Hash.String()] = &newRoot

		return nil
	}

	newRootObj, err := w.updateTreeEntry(w.lastTree, current)
	if err != nil {
		return fmt.Errorf("updating root tree: %w", err)
	}
	w.treeCache[newRootObj.Hash.String()] = newRootObj
	w.lastTree = newRootObj

	return nil
}

func (w *refWriter) updateTreeEntry(obj *protocol.PackfileObject, current protocol.PackfileTreeEntry) (*protocol.PackfileObject, error) {
	if obj.Type != protocol.ObjectTypeTree {
		return nil, errors.New("object is not a tree")
	}

	combinedEntries := append(make([]protocol.PackfileTreeEntry, 0, len(obj.Tree)+1), obj.Tree...)
	combinedEntries = append(combinedEntries, current)
	newObj, err := protocol.BuildTreeObject(crypto.SHA1, combinedEntries)
	if err != nil {
		return nil, fmt.Errorf("building tree object: %w", err)
	}

	w.writer.AddObject(newObj)

	return &newObj, nil
}
