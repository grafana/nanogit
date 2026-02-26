package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/cli/internal/auth"
	"github.com/grafana/nanogit/cli/internal/client"
	"github.com/grafana/nanogit/cli/internal/output"
	"github.com/grafana/nanogit/cli/internal/refparse"
	"github.com/spf13/cobra"
)

var (
	cloneRef          string
	cloneIncludePaths []string
	cloneExcludePaths []string
	cloneBatchSize    int
	cloneConcurrency  int
)

var cloneCmd = &cobra.Command{
	Use:   "clone <url> <destination>",
	Short: "Clone a repository to local filesystem",
	Long: `Clone a repository to local filesystem.

The --ref argument can be:
  - A branch name (e.g., "main") - will try refs/heads/main
  - A tag name (e.g., "v1.0.0") - will try refs/tags/v1.0.0
  - A full reference path (e.g., "refs/heads/main")
  - A commit hash (40 hex characters)

Examples:
  # Basic clone with short branch name
  nanogit clone https://github.com/grafana/nanogit /tmp/repo

  # Clone specific branch
  nanogit clone https://github.com/grafana/nanogit /tmp/repo --ref develop

  # Clone at specific commit
  nanogit clone https://github.com/grafana/nanogit /tmp/repo --ref abc123...

  # Clone with path filtering
  nanogit clone https://github.com/grafana/nanogit /tmp/repo \
    --include-paths "src/**,docs/**" \
    --exclude-paths "**/*.test.go"

  # Performance tuning for large repos
  nanogit clone https://github.com/grafana/nanogit /tmp/repo \
    --batch-size 50 --concurrency 8

  # JSON output
  nanogit clone https://github.com/grafana/nanogit /tmp/repo --json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		destination := args[1]

		// Setup authentication
		authConfig := auth.FromEnvironment()
		authConfig.Merge(token, username, password)

		// Create client
		ctx := context.Background()
		c, err := client.New(ctx, url, authConfig)
		if err != nil {
			return err
		}

		// Show progress for human-readable output
		if getOutputFormat() != "json" {
			fmt.Printf("Cloning %s...\n", url)
		}

		// Resolve ref or commit hash
		commitHash, err := refparse.ResolveRefOrHash(ctx, c, cloneRef)
		if err != nil {
			return fmt.Errorf("resolving ref or commit %s: %w", cloneRef, err)
		}

		if getOutputFormat() != "json" {
			fmt.Printf("✓ Resolved %s -> %s\n", cloneRef, commitHash.String()[:8]+"...")
		}

		// Parse include/exclude paths
		var includePaths []string
		var excludePaths []string

		if len(cloneIncludePaths) > 0 {
			for _, paths := range cloneIncludePaths {
				includePaths = append(includePaths, strings.Split(paths, ",")...)
			}
		}

		if len(cloneExcludePaths) > 0 {
			for _, paths := range cloneExcludePaths {
				excludePaths = append(excludePaths, strings.Split(paths, ",")...)
			}
		}

		// Clone repository
		result, err := c.Clone(ctx, nanogit.CloneOptions{
			Path:         destination,
			Hash:         commitHash,
			IncludePaths: includePaths,
			ExcludePaths: excludePaths,
			BatchSize:    cloneBatchSize,
			Concurrency:  cloneConcurrency,
		})
		if err != nil {
			return fmt.Errorf("cloning repository: %w", err)
		}

		// Output result
		formatter := output.Get(getOutputFormat())
		return formatter.FormatCloneResult(result)
	},
}

func init() {
	cloneCmd.Flags().StringVar(&cloneRef, "ref", "main", "Branch or tag to clone")
	cloneCmd.Flags().StringSliceVar(&cloneIncludePaths, "include-paths", nil, "Glob patterns to include (comma-separated)")
	cloneCmd.Flags().StringSliceVar(&cloneExcludePaths, "exclude-paths", nil, "Glob patterns to exclude (comma-separated)")
	cloneCmd.Flags().IntVar(&cloneBatchSize, "batch-size", 0, "Blob fetch batch size (0=sequential)")
	cloneCmd.Flags().IntVar(&cloneConcurrency, "concurrency", 0, "Parallel fetch workers (0=sequential)")
	rootCmd.AddCommand(cloneCmd)
}
