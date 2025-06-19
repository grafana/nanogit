package clients

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol/hash"
)

// NanogitClient implements the GitClient interface using nanogit
type NanogitClient struct {
	clients map[string]nanogit.Client // Cache clients by repo URL
}

// NewNanogitClient creates a new nanogit client
func NewNanogitClient() *NanogitClient {
	return &NanogitClient{
		clients: make(map[string]nanogit.Client),
	}
}

// getOrCreateClient gets a client for the repository, creating one if needed
func (c *NanogitClient) getOrCreateClient(repoURL string) (nanogit.Client, error) {
	if client, exists := c.clients[repoURL]; exists {
		return client, nil
	}

	// Parse URL to extract credentials
	u, err := url.Parse(repoURL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	// Remove credentials from URL for client creation
	cleanURL := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   u.Path,
	}

	var opts []options.Option
	if u.User != nil {
		username := u.User.Username()
		password, _ := u.User.Password()
		opts = append(opts, options.WithBasicAuth(username, password))
	}

	client, err := nanogit.NewHTTPClient(cleanURL.String(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create nanogit client: %w", err)
	}

	c.clients[repoURL] = client
	return client, nil
}

// getMainRef gets the main branch reference
func (c *NanogitClient) getMainRef(ctx context.Context, client nanogit.Client) (nanogit.Ref, error) {
	refs, err := client.ListRefs(ctx)
	if err != nil {
		return nanogit.Ref{}, fmt.Errorf("failed to list refs: %w", err)
	}

	for _, ref := range refs {
		if ref.Name == "refs/heads/main" {
			return ref, nil
		}
	}
	return nanogit.Ref{}, fmt.Errorf("main branch not found")
}

// Name returns the client name
func (c *NanogitClient) Name() string {
	return "nanogit"
}

// CreateFile creates a new file in the repository
func (c *NanogitClient) CreateFile(ctx context.Context, repoURL, path, content, message string) error {
	client, err := c.getOrCreateClient(repoURL)
	if err != nil {
		return err
	}

	// Get the main branch reference
	mainRef, err := c.getMainRef(ctx, client)
	if err != nil {
		return err
	}

	// Create staged writer
	writer, err := client.NewStagedWriter(ctx, mainRef)
	if err != nil {
		return fmt.Errorf("failed to create staged writer: %w", err)
	}

	// Create the blob
	_, err = writer.CreateBlob(ctx, path, []byte(content))
	if err != nil {
		return fmt.Errorf("failed to create blob: %w", err)
	}

	// Commit the changes
	author := nanogit.Author{
		Name:  "Performance Test",
		Email: "test@example.com",
	}
	committer := nanogit.Committer{
		Name:  "Performance Test",
		Email: "test@example.com",
	}

	_, err = writer.Commit(ctx, message, author, committer)
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push the changes
	err = writer.Push(ctx)
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// UpdateFile updates an existing file in the repository
func (c *NanogitClient) UpdateFile(ctx context.Context, repoURL, path, content, message string) error {
	client, err := c.getOrCreateClient(repoURL)
	if err != nil {
		return err
	}

	// Get the main branch reference
	mainRef, err := c.getMainRef(ctx, client)
	if err != nil {
		return err
	}

	// Create staged writer
	writer, err := client.NewStagedWriter(ctx, mainRef)
	if err != nil {
		return fmt.Errorf("failed to create staged writer: %w", err)
	}

	// Update the blob
	_, err = writer.UpdateBlob(ctx, path, []byte(content))
	if err != nil {
		return fmt.Errorf("failed to update blob: %w", err)
	}

	// Commit the changes
	author := nanogit.Author{
		Name:  "Performance Test",
		Email: "test@example.com",
	}
	committer := nanogit.Committer{
		Name:  "Performance Test",
		Email: "test@example.com",
	}

	_, err = writer.Commit(ctx, message, author, committer)
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push the changes
	err = writer.Push(ctx)
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// DeleteFile deletes a file from the repository
func (c *NanogitClient) DeleteFile(ctx context.Context, repoURL, path, message string) error {
	client, err := c.getOrCreateClient(repoURL)
	if err != nil {
		return err
	}

	// Get the main branch reference
	mainRef, err := c.getMainRef(ctx, client)
	if err != nil {
		return err
	}

	// Create staged writer
	writer, err := client.NewStagedWriter(ctx, mainRef)
	if err != nil {
		return fmt.Errorf("failed to create staged writer: %w", err)
	}

	// Delete the blob
	_, err = writer.DeleteBlob(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to delete blob: %w", err)
	}

	// Commit the changes
	author := nanogit.Author{
		Name:  "Performance Test",
		Email: "test@example.com",
	}
	committer := nanogit.Committer{
		Name:  "Performance Test",
		Email: "test@example.com",
	}

	_, err = writer.Commit(ctx, message, author, committer)
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push the changes
	err = writer.Push(ctx)
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// CompareCommits compares two commits and returns the differences
func (c *NanogitClient) CompareCommits(ctx context.Context, repoURL, base, head string) (*CommitComparison, error) {
	client, err := c.getOrCreateClient(repoURL)
	if err != nil {
		return nil, err
	}

	// Resolve commit hashes from refs (like HEAD~1, HEAD, etc.)
	baseHash, err := c.resolveCommitRef(ctx, client, base)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base commit %s: %w", base, err)
	}

	headHash, err := c.resolveCommitRef(ctx, client, head)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve head commit %s: %w", head, err)
	}

	// Compare commits using nanogit
	commitFiles, err := client.CompareCommits(ctx, baseHash, headHash)
	if err != nil {
		return nil, fmt.Errorf("failed to compare commits: %w", err)
	}

	// Convert to our format
	comparison := &CommitComparison{
		FilesChanged: len(commitFiles),
		Files:        make([]FileChangeSummary, 0, len(commitFiles)),
	}

	for _, file := range commitFiles {
		var status string
		switch file.Status {
		case "A":
			status = "added"
		case "M":
			status = "modified"
		case "D":
			status = "deleted"
		default:
			status = "modified"
		}

		comparison.Files = append(comparison.Files, FileChangeSummary{
			Path:      file.Path,
			Status:    status,
			Additions: 0, // nanogit doesn't provide line-level diffs
			Deletions: 0, // nanogit doesn't provide line-level diffs
		})
	}

	return comparison, nil
}

// resolveCommitRef resolves a commit reference (like HEAD, HEAD~1) to a hash
func (c *NanogitClient) resolveCommitRef(ctx context.Context, client nanogit.Client, ref string) (hash.Hash, error) {
	// For simple cases, try to parse as hash directly
	if len(ref) == 40 {
		if h, err := hash.FromHex(ref); err == nil {
			return h, nil
		}
	}

	// For HEAD and other references, get the main branch
	refs, err := client.ListRefs(ctx)
	if err != nil {
		return hash.Hash{}, fmt.Errorf("failed to list refs: %w", err)
	}

	var mainRef nanogit.Ref
	for _, r := range refs {
		if r.Name == "refs/heads/main" {
			mainRef = r
			break
		}
	}
	if mainRef.Name == "" {
		return hash.Hash{}, fmt.Errorf("main branch not found")
	}

	// For HEAD, return the main branch hash
	if ref == "HEAD" {
		return mainRef.Hash, nil
	}

	// For HEAD~N, we need to walk back through commits
	if strings.HasPrefix(ref, "HEAD~") {
		stepsBack := 1
		if len(ref) > 5 {
			if n, err := fmt.Sscanf(ref, "HEAD~%d", &stepsBack); err != nil || n != 1 {
				return hash.Hash{}, fmt.Errorf("invalid ref format: %s", ref)
			}
		}

		currentHash := mainRef.Hash
		for i := 0; i < stepsBack; i++ {
			commit, err := client.GetCommit(ctx, currentHash)
			if err != nil {
				return hash.Hash{}, fmt.Errorf("failed to get commit: %w", err)
			}
			if commit.Parent.String() == "" {
				return hash.Hash{}, fmt.Errorf("reached root commit")
			}
			currentHash = commit.Parent // Take parent
		}
		return currentHash, nil
	}

	return hash.Hash{}, fmt.Errorf("unsupported ref format: %s", ref)
}

// GetFlatTree returns a flat listing of all files in the repository at a given ref
func (c *NanogitClient) GetFlatTree(ctx context.Context, repoURL, ref string) (*TreeResult, error) {
	client, err := c.getOrCreateClient(repoURL)
	if err != nil {
		return nil, err
	}

	// Resolve the ref to a commit hash
	commitHash, err := c.resolveCommitRef(ctx, client, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve ref %s: %w", ref, err)
	}

	// Get the commit to get its tree hash
	commit, err := client.GetCommit(ctx, commitHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Get the flat tree
	flatTree, err := client.GetFlatTree(ctx, commit.Tree)
	if err != nil {
		return nil, fmt.Errorf("failed to get flat tree: %w", err)
	}

	// Convert to our format
	result := &TreeResult{
		Files: make([]TreeFile, 0, len(flatTree.Entries)),
		Count: len(flatTree.Entries),
	}

	for _, entry := range flatTree.Entries {
		result.Files = append(result.Files, TreeFile{
			Path: entry.Path,
			Size: 0, // nanogit FlatTreeEntry doesn't provide size
			Type: "blob",
		})
	}

	return result, nil
}

// BulkCreateFiles creates multiple files in a single commit
func (c *NanogitClient) BulkCreateFiles(ctx context.Context, repoURL string, files []FileChange, message string) error {
	client, err := c.getOrCreateClient(repoURL)
	if err != nil {
		return err
	}

	// Get the main branch reference
	mainRef, err := c.getMainRef(ctx, client)
	if err != nil {
		return err
	}

	// Create staged writer
	writer, err := client.NewStagedWriter(ctx, mainRef)
	if err != nil {
		return fmt.Errorf("failed to create staged writer: %w", err)
	}

	// Process all files
	for _, file := range files {
		switch strings.ToLower(file.Action) {
		case "create":
			_, err = writer.CreateBlob(ctx, file.Path, []byte(file.Content))
			if err != nil {
				return fmt.Errorf("failed to create blob %s: %w", file.Path, err)
			}
		case "update":
			_, err = writer.UpdateBlob(ctx, file.Path, []byte(file.Content))
			if err != nil {
				return fmt.Errorf("failed to update blob %s: %w", file.Path, err)
			}
		case "delete":
			_, err = writer.DeleteBlob(ctx, file.Path)
			if err != nil {
				return fmt.Errorf("failed to delete blob %s: %w", file.Path, err)
			}
		}
	}

	// Commit all changes
	author := nanogit.Author{
		Name:  "Performance Test",
		Email: "test@example.com",
	}
	committer := nanogit.Committer{
		Name:  "Performance Test",
		Email: "test@example.com",
	}

	_, err = writer.Commit(ctx, message, author, committer)
	if err != nil {
		return fmt.Errorf("failed to commit bulk changes: %w", err)
	}

	// Push the changes
	err = writer.Push(ctx)
	if err != nil {
		return fmt.Errorf("failed to push bulk changes: %w", err)
	}

	return nil
}
