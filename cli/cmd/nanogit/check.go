package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:   "check <repository>",
	Short: "Check if a Git server is compatible with nanogit",
	Long: `Check if a Git server supports Git protocol v2, which is required by nanogit.

This command helps you determine if a Git repository URL is compatible with nanogit
before attempting other operations. nanogit requires Git Smart HTTP Protocol v2.

Supported providers: GitHub, GitLab, Bitbucket, and others with protocol v2 support.
Not supported: Azure DevOps (protocol v1 only), older Git servers.

Examples:
  # Check GitHub repository
  nanogit check https://github.com/grafana/nanogit.git

  # Check with authentication
  nanogit check https://github.com/user/private-repo.git --token <token>

  # Check Azure DevOps (will show as incompatible)
  nanogit check https://dev.azure.com/org/project/_git/repo

  # Output as JSON
  nanogit --json check https://github.com/grafana/nanogit.git`,
	Args: cobra.ExactArgs(1),
	RunE: runCheck,
}

func runCheck(cmd *cobra.Command, args []string) error {
	repoURL := args[0]
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

	// Check server compatibility
	compatible, err := client.IsServerCompatible(ctx)
	if err != nil {
		return fmt.Errorf("failed to check compatibility: %w", err)
	}

	// Output results
	if globalJSON {
		return outputCheckJSON(repoURL, compatible)
	}
	return outputCheckHuman(repoURL, compatible)
}

type checkResultJSON struct {
	Repository string `json:"repository"`
	Compatible bool   `json:"compatible"`
	Protocol   string `json:"protocol"`
	Message    string `json:"message"`
}

func outputCheckJSON(repoURL string, compatible bool) error {
	result := checkResultJSON{
		Repository: repoURL,
		Compatible: compatible,
	}

	if compatible {
		result.Protocol = "v2"
		result.Message = "Server supports Git protocol v2 and is compatible with nanogit"
	} else {
		result.Protocol = "v1"
		result.Message = "Server only supports Git protocol v1. nanogit requires protocol v2. Please use a different Git provider (GitHub, GitLab, Bitbucket) or standard git CLI for this repository."
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func outputCheckHuman(repoURL string, compatible bool) error {
	fmt.Printf("Checking compatibility for: %s\n\n", repoURL)

	if compatible {
		fmt.Printf("✅ Compatible - Server supports Git protocol v2\n\n")
		fmt.Printf("This server is compatible with nanogit. You can use:\n")
		fmt.Printf("  • nanogit ls-remote\n")
		fmt.Printf("  • nanogit ls-tree\n")
		fmt.Printf("  • nanogit cat-file\n")
		fmt.Printf("  • nanogit clone\n")
		return nil
	}

	// Incompatible
	fmt.Printf("❌ Not Compatible - Server only supports Git protocol v1\n\n")
	fmt.Printf("nanogit requires Git Smart HTTP Protocol v2, which this server does not support.\n\n")
	fmt.Printf("Supported providers:\n")
	fmt.Printf("  ✅ GitHub\n")
	fmt.Printf("  ✅ GitLab\n")
	fmt.Printf("  ✅ Bitbucket\n")
	fmt.Printf("  ✅ Gitea (recent versions)\n\n")
	fmt.Printf("Not supported:\n")
	fmt.Printf("  ❌ Azure DevOps (protocol v1 only)\n")
	fmt.Printf("  ❌ Older Git servers without v2 support\n\n")
	fmt.Printf("For this repository, please use standard git CLI instead.\n")

	return nil
}
