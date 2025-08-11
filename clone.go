package nanogit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/client"
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
// It creates the necessary directory structure and downloads blob content in batches for better performance.
func (c *httpClient) writeFilesToDisk(ctx context.Context, basePath string, tree *FlatTree) error {
	logger := log.FromContext(ctx)
	logger.Debug("Writing files to disk",
		"base_path", basePath,
		"file_count", len(tree.Entries))

	// Create the base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("create base directory %s: %w", basePath, err)
	}

	// Collect all blob hashes and create path mapping
	blobEntries := make([]FlatTreeEntry, 0, len(tree.Entries))
	blobHashes := make([]hash.Hash, 0, len(tree.Entries))
	hashToEntry := make(map[string]FlatTreeEntry)

	for _, entry := range tree.Entries {
		// Only process blob entries (files)
		if entry.Type == protocol.ObjectTypeBlob {
			blobEntries = append(blobEntries, entry)
			blobHashes = append(blobHashes, entry.Hash)
			hashToEntry[entry.Hash.String()] = entry
		}
	}

	if len(blobHashes) == 0 {
		logger.Debug("No blob files to write")
		return nil
	}

	// Fetch all blobs in batches for better performance
	blobMap, err := c.fetchBlobsBatched(ctx, blobHashes)
	if err != nil {
		return fmt.Errorf("fetch blobs in batches: %w", err)
	}

	logger.Debug("Blob fetch summary",
		"requested_blobs", len(blobHashes),
		"received_blobs", len(blobMap),
		"expected_files", len(blobEntries))

	// Check if we got all the blobs we requested
	if len(blobMap) != len(blobHashes) {
		logger.Warn("Blob count mismatch",
			"requested", len(blobHashes),
			"received", len(blobMap))
		
		// Find missing blobs
		for _, expectedHash := range blobHashes {
			if _, found := blobMap[expectedHash.String()]; !found {
				if entry, exists := hashToEntry[expectedHash.String()]; exists {
					logger.Error("Missing blob for file",
						"file_path", entry.Path,
						"blob_hash", expectedHash.String())
				}
			}
		}
	}

	// Write all files to disk
	filesWritten := 0
	for _, blobObj := range blobMap {
		entry, exists := hashToEntry[blobObj.Hash.String()]
		if !exists {
			logger.Warn("Received unexpected blob",
				"blob_hash", blobObj.Hash.String())
			continue // Skip unexpected blobs
		}

		filePath := filepath.Join(basePath, entry.Path)

		// Create parent directories if needed
		parentDir := filepath.Dir(filePath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("create parent directory for %s: %w", entry.Path, err)
		}

		// Write the file content
		if err := os.WriteFile(filePath, blobObj.Data, 0644); err != nil {
			return fmt.Errorf("write file %s: %w", entry.Path, err)
		}

		filesWritten++
		logger.Debug("File written",
			"path", entry.Path,
			"size", len(blobObj.Data))
	}

	logger.Debug("All files written to disk",
		"base_path", basePath,
		"expected_files", len(blobEntries),
		"files_written", filesWritten)

	// Final validation
	if filesWritten != len(blobEntries) {
		logger.Error("File count mismatch after writing",
			"expected_files", len(blobEntries),
			"files_written", filesWritten)
		return fmt.Errorf("expected to write %d files but only wrote %d", len(blobEntries), filesWritten)
	}

	return nil
}

