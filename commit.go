package nanogit

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/client"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/storage"
)

// Author represents the person who created the changes in the commit.
// It includes their name, email, and the timestamp of when they made the changes.
// This is typically the person who wrote the code or made the modifications.
type Author struct {
	// Name is the full name of the author (e.g., "John Doe")
	Name string
	// Email is the email address of the author (e.g., "john@example.com")
	Email string
	// Time is when the changes were originally made by the author
	Time time.Time
}

// Committer represents the person who created the commit object.
// This is often the same as the author, but can be different in cases
// where someone else commits changes on behalf of the author (e.g., via patches).
type Committer struct {
	// Name is the full name of the committer (e.g., "Jane Smith")
	Name string
	// Email is the email address of the committer (e.g., "jane@example.com")
	Email string
	// Time is when the commit object was created
	Time time.Time
}

// Commit represents a Git commit object.
// It contains metadata about the commit, including the author, committer,
// commit message, and references to the parent commits and tree.
type Commit struct {
	// Hash is the SHA-1 hash of the commit object
	Hash hash.Hash
	// Tree is the hash of the root tree object that represents the state
	// of the repository at the time of the commit
	Tree hash.Hash
	// Parent is the hash of the parent commit
	// TODO: Merge commits can have multiple parents, but currently only single parent is supported
	Parent hash.Hash
	// Author is the person who created the changes in the commit
	Author Author
	// Committer is the person who created the commit object
	Committer Committer
	// Message is the commit message that describes the changes made in this commit
	Message string
}

// Time returns the timestamp when the commit object was created.
// This is equivalent to the committer's timestamp, as the committer is the person
// who actually created the commit object in the repository. For most commits,
// this will be the same as the author time, but they can differ in some workflows.
//
// Returns:
//   - time.Time: The timestamp when the commit was created
func (c *Commit) Time() time.Time {
	return c.Committer.Time
}

// CommitFile represents a file change between two commits.
// It contains information about how a file was modified, including its path,
// mode, hash, and the type of change (added, modified, deleted, etc.).
type CommitFile struct {
	// Path of the file in the head commit
	Path string
	// Mode is the file mode in the head commit (e.g., 100644 for regular files)
	Mode uint32
	// OldMode is the original file mode in the base commit (for modified files)
	OldMode uint32
	// Hash is the file hash in the head commit
	Hash hash.Hash
	// OldHash is the original file hash in the base commit (for modified files)
	OldHash hash.Hash
	// Status indicates the type of file change (added, modified, deleted, etc.)
	Status protocol.FileStatus
}

