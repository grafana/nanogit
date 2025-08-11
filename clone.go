package nanogit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

// CloneOptions provides configuration options for repository cloning operations.
// It supports flexible folder filtering to optimize clones of large repositories
// by only including or excluding specific paths, which is ideal for CI environments
// with no caching where only certain directories are needed.
type CloneOptions struct {
	// Path specifies the local filesystem path where files should be written.
	// This field is required for clone operations.
	Path string

	// Hash specifies the commit hash to clone from.
	// Use client.GetRef() to resolve branch/tag names to hashes first.
	Hash hash.Hash

	// IncludePaths specifies which paths to include in the clone.
	// Only files and directories matching these patterns will be included.
	// Supports glob patterns (e.g., "src/**", "*.go", "docs/api/*").
	// If empty, all paths are included (unless excluded by ExcludePaths).
	IncludePaths []string

	// ExcludePaths specifies which paths to exclude from the clone.
	// Files and directories matching these patterns will be excluded.
	// Supports glob patterns (e.g., "node_modules/**", "*.tmp", "test/**").
	// ExcludePaths takes precedence over IncludePaths.
	ExcludePaths []string
}

// CloneResult contains the results of a clone operation.
type CloneResult struct {
	// Path is the local filesystem path where files were written.
	Path string

	// Commit is the commit object that was cloned.
	Commit *Commit

	// FlatTree contains all files and directories in the cloned repository,
	// filtered according to the CloneOptions.
	FlatTree *FlatTree

	// TotalFiles is the total number of files in the cloned tree.
	TotalFiles int

	// FilteredFiles is the number of files after applying include/exclude filters.
	FilteredFiles int
}

// Clone clones a repository for the given reference with optional path filtering.
// This method is optimized for CI environments and large repositories where only
// specific directories are needed. It supports flexible include/exclude patterns
// to minimize the amount of data fetched and processed.
//
// The clone operation:
//  1. Resolves the specified ref to a commit hash
//  2. Fetches the commit and tree objects
//  3. Applies include/exclude filters to the tree structure
//  4. Returns the filtered tree with only the requested paths
//
// Parameters:
//   - ctx: Context for the operation
//   - opts: Clone options including ref, depth, and path filters
//
// Returns:
//   - *CloneResult: Contains the cloned commit and filtered tree
//   - error: Error if clone operation fails
//
// Example:
//
//	// Get the commit hash for main branch
//	ref, err := client.GetRef(ctx, "main")
//	if err != nil {
//	    return err
//	}
//
//	// Clone only the src/ and docs/ directories
//	result, err := client.Clone(ctx, nanogit.CloneOptions{
//	    Path: "/tmp/repo",
//	    Hash: ref.Hash,
//	    IncludePaths: []string{"src/**", "docs/**"},
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Cloned %d files from commit %s\n",
//	    result.FilteredFiles, result.Commit.Hash.String()[:8])
func (c *httpClient) Clone(ctx context.Context, opts CloneOptions) (*CloneResult, error) {
	logger := log.FromContext(ctx)
	// Validate that hash is provided
	if opts.Hash == hash.Zero {
		return nil, fmt.Errorf("commit hash is required - use client.GetRef() to resolve branch/tag names to hashes")
	}

	logger.Debug("Starting clone operation",
		"commit_hash", opts.Hash.String(),
		"include_paths", opts.IncludePaths,
		"exclude_paths", opts.ExcludePaths)

	// Get the commit object
	commit, err := c.GetCommit(ctx, opts.Hash)
	if err != nil {
		return nil, fmt.Errorf("get commit %s: %w", opts.Hash.String(), err)
	}

	// Get the full tree structure
	fullTree, err := c.GetFlatTree(ctx, commit.Hash)
	if err != nil {
		return nil, fmt.Errorf("get tree for commit %s: %w", commit.Hash.String(), err)
	}

	logger.Debug("Retrieved full tree",
		"commit_hash", commit.Hash.String(),
		"total_entries", len(fullTree.Entries))

	// Validate that path is provided
	if opts.Path == "" {
		return nil, fmt.Errorf("clone path is required")
	}

	// Apply path filters to the tree
	filteredTree, err := c.filterTree(fullTree, opts.IncludePaths, opts.ExcludePaths)
	if err != nil {
		return nil, fmt.Errorf("filter tree: %w", err)
	}

	// Write files to filesystem
	err = c.writeFilesToDisk(ctx, opts.Path, filteredTree)
	if err != nil {
		return nil, fmt.Errorf("write files to disk: %w", err)
	}

	result := &CloneResult{
		Path:          opts.Path,
		Commit:        commit,
		FlatTree:      filteredTree,
		TotalFiles:    len(fullTree.Entries),
		FilteredFiles: len(filteredTree.Entries),
	}

	logger.Debug("Clone completed",
		"commit_hash", commit.Hash.String(),
		"total_files", result.TotalFiles,
		"filtered_files", result.FilteredFiles,
		"output_path", opts.Path)

	return result, nil
}

