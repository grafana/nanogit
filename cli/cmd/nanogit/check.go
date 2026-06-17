package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:   "check [<repository>]",
	Short: "Check if a Git server is compatible with nanogit",
	Long: `Check if a Git server supports Git protocol v2, which is required by nanogit.

This command helps you determine if a Git repository URL is compatible with nanogit
before attempting other operations. nanogit requires Git Smart HTTP Protocol v2.

Many modern Git hosting providers support protocol v2, but some older servers or
certain cloud providers may only support protocol v1.

The repository argument is optional when NANOGIT_REPO is set.

Examples:
  # Check repository compatibility
  nanogit check https://example.com/repo.git

  # Use NANOGIT_REPO env var instead of repeating the URL
  export NANOGIT_REPO=https://example.com/repo.git
  nanogit check

  # Check with authentication
  nanogit check https://example.com/private-repo.git --token <token>

  # Output as JSON
  nanogit --json check https://example.com/repo.git`,
	Args: repoArgs(1),
	RunE: runCheck,
}

func runCheck(cmd *cobra.Command, args []string) error {
	repoURL, _ := resolveRepoURL(args, 1)

	ctx, client, err := setupClient(context.Background(), repoURL)
	if err != nil {
		return err
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
		result.Message = "Server only supports Git protocol v1. nanogit requires protocol v2. Please use a Git provider with v2 support or use standard git CLI for this repository."
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
	fmt.Printf("Options:\n")
	fmt.Printf("  • Use a Git hosting provider with protocol v2 support\n")
	fmt.Printf("  • Use standard git CLI for this repository\n\n")
	fmt.Printf("Note: Most modern Git hosting providers support protocol v2.\n")
	fmt.Printf("Older servers or certain cloud providers may only support v1.\n")

	return nil
}
