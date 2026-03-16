package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
	"github.com/spf13/cobra"
)

var (
	lsRemoteHeads bool
	lsRemoteTags  bool
	lsRemoteJSON  bool
	lsRemoteToken string
)

func init() {
	rootCmd.AddCommand(lsRemoteCmd)

	lsRemoteCmd.Flags().BoolVar(&lsRemoteHeads, "heads", false, "Show only branch references (refs/heads/*)")
	lsRemoteCmd.Flags().BoolVar(&lsRemoteTags, "tags", false, "Show only tag references (refs/tags/*)")
	lsRemoteCmd.Flags().BoolVar(&lsRemoteJSON, "json", false, "Output results in JSON format")
	lsRemoteCmd.Flags().StringVar(&lsRemoteToken, "token", "", "Authentication token (can also use NANOGIT_TOKEN env var)")
}

var lsRemoteCmd = &cobra.Command{
	Use:   "ls-remote <repository>",
	Short: "List references in a remote repository",
	Long: `List references (branches and tags) from a remote Git repository.

Examples:
  # List all references
  nanogit ls-remote https://github.com/grafana/nanogit

  # List only branches
  nanogit ls-remote https://github.com/grafana/nanogit --heads

  # List only tags
  nanogit ls-remote https://github.com/grafana/nanogit --tags

  # Output as JSON
  nanogit ls-remote https://github.com/grafana/nanogit --json

  # With authentication
  nanogit ls-remote https://github.com/user/private-repo --token <token>
  NANOGIT_TOKEN=<token> nanogit ls-remote https://github.com/user/private-repo`,
	Args: cobra.ExactArgs(1),
	RunE: runLsRemote,
}

func runLsRemote(cmd *cobra.Command, args []string) error {
	repoURL := args[0]
	ctx := context.Background()

	// Get authentication token from flag or environment
	token := lsRemoteToken
	if token == "" {
		token = os.Getenv("NANOGIT_TOKEN")
	}

	// Create client with optional authentication
	var client nanogit.Client
	var err error
	if token != "" {
		client, err = nanogit.NewHTTPClient(repoURL, options.WithBasicAuth("", token))
	} else {
		client, err = nanogit.NewHTTPClient(repoURL)
	}
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// List all references
	refs, err := client.ListRefs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list references: %w", err)
	}

	// Filter references based on flags
	filteredRefs := filterRefs(refs)

	// Output results
	if lsRemoteJSON {
		return outputJSON(filteredRefs)
	}
	return outputHuman(filteredRefs)
}

func filterRefs(refs []nanogit.Ref) []nanogit.Ref {
	// If no filter flags are set, return all refs
	if !lsRemoteHeads && !lsRemoteTags {
		return refs
	}

	var filtered []nanogit.Ref
	for _, ref := range refs {
		if lsRemoteHeads && strings.HasPrefix(ref.Name, "refs/heads/") {
			filtered = append(filtered, ref)
		}
		if lsRemoteTags && strings.HasPrefix(ref.Name, "refs/tags/") {
			filtered = append(filtered, ref)
		}
	}
	return filtered
}

// refJSON is a JSON-friendly representation of a Git reference
type refJSON struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
}

func outputJSON(refs []nanogit.Ref) error {
	// Convert refs to JSON-friendly format
	jsonRefs := make([]refJSON, len(refs))
	for i, ref := range refs {
		jsonRefs[i] = refJSON{
			Name: ref.Name,
			Hash: ref.Hash.String(),
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonRefs)
}

func outputHuman(refs []nanogit.Ref) error {
	for _, ref := range refs {
		fmt.Printf("%s\t%s\n", ref.Hash, ref.Name)
	}
	return nil
}
