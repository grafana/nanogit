package nanogit

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

// Author represents the person who created the changes in the commit.
// It includes their name, email, and the timestamp of when they made the changes.
type Author struct {
	Name  string
	Email string
	Time  time.Time
}

// Committer represents the person who created the commit object.
// This is often the same as the author, but can be different in cases
// where someone else commits changes on behalf of the author.
type Committer struct {
	Name  string
	Email string
	Time  time.Time
}

// Commit represents a Git commit object.
// It contains metadata about the commit, including the author, committer,
// commit message, and references to the parent commits and tree.
type Commit struct {
	// Hash is the hash of the commit object.
	Hash hash.Hash
	// Tree is the hash of the root tree object that represents the state
	// of the repository at the time of the commit.
	Tree hash.Hash

	// Parent is a list of hashes of parent commits.
	// TODO: Merge commits can have multiple parents.
	Parent hash.Hash

	// Author is the person who created the changes in the commit.
	Author Author

	// Committer is the person who created the commit object.
	Committer Committer

	// Message is the commit message that describes the changes made in this commit.
	Message string
}

// Time returns the time when the commit was created.
// This is the same as the committer's time, as the committer is the person
// who actually created the commit object in the repository.
func (c *Commit) Time() time.Time {
	return c.Committer.Time
}

// CommitFile represents a file change between two commits.
// It contains information about how a file was modified, including its path,
// mode, hash, and the type of change (added, modified, deleted, etc.).
type CommitFile struct {
	Path    string              // Path of the file in the head commit
	Mode    uint32              // File mode in the head commit (e.g., 100644 for regular files)
	OldMode uint32              // Original mode in the base commit (for modified files)
	Hash    hash.Hash           // File hash in the head commit
	OldHash hash.Hash           // Original hash in the base commit (for modified files)
	Status  protocol.FileStatus // Status of the file change (added, modified, deleted, etc.)
}

// CompareCommits compares two commits and returns the differences between them.
// It takes a base commit and a head commit, and returns a list of file changes
// that occurred between them. The changes include:
//   - Added files (present in head but not in base)
//   - Modified files (different content or mode between base and head)
//   - Deleted files (present in base but not in head)
//
// The returned changes are sorted by file path for consistent ordering.
func (c *clientImpl) CompareCommits(ctx context.Context, baseCommit, headCommit hash.Hash) ([]CommitFile, error) {
	// Get both commits
	base, err := c.getObject(ctx, baseCommit)
	if err != nil {
		return nil, fmt.Errorf("getting base commit: %w", err)
	}

	if base.Type != protocol.ObjectTypeCommit {
		return nil, errors.New("base commit is not a commit")
	}

	head, err := c.getObject(ctx, headCommit)
	if err != nil {
		return nil, fmt.Errorf("getting head commit: %w", err)
	}

	if head.Type != protocol.ObjectTypeCommit {
		return nil, errors.New("head commit is not a commit")
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

// compareTrees recursively compares two trees and collects changes between them.
// It builds maps of entries from both trees and compares them to identify:
//   - Files that exist in the head tree but not in the base tree (added)
//   - Files that exist in both trees but have different content or mode (modified)
//   - Files that exist in the base tree but not in the head tree (deleted)
//
// The function returns a sorted list of changes, with each change containing
// the relevant file information and status.
func (c *clientImpl) compareTrees(base, head *Tree) ([]CommitFile, error) {
	changes := make([]CommitFile, 0)

	// Build maps for efficient lookup
	inHead := make(map[string]TreeEntry)
	for _, entry := range head.Entries {
		inHead[entry.Path] = entry
	}

	inBase := make(map[string]TreeEntry)
	for _, entry := range base.Entries {
		inBase[entry.Path] = entry
	}

	// Check for added and modified files
	for _, entry := range head.Entries {
		if _, ok := inBase[entry.Path]; !ok {
			// File exists in head but not in base - it was added
			changes = append(changes, CommitFile{
				Path:   entry.Path,
				Status: protocol.FileStatusAdded,
				Mode:   entry.Mode,
				Hash:   entry.Hash,
			})
		} else if !inBase[entry.Path].Hash.Is(entry.Hash) && entry.Type != protocol.ObjectTypeTree {
			// File exists in both but has different content - it was modified
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

	// Check for deleted files
	for _, entry := range base.Entries {
		if _, ok := inHead[entry.Path]; !ok {
			// File exists in base but not in head - it was deleted
			changes = append(changes, CommitFile{
				Path:   entry.Path,
				Status: protocol.FileStatusDeleted,
				Mode:   entry.Mode,
				Hash:   entry.Hash,
			})
		}
	}

	// Sort changes by path for consistent ordering
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Path < changes[j].Path
	})

	return changes, nil
}

// GetCommit returns a commit object from the repository.
func (c *clientImpl) GetCommit(ctx context.Context, hash hash.Hash) (*Commit, error) {
	commit, err := c.getObject(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("getting commit: %w", err)
	}

	if commit.Type != protocol.ObjectTypeCommit {
		return nil, errors.New("commit is not a commit")
	}

	authorTime, err := commit.Commit.Author.Time()
	if err != nil {
		return nil, fmt.Errorf("parsing author time: %w", err)
	}

	committerTime, err := commit.Commit.Committer.Time()
	if err != nil {
		return nil, fmt.Errorf("parsing committer time: %w", err)
	}

	return &Commit{
		Hash:   commit.Hash,
		Tree:   commit.Commit.Tree,
		Parent: commit.Commit.Parent,
		Author: Author{
			Name:  commit.Commit.Author.Name,
			Email: commit.Commit.Author.Email,
			Time:  authorTime,
		},
		Committer: Committer{
			Name:  commit.Commit.Committer.Name,
			Email: commit.Commit.Committer.Email,
			Time:  committerTime,
		},
		Message: strings.TrimSpace(commit.Commit.Message),
	}, nil
}
