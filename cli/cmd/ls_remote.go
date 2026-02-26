package cmd

import (
	"context"
	"strings"

	"github.com/grafana/nanogit/cli/internal/auth"
	"github.com/grafana/nanogit/cli/internal/client"
	"github.com/grafana/nanogit/cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	lsRemoteHeads bool
	lsRemoteTags  bool
)

var lsRemoteCmd = &cobra.Command{
	Use:   "ls-remote <url>",
	Short: "List references in a remote repository",
	Long: `List references (branches and tags) in a remote repository.

Examples:
  # List all references
  nanogit ls-remote https://github.com/grafana/nanogit

  # List only branches
  nanogit ls-remote https://github.com/grafana/nanogit --heads

  # List only tags
  nanogit ls-remote https://github.com/grafana/nanogit --tags

  # JSON output
  nanogit ls-remote https://github.com/grafana/nanogit --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]

		// Setup authentication
		authConfig := auth.FromEnvironment()
		authConfig.Merge(token, username, password)

		// Create client
		ctx := context.Background()
		c, err := client.New(ctx, url, authConfig)
		if err != nil {
			return err
		}

		// List references
		refs, err := c.ListRefs(ctx)
		if err != nil {
			return err
		}

		// Filter based on flags
		if lsRemoteHeads || lsRemoteTags {
			filtered := refs[:0]
			for _, ref := range refs {
				if lsRemoteHeads && strings.HasPrefix(ref.Name, "refs/heads/") {
					filtered = append(filtered, ref)
				} else if lsRemoteTags && strings.HasPrefix(ref.Name, "refs/tags/") {
					filtered = append(filtered, ref)
				}
			}
			refs = filtered
		}

		// Output results
		formatter := output.Get(getOutputFormat())
		return formatter.FormatRefs(refs)
	},
}

func init() {
	lsRemoteCmd.Flags().BoolVar(&lsRemoteHeads, "heads", false, "Show only branches (refs/heads/)")
	lsRemoteCmd.Flags().BoolVar(&lsRemoteTags, "tags", false, "Show only tags (refs/tags/)")
	rootCmd.AddCommand(lsRemoteCmd)
}
