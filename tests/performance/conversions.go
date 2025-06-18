package performance

import "github.com/grafana/nanogit/tests/performance/clients"

// Convert client types to performance types to avoid import cycles

func convertCommitComparison(c *clients.CommitComparison) *CommitComparison {
	if c == nil {
		return nil
	}
	
	files := make([]FileChangeSummary, len(c.Files))
	for i, f := range c.Files {
		files[i] = FileChangeSummary{
			Path:      f.Path,
			Status:    f.Status,
			Additions: f.Additions,
			Deletions: f.Deletions,
		}
	}
	
	return &CommitComparison{
		FilesChanged: c.FilesChanged,
		Additions:    c.Additions,
		Deletions:    c.Deletions,
		Files:        files,
	}
}

func convertTreeResult(t *clients.TreeResult) *TreeResult {
	if t == nil {
		return nil
	}
	
	files := make([]TreeFile, len(t.Files))
	for i, f := range t.Files {
		files[i] = TreeFile{
			Path: f.Path,
			Size: f.Size,
			Type: f.Type,
		}
	}
	
	return &TreeResult{
		Files: files,
		Count: t.Count,
	}
}

func convertFileChanges(files []FileChange) []clients.FileChange {
	clientFiles := make([]clients.FileChange, len(files))
	for i, f := range files {
		clientFiles[i] = clients.FileChange{
			Path:    f.Path,
			Content: f.Content,
			Action:  f.Action,
		}
	}
	return clientFiles
}