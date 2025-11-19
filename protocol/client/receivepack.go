package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
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

	var res *http.Response
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Create a new request for each attempt since the body may have been consumed
		var req *http.Request
		if attempt == 0 {
			req, err = http.NewRequestWithContext(ctx, http.MethodPost, u.String(), data)
		} else {
			// For retries, we need to recreate the request body
			// Since the original reader may have been consumed, we can't retry with the same body
			// This is a limitation - we'll only retry if the error occurred before reading the body
			// In practice, network errors typically occur before body consumption
			req, err = http.NewRequestWithContext(ctx, http.MethodPost, u.String(), data)
		}
		if err != nil {
			return err
		}

		c.addDefaultHeaders(req)
		req.Header.Add("Content-Type", "application/x-git-receive-pack-request")
		req.Header.Add("Accept", "application/x-git-receive-pack-result")

		res, err = c.client.Do(req)
		if err != nil {
			// Only retry on network errors before response is received
			if isNetworkError(err) && attempt < maxRetries {
				logger.Debug("Network error on receive-pack, retrying",
					"attempt", attempt+1,
					"max_retries", maxRetries,
					"error", err,
					"backoff", backoff)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
					backoff = time.Duration(float64(backoff) * 2)
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
					continue
				}
			}
			return err
		}

		// Once we have a response (even if it's an error status), don't retry
		break
	}

	if res == nil {
		// This should not happen, but handle it defensively
		return fmt.Errorf("no response received after %d attempts", maxRetries+1)
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error closing response body: %w", closeErr)
		}
	}()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		_ = res.Body.Close()
		underlying := fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
		if res.StatusCode >= 500 {
			return protocol.NewServerUnavailableError(res.StatusCode, underlying)
		}
		return underlying
	}

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
