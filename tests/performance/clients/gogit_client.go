package clients

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/utils/merkletrie"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-billy/v5/memfs"
)

// GoGitClient implements the GitClient interface using go-git
type GoGitClient struct {
	repos map[string]*git.Repository // Cache repos by URL
	auths map[string]*http.BasicAuth  // Cache auth info by URL
}

// NewGoGitClient creates a new go-git client
func NewGoGitClient() *GoGitClient {
	return &GoGitClient{
		repos: make(map[string]*git.Repository),
		auths: make(map[string]*http.BasicAuth),
	}
}

// Name returns the client name
func (c *GoGitClient) Name() string {
	return "go-git"
}

// getOrCloneRepo gets a repository from cache or clones it
func (c *GoGitClient) getOrCloneRepo(ctx context.Context, repoURL string) (*git.Repository, error) {
	if repo, exists := c.repos[repoURL]; exists {
		return repo, nil
	}

	// Parse URL to extract credentials and clean URL
	u, err := url.Parse(repoURL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	// Prepare clone options
	cloneOpts := &git.CloneOptions{
		URL: repoURL,
	}

	// Add authentication if present in URL
	if u.User != nil {
		username := u.User.Username()
		password, _ := u.User.Password()
		auth := &http.BasicAuth{
			Username: username,
			Password: password,
		}
		cloneOpts.Auth = auth
		
		// Store auth info for later push operations
		c.auths[repoURL] = auth
		
		// Use clean URL without credentials
		cleanURL := &url.URL{
			Scheme: u.Scheme,
			Host:   u.Host,
			Path:   u.Path,
		}
		cloneOpts.URL = cleanURL.String()
	}

	// Create in-memory filesystem for worktree
	fs := memfs.New()
	
	// Create .git directory for storage
	gitDir, err := fs.Chroot("/.git")
	if err != nil {
		return nil, fmt.Errorf("failed to create .git directory: %w", err)
	}
	
	// Create storage with filesystem support
	storage := filesystem.NewStorage(gitDir, cache.NewObjectLRUDefault())

	// Clone repository with filesystem storage to support worktree
	repo, err := git.CloneContext(ctx, storage, fs, cloneOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	c.repos[repoURL] = repo
	return repo, nil
}

// pushChanges pushes local changes to the remote repository
func (c *GoGitClient) pushChanges(ctx context.Context, repo *git.Repository, repoURL string) error {
	// Parse URL to get clean URL for push
	u, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("invalid repository URL: %w", err)
	}

	cleanURL := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   u.Path,
	}

	pushOpts := &git.PushOptions{
		RemoteName: "origin",
		RemoteURL:  cleanURL.String(),
	}

	// Add authentication if available
	if auth, exists := c.auths[repoURL]; exists {
		pushOpts.Auth = auth
	}

	return repo.PushContext(ctx, pushOpts)
}

