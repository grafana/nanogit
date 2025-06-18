package performance

import (
	"context"

	"github.com/grafana/nanogit/tests/performance/clients"
)

// ClientWrapper wraps the client implementations to handle type conversions
type ClientWrapper struct {
	name     string
	nanogit  *clients.NanogitClient
	gogit    *clients.GoGitClient
	gitcli   *clients.GitCLIClient
}

func NewNanogitClientWrapper() GitClient {
	return &ClientWrapper{
		name:    "nanogit",
		nanogit: clients.NewNanogitClient(),
	}
}

func NewGoGitClientWrapper() GitClient {
	return &ClientWrapper{
		name:  "go-git",
		gogit: clients.NewGoGitClient(),
	}
}

func NewGitCLIClientWrapper() (GitClient, error) {
	gitcli, err := clients.NewGitCLIClient()
	if err != nil {
		return nil, err
	}
	return &ClientWrapper{
		name:   "git-cli",
		gitcli: gitcli,
	}, nil
}

func (w *ClientWrapper) Name() string {
	return w.name
}

func (w *ClientWrapper) CreateFile(ctx context.Context, repoURL, path, content, message string) error {
	switch {
	case w.nanogit != nil:
		return w.nanogit.CreateFile(ctx, repoURL, path, content, message)
	case w.gogit != nil:
		return w.gogit.CreateFile(ctx, repoURL, path, content, message)
	case w.gitcli != nil:
		return w.gitcli.CreateFile(ctx, repoURL, path, content, message)
	default:
		return nil
	}
}

func (w *ClientWrapper) UpdateFile(ctx context.Context, repoURL, path, content, message string) error {
	switch {
	case w.nanogit != nil:
		return w.nanogit.UpdateFile(ctx, repoURL, path, content, message)
	case w.gogit != nil:
		return w.gogit.UpdateFile(ctx, repoURL, path, content, message)
	case w.gitcli != nil:
		return w.gitcli.UpdateFile(ctx, repoURL, path, content, message)
	default:
		return nil
	}
}

func (w *ClientWrapper) DeleteFile(ctx context.Context, repoURL, path, message string) error {
	switch {
	case w.nanogit != nil:
		return w.nanogit.DeleteFile(ctx, repoURL, path, message)
	case w.gogit != nil:
		return w.gogit.DeleteFile(ctx, repoURL, path, message)
	case w.gitcli != nil:
		return w.gitcli.DeleteFile(ctx, repoURL, path, message)
	default:
		return nil
	}
}

func (w *ClientWrapper) CompareCommits(ctx context.Context, repoURL, base, head string) (*CommitComparison, error) {
	switch {
	case w.nanogit != nil:
		result, err := w.nanogit.CompareCommits(ctx, repoURL, base, head)
		return convertCommitComparison(result), err
	case w.gogit != nil:
		result, err := w.gogit.CompareCommits(ctx, repoURL, base, head)
		return convertCommitComparison(result), err
	case w.gitcli != nil:
		result, err := w.gitcli.CompareCommits(ctx, repoURL, base, head)
		return convertCommitComparison(result), err
	default:
		return nil, nil
	}
}

func (w *ClientWrapper) GetFlatTree(ctx context.Context, repoURL, ref string) (*TreeResult, error) {
	switch {
	case w.nanogit != nil:
		result, err := w.nanogit.GetFlatTree(ctx, repoURL, ref)
		return convertTreeResult(result), err
	case w.gogit != nil:
		result, err := w.gogit.GetFlatTree(ctx, repoURL, ref)
		return convertTreeResult(result), err
	case w.gitcli != nil:
		result, err := w.gitcli.GetFlatTree(ctx, repoURL, ref)
		return convertTreeResult(result), err
	default:
		return nil, nil
	}
}

func (w *ClientWrapper) BulkCreateFiles(ctx context.Context, repoURL string, files []FileChange, message string) error {
	clientFiles := convertFileChanges(files)
	
	switch {
	case w.nanogit != nil:
		return w.nanogit.BulkCreateFiles(ctx, repoURL, clientFiles, message)
	case w.gogit != nil:
		return w.gogit.BulkCreateFiles(ctx, repoURL, clientFiles, message)
	case w.gitcli != nil:
		return w.gitcli.BulkCreateFiles(ctx, repoURL, clientFiles, message)
	default:
		return nil
	}
}