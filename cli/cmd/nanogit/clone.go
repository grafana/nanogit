package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/grafana/nanogit"
	"github.com/spf13/cobra"
)

// cloneArgs validates the positional arguments for `clone`. It accepts 1 or 2
// args when NANOGIT_REPO is unset (the original behaviour) and additionally
// allows 0 args when NANOGIT_REPO supplies the URL.
func cloneArgs(_ *cobra.Command, args []string) error {
	envSet := os.Getenv(repoEnv) != ""
	switch len(args) {
	case 0:
		if !envSet {
			return fmt.Errorf("accepts 1-2 arg(s), received 0 (or set %s)", repoEnv)
		}
		return nil
	case 1, 2:
		return nil
	default:
		return fmt.Errorf("accepts at most 2 arg(s), received %d", len(args))
	}
}

// resolveCloneArgs picks the repo URL and destination from positional args,
// falling back to NANOGIT_REPO. With a single positional arg and env set, the
// URL-vs-path disambiguation uses looksLikeRepoURL.
func resolveCloneArgs(args []string) (repoURL, dest string) {
	env := os.Getenv(repoEnv)
	switch len(args) {
	case 0:
		return env, "."
	case 1:
		if env != "" && !looksLikeRepoURL(args[0]) {
			return env, args[0]
		}
		return args[0], "."
	default:
		return args[0], args[1]
	}
}

var (
	cloneRef         string
	cloneInclude     []string
	cloneExclude     []string
	cloneBatchSize   int
	cloneConcurrency int
)

func init() {
	rootCmd.AddCommand(cloneCmd)

	cloneCmd.Flags().StringVar(&cloneRef, "ref", "", "Git reference to clone (branch, tag, or commit). Defaults to HEAD.")
	cloneCmd.Flags().StringSliceVar(&cloneInclude, "include", nil, "Include paths (glob patterns, e.g., 'src/**', '*.go')")
	cloneCmd.Flags().StringSliceVar(&cloneExclude, "exclude", nil, "Exclude paths (glob patterns, e.g., 'node_modules/**', '*.tmp')")
	cloneCmd.Flags().IntVar(&cloneBatchSize, "batch-size", 50, "Number of blobs to fetch per request (default 50)")
	cloneCmd.Flags().IntVar(&cloneConcurrency, "concurrency", 10, "Number of parallel blob fetches (default 10)")
}

var cloneCmd = &cobra.Command{
	Use:   "clone [<repository>] [<destination>]",
	Short: "Clone a repository to a local directory",
	Long: `Clone a repository to a local directory.

By default, clones the default branch (HEAD). You can specify a different ref
using the --ref flag.

Supports path filtering with glob patterns to clone only specific files or directories.

The repository argument is optional when NANOGIT_REPO is set. When it is set
and a single positional argument is passed, it is treated as the destination
path unless it looks like a URL (contains "://").

Examples:
  # Clone to current directory (uses HEAD/default branch)
  nanogit clone https://github.com/grafana/nanogit.git

  # Use NANOGIT_REPO env var instead of passing the URL
  export NANOGIT_REPO=https://github.com/grafana/nanogit.git
  nanogit clone ./my-repo

  # Clone to specific directory
  nanogit clone https://github.com/grafana/nanogit.git ./my-repo

  # Clone specific branch
  nanogit clone https://github.com/grafana/nanogit.git --ref main ./my-repo

  # Clone specific tag
  nanogit clone https://github.com/grafana/nanogit.git --ref v1.0.0 ./my-repo

  # Clone only specific directories
  nanogit clone https://github.com/grafana/nanogit.git ./my-repo --include 'src/**' --include 'docs/**'

  # Clone with batching and concurrency for better performance
  nanogit clone https://github.com/grafana/nanogit.git ./my-repo --batch-size 100 --concurrency 20`,
	Args: cloneArgs,
	RunE: runClone,
}

