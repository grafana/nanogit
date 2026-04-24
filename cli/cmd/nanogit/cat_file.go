package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/grafana/nanogit"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(catFileCmd)
}

var catFileCmd = &cobra.Command{
	Use:   "cat-file <repository> <ref> <path>",
	Short: "Display the contents of a file",
	Long: `Display the contents of a file from a Git repository at a specific reference.

The ref can be a branch name, tag name, or commit hash.
The path is the file path within the repository.

Examples:
  # Display file contents
  nanogit cat-file https://github.com/grafana/nanogit.git main README.md

  # Display file from a specific tag
  nanogit cat-file https://github.com/grafana/nanogit.git v1.0.0 docs/api.md

  # Display file from a commit hash
  nanogit cat-file https://github.com/grafana/nanogit.git abc123 src/main.go

  # Output with metadata in JSON format
  nanogit cat-file https://github.com/grafana/nanogit.git main README.md --json

  # With authentication
  nanogit cat-file https://github.com/user/private-repo.git main file.txt --token <token>
  NANOGIT_TOKEN=<token> nanogit cat-file https://github.com/user/private-repo.git main file.txt`,
	Args: cobra.ExactArgs(3),
	RunE: runCatFile,
}

func runCatFile(cmd *cobra.Command, args []string) error {
	repoURL := args[0]
	ref := args[1]
	filePath := args[2]

	ctx, client, err := setupClient(context.Background(), repoURL)
	if err != nil {
		return err
	}

	// Resolve ref to get the commit hash
	commitHash, err := resolveRef(ctx, client, ref)
	if err != nil {
		return fmt.Errorf("failed to resolve ref %q: %w", ref, err)
	}

	// Get the commit to find the tree hash
	commit, err := client.GetCommit(ctx, commitHash)
	if err != nil {
		return fmt.Errorf("failed to get commit: %w", err)
	}

	// Get the file blob by path
	blob, err := client.GetBlobByPath(ctx, commit.Tree, filePath)
	if err != nil {
		return fmt.Errorf("failed to get file %q: %w", filePath, err)
	}

	// Output the file contents
	if globalJSON {
		return outputBlobJSON(blob, filePath)
	}
	return outputBlobRaw(blob)
}

type blobJSON struct {
	Path    string `json:"path"`
	Hash    string `json:"hash"`
	Size    int    `json:"size"`
	Content string `json:"content"`
}

func outputBlobJSON(blob *nanogit.Blob, path string) error {
	output := blobJSON{
		Path:    path,
		Hash:    blob.Hash.String(),
		Size:    len(blob.Content),
		Content: string(blob.Content),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputBlobRaw(blob *nanogit.Blob) error {
	_, err := os.Stdout.Write(blob.Content)
	return err
}