// CompareCommits compares two commits and returns the differences between them.
// This method performs a comprehensive diff between two commits, analyzing
// all file changes that occurred between the base and head commits.
//
// The comparison includes:
//   - Added files (present in head but not in base)
//   - Modified files (different content or mode between base and head)
//   - Deleted files (present in base but not in head)
//
// Parameters:
//   - ctx: Context for the operation
//   - baseCommit: Hash of the base commit (older commit)
//   - headCommit: Hash of the head commit (newer commit)
//
// Returns:
//   - []CommitFile: Sorted list of file changes between the commits
//   - error: Error if either commit cannot be found or comparison fails
//
// Example:
//
//	changes, err := client.CompareCommits(ctx, oldCommit, newCommit)
//	if err != nil {
//	    return err
//	}
//	for _, change := range changes {
//	    fmt.Printf("%s: %s\n", change.Status, change.Path)
//	}
func (c *httpClient) CompareCommits(ctx context.Context, baseCommit, headCommit hash.Hash) ([]CommitFile, error) {
	// Ensure storage as it's a complex operation with multiple calls
	// and we may get more objects in the same request than expected in some responses
	ctx, _ = storage.FromContextOrInMemory(ctx)

	// Get both trees
	baseTree, err := c.GetFlatTree(ctx, baseCommit)
	if err != nil {
		return nil, fmt.Errorf("getting base tree: %w", err)
	}

	headTree, err := c.GetFlatTree(ctx, headCommit)
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
func (c *httpClient) compareTrees(base, head *FlatTree) ([]CommitFile, error) {
	changes := make([]CommitFile, 0)

	// Build maps for efficient lookup
	inHead := make(map[string]FlatTreeEntry)
	for _, entry := range head.Entries {
		inHead[entry.Path] = entry
	}

	inBase := make(map[string]FlatTreeEntry)
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

// GetCommit retrieves a specific commit object from the repository by its hash.
// This method fetches the complete commit information including metadata,
// author, committer, message, and references to parent commits and tree.
//
// Parameters:
//   - ctx: Context for the operation
//   - hash: SHA-1 hash of the commit to retrieve
//
// Returns:
//   - *Commit: The commit object with all metadata
//   - error: Error if the commit is not found or cannot be retrieved
//
// Example:
//
//	commit, err := client.GetCommit(ctx, commitHash)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Commit by %s: %s\n", commit.Author.Name, commit.Message)
func (c *httpClient) GetCommit(ctx context.Context, commitHash hash.Hash) (*Commit, error) {
	objects, err := c.Fetch(ctx, client.FetchOptions{
		NoProgress:   true,
		NoBlobFilter: true,
		Want:         []hash.Hash{commitHash},
		Deepen:       1,
		Shallow:      true,
		Done:         true,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching commit objects: %w", err)
	}

	if len(objects) == 0 {
		return nil, NewObjectNotFoundError(commitHash)
	}

	var foundObj *protocol.PackfileObject
	for _, obj := range objects {
		// Skip tree objects that are included in the response despite the blob:none filter.
		// Most Git servers don't support tree:0 filter specification, so we may receive
		// recursive tree objects that we need to filter out.
		if obj.Type == protocol.ObjectTypeTree {
			continue
		}

		if obj.Type != protocol.ObjectTypeCommit {
			return nil, NewUnexpectedObjectTypeError(commitHash, protocol.ObjectTypeCommit, obj.Type)
		}

		// we got more commits than expected
		if foundObj != nil {
			return nil, NewUnexpectedObjectCountError(1, []*protocol.PackfileObject{foundObj, obj})
		}

		if obj.Hash.Is(commitHash) {
			foundObj = obj
		}
	}

	if foundObj == nil {
		return nil, NewObjectNotFoundError(commitHash)
	}

	return packfileObjectToCommit(foundObj)
}

func packfileObjectToCommit(commit *protocol.PackfileObject) (*Commit, error) {
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

// ListCommitsOptions provides filtering and pagination options for listing commits.
// Similar to GitHub's API, it allows limiting results, filtering by path, and pagination.
type ListCommitsOptions struct {
	// PerPage specifies the number of commits to return per page
	// If 0, defaults to 30. Maximum allowed is 100
	PerPage int
	// Page specifies which page of results to return (1-based)
	// If 0, defaults to 1
	Page int
	// Path filters commits to only those that affect the specified file or directory path
	// If empty, all commits are included
	Path string
	// Since filters commits to only those created after this time
	// If zero, no time filtering is applied
	Since time.Time
	// Until filters commits to only those created before this time
	// If zero, no time filtering is applied
	Until time.Time
}

// ListCommits retrieves a list of commits starting from the specified commit,
// walking backwards through the commit history. This method supports filtering
// and pagination similar to GitHub's API, allowing you to traverse repository
// history efficiently.
//
// The method traverses the commit graph starting from the specified commit,
// following parent links to build a chronological list of commits. It supports
// various filters to narrow down results and pagination for large histories.
//
// Parameters:
//   - ctx: Context for the operation
//   - startCommit: Hash of the commit to start traversal from (typically HEAD)
//   - options: Filtering and pagination options
//
// Returns:
//   - []Commit: List of commits matching the specified criteria
//   - error: Error if traversal fails or commits cannot be retrieved
//
// Example:
//
//	// Get the latest 10 commits on main branch
//	options := nanogit.ListCommitsOptions{
//	    PerPage: 10,
//	    Page:    1,
//	}
//	commits, err := client.ListCommits(ctx, mainBranchHash, options)
//	if err != nil {
//	    return err
//	}
//	for _, commit := range commits {
//	    fmt.Printf("%s: %s\n", commit.Hash.String()[:8], commit.Message)
//	}
func (c *httpClient) ListCommits(ctx context.Context, startCommit hash.Hash, options ListCommitsOptions) ([]Commit, error) {
	// TODO: optimize this one
	// Set defaults for pagination
	perPage := options.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	page := options.Page
	if page <= 0 {
		page = 1
	}

	// Calculate how many commits to skip and collect
	skip := (page - 1) * perPage
	collect := perPage

	var commitObjs []*protocol.PackfileObject
	visited := make(map[string]bool)
	queue := []hash.Hash{startCommit}

	// Ensure storage as it's a complex operation with multiple calls
	// and we may get more objects in the same request than expected in some responses
	ctx, allObjects := storage.FromContextOrInMemory(ctx)

	for len(queue) > 0 && len(commitObjs) < skip+collect {
		currentHash := queue[0]
		queue = queue[1:]

		// Skip if already visited (handle merge commits)
		if visited[currentHash.String()] {
			continue
		}
		visited[currentHash.String()] = true

		// Get the commit object
		objects, err := c.Fetch(ctx, client.FetchOptions{
			NoProgress:   true,
			NoBlobFilter: true,
			Want:         []hash.Hash{currentHash},
			Deepen:       perPage,
			Done:         true,
		})
		if err != nil {
			return nil, fmt.Errorf("getting commit %s in queue: %w", currentHash.String(), err)
		}

		// Try to find it in the objects we got but if not, get it from the storage
		commit, ok := objects[currentHash.String()]
		if !ok || commit.Type != protocol.ObjectTypeCommit {
			commit, ok = allObjects.Get(currentHash)
			if !ok || commit.Type != protocol.ObjectTypeCommit {
				return nil, fmt.Errorf("commit %s not found", currentHash.String())
			}
		}

		// Apply filters
		matches, err := c.commitMatchesFilters(ctx, commit, &options, allObjects)
		if err != nil {
			return nil, fmt.Errorf("check commit filters: %w", err)
		}

		if !matches {
			// Add parent to queue for continued traversal
			// TODO: handle merge commits
			if !commit.Commit.Parent.Is(hash.Zero) {
				queue = append(queue, commit.Commit.Parent)
			}
			continue
		}

		// Add to results
		commitObjs = append(commitObjs, commit)

		// Continue with parent commit
		// TODO: handle merge commits
		if !commit.Commit.Parent.Is(hash.Zero) {
			queue = append(queue, commit.Commit.Parent)
		}
	}

	// Apply pagination
	if skip >= len(commitObjs) {
		return []Commit{}, nil
	}

	// Convert to Commit objects
	end := min(skip+collect, len(commitObjs))
	commits := make([]Commit, 0, end-skip)
	for _, obj := range commitObjs[skip:end] {
		commit, err := packfileObjectToCommit(obj)
		if err != nil {
			return nil, fmt.Errorf("converting packfile object to commit: %w", err)
		}
		commits = append(commits, *commit)
	}

	return commits, nil
}

// commitMatchesFilters checks if a commit matches the specified filters.
func (c *httpClient) commitMatchesFilters(ctx context.Context, commit *protocol.PackfileObject, options *ListCommitsOptions, allObjects storage.PackfileStorage) (bool, error) {
	logger := log.FromContext(ctx)
	commitTime, err := commit.Commit.Author.Time()
	if err != nil {
		logger.Debug("error parsing commit time", "commit", commit.Hash.String(), "error", err.Error())
		return false, fmt.Errorf("parsing commit time: %w", err)
	}

	// Check time filters
	if !options.Since.IsZero() && commitTime.Before(options.Since) {
		return false, nil
	}

	if !options.Until.IsZero() && commitTime.After(options.Until) {
		return false, nil
	}

	// Check path filter
	if options.Path != "" {
		affected, err := c.commitAffectsPath(ctx, commit, options.Path, allObjects)
		if err != nil {
			// Log error but don't fail the entire operation
			// TODO: should we handle this differently?
			logger.Error("error checking if commit affects path", "commit", commit.Hash.String(), "path", options.Path, "error", err.Error())
			return false, nil
		}
		if !affected {
			return false, nil
		}
	}

	return true, nil
}

// commitAffectsPath checks if a commit affects the specified path by comparing with the hash of that path in the parent commit.
// TODO: make it work for merge commits
func (c *httpClient) commitAffectsPath(ctx context.Context, commit *protocol.PackfileObject, path string, allObjects storage.PackfileStorage) (bool, error) {
	// For the initial commit (no parent), check if the path exists
	if commit.Commit.Parent.Is(hash.Zero) {
		parentHash, err := c.hashForPath(ctx, commit.Hash, path, allObjects)
		if err != nil {
			return false, fmt.Errorf("hash for path: %w", err)
		}

		return !parentHash.Is(hash.Zero), nil
	}

	pathHashParent, err := c.hashForPath(ctx, commit.Commit.Parent, path, allObjects)
	if err != nil {
		return false, fmt.Errorf("hash for path: %w", err)
	}

	pathHashCommit, err := c.hashForPath(ctx, commit.Hash, path, allObjects)
	if err != nil {
		return false, fmt.Errorf("hash for path: %w", err)
	}

	return !pathHashParent.Is(pathHashCommit), nil
}

// walkPathToTreeHash walks the path to find the tree hash
// if the object is not in the storage, it will be fetched.
// All objects returned by the client will be added to the storage.
// If the object is not found, hash.Zero will be returned.
// If the object is a tree, the hash of the tree will be returned.
// If the object is a blob, the hash of the blob will be returned.
// Otherwise, return an error.
func (c *httpClient) hashForPath(ctx context.Context, commitHash hash.Hash, path string, allObjects storage.PackfileStorage) (hash.Hash, error) {
	commit, ok := allObjects.Get(commitHash)
	if !ok {
		objects, err := c.Fetch(ctx, client.FetchOptions{
			NoProgress:   true,
			NoBlobFilter: true,
			Want:         []hash.Hash{commitHash},
			Shallow:      true,
			Done:         true,
		})

		if err != nil {
			return hash.Zero, fmt.Errorf("getting commit to get hash for path: %w", err)
		}

		// Try to find it in the objects we got but if not, get it from the storage
		commit, ok = objects[commitHash.String()]
		if !ok || commit.Type != protocol.ObjectTypeCommit {
			commit, ok = allObjects.Get(commitHash)
			if !ok || commit.Type != protocol.ObjectTypeCommit {
				return hash.Zero, fmt.Errorf("commit %s not found", commitHash.String())
			}
		}
	}

	logger := log.FromContext(ctx)
	logger.Debug("hashForPath", "commit", commitHash.String(), "path", path, "allObjects", allObjects.GetAllKeys(), "commit", commit)
	treeHash := commit.Commit.Tree
	tree, err := c.GetTree(ctx, treeHash)
	if err != nil {
		return hash.Zero, fmt.Errorf("getting tree: %w", err)
	}

	// If path is empty, return the tree hash
	if path == "" {
		return treeHash, nil
	}

	// Split path into components
	components := strings.Split(path, "/")
	currentTree := tree

	// Walk through each path component
	for i, component := range components {
		component = strings.TrimSpace(component)
		if component == "" {
			return hash.Zero, errors.New("path component is empty")
		}

		// Find the entry in the current tree
		var found bool
		var entryHash hash.Hash
		for _, entry := range currentTree.Entries {
			if entry.Name == component {
				found = true
				entryHash = entry.Hash
				break
			}
		}

		if !found {
			return hash.Zero, nil
		}

		// If this is the last component, return its hash
		if i == len(components)-1 {
			return entryHash, nil
		}

		// Otherwise, get the next tree
		nextTree, err := c.GetTree(ctx, entryHash)
		if err != nil {
			return hash.Zero, fmt.Errorf("getting tree: %w", err)
		}

		currentTree = nextTree
	}

	return hash.Zero, nil
}
