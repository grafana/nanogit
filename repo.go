package nanogit

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol/client"
)

// RepoExists checks if the repository exists on the server.
// It attempts to fetch the repository's refs to determine if it exists.
//
// Returns:
//   - true if the repository exists and is accessible
//   - false if the repository does not exist (404)
//   - error if there are any other connection or protocol issues
func (c *httpClient) RepoExists(ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Check repository existence")

	err := c.SmartInfo(ctx, "git-upload-pack")
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			logger.Debug("Repository not found")
			return false, nil
		}
		return false, fmt.Errorf("check repository info: %w", err)
	}

	logger.Debug("Repository exists")
	return true, nil
}

// IsProtocolCompatible checks if the Git server supports protocol v2.
// This method can be called before performing Git operations to ensure
// the server is compatible with nanogit.
//
// Returns:
//   - (true, nil) if the server supports protocol v2 or if version cannot be determined (unknown)
//   - (false, ErrProtocolV1NotSupported) if the server only supports protocol v1
//   - (false, error) if there are connection issues or other problems
//
// Protocol v1-only servers will fail with ErrProtocolV1NotSupported.
// Unknown protocol versions are treated as compatible to allow operations to proceed.
//
// Most modern Git servers support protocol v2 (introduced in Git 2.18, 2018).
func (c *httpClient) IsProtocolCompatible(ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Check protocol compatibility")

	version, err := c.CheckProtocolVersion(ctx, "git-upload-pack")
	if err != nil {
		logger.Debug("Protocol check failed", "error", err)
		return false, err
	}

	logger.Debug("Protocol version detected", "version", version)
	// v2 is compatible, Unknown is treated as compatible (allow operations to proceed)
	// Only v1 is incompatible (would have returned an error above)
	return version != client.ProtocolVersionV1, nil
}
