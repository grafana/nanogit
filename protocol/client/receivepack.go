package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
)

// ReceivePack sends a POST request to the git-receive-pack endpoint.
// This endpoint is used to send objects to the remote repository.
func (c *rawClient) ReceivePack(ctx context.Context, data io.Reader) ([]byte, error) {

	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/git-receive-pack.
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := c.base.JoinPath("git-receive-pack")
	logger := log.FromContext(ctx)
	logger.Debug("Receive-pack", "url", u.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), data)
	if err != nil {
		return nil, err
	}

	c.addDefaultHeaders(req)
	req.Header.Add("Content-Type", "application/x-git-receive-pack-request")
	req.Header.Add("Accept", "application/x-git-receive-pack-result")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	logger.Debug("Receive-pack response",
		"status", res.StatusCode,
		"statusText", res.Status,
		"responseSize", len(responseBody))
	logger.Debug("Receive-pack raw response", "responseBody", string(responseBody))

	// Check for Git protocol errors in receive-pack response
	if protocolErr := checkReceivePackErrors(responseBody); protocolErr != nil {
		logger.Debug("Protocol error detected in receive-pack response", "error", protocolErr.Error())
		return responseBody, protocolErr
	}

	return responseBody, nil
}

// checkReceivePackErrors checks a receive-pack response for Git protocol errors.
// This function specifically looks for error patterns that can appear in receive-pack
// responses, including side-band error messages and reference update failures.
func checkReceivePackErrors(responseBody []byte) error {
	// Look for specific patterns in the response that indicate errors
	responseStr := string(responseBody)
	
	// Check for "error:" messages (which might be in side-band packets)
	if strings.Contains(responseStr, "error: cannot lock ref") {
		// Extract the error message
		lines := strings.Split(responseStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "error: cannot lock ref") {
				// Find the actual error message by skipping packet length and side-band channel
				if idx := strings.Index(line, "error:"); idx >= 0 {
					message := strings.TrimSpace(line[idx+6:]) // Remove "error:" prefix
					return protocol.NewGitServerError([]byte(line), "error", message)
				}
			}
		}
	}
	
	// Check for "ng" (no good) reference update failures
	if strings.Contains(responseStr, "ng refs/") {
		lines := strings.Split(responseStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "ng refs/") {
				// Find the ng message by skipping packet length
				if idx := strings.Index(line, "ng "); idx >= 0 {
					parts := strings.SplitN(line[idx+3:], " ", 2)
					var refName, reason string
					if len(parts) >= 1 {
						refName = strings.TrimSpace(parts[0])
					}
					if len(parts) >= 2 {
						reason = strings.TrimSpace(parts[1])
					} else {
						reason = "update failed"
					}
					return protocol.NewGitReferenceUpdateError([]byte(line), refName, reason)
				}
			}
		}
	}
	
	return nil
}
