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

// ProtocolVersion detects which Git protocol version the server supports.
// This method can be called before performing Git operations to check
// server capabilities.
//
// Returns:
//   - ProtocolVersionV2: Server supports protocol v2 (modern servers)
//   - ProtocolVersionV1: Server only supports protocol v1 (legacy servers)
//   - ProtocolVersionUnknown: Version could not be determined
//   - error: Connection issues or other problems
//
// Most modern Git servers support protocol v2 (introduced in Git 2.18, 2018).
// Nanogit requires protocol v2 for full functionality. Callers should check
// the returned version and decide how to handle v1 or unknown servers.
func (c *httpClient) ProtocolVersion(ctx context.Context) (client.ProtocolVersion, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Detecting protocol version")

	version, err := c.CheckProtocolVersion(ctx, "git-upload-pack")
	if err != nil {
		logger.Debug("Protocol detection failed", "error", err)
		return client.ProtocolVersionUnknown, err
	}

	logger.Debug("Protocol version detected", "version", version)
	return version, nil
}
