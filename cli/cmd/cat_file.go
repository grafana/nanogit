package cmd

import (
	"context"
	"fmt"

	"github.com/grafana/nanogit/cli/internal/auth"
	"github.com/grafana/nanogit/cli/internal/client"
	"github.com/grafana/nanogit/cli/internal/output"
	"github.com/grafana/nanogit/cli/internal/refparse"
	"github.com/spf13/cobra"
)

var (
	catFileShowType bool
	catFileShowSize bool
)

var catFileCmd = &cobra.Command{
	Use:   "cat-file <url> <ref|commit> <path>",
	Short: "Output file contents from a repository",
	Long: `Output file contents from a repository at the specified reference or commit hash.

The ref argument can be:
  - A branch name (e.g., "main") - will try refs/heads/main
  - A tag name (e.g., "v1.0.0") - will try refs/tags/v1.0.0
  - A full reference path (e.g., "refs/heads/main")
  - A commit hash (40 hex characters)

Examples:
  # Output file content with short branch name
  nanogit cat-file https://github.com/grafana/nanogit main README.md

  # Output file content with commit hash
  nanogit cat-file https://github.com/grafana/nanogit abc123... README.md

  # Show file type and size
  nanogit cat-file https://github.com/grafana/nanogit main README.md --show-type --show-size

  # JSON output
  nanogit cat-file https://github.com/grafana/nanogit main README.md --json`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		refName := args[1]
		path := args[2]

		// Setup authentication
		authConfig := auth.FromEnvironment()
		authConfig.Merge(token, username, password)

		// Create client
		ctx := context.Background()
		c, err := client.New(ctx, url, authConfig)
		if err != nil {
			return err
		}

		// Resolve ref or commit hash
		commitHash, err := refparse.ResolveRefOrHash(ctx, c, refName)
		if err != nil {
			return fmt.Errorf("resolving ref or commit %s: %w", refName, err)
		}

		// Get commit to extract tree hash
		commit, err := c.GetCommit(ctx, commitHash)
		if err != nil {
			return fmt.Errorf("getting commit: %w", err)
		}

		// Get blob content
		blob, err := c.GetBlobByPath(ctx, commit.Tree, path)
		if err != nil {
			return fmt.Errorf("getting blob: %w", err)
		}

		// Show type and/or size if requested (human-readable mode only)
		if (catFileShowType || catFileShowSize) && getOutputFormat() != "json" {
			if catFileShowType {
				fmt.Println("blob")
			}
			if catFileShowSize {
				fmt.Println(len(blob.Content))
			}
		}

		// Output content
		formatter := output.Get(getOutputFormat())
		return formatter.FormatBlobContent(path, blob.Hash, blob.Content)
	},
}

func init() {
	catFileCmd.Flags().BoolVar(&catFileShowType, "show-type", false, "Show object type")
	catFileCmd.Flags().BoolVar(&catFileShowSize, "show-size", false, "Show object size")
	rootCmd.AddCommand(catFileCmd)
}
