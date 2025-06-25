package client

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/grafana/nanogit/log"
)

// ReceivePack sends a POST request to the git-receive-pack endpoint.
// This endpoint is used to send objects to the remote repository.
// The data parameter is streamed to the server, and the response is returned as a ReadCloser.
// The caller is responsible for closing the returned ReadCloser and handling any protocol errors.
func (c *rawClient) ReceivePack(ctx context.Context, data io.Reader) (response io.ReadCloser, err error) {
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

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		res.Body.Close()
		return nil, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	logger.Debug("Receive-pack response",
		"status", res.StatusCode,
		"statusText", res.Status)

	// Note: Protocol error checking will need to be done by the caller
	// since we're now returning a stream instead of buffered data
	return res.Body, nil
}
