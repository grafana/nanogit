package main

import (
	"context"
	"fmt"
	"os"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
	"github.com/spf13/cobra"
)

var (
	cloneInclude  []string
	cloneExclude  []string
	cloneUsername string
	cloneToken    string
)

func init() {
	rootCmd.AddCommand(cloneCmd)

	cloneCmd.Flags().StringSliceVar(&cloneInclude, "include", nil, "Include paths (glob patterns, e.g., 'src/**', '*.go')")
	cloneCmd.Flags().StringSliceVar(&cloneExclude, "exclude", nil, "Exclude paths (glob patterns, e.g., 'node_modules/**', '*.tmp')")
	cloneCmd.Flags().StringVar(&cloneUsername, "username", "", "Authentication username (can also use NANOGIT_USERNAME env var, defaults to 'git')")
	cloneCmd.Flags().StringVar(&cloneToken, "token", "", "Authentication token (can also use NANOGIT_TOKEN env var)")
}

var cloneCmd = &cobra.Command{
	Use:   "clone <repository> <ref> <destination>",
	Short: "Clone a repository to a local directory",
	Long: `Clone a repository from a specific reference to a local directory.

The ref can be a branch name, tag name, or commit hash.
Supports path filtering with glob patterns to clone only specific files or directories.

Examples:
  # Clone entire repository
  nanogit clone https://github.com/grafana/nanogit.git main ./my-repo

  # Clone only specific directories
  nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --include 'src/**' --include 'docs/**'

  # Clone excluding certain paths
  nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --exclude 'node_modules/**' --exclude '*.tmp'

  # Clone with include and exclude patterns
  nanogit clone https://github.com/grafana/nanogit.git main ./my-repo --include 'src/**' --exclude '*.test.go'

  # Clone from a specific tag
  nanogit clone https://github.com/grafana/nanogit.git v1.0.0 ./my-repo

  # With authentication
  nanogit clone https://github.com/user/private-repo.git main ./my-repo --token <token>
  NANOGIT_TOKEN=<token> nanogit clone https://github.com/user/private-repo.git main ./my-repo`,
	Args: cobra.ExactArgs(3),
	RunE: runClone,
}

func runClone(cmd *cobra.Command, args []string) error {
	repoURL := args[0]
	ref := args[1]
	destPath := args[2]
	ctx := context.Background()

	// Get authentication credentials from flags or environment
	token := cloneToken
	if token == "" {
		token = os.Getenv("NANOGIT_TOKEN")
	}

	username := cloneUsername
	if username == "" {
		username = os.Getenv("NANOGIT_USERNAME")
	}
	if username == "" {
		username = "git"
	}

	// Create client with optional authentication
	var client nanogit.Client
	var err error
	if token != "" {
		client, err = nanogit.NewHTTPClient(repoURL,
			options.WithBasicAuth(username, token),
			options.WithoutGitSuffix())
	} else {
		client, err = nanogit.NewHTTPClient(repoURL, options.WithoutGitSuffix())
	}
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Resolve ref to get the commit hash
	commitHash, err := resolveRef(ctx, client, ref)
	if err != nil {
		return fmt.Errorf("failed to resolve ref %q: %w", ref, err)
	}

	fmt.Printf("Cloning %s at %s to %s...\n", repoURL, ref, destPath)

	// Prepare clone options
	cloneOpts := nanogit.CloneOptions{
		Path:         destPath,
		Hash:         commitHash,
		IncludePaths: cloneInclude,
		ExcludePaths: cloneExclude,
	}

	// Clone the repository
	result, err := client.Clone(ctx, cloneOpts)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Display results
	fmt.Printf("\nClone complete!\n")
	fmt.Printf("  Commit:   %s\n", result.Commit.Hash.String()[:8])
	fmt.Printf("  Message:  %s\n", firstLine(result.Commit.Message))
	fmt.Printf("  Author:   %s <%s>\n", result.Commit.Author.Name, result.Commit.Author.Email)
	fmt.Printf("  Files:    %d of %d cloned to %s\n", result.FilteredFiles, result.TotalFiles, result.Path)

	if len(cloneInclude) > 0 || len(cloneExclude) > 0 {
		fmt.Printf("\nPath filtering applied:\n")
		if len(cloneInclude) > 0 {
			fmt.Printf("  Include: %v\n", cloneInclude)
		}
		if len(cloneExclude) > 0 {
			fmt.Printf("  Exclude: %v\n", cloneExclude)
		}
	}

	return nil
}

// firstLine returns the first line of a multi-line string
func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	return s
}
