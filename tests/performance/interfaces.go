package performance

import (
	"context"
	"time"
)

// GitClient represents a common interface for all Git implementations
type GitClient interface {
	Name() string
	CreateFile(ctx context.Context, repoURL, path, content, message string) error
	UpdateFile(ctx context.Context, repoURL, path, content, message string) error
	DeleteFile(ctx context.Context, repoURL, path, message string) error
	CompareCommits(ctx context.Context, repoURL, base, head string) (*CommitComparison, error)
	GetFlatTree(ctx context.Context, repoURL, ref string) (*TreeResult, error)
	BulkCreateFiles(ctx context.Context, repoURL string, files []FileChange, message string) error
}

// FileChange represents a file operation for bulk operations
type FileChange struct {
	Path    string
	Content string
	Action  string // "create", "update", "delete"
}

// CommitComparison represents the result of comparing two commits
type CommitComparison struct {
	FilesChanged int
	Additions    int
	Deletions    int
	Files        []FileChangeSummary
}

// FileChangeSummary represents a file change in a commit comparison
type FileChangeSummary struct {
	Path      string
	Status    string // "added", "modified", "deleted"
	Additions int
	Deletions int
}

// TreeResult represents a flat tree listing
type TreeResult struct {
	Files []TreeFile
	Count int
}

// TreeFile represents a file in a tree
type TreeFile struct {
	Path string
	Size int64
	Type string // "blob", "tree"
}

// BenchmarkResult represents the result of a single benchmark run
type BenchmarkResult struct {
	Client       string        `json:"client"`
	Operation    string        `json:"operation"`
	Scenario     string        `json:"scenario"`
	Duration     time.Duration `json:"duration"`
	MemoryUsed   int64         `json:"memory_used"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	RepoSize     string        `json:"repo_size"`
	FileCount    int           `json:"file_count"`
	ExtraMetrics map[string]interface{} `json:"extra_metrics,omitempty"`
}

// BenchmarkConfig represents configuration for a benchmark run
type BenchmarkConfig struct {
	RepoURL     string
	RepoSize    string // "small", "medium", "large"
	FileCount   int
	Iterations  int
	Timeout     time.Duration
}