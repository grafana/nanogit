package client

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/retry"
)

// UploadPack sends a POST request to the git-upload-pack endpoint.
// This endpoint is used to fetch objects and refs from the remote repository.
// The data parameter is streamed to the server, and the response is returned as a ReadCloser.
// The caller is responsible for closing the returned ReadCloser.
func (c *rawClient) UploadPack(ctx context.Context, data io.Reader) (response io.ReadCloser, err error) {

	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/git-upload-pack.
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := c.base.JoinPath("git-upload-pack").String()

	logger := log.FromContext(ctx)
	logger.Debug("Upload-pack", "url", u)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, data)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-git-upload-pack-request")
	c.addDefaultHeaders(req)

	// For POST requests, we can only retry on network errors, not 5xx responses,
	// because the request body is consumed and cannot be re-read.
	// Unwrap httpRetrier if present to get the underlying retrier that only retries on network errors.
	postCtx := ctx
	if retrier := retry.FromContext(ctx); retrier != nil {
		networkOnlyRetrier := UnwrapHTTPRetrier(retrier)
		postCtx = retry.ToContext(ctx, networkOnlyRetrier)
	}
	res, err := retry.Do(postCtx, func() (*http.Response, error) {
		return c.client.Do(req)
		// Only retry network errors - 5xx responses cannot be retried for POST
		// requests because the request body has been consumed.
	})
	if err != nil {
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if closeErr := res.Body.Close(); closeErr != nil {
			logger.Error("error closing response body", "error", closeErr)
		}

		underlying := fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
		if res.StatusCode >= 500 {
			return nil, NewServerUnavailableError(res.StatusCode, underlying)
		}
		return nil, underlying
	}

	logger.Debug("Upload-pack response",
		"status", res.StatusCode,
		"statusText", res.Status)

	return res.Body, nil
}
