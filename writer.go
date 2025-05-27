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

func (c *httpClient) NewStagedWriter(ctx context.Context, ref Ref) (StagedWriter, error) {
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

	entries := make(map[string]*FlatTreeEntry, len(currentTree.Entries))
	for _, entry := range currentTree.Entries {
		entries[entry.Path] = &entry
	}

	// Create a packfile writer
	writer := protocol.NewPackfileWriter(crypto.SHA1)
	return &stagedWriter{
		httpClient: c,
		ref:        ref,
		writer:     writer,
		lastCommit: commit,
		lastTree:   treeObj,
		// TODO: I think we only need one
		treeCache:   cache,
		treeEntries: entries,
	}, nil
}

type stagedWriter struct {
	*httpClient
	ref         Ref
	writer      *protocol.PackfileWriter
	lastCommit  *Commit
	lastTree    *protocol.PackfileObject
	treeCache   map[string]*protocol.PackfileObject
	treeEntries map[string]*FlatTreeEntry
}

// CreateBlob creates a new blob in the specified path.
func (w *stagedWriter) CreateBlob(ctx context.Context, path string, content []byte) (hash.Hash, error) {
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

func (w *stagedWriter) UpdateBlob(ctx context.Context, path string, content []byte) (hash.Hash, error) {
	if w.treeEntries[path] == nil {
		return nil, errors.New("blob at that path does not exist")
	}

	// Create the blob for the file content
	blobHash, err := w.writer.AddBlob(content)
	if err != nil {
		return nil, fmt.Errorf("creating blob: %w", err)
	}

	w.logger.Debug("created blob", "hash", blobHash.String())
	w.treeEntries[path] = &FlatTreeEntry{
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

func (w *stagedWriter) DeleteBlob(ctx context.Context, path string) (hash.Hash, error) {
	if w.treeEntries[path] == nil {
		return nil, errors.New("blob at that path does not exist")
	}

	if w.treeEntries[path].Type != protocol.ObjectTypeBlob {
		return nil, errors.New("entry at that path is not a blob")
	}
	blobHash := w.treeEntries[path].Hash

	w.logger.Debug("deleting blob", "path", path)

	// Remove the entry from our tracking
	delete(w.treeEntries, path)

	// Update the tree structure to remove the entry
	if err := w.removeBlobFromTree(ctx, path); err != nil {
		return nil, fmt.Errorf("removing blob from tree: %w", err)
	}

	return blobHash, nil
}

func (w *stagedWriter) DeleteTree(ctx context.Context, path string) (hash.Hash, error) {
	if w.treeEntries[path] == nil {
		return nil, errors.New("tree at that path does not exist")
	}

	if w.treeEntries[path].Type != protocol.ObjectTypeTree {
		return nil, errors.New("entry at that path is not a tree")
	}
	treeHash := w.treeEntries[path].Hash

	w.logger.Debug("deleting tree", "path", path)

	// Find and remove all entries that start with this path
	pathPrefix := path + "/"
	var entriesToDelete []string

	for entryPath := range w.treeEntries {
		if entryPath == path || strings.HasPrefix(entryPath, pathPrefix) {
			entriesToDelete = append(entriesToDelete, entryPath)
		}
	}

	// Remove all entries under this tree
	for _, entryPath := range entriesToDelete {
		w.logger.Debug("removing entry", "path", entryPath)
		delete(w.treeEntries, entryPath)
	}

	// Update the tree structure to remove the directory entry
	if err := w.removeTreeFromTree(ctx, path); err != nil {
		return nil, fmt.Errorf("removing tree from tree: %w", err)
	}

	return treeHash, nil
}

func (w *stagedWriter) Commit(ctx context.Context, message string, author Author, committer Committer) (*Commit, error) {
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

func (w *stagedWriter) Push(ctx context.Context) error {
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
func (w *stagedWriter) addMissingOrStaleTreeEntries(ctx context.Context, path string, blobHash hash.Hash) error {
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
			w.treeEntries[currentPath] = &FlatTreeEntry{
				Path: currentPath,
				Hash: treeObj.Hash,
				Type: protocol.ObjectTypeTree,
			}
		} else {
			// If tree exists, add our entries to it
			existingTree, ok := w.treeCache[existingObj.Hash.String()]
			if !ok {
				w.logger.Info("fetch tree not found in cache", "path", currentPath, "hash", existingObj.Hash.String())
				var err error
				existingTree, err = w.getObject(ctx, existingObj.Hash)
				if err != nil {
					return fmt.Errorf("getting existing tree %s: %w", currentPath, err)
				}
				w.treeCache[existingObj.Hash.String()] = existingTree
				w.logger.Info("tree object found in remote", "path", currentPath, "hash", existingObj.Hash.String(), "entries", len(existingTree.Tree))
			} else {
				w.logger.Debug("tree object found in cache", "path", currentPath, "hash", existingObj.Hash.String(), "entries", len(existingTree.Tree))
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
			w.treeEntries[currentPath] = &FlatTreeEntry{
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

func (w *stagedWriter) updateTreeEntry(obj *protocol.PackfileObject, current protocol.PackfileTreeEntry) (*protocol.PackfileObject, error) {
	if obj == nil {
		return nil, errors.New("object is nil")
	}

	if obj.Type != protocol.ObjectTypeTree {
		return nil, errors.New("object is not a tree")
	}

	// Create a new slice for the updated entries
	combinedEntries := make([]protocol.PackfileTreeEntry, 0, len(obj.Tree))

	// Add all entries except the one we're updating
	for _, entry := range obj.Tree {
		if entry.FileName != current.FileName {
			combinedEntries = append(combinedEntries, entry)
		}
	}

	// Add the new/updated entry
	combinedEntries = append(combinedEntries, current)

	newObj, err := protocol.BuildTreeObject(crypto.SHA1, combinedEntries)
	if err != nil {
		return nil, fmt.Errorf("building tree object: %w", err)
	}

	w.writer.AddObject(newObj)

	return &newObj, nil
}

func (w *stagedWriter) removeBlobFromTree(ctx context.Context, path string) error {
	// Split the path into parts
	pathParts := strings.Split(path, "/")
	if len(pathParts) == 0 {
		return errors.New("empty path")
	}

	// Get the file name and directory parts
	fileName := pathParts[len(pathParts)-1]
	dirParts := pathParts[:len(pathParts)-1]

	// First, remove the file from its immediate parent directory
	if len(dirParts) == 0 {
		// File is in root directory
		newRootObj, err := w.removeTreeEntry(w.lastTree, fileName)
		if err != nil {
			return fmt.Errorf("removing file from root tree: %w", err)
		}
		w.lastTree = newRootObj
		w.treeCache[newRootObj.Hash.String()] = newRootObj
		w.logger.Debug("removed file from root", "file", fileName, "new_root_hash", newRootObj.Hash.String())
		return nil
	}

	// File is in a subdirectory - we need to update the tree hierarchy
	// Start from the immediate parent and work up to root
	var updatedChildHash hash.Hash

	for i := len(dirParts) - 1; i >= 0; i-- {
		currentPath := strings.Join(dirParts[:i+1], "/")

		// Get the tree we need to modify
		existingObj, exists := w.treeEntries[currentPath]
		if !exists {
			return fmt.Errorf("parent directory %s does not exist", currentPath)
		}

		if existingObj.Type != protocol.ObjectTypeTree {
			return errors.New("parent path is not a tree")
		}

		// Get tree object from cache or fetch it
		treeObj, ok := w.treeCache[existingObj.Hash.String()]
		if !ok {
			var err error
			treeObj, err = w.getObject(ctx, existingObj.Hash)
			if err != nil {
				return fmt.Errorf("getting tree %s: %w", currentPath, err)
			}
			w.treeCache[existingObj.Hash.String()] = treeObj
		}

		var newObj *protocol.PackfileObject
		var err error

		if i == len(dirParts)-1 {
			// This is the immediate parent - remove the file
			newObj, err = w.removeTreeEntry(treeObj, fileName)
			if err != nil {
				return fmt.Errorf("removing file from tree %s: %w", currentPath, err)
			}
			w.logger.Debug("removed file from parent tree", "path", currentPath, "file", fileName, "new_hash", newObj.Hash.String())
		} else {
			// This is an ancestor directory - update with new child hash
			childDirName := dirParts[i+1]
			childEntry := protocol.PackfileTreeEntry{
				FileMode: 0o40000, // Directory mode
				FileName: childDirName,
				Hash:     updatedChildHash.String(),
			}
			newObj, err = w.updateTreeEntry(treeObj, childEntry)
			if err != nil {
				return fmt.Errorf("updating tree %s with new child: %w", currentPath, err)
			}
			w.logger.Debug("updated parent tree with new child", "path", currentPath, "child", childDirName, "child_hash", updatedChildHash.String(), "new_hash", newObj.Hash.String())
		}

		// Store the new tree hash for the next iteration
		updatedChildHash = newObj.Hash

		// Update our references
		w.treeCache[newObj.Hash.String()] = newObj
		w.treeEntries[currentPath] = &FlatTreeEntry{
			Path: currentPath,
			Hash: newObj.Hash,
			Type: protocol.ObjectTypeTree,
		}
	}

	// Finally, update the root tree with the new top-level directory hash
	rootDirName := dirParts[0]
	rootDirEntry := protocol.PackfileTreeEntry{
		FileMode: 0o40000, // Directory mode
		FileName: rootDirName,
		Hash:     updatedChildHash.String(),
	}

	newRootObj, err := w.updateTreeEntry(w.lastTree, rootDirEntry)
	if err != nil {
		return fmt.Errorf("updating root tree: %w", err)
	}

	w.lastTree = newRootObj
	w.treeCache[newRootObj.Hash.String()] = newRootObj
	w.logger.Debug("updated root tree", "dir", rootDirName, "dir_hash", updatedChildHash.String(), "new_root_hash", newRootObj.Hash.String())

	return nil
}

func (w *stagedWriter) removeTreeFromTree(ctx context.Context, path string) error {
	// Split the path into parts
	pathParts := strings.Split(path, "/")
	if len(pathParts) == 0 {
		return errors.New("empty path")
	}

	// Get the directory name and parent directory parts
	dirName := pathParts[len(pathParts)-1]
	parentParts := pathParts[:len(pathParts)-1]

	// First, remove the directory from its immediate parent
	if len(parentParts) == 0 {
		// Directory is in root
		newRootObj, err := w.removeTreeEntry(w.lastTree, dirName)
		if err != nil {
			return fmt.Errorf("removing directory from root tree: %w", err)
		}
		w.lastTree = newRootObj
		w.treeCache[newRootObj.Hash.String()] = newRootObj
		w.logger.Debug("removed directory from root", "dir", dirName, "new_root_hash", newRootObj.Hash.String())
		return nil
	}

	// Directory is in a subdirectory - we need to update the tree hierarchy
	// Start from the immediate parent and work up to root
	var updatedChildHash hash.Hash

	for i := len(parentParts) - 1; i >= 0; i-- {
		currentPath := strings.Join(parentParts[:i+1], "/")

		// Get the tree we need to modify
		existingObj, exists := w.treeEntries[currentPath]
		if !exists {
			return fmt.Errorf("parent directory %s does not exist", currentPath)
		}

		if existingObj.Type != protocol.ObjectTypeTree {
			return errors.New("parent path is not a tree")
		}

		// Get tree object from cache or fetch it
		treeObj, ok := w.treeCache[existingObj.Hash.String()]
		if !ok {
			var err error
			treeObj, err = w.getObject(ctx, existingObj.Hash)
			if err != nil {
				return fmt.Errorf("getting tree %s: %w", currentPath, err)
			}
			w.treeCache[existingObj.Hash.String()] = treeObj
		}

		var newObj *protocol.PackfileObject
		var err error

		if i == len(parentParts)-1 {
			// This is the immediate parent - remove the directory
			newObj, err = w.removeTreeEntry(treeObj, dirName)
			if err != nil {
				return fmt.Errorf("removing directory from tree %s: %w", currentPath, err)
			}
			w.logger.Debug("removed directory from parent tree", "path", currentPath, "dir", dirName, "new_hash", newObj.Hash.String())
		} else {
			// This is an ancestor directory - update with new child hash
			childDirName := parentParts[i+1]
			childEntry := protocol.PackfileTreeEntry{
				FileMode: 0o40000, // Directory mode
				FileName: childDirName,
				Hash:     updatedChildHash.String(),
			}
			newObj, err = w.updateTreeEntry(treeObj, childEntry)
			if err != nil {
				return fmt.Errorf("updating tree %s with new child: %w", currentPath, err)
			}
			w.logger.Debug("updated parent tree with new child", "path", currentPath, "child", childDirName, "child_hash", updatedChildHash.String(), "new_hash", newObj.Hash.String())
		}

		// Store the new tree hash for the next iteration
		updatedChildHash = newObj.Hash

		// Update our references
		w.treeCache[newObj.Hash.String()] = newObj
		w.treeEntries[currentPath] = &FlatTreeEntry{
			Path: currentPath,
			Hash: newObj.Hash,
			Type: protocol.ObjectTypeTree,
		}
	}

	// Finally, update the root tree with the new top-level directory hash
	rootDirName := parentParts[0]
	rootDirEntry := protocol.PackfileTreeEntry{
		FileMode: 0o40000, // Directory mode
		FileName: rootDirName,
		Hash:     updatedChildHash.String(),
	}

	newRootObj, err := w.updateTreeEntry(w.lastTree, rootDirEntry)
	if err != nil {
		return fmt.Errorf("updating root tree: %w", err)
	}

	w.lastTree = newRootObj
	w.treeCache[newRootObj.Hash.String()] = newRootObj
	w.logger.Debug("updated root tree", "dir", rootDirName, "dir_hash", updatedChildHash.String(), "new_root_hash", newRootObj.Hash.String())

	return nil
}

func (w *stagedWriter) removeTreeEntry(obj *protocol.PackfileObject, targetFileName string) (*protocol.PackfileObject, error) {
	if obj == nil {
		return nil, errors.New("object is nil")
	}

	if obj.Type != protocol.ObjectTypeTree {
		return nil, errors.New("object is not a tree")
	}

	// Create a new slice excluding the target entry
	filteredEntries := make([]protocol.PackfileTreeEntry, 0, len(obj.Tree))
	found := false

	for _, entry := range obj.Tree {
		if entry.FileName != targetFileName {
			filteredEntries = append(filteredEntries, entry)
		} else {
			found = true
		}
	}

	if !found {
		// Entry not found in tree, but this might be okay for intermediate trees
		// Return the original object unchanged
		return obj, nil
	}

	// Build new tree object with the filtered entries
	newObj, err := protocol.BuildTreeObject(crypto.SHA1, filteredEntries)
	if err != nil {
		return nil, fmt.Errorf("building tree object: %w", err)
	}

	w.writer.AddObject(newObj)

	return &newObj, nil
}
