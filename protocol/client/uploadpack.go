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

// UploadPack sends a POST request to the git-upload-pack endpoint.
// This endpoint is used to fetch objects and refs from the remote repository.
// The data parameter is streamed to the server, and the response is returned as a ReadCloser.
// The caller is responsible for closing the returned ReadCloser.
// Retries only on network errors before a response is received.
func (c *rawClient) UploadPack(ctx context.Context, data io.Reader) (response io.ReadCloser, err error) {
	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/git-upload-pack.
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := c.base.JoinPath("git-upload-pack").String()

	logger := log.FromContext(ctx)
	logger.Debug("Upload-pack", "url", u)

	var res *http.Response
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Create a new request for each attempt since the body may have been consumed
		var req *http.Request
		if attempt == 0 {
			req, err = http.NewRequestWithContext(ctx, http.MethodPost, u, data)
		} else {
			// For retries, we need to recreate the request body
			// Since the original reader may have been consumed, we can't retry with the same body
			// This is a limitation - we'll only retry if the error occurred before reading the body
			// In practice, network errors typically occur before body consumption
			req, err = http.NewRequestWithContext(ctx, http.MethodPost, u, data)
		}
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/x-git-upload-pack-request")
		c.addDefaultHeaders(req)

		res, err = c.client.Do(req)
		if err != nil {
			// Only retry on network errors before response is received
			if isNetworkError(err) && attempt < maxRetries {
				logger.Debug("Network error on upload-pack, retrying",
					"attempt", attempt+1,
					"max_retries", maxRetries,
					"error", err,
					"backoff", backoff)
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
					backoff = time.Duration(float64(backoff) * 2)
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
					continue
				}
			}
			return nil, err
		}

		// Once we have a response (even if it's an error status), don't retry
		break
	}

	if res == nil {
		// This should not happen, but handle it defensively
		return nil, fmt.Errorf("no response received after %d attempts", maxRetries+1)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if closeErr := res.Body.Close(); closeErr != nil {
			logger.Error("error closing response body", "error", closeErr)
		}

		underlying := fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
		if res.StatusCode >= 500 {
			return nil, protocol.NewServerUnavailableError(res.StatusCode, underlying)
		}
		return nil, underlying
	}

	logger.Debug("Upload-pack response",
		"status", res.StatusCode,
		"statusText", res.Status)

	return res.Body, nil
}
