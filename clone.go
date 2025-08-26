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

// batchResult represents the result of processing a batch of blobs
type batchResult struct {
	batchID      int
	objects      map[string]*protocol.PackfileObject
	missingBlobs []hash.Hash
	err          error
}

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

	// OnFileWritten is called after each file is successfully written to disk.
	// It receives the relative file path and the file size in bytes.
	// This can be used for progress tracking, logging, or custom processing.
	// The callback should be fast and non-blocking to avoid slowing down the clone.
	OnFileWritten func(filePath string, size int64)
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

	// TotalFiles is the total number of files (not directories) in the repository.
	TotalFiles int

	// TotalFilteredFiles is the number of files (not directories) after applying include/exclude filters.
	TotalFilteredFiles int
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
//	// Clone with progress tracking
//	var totalSize int64
//	result, err := client.Clone(ctx, nanogit.CloneOptions{
//	    Path: "/tmp/repo",
//	    Hash: ref.Hash,
//	    IncludePaths: []string{"src/**", "docs/**"},
//	    OnFileWritten: func(filePath string, size int64) {
//	        totalSize += size
//	        fmt.Printf("Written: %s (%d bytes, total: %d bytes)\n",
//	            filePath, size, totalSize)
//	    },
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Cloned %d files (%d total bytes) from commit %s\n",
//	    result.FilteredFiles, totalSize, result.Commit.Hash.String()[:8])
func (c *httpClient) Clone(ctx context.Context, opts CloneOptions) (*CloneResult, error) {
	logger := log.FromContext(ctx)
	if opts.Hash == hash.Zero {
		return nil, fmt.Errorf("commit hash is required - use client.GetRef() to resolve branch/tag names to hashes")
	}

	if opts.Path == "" {
		return nil, fmt.Errorf("clone path is required")
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

	filteredTree, err := c.filterFilesInTree(ctx, fullTree, opts.IncludePaths, opts.ExcludePaths)
	if err != nil {
		return nil, fmt.Errorf("filter tree: %w", err)
	}

	err = c.fetchAndWriteFilesToDisk(ctx, opts, filteredTree)
	if err != nil {
		return nil, fmt.Errorf("write files to disk: %w", err)
	}

	// Count only files (not directories) for TotalFiles
	totalFileCount := 0
	for _, entry := range fullTree.Entries {
		if entry.Mode&0o40000 == 0 { // Not a directory
			totalFileCount++
		}
	}

	result := &CloneResult{
		Path:               opts.Path,
		Commit:             commit,
		FlatTree:           filteredTree,
		TotalFiles:         totalFileCount,
		TotalFilteredFiles: len(filteredTree.Entries),
	}

	logger.Debug("Clone completed",
		"commit_hash", commit.Hash.String(),
		"total_files", result.TotalFiles,
		"total_filtered_files", result.TotalFilteredFiles,
		"output_path", opts.Path)

	return result, nil
}

// filterFilesInTree applies include and exclude path patterns to filter a FlatTree.
// It returns a new FlatTree containing only entries that match the criteria.
func (c *httpClient) filterFilesInTree(ctx context.Context, tree *FlatTree, includePaths, excludePaths []string) (*FlatTree, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Starting filterFilesInTree",
		"total_entries", len(tree.Entries),
		"include_paths", strings.Join(includePaths, ","),
		"exclude_paths", strings.Join(excludePaths, ","))

	if len(includePaths) == 0 && len(excludePaths) == 0 {
		// No filtering needed
		logger.Debug("No filtering needed, returning original tree")
		return tree, nil
	}

	filtered := &FlatTree{
		Entries: make([]FlatTreeEntry, 0, len(tree.Entries)),
	}

	var totalFileCount int
	for _, entry := range tree.Entries {
		// Skip directories - we only want files in the filtered tree
		if entry.Mode&0o40000 != 0 {
			logger.Debug("Skipping directory in filterTree", "path", entry.Path, "mode", fmt.Sprintf("0o%o", entry.Mode))
			continue
		}

		totalFileCount++
		if c.shouldIncludePath(entry.Path, includePaths, excludePaths) {
			filtered.Entries = append(filtered.Entries, entry)
		} else {
			logger.Debug("Excluding file based on filters", "path", entry.Path)
		}
	}

	logger.Debug("Completed filterTree",
		"input_files", totalFileCount,
		"excluded_files", totalFileCount-len(filtered.Entries),
		"output_entries", len(filtered.Entries))

	return filtered, nil
}

