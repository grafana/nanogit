package client

import (
	"context"
	"fmt"
	"io"
	"net/http"

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
	if _, _, err := protocol.ParsePack(responseBody); err != nil {
		logger.Debug("Protocol error detected in receive-pack response", "error", err.Error())
		return responseBody, err
	}

	return responseBody, nil
}