// CreateFile creates a new file in the repository
func (c *GoGitClient) CreateFile(ctx context.Context, repoURL, path, content, message string) error {
	repo, err := c.getOrCloneRepo(ctx, repoURL)
	if err != nil {
		return err
	}

	// Get worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Create file
	file, err := worktree.Filesystem.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write([]byte(content)); err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	// Add file to staging
	if _, err := worktree.Add(path); err != nil {
		return fmt.Errorf("failed to add file to staging: %w", err)
	}

	// Commit
	_, err = worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Performance Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push changes to remote
	if err := c.pushChanges(ctx, repo, repoURL); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

// UpdateFile updates an existing file in the repository
func (c *GoGitClient) UpdateFile(ctx context.Context, repoURL, path, content, message string) error {
	repo, err := c.getOrCloneRepo(ctx, repoURL)
	if err != nil {
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Update file
	file, err := worktree.Filesystem.OpenFile(path, 1, 0644) // O_WRONLY
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}

	if _, err := file.Write([]byte(content)); err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	// Add and commit
	if _, err := worktree.Add(path); err != nil {
		return fmt.Errorf("failed to add file to staging: %w", err)
	}

	_, err = worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Performance Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// DeleteFile deletes a file from the repository
func (c *GoGitClient) DeleteFile(ctx context.Context, repoURL, path, message string) error {
	repo, err := c.getOrCloneRepo(ctx, repoURL)
	if err != nil {
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Remove file
	if err := worktree.Filesystem.Remove(path); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	// Stage deletion
	if _, err := worktree.Add(path); err != nil {
		return fmt.Errorf("failed to stage deletion: %w", err)
	}

	// Commit
	_, err = worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Performance Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push changes to remote
	if err := c.pushChanges(ctx, repo, repoURL); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

// CompareCommits compares two commits and returns the differences
func (c *GoGitClient) CompareCommits(ctx context.Context, repoURL, base, head string) (*CommitComparison, error) {
	repo, err := c.getOrCloneRepo(ctx, repoURL)
	if err != nil {
		return nil, err
	}

	// Get commits
	baseCommit, err := repo.CommitObject(plumbing.NewHash(base))
	if err != nil {
		return nil, fmt.Errorf("failed to get base commit: %w", err)
	}

	headCommit, err := repo.CommitObject(plumbing.NewHash(head))
	if err != nil {
		return nil, fmt.Errorf("failed to get head commit: %w", err)
	}

	// Get trees
	baseTree, err := baseCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get base tree: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get head tree: %w", err)
	}

	// Compare trees
	changes, err := object.DiffTree(baseTree, headTree)
	if err != nil {
		return nil, fmt.Errorf("failed to diff trees: %w", err)
	}

	comparison := &CommitComparison{
		Files: make([]FileChangeSummary, 0, len(changes)),
	}

	for _, change := range changes {
		var status string
		action, err := change.Action()
		if err != nil {
			continue // Skip if we can't get action
		}
		switch action {
		case merkletrie.Insert:
			status = "added"
		case merkletrie.Modify:
			status = "modified"
		case merkletrie.Delete:
			status = "deleted"
		}

		// Get patch for line counts
		patch, err := change.Patch()
		if err != nil {
			continue // Skip if we can't get patch
		}

		stats := patch.Stats()
		additions := 0
		deletions := 0
		if len(stats) > 0 {
			additions = stats[0].Addition
			deletions = stats[0].Deletion
		}

		comparison.Files = append(comparison.Files, FileChangeSummary{
			Path:      change.To.Name,
			Status:    status,
			Additions: additions,
			Deletions: deletions,
		})

		comparison.Additions += additions
		comparison.Deletions += deletions
	}

	comparison.FilesChanged = len(comparison.Files)
	return comparison, nil
}

// GetFlatTree returns a flat listing of all files in the repository at a given ref
func (c *GoGitClient) GetFlatTree(ctx context.Context, repoURL, ref string) (*TreeResult, error) {
	repo, err := c.getOrCloneRepo(ctx, repoURL)
	if err != nil {
		return nil, err
	}

	// Resolve reference
	var hash plumbing.Hash
	if ref == "HEAD" {
		headRef, err := repo.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to get HEAD: %w", err)
		}
		hash = headRef.Hash()
	} else {
		hash = plumbing.NewHash(ref)
	}

	// Get commit
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Get tree
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	result := &TreeResult{
		Files: make([]TreeFile, 0),
	}

	// Walk tree
	err = tree.Files().ForEach(func(f *object.File) error {
		result.Files = append(result.Files, TreeFile{
			Path: f.Name,
			Size: f.Size,
			Type: "blob",
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk tree: %w", err)
	}

	result.Count = len(result.Files)
	return result, nil
}

// BulkCreateFiles creates multiple files in a single commit
func (c *GoGitClient) BulkCreateFiles(ctx context.Context, repoURL string, files []FileChange, message string) error {
	repo, err := c.getOrCloneRepo(ctx, repoURL)
	if err != nil {
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Process all files
	for _, fileChange := range files {
		switch strings.ToLower(fileChange.Action) {
		case "create", "update":
			file, err := worktree.Filesystem.Create(fileChange.Path)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", fileChange.Path, err)
			}
			
			if _, err := file.Write([]byte(fileChange.Content)); err != nil {
				file.Close()
				return fmt.Errorf("failed to write file %s: %w", fileChange.Path, err)
			}
			file.Close()
			
			if _, err := worktree.Add(fileChange.Path); err != nil {
				return fmt.Errorf("failed to add file %s: %w", fileChange.Path, err)
			}

		case "delete":
			if err := worktree.Filesystem.Remove(fileChange.Path); err != nil {
				return fmt.Errorf("failed to remove file %s: %w", fileChange.Path, err)
			}
			
			if _, err := worktree.Add(fileChange.Path); err != nil {
				return fmt.Errorf("failed to stage deletion %s: %w", fileChange.Path, err)
			}
		}
	}

	// Commit all changes
	_, err = worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Performance Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit bulk changes: %w", err)
	}

	// Push changes to remote
	if err := c.pushChanges(ctx, repo, repoURL); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}