func runClone(cmd *cobra.Command, args []string) error {
	repoURL, destPath := resolveCloneArgs(args)

	// Determine ref (default to HEAD)
	ref := cloneRef
	if ref == "" {
		ref = "HEAD"
	}

	ctx, client, err := setupClient(context.Background(), repoURL)
	if err != nil {
		return err
	}

	// Resolve ref to get the commit hash
	commitHash, err := resolveRef(ctx, client, ref)
	if err != nil {
		return fmt.Errorf("failed to resolve ref %q: %w", ref, err)
	}

	if !globalJSON {
		fmt.Fprintf(os.Stderr, "Cloning %s at %s to %s...\n", repoURL, ref, destPath)
	}

	// Prepare clone options
	cloneOpts := nanogit.CloneOptions{
		Path:         destPath,
		Hash:         commitHash,
		IncludePaths: cloneInclude,
		ExcludePaths: cloneExclude,
		BatchSize:    cloneBatchSize,
		Concurrency:  cloneConcurrency,
	}

	// Clone the repository
	result, err := client.Clone(ctx, cloneOpts)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Display results
	if globalJSON {
		return outputCloneJSON(result, cloneBatchSize, cloneConcurrency, cloneInclude, cloneExclude)
	}
	return outputCloneHuman(result, cloneBatchSize, cloneConcurrency, cloneInclude, cloneExclude)
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

type cloneResultJSON struct {
	Commit struct {
		Hash    string `json:"hash"`
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"commit"`
	Path  string `json:"path"`
	Files struct {
		Total    int `json:"total"`
		Filtered int `json:"filtered"`
	} `json:"files"`
	Performance struct {
		BatchSize   int `json:"batch_size"`
		Concurrency int `json:"concurrency"`
	} `json:"performance"`
	PathFiltering *struct {
		Include []string `json:"include,omitempty"`
		Exclude []string `json:"exclude,omitempty"`
	} `json:"path_filtering,omitempty"`
}

func outputCloneJSON(result *nanogit.CloneResult, batchSize, concurrency int, include, exclude []string) error {
	output := cloneResultJSON{}
	output.Commit.Hash = result.Commit.Hash.String()
	output.Commit.Message = result.Commit.Message
	output.Commit.Author.Name = result.Commit.Author.Name
	output.Commit.Author.Email = result.Commit.Author.Email
	output.Path = result.Path
	output.Files.Total = result.TotalFiles
	output.Files.Filtered = result.FilteredFiles
	output.Performance.BatchSize = batchSize
	output.Performance.Concurrency = concurrency

	if len(include) > 0 || len(exclude) > 0 {
		output.PathFiltering = &struct {
			Include []string `json:"include,omitempty"`
			Exclude []string `json:"exclude,omitempty"`
		}{
			Include: include,
			Exclude: exclude,
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputCloneHuman(result *nanogit.CloneResult, batchSize, concurrency int, include, exclude []string) error {
	fmt.Printf("\nClone complete!\n")
	fmt.Printf("  Commit:      %s\n", result.Commit.Hash.String())
	fmt.Printf("  Message:     %s\n", firstLine(result.Commit.Message))
	fmt.Printf("  Author:      %s <%s>\n", result.Commit.Author.Name, result.Commit.Author.Email)
	fmt.Printf("  Files:       %d of %d cloned to %s\n", result.FilteredFiles, result.TotalFiles, result.Path)
	fmt.Printf("  Batch size:  %d\n", batchSize)
	fmt.Printf("  Concurrency: %d\n", concurrency)

	if len(include) > 0 || len(exclude) > 0 {
		fmt.Printf("\nPath filtering applied:\n")
		if len(include) > 0 {
			fmt.Printf("  Include: %v\n", include)
		}
		if len(exclude) > 0 {
			fmt.Printf("  Exclude: %v\n", exclude)
		}
	}

	return nil
}
