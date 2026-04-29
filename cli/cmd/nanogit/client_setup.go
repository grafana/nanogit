package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/spf13/cobra"
)

// globalReceivePackCaps and globalNegotiateCaps back the shared write-side
// flags wired through buildClientOptions. They are populated by
// addWriteFlags so any new write subcommand picks them up uniformly without
// re-declaring per-command duplicates.
var (
	globalReceivePackCaps []string
	globalNegotiateCaps   bool
)

// addWriteFlags registers the receive-pack-capability and
// negotiate-capabilities flags on cmd. Write subcommands (put-file today;
// future ones tomorrow) call this in init() so the flags stay in lockstep
// across commands and so buildClientOptions can read them from one place.
func addWriteFlags(cmd *cobra.Command) {
	cmd.Flags().StringSliceVar(&globalReceivePackCaps, "receive-pack-capability", nil,
		"Advertised receive-pack capability (repeatable). When set, replaces the default set entirely. "+
			"Common values include report-status-v2, side-band-64k, quiet, object-format=sha1, and agent=<name>. "+
			"Arbitrary capability tokens are also accepted as an escape hatch for advanced use. "+
			"Example: --receive-pack-capability=report-status-v2 --receive-pack-capability=quiet "+
			"--receive-pack-capability=object-format=sha1 --receive-pack-capability=agent=nanogit "+
			"(drops side-band-64k to work around servers that wrap report-status in side-band channel 1).")
	cmd.Flags().BoolVar(&globalNegotiateCaps, "negotiate-capabilities", false,
		"Fetch the server's advertised receive-pack capabilities once and advertise the intersection "+
			"with the desired set on subsequent ref updates. Composes with --receive-pack-capability: "+
			"the listed (or default) set is filtered against what the server actually offers. Off by default.")
}

// buildClientOptions converts the shared write-side flags into nanogit
// options. When no capabilities are supplied and negotiation is off the
// returned slice is empty and the caller keeps the library defaults. Each
// raw capability is trimmed; empty entries are rejected so "a,,b" does not
// silently inject a blank capability.
func buildClientOptions() ([]options.Option, error) {
	var opts []options.Option
	if len(globalReceivePackCaps) > 0 {
		caps := make([]protocol.Capability, 0, len(globalReceivePackCaps))
		for _, raw := range globalReceivePackCaps {
			v := strings.TrimSpace(raw)
			if v == "" {
				return nil, errors.New("--receive-pack-capability value cannot be empty")
			}
			caps = append(caps, protocol.Capability(v))
		}
		opts = append(opts, options.WithReceivePackCapabilities(caps...))
	}
	if globalNegotiateCaps {
		opts = append(opts, options.WithCapabilityNegotiation())
	}
	return opts, nil
}

// setupClient resolves authentication from flags/env, constructs a nanogit
// Client, and returns a context that carries the CLI logger so library
// internals can emit Info/Debug that the user can see with -v or
// NANOGIT_TRACE=1. Callers may pass extra nanogit options (e.g.
// options.WithReceivePackCapabilities) that are appended after the default
// auth / suffix options.
func setupClient(ctx context.Context, repoURL string, extra ...options.Option) (context.Context, nanogit.Client, error) {
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

	ctx = log.ToContext(ctx, newCLILogger(globalVerbose))

	opts := make([]options.Option, 0, 2+len(extra))
	if token != "" {
		opts = append(opts, options.WithBasicAuth(username, token))
	}
	opts = append(opts, options.WithoutGitSuffix())
	opts = append(opts, extra...)

	client, err := nanogit.NewHTTPClient(repoURL, opts...)
	if err != nil {
		return ctx, nil, fmt.Errorf("failed to create client: %w", err)
	}
	return ctx, client, nil
}