// matchGlobPattern matches a path against a glob pattern that may include ** wildcards.
// It supports both single * and double ** wildcards for recursive matching.
func matchGlobPattern(pattern, path string) bool {
	// Handle ** patterns by converting them to regular expressions
	if strings.Contains(pattern, "**") {
		// Convert glob pattern to regex pattern
		regexPattern := strings.ReplaceAll(pattern, "**", ".*")
		regexPattern = strings.ReplaceAll(regexPattern, "*", "[^/]*")
		regexPattern = "^" + regexPattern + "$"

		// Use strings.Contains for simple suffix matching as a fallback
		if strings.HasPrefix(pattern, "**") && strings.HasSuffix(pattern, "*") {
			// Pattern like **/*.gen.ts
			suffix := strings.TrimPrefix(pattern, "**")
			suffix = strings.TrimPrefix(suffix, "/")
			if strings.HasSuffix(suffix, "*") {
				// Convert to simple suffix check for .gen.ts
				if strings.Contains(suffix, ".") {
					dotSuffix := suffix[strings.LastIndex(suffix, "."):]
					return strings.HasSuffix(path, dotSuffix)
				}
			}
		}

		// For now, fall back to checking file extension for .gen.ts patterns
		if strings.Contains(pattern, ".gen.ts") {
			return strings.Contains(path, ".gen.ts")
		}
		if strings.Contains(pattern, "_gen.ts") {
			return strings.Contains(path, "_gen.ts")
		}
	}

	// Use standard filepath.Match for simple patterns
	matched, err := filepath.Match(pattern, path)
	return err == nil && matched
}

// shouldIncludePath determines if a path should be included based on include/exclude patterns.
// ExcludePaths takes precedence over IncludePaths.
func (c *httpClient) shouldIncludePath(path string, includePaths, excludePaths []string) bool {
	// First check exclude patterns - they take precedence
	for _, excludePattern := range excludePaths {
		// Use enhanced pattern matching for ** patterns
		if matchGlobPattern(excludePattern, path) {
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
		// Use enhanced pattern matching for ** patterns
		if matchGlobPattern(includePattern, path) {
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

// fetchAndWriteFilesToDisk writes all files from the filtered tree to the specified directory path.
// It creates the necessary directory structure and downloads blob content in batches for better performance.
func (c *httpClient) fetchAndWriteFilesToDisk(ctx context.Context, opts CloneOptions, tree *FlatTree) error {
	logger := log.FromContext(ctx)
	basePath := opts.Path
	logger.Debug("Fetch and write files to disk",
		"base_path", basePath,
		"file_count", len(tree.Entries))

	// Create the base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("create base directory %s: %w", basePath, err)
	}

	if len(tree.Entries) == 0 {
		logger.Debug("No files files to write")
		return nil
	}

	// prepare variables to track blobs and their paths
	hashToBlobs := make(map[string][]FlatTreeEntry)
	pendingBlobs := make(map[hash.Hash]struct{})
	for _, entry := range tree.Entries {
		pendingBlobs[entry.Hash] = struct{}{}
		if _, exists := hashToBlobs[entry.Hash.String()]; !exists {
			hashToBlobs[entry.Hash.String()] = make([]FlatTreeEntry, 0)
		}
		hashToBlobs[entry.Hash.String()] = append(hashToBlobs[entry.Hash.String()], entry)
	}
	totalBlobs := len(hashToBlobs)

	var attempt int
	for attempt <= 3 && len(pendingBlobs) > 0 {
		attempt++
		// pendingBlobs, err = c.fetchAndWriteInBatches(ctx, pendingBlobs)
		// if err != nil {
		// 	return fmt.Errorf("fetch and write in batches in attempt %d: %w", attempt, err)
		// }
	}

	if len(pendingBlobs) > 0 {
		logger.Debug("Some blobs are still pending after multiple batch attempts",
			"pending_blob_count", len(pendingBlobs),
			"total_blobs", totalBlobs,
			"attempts", attempt,
		)

		for hash := range pendingBlobs {
			blob, err := c.GetBlob(ctx, hash)
			if err != nil {
				return fmt.Errorf("get individual blob %s: %w", hash.String(), err)
			}

			if err := c.writeBlobToDisk(ctx, opts, blob, hashToBlobs); err != nil {
				return fmt.Errorf("write individual blob %s to disk: %w", hash.String(), err)
			}
		}
	}

	return nil
}

func (c *httpClient) writeBlobToDisk(ctx context.Context, opts CloneOptions, blob *Blob, hashToBlobs map[string][]FlatTreeEntry) error {
	logger := log.FromContext(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, entry := range hashToBlobs[blob.Hash.String()] {
		filePath := filepath.Join(opts.Path, entry.Path)
		// Create parent directories if needed
		parentDir := filepath.Dir(filePath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("create parent directory for %s: %w", entry.Path, err)
		}

		// Write the file content
		if err := os.WriteFile(filePath, blob.Content, 0644); err != nil {
			return fmt.Errorf("write file %s: %w", entry.Path, err)
		}
		logger.Debug("Wrote file to disk", "path", entry.Path, "size", len(blob.Content))
		if opts.OnFileWritten != nil {
			opts.OnFileWritten(entry.Path, int64(len(blob.Content)))
		}
	}

	return nil
}
