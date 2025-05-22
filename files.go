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

	// Get the current tree if we have a parent commit
	var currentTree *Tree
	if ref.Hash != "" {
		parentHash, err := hash.FromHex(ref.Hash)
		if err != nil {
			return fmt.Errorf("parsing parent hash: %w", err)
		}
		currentTree, err = c.GetTree(ctx, parentHash)
		if err != nil {
			return fmt.Errorf("getting current tree: %w", err)
		}
	}

	// Split the path into parts
	pathParts := strings.Split(path, "/")
	if len(pathParts) == 0 {
		return errors.New("empty path")
	}

	// Start with the root tree entries
	var entries []protocol.PackfileTreeEntry
	if currentTree != nil {
		// Copy existing entries, but don't add them to the packfile
		for _, entry := range currentTree.Entries {
			entries = append(entries, protocol.PackfileTreeEntry{
				FileMode: entry.Mode,
				FileName: entry.Name,
				Hash:     entry.Hash.String(),
			})
		}
	}

	// Process each part of the path
	currentPath := ""
	for i, part := range pathParts {
		if i == len(pathParts)-1 {
			// This is the file - add it to the current level
			entries = append(entries, protocol.PackfileTreeEntry{
				FileMode: 0o100644, // Regular file
				FileName: part,
				Hash:     blobHash.String(),
			})
		} else {
			// This is a directory
			currentPath += part + "/"

			// Check if this directory already exists in the current level
			var dirEntry *protocol.PackfileTreeEntry
			for j, entry := range entries {
				if entry.FileName == part && entry.FileMode == 0o40000 {
					dirEntry = &entries[j]
					break
				}
			}

			if dirEntry == nil {
				// Directory doesn't exist - create a new empty tree
				emptyTreeHash, err := writer.AddTree(nil)
				if err != nil {
					return fmt.Errorf("creating tree for %s: %w", currentPath, err)
				}
				entries = append(entries, protocol.PackfileTreeEntry{
					FileMode: 0o40000, // Directory
					FileName: part,
					Hash:     emptyTreeHash.String(),
				})
				dirEntry = &entries[len(entries)-1]
			} else {
				// Directory exists - get its tree
				dirHash, err := hash.FromHex(dirEntry.Hash)
				if err != nil {
					return fmt.Errorf("parsing directory hash: %w", err)
				}

				dirTree, err := c.GetTree(ctx, dirHash)
				if err != nil {
					return fmt.Errorf("getting tree for %s: %w", currentPath, err)
				}

				// Create a new tree with the existing entries
				dirEntries := make([]protocol.PackfileTreeEntry, len(dirTree.Entries))
				for j, entry := range dirTree.Entries {
					dirEntries[j] = protocol.PackfileTreeEntry{
						FileMode: entry.Mode,
						FileName: entry.Name,
						Hash:     entry.Hash.String(),
					}
				}

				// Add the new tree
				newTreeHash, err := writer.AddTree(dirEntries)
				if err != nil {
					return fmt.Errorf("creating tree for %s: %w", currentPath, err)
				}

				// Update the directory entry with the new tree hash
				dirEntry.Hash = newTreeHash.String()
			}
		}
	}

	// Create the root tree
	rootTreeHash, err := writer.AddTree(entries)
	if err != nil {
		return fmt.Errorf("creating root tree: %w", err)
	}

	// Create the commit
	var parentHash hash.Hash
	if ref.Hash != "" {
		parentHash, err = hash.FromHex(ref.Hash)
		if err != nil {
			return fmt.Errorf("parsing parent hash: %w", err)
		}
	}

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