// fetchBlobsBatched efficiently fetches multiple blobs in batches to reduce HTTP requests
// and improve performance for cloning operations with many files.
func (c *httpClient) fetchBlobsBatched(ctx context.Context, blobHashes []hash.Hash) (map[string]*protocol.PackfileObject, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Fetching blobs in batches",
		"total_blobs", len(blobHashes),
		"batch_size", 100) // We'll use a reasonable batch size

	if len(blobHashes) == 0 {
		return make(map[string]*protocol.PackfileObject), nil
	}

	// For small numbers of blobs, fetch them all at once
	const maxSingleBatch = 100
	if len(blobHashes) <= maxSingleBatch {
		return c.fetchBlobBatch(ctx, blobHashes)
	}

	// For larger numbers, split into multiple batches and process sequentially (to debug issues)
	const batchSize = 50
	numBatches := (len(blobHashes) + batchSize - 1) / batchSize
	
	logger.Debug("Splitting into multiple batches (sequential processing)",
		"batch_count", numBatches,
		"batch_size", batchSize)

	// Process batches sequentially to avoid race conditions
	allObjects := make(map[string]*protocol.PackfileObject)
	for i := 0; i < numBatches; i++ {
		start := i * batchSize
		end := start + batchSize
		if end > len(blobHashes) {
			end = len(blobHashes)
		}
		
		batchHashes := blobHashes[start:end]
		
		logger.Debug("Processing batch",
			"batch_id", i,
			"blob_count", len(batchHashes))
		
		objects, err := c.fetchBlobBatch(ctx, batchHashes)
		if err != nil {
			logger.Error("Batch fetch failed",
				"batch_id", i,
				"error", err)
			return nil, fmt.Errorf("batch %d failed: %w", i, err)
		}

		logger.Debug("Batch fetch completed",
			"batch_id", i,
			"requested", len(batchHashes),
			"received", len(objects))

		// Check for missing blobs in this batch and try to fetch them individually
		if len(objects) != len(batchHashes) {
			logger.Warn("Batch blob count mismatch - trying individual fetch for missing blobs",
				"batch_id", i,
				"requested", len(batchHashes),
				"received", len(objects))
			
			// Find missing blobs and try to fetch them individually
			missingHashes := make([]hash.Hash, 0)
			for _, expectedHash := range batchHashes {
				if _, found := objects[expectedHash.String()]; !found {
					missingHashes = append(missingHashes, expectedHash)
					logger.Debug("Will retry missing blob individually",
						"batch_id", i,
						"blob_hash", expectedHash.String())
				}
			}
			
			// Try to fetch missing blobs individually using the old GetBlob method
			logger.Debug("Fetching missing blobs individually",
				"batch_id", i,
				"missing_count", len(missingHashes))
			
			for _, missingHash := range missingHashes {
				logger.Debug("Fetching individual blob",
					"blob_hash", missingHash.String())
				
				// Use the old GetBlob method as fallback
				blob, err := c.GetBlob(ctx, missingHash)
				if err != nil {
					logger.Error("Failed to fetch missing blob individually",
						"blob_hash", missingHash.String(),
						"error", err)
					continue // Skip this blob, but continue with others
				}
				
				// Convert Blob to PackfileObject format
				blobObj := &protocol.PackfileObject{
					Hash: missingHash,
					Type: protocol.ObjectTypeBlob,
					Data: blob.Content,
				}
				
				objects[missingHash.String()] = blobObj
				logger.Debug("Successfully fetched missing blob individually",
					"blob_hash", missingHash.String(),
					"size", len(blob.Content))
			}
			
			logger.Debug("Individual fetch completed",
				"batch_id", i,
				"missing_attempted", len(missingHashes),
				"total_in_batch_now", len(objects))
		}

		// Merge objects from this batch
		for hash, obj := range objects {
			allObjects[hash] = obj
		}

		logger.Debug("Batch completed",
			"batch_id", i,
			"objects_fetched", len(objects),
			"total_objects_so_far", len(allObjects))
	}

	logger.Debug("All batches completed",
		"total_objects", len(allObjects),
		"total_batches", numBatches)

	return allObjects, nil
}

// fetchBlobBatch fetches a single batch of blobs using the underlying Fetch API
func (c *httpClient) fetchBlobBatch(ctx context.Context, blobHashes []hash.Hash) (map[string]*protocol.PackfileObject, error) {
	if len(blobHashes) == 0 {
		return make(map[string]*protocol.PackfileObject), nil
	}

	logger := log.FromContext(ctx)
	logger.Debug("Fetching blob batch",
		"blob_count", len(blobHashes))

	// Use the existing Fetch method to get multiple blobs at once
	objects, err := c.Fetch(ctx, client.FetchOptions{
		NoProgress:     true,
		Want:           blobHashes,
		Done:           true,
		NoExtraObjects: true, // Only get the blobs we requested
	})
	if err != nil {
		return nil, fmt.Errorf("fetch blob batch: %w", err)
	}

	// Verify we only got blob objects
	blobObjects := make(map[string]*protocol.PackfileObject)
	for hash, obj := range objects {
		if obj.Type != protocol.ObjectTypeBlob {
			logger.Warn("Unexpected object type in blob batch",
				"hash", hash,
				"expected_type", protocol.ObjectTypeBlob,
				"actual_type", obj.Type)
			continue
		}
		blobObjects[hash] = obj
	}

	logger.Debug("Blob batch fetched successfully",
		"requested_count", len(blobHashes),
		"received_count", len(blobObjects))

	return blobObjects, nil
}
