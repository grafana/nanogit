package clients

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
