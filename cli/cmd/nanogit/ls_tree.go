package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/spf13/cobra"
)

var (
	lsTreeRecursive bool
	lsTreeLong      bool
	lsTreePath      string
)

func init() {
	rootCmd.AddCommand(lsTreeCmd)

	lsTreeCmd.Flags().BoolVarP(&lsTreeRecursive, "recursive", "r", false, "List tree contents recursively")
	lsTreeCmd.Flags().BoolVarP(&lsTreeLong, "long", "l", false, "Show detailed information (mode, type, hash)")
	lsTreeCmd.Flags().StringVar(&lsTreePath, "path", "", "Path within the tree to list (defaults to root)")
}

var lsTreeCmd = &cobra.Command{
	Use:   "ls-tree <repository> <ref>",
	Short: "List the contents of a tree object",
	Long: `List the contents of a tree object from a Git repository at a specific reference.

The ref can be a branch name, tag name, or commit hash.

Examples:
  # List files at root of main branch
  nanogit ls-tree https://github.com/grafana/nanogit.git main

  # List files in a specific directory
  nanogit ls-tree https://github.com/grafana/nanogit.git main --path docs

  # List all files recursively
  nanogit ls-tree https://github.com/grafana/nanogit.git main --recursive

  # Show detailed information
  nanogit ls-tree https://github.com/grafana/nanogit.git v1.0.0 --long

  # Output as JSON
  nanogit ls-tree https://github.com/grafana/nanogit.git main --json

  # With authentication
  nanogit ls-tree https://github.com/user/private-repo.git main --token <token>
  NANOGIT_TOKEN=<token> nanogit ls-tree https://github.com/user/private-repo.git main`,
	Args: cobra.ExactArgs(2),
	RunE: runLsTree,
}

func runLsTree(cmd *cobra.Command, args []string) error {
	repoURL := args[0]
	ref := args[1]
	ctx := context.Background()

	// Get authentication credentials from flags or environment
	token := globalToken
	if token == "" {
		token = os.Getenv("NANOGIT_TOKEN")
	}

	username := globalUsername
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

	// Get the commit to find the tree hash
	commit, err := client.GetCommit(ctx, commitHash)
	if err != nil {
		return fmt.Errorf("failed to get commit: %w", err)
	}

	// Get tree contents based on flags
	if lsTreeRecursive {
		return listRecursiveTree(ctx, client, commitHash, lsTreePath)
	}
	return listTree(ctx, client, commit.Tree, lsTreePath)
}

func listTree(ctx context.Context, client nanogit.Client, treeHash hash.Hash, path string) error {
	var tree *nanogit.Tree
	var err error

	if path != "" {
		// Get tree at specific path
		tree, err = client.GetTreeByPath(ctx, treeHash, path)
		if err != nil {
			return fmt.Errorf("failed to get tree at path %q: %w", path, err)
		}
	} else {
		// Get root tree
		tree, err = client.GetTree(ctx, treeHash)
		if err != nil {
			return fmt.Errorf("failed to get tree: %w", err)
		}
	}

	if globalJSON {
		return outputTreeJSON(tree.Entries)
	}
	return outputTreeHuman(tree.Entries)
}

func listRecursiveTree(ctx context.Context, client nanogit.Client, commitHash hash.Hash, path string) error {
	flatTree, err := client.GetFlatTree(ctx, commitHash)
	if err != nil {
		return fmt.Errorf("failed to get flat tree: %w", err)
	}

	// Filter by path if specified
	entries := flatTree.Entries
	if path != "" {
		entries = filterEntriesByPath(entries, path)
	}

	if globalJSON {
		return outputFlatTreeJSON(entries)
	}
	return outputFlatTreeHuman(entries)
}

func filterEntriesByPath(entries []nanogit.FlatTreeEntry, prefix string) []nanogit.FlatTreeEntry {
	// Normalize prefix
	prefix = strings.Trim(prefix, "/")
	if prefix != "" {
		prefix = prefix + "/"
	}

	var filtered []nanogit.FlatTreeEntry
	for _, entry := range entries {
		if strings.HasPrefix(entry.Path, prefix) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

type treeEntryJSON struct {
	Name string `json:"name"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	Hash string `json:"hash"`
}

type flatTreeEntryJSON struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	Hash string `json:"hash"`
}

func outputTreeJSON(entries []nanogit.TreeEntry) error {
	jsonEntries := make([]treeEntryJSON, len(entries))
	for i, entry := range entries {
		jsonEntries[i] = treeEntryJSON{
			Name: entry.Name,
			Mode: fmt.Sprintf("%06o", entry.Mode),
			Type: objectTypeToString(entry.Type),
			Hash: entry.Hash.String(),
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonEntries)
}

func outputFlatTreeJSON(entries []nanogit.FlatTreeEntry) error {
	jsonEntries := make([]flatTreeEntryJSON, len(entries))
	for i, entry := range entries {
		jsonEntries[i] = flatTreeEntryJSON{
			Name: entry.Name,
			Path: entry.Path,
			Mode: fmt.Sprintf("%06o", entry.Mode),
			Type: objectTypeToString(entry.Type),
			Hash: entry.Hash.String(),
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonEntries)
}

func outputTreeHuman(entries []nanogit.TreeEntry) error {
	for _, entry := range entries {
		if lsTreeLong {
			fmt.Printf("%06o %s %s\t%s\n",
				entry.Mode,
				objectTypeToString(entry.Type),
				entry.Hash.String(),
				entry.Name)
		} else {
			fmt.Printf("%s\n", entry.Name)
		}
	}
	return nil
}

func outputFlatTreeHuman(entries []nanogit.FlatTreeEntry) error {
	for _, entry := range entries {
		if lsTreeLong {
			fmt.Printf("%06o %s %s\t%s\n",
				entry.Mode,
				objectTypeToString(entry.Type),
				entry.Hash.String(),
				entry.Path)
		} else {
			fmt.Printf("%s\n", entry.Path)
		}
	}
	return nil
}

// resolveRef resolves a ref name to a commit hash.
// It tries multiple strategies:
// 1. Try as full ref name (refs/heads/main, refs/tags/v1.0.0)
// 2. Try as branch name (main -> refs/heads/main)
// 3. Try as tag name (v1.0.0 -> refs/tags/v1.0.0)
// 4. Try as commit hash directly
func objectTypeToString(objType protocol.ObjectType) string {
	switch objType {
	case protocol.ObjectTypeBlob:
		return "blob"
	case protocol.ObjectTypeTree:
		return "tree"
	case protocol.ObjectTypeCommit:
		return "commit"
	case protocol.ObjectTypeTag:
		return "tag"
	default:
		return "unknown"
	}
}