// filterTree applies include and exclude path patterns to filter a FlatTree.
// It returns a new FlatTree containing only entries that match the criteria.
func (c *httpClient) filterTree(tree *FlatTree, includePaths, excludePaths []string) (*FlatTree, error) {
	if len(includePaths) == 0 && len(excludePaths) == 0 {
		// No filtering needed
		return tree, nil
	}

	filtered := &FlatTree{
		Entries: make([]FlatTreeEntry, 0, len(tree.Entries)),
	}

	for _, entry := range tree.Entries {
		included := c.shouldIncludePath(entry.Path, includePaths, excludePaths)
		if included {
			filtered.Entries = append(filtered.Entries, entry)
		}
	}

	return filtered, nil
}

// shouldIncludePath determines if a path should be included based on include/exclude patterns.
// ExcludePaths takes precedence over IncludePaths.
func (c *httpClient) shouldIncludePath(path string, includePaths, excludePaths []string) bool {
	// First check exclude patterns - they take precedence
	for _, excludePattern := range excludePaths {
		if matched, err := filepath.Match(excludePattern, path); err == nil && matched {
			return false
		}

		// Also check if the path starts with the exclude pattern (for directory exclusions)
		if strings.HasSuffix(excludePattern, "/**") {
			prefix := strings.TrimSuffix(excludePattern, "/**")
			if strings.HasPrefix(path, prefix+"/") || path == prefix {
				return false
			}
		}
	}

	// If no include patterns specified, include everything not excluded
	if len(includePaths) == 0 {
		return true
	}

	// Check include patterns
	for _, includePattern := range includePaths {
		if matched, err := filepath.Match(includePattern, path); err == nil && matched {
			return true
		}

		// Also check if the path starts with the include pattern (for directory inclusions)
		if strings.HasSuffix(includePattern, "/**") {
			prefix := strings.TrimSuffix(includePattern, "/**")
			if strings.HasPrefix(path, prefix+"/") || path == prefix {
				return true
			}
		}
	}

	return false
}

// writeFilesToDisk writes all files from the filtered tree to the specified directory path.
// It creates the necessary directory structure and downloads blob content for each file.
func (c *httpClient) writeFilesToDisk(ctx context.Context, basePath string, tree *FlatTree) error {
	logger := log.FromContext(ctx)
	logger.Debug("Writing files to disk",
		"base_path", basePath,
		"file_count", len(tree.Entries))

	// Create the base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("create base directory %s: %w", basePath, err)
	}

	// Process each file in the tree
	for _, entry := range tree.Entries {
		// Skip tree entries (directories are created automatically when needed)
		if entry.Type != protocol.ObjectTypeBlob {
			continue
		}

		filePath := filepath.Join(basePath, entry.Path)

		// Create parent directories if needed
		parentDir := filepath.Dir(filePath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("create parent directory for %s: %w", entry.Path, err)
		}

		// Get the blob content
		blob, err := c.GetBlob(ctx, entry.Hash)
		if err != nil {
			return fmt.Errorf("get blob content for %s: %w", entry.Path, err)
		}

		// Write the file content
		if err := os.WriteFile(filePath, blob.Content, 0644); err != nil {
			return fmt.Errorf("write file %s: %w", entry.Path, err)
		}

		logger.Debug("File written",
			"path", entry.Path,
			"size", len(blob.Content))
	}

	logger.Debug("All files written to disk",
		"base_path", basePath,
		"file_count", len(tree.Entries))

	return nil
}
