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
// The data parameter is streamed to the server, and the response is parsed internally.
// Returns an error if the HTTP request fails or if Git protocol errors are detected.
func (c *rawClient) ReceivePack(ctx context.Context, data io.Reader) error {
	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/git-receive-pack.
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := c.base.JoinPath("git-receive-pack")
	logger := log.FromContext(ctx)
	logger.Debug("Receive-pack", "url", u.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), data)
	if err != nil {
		return err
	}

	c.addDefaultHeaders(req)
	req.Header.Add("Content-Type", "application/x-git-receive-pack-request")
	req.Header.Add("Accept", "application/x-git-receive-pack-result")

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		res.Body.Close()
		return fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}
	defer res.Body.Close()

	logger.Debug("Receive-pack response",
		"status", res.StatusCode,
		"statusText", res.Status)

	// Parse the response to check for Git protocol errors
	_, _, err = protocol.ParsePack(res.Body)
	if err != nil {
		return fmt.Errorf("git protocol error: %w", err)
	}

	return nil
}
