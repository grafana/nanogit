package client

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/retry"
)

// ReceivePack sends a POST request to the git-receive-pack endpoint.
// This endpoint is used to send objects to the remote repository.
// The data parameter is streamed to the server, and the response is parsed internally.
// Returns an error if the HTTP request fails or if Git protocol errors are detected.
// Retries only on network errors before a response is received.
func (c *rawClient) ReceivePack(ctx context.Context, data io.Reader) (err error) {
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

	// For POST requests, we can only retry on network errors, not 5xx responses,
	// because the request body is consumed and cannot be re-read.
	res, err := retry.Do(ctx, func() (*http.Response, error) {
		res, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			_ = res.Body.Close()
			underlying := fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
			if res.StatusCode >= 500 {
				return nil, NewServerUnavailableError(res.StatusCode, underlying)
			}
			return nil, underlying
		}

		return res, nil
	})
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error closing response body: %w", closeErr)
		}
	}()

	logger.Debug("Receive-pack response",
		"status", res.StatusCode,
		"statusText", res.Status)

	parser := protocol.NewParser(res.Body)
	for {
		if _, err := parser.Next(); err != nil {
			if err == io.EOF {
				return nil
			}

			return fmt.Errorf("git protocol error: %w", err)
		}
	}
}
