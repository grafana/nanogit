package cmd

import (
	"context"
	"fmt"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/cli/internal/auth"
	"github.com/grafana/nanogit/cli/internal/client"
	"github.com/grafana/nanogit/cli/internal/output"
	"github.com/grafana/nanogit/cli/internal/refparse"
	"github.com/spf13/cobra"
)

var (
	lsTreeRecursive bool
	lsTreeLong      bool
)

var lsTreeCmd = &cobra.Command{
	Use:   "ls-tree <url> <ref|commit> [path]",
	Short: "List contents of a tree object",
	Long: `List contents of a tree object at the specified reference or commit hash.

The ref argument can be:
  - A branch name (e.g., "main") - will try refs/heads/main
  - A tag name (e.g., "v1.0.0") - will try refs/tags/v1.0.0
  - A full reference path (e.g., "refs/heads/main")
  - A commit hash (40 hex characters)

Examples:
  # List root directory with short branch name
  nanogit ls-tree https://github.com/grafana/nanogit main

  # List with commit hash
  nanogit ls-tree https://github.com/grafana/nanogit abc123...

  # List specific directory
  nanogit ls-tree https://github.com/grafana/nanogit main src/

  # Recursive listing
  nanogit ls-tree https://github.com/grafana/nanogit main --recursive

  # Show detailed output (mode, type, hash)
  nanogit ls-tree https://github.com/grafana/nanogit main --long

  # JSON output
  nanogit ls-tree https://github.com/grafana/nanogit main --json`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		refName := args[1]
		path := ""
		if len(args) > 2 {
			path = args[2]
		}

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

		// Get tree entries
		var entries []nanogit.FlatTreeEntry
		if lsTreeRecursive {
			// Use GetFlatTree for recursive listing
			flatTree, err := c.GetFlatTree(ctx, commit.Hash)
			if err != nil {
				return fmt.Errorf("getting flat tree: %w", err)
			}

			// Filter by path if specified
			if path != "" {
				filtered := flatTree.Entries[:0]
				for _, entry := range flatTree.Entries {
					if entry.Path == path || (len(entry.Path) > len(path) && entry.Path[:len(path)] == path) {
						filtered = append(filtered, entry)
					}
				}
				entries = filtered
			} else {
				entries = flatTree.Entries
			}
		} else {
			// Non-recursive: get single tree
			var tree *nanogit.Tree
			if path == "" {
				tree, err = c.GetTree(ctx, commit.Tree)
			} else {
				tree, err = c.GetTreeByPath(ctx, commit.Tree, path)
			}
			if err != nil {
				return fmt.Errorf("getting tree: %w", err)
			}

			// Convert Tree entries to FlatTreeEntry format for consistent output
			entries = make([]nanogit.FlatTreeEntry, len(tree.Entries))
			for i, entry := range tree.Entries {
				entries[i] = nanogit.FlatTreeEntry{
					Name: entry.Name,
					Path: entry.Name,
					Mode: entry.Mode,
					Hash: entry.Hash,
					Type: entry.Type,
				}
			}
		}

		// Format output based on --long flag
		formatter := output.Get(getOutputFormat())
		if !lsTreeLong && getOutputFormat() != "json" {
			// Simple format: just names
			for _, entry := range entries {
				fmt.Println(entry.Path)
			}
			return nil
		}

		// Detailed format or JSON
		return formatter.FormatTreeEntries(entries)
	},
}

func init() {
	lsTreeCmd.Flags().BoolVarP(&lsTreeRecursive, "recursive", "r", false, "Recursively list all files")
	lsTreeCmd.Flags().BoolVarP(&lsTreeLong, "long", "l", false, "Show detailed info (mode, type, hash)")
	rootCmd.AddCommand(lsTreeCmd)
}
