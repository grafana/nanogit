package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version can be set during build via: go build -ldflags "-X main.Version=v1.0.0"
	Version = "dev"
)

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
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			fmt.Fprintf(os.Stderr, "Error displaying help: %v\n", err)
		}
	},
}
