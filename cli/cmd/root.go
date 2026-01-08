package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	token    string
	username string
	password string
	jsonOut  bool
	debug    bool
)

var rootCmd = &cobra.Command{
	Use:   "nanogit",
	Short: "A lightweight Git client for cloud-native environments",
	Long: `nanogit is a lightweight, HTTPS-only Git implementation designed for
cloud-native environments. It provides essential Git operations optimized
for server-side usage.

Authentication can be provided via flags or environment variables:
  - NANOGIT_TOKEN: General token for any provider
  - GITHUB_TOKEN:  GitHub-specific token
  - GITLAB_TOKEN:  GitLab-specific token
  - NANOGIT_USERNAME + NANOGIT_PASSWORD: Basic auth`,
	SilenceUsage: true,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "Authentication token")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "Username for basic auth")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "Password for basic auth")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")

	// Set up persistent pre-run to configure logging
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if debug {
			os.Setenv("NANOGIT_LOG_LEVEL", "debug")
		}
		return nil
	}
}

// getOutputFormat returns "json" if json flag is set, otherwise "human"
func getOutputFormat() string {
	if jsonOut {
		return "json"
	}
	return "human"
}

// exitWithError prints an error and exits with code 1
func exitWithError(err error) {
	if jsonOut {
		fmt.Fprintf(os.Stderr, `{"error": "%s"}`+"\n", err.Error())
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	os.Exit(1)
}
