package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version can be set during build via: go build -ldflags "-X main.Version=v1.0.0"
	Version = "dev"
	// Commit is the git commit hash (set by GoReleaser)
	Commit = "unknown"
	// Date is the build date (set by GoReleaser)
	Date = "unknown"

	// Global flags available to all commands
	globalUsername string
	globalToken    string
	globalJSON     bool
	globalVerbose  bool

	// DoS-protection byte caps. 0 (the default) means "no limit", which
	// matches the library's default. Each flag corresponds to one field
	// of options.Limits.
	globalMaxBytesSingleObject int64
	globalMaxBytesMultiObject  int64
	globalMaxBytesRefs         int64
	globalMaxBytesReceivePack  int64
)

func init() {
	// Add persistent flags that are available to all subcommands
	rootCmd.PersistentFlags().StringVar(&globalUsername, "username", "", "Authentication username (can also use NANOGIT_USERNAME env var, defaults to 'git')")
	rootCmd.PersistentFlags().StringVar(&globalToken, "token", "", "Authentication token (can also use NANOGIT_TOKEN env var)")
	rootCmd.PersistentFlags().BoolVar(&globalJSON, "json", false, "Output results in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&globalVerbose, "verbose", "v", false, "Be verbose (emit Info-level logs to stderr; set NANOGIT_TRACE=1 for Debug/wire detail)")

	rootCmd.PersistentFlags().Int64Var(&globalMaxBytesSingleObject, "max-bytes-single-object", 0, "Cap (bytes) on responses to single-object fetches: GetBlob, GetTree, GetCommit. 0 = no limit.")
	rootCmd.PersistentFlags().Int64Var(&globalMaxBytesMultiObject, "max-bytes-multi-object", 0, "Cap (bytes) on responses to multi-object fetches: GetFlatTree, ListCommits, CompareCommits, Clone. 0 = no limit.")
	rootCmd.PersistentFlags().Int64Var(&globalMaxBytesRefs, "max-bytes-refs", 0, "Cap (bytes) on ref-listing and protocol-detection responses. 0 = no limit (1 MB floor still applies to the protocol-detection path).")
	rootCmd.PersistentFlags().Int64Var(&globalMaxBytesReceivePack, "max-bytes-receive-pack", 0, "Cap (bytes) on the server's reply to a receive-pack push. 0 = no limit.")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "nanogit",
	Short: "A lightweight Git client for cloud-native environments",
	Long: `nanogit is a lightweight, HTTPS-only Git implementation written in Go,
designed for cloud-native environments. It provides essential Git operations
optimized for server-side usage with pluggable storage backends.

For more information, visit: https://github.com/grafana/nanogit`,
	Version: buildVersion(),
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			fmt.Fprintf(os.Stderr, "Error displaying help: %v\n", err)
		}
	},
}

func buildVersion() string {
	if Commit != "unknown" && Date != "unknown" {
		return fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date)
	}
	return Version
}
