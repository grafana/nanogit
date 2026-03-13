package nanogit

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/log"
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

// IsServerCompatible checks if the server supports Git protocol v2, which is required by nanogit.
//
// Returns true if the server supports protocol v2, false if it only supports v1.
// Returns an error if the protocol version cannot be determined or if there are connection/authentication issues.
//
// Most modern Git servers support protocol v2 (introduced in Git 2.18, 2018).
// Nanogit requires protocol v2 for full functionality.
func (c *httpClient) IsServerCompatible(ctx context.Context) (bool, error) {
	return c.RawClient.IsServerCompatible(ctx)
}
