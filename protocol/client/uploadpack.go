package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/grafana/nanogit/log"
)

// UploadPack sends a POST request to the git-upload-pack endpoint.
// This endpoint is used to fetch objects and refs from the remote repository.
func (c *rawClient) UploadPack(ctx context.Context, data []byte) ([]byte, error) {
	body := bytes.NewReader(data)

	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/git-upload-pack.
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := c.base.JoinPath("git-upload-pack").String()

	logger := log.FromContext(ctx)
	logger.Debug("Upload-pack", "url", u, "requestSize", len(data))
	logger.Debug("Upload-pack raw request", "requestBody", string(data))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-git-upload-pack-request")
	c.addDefaultHeaders(req)

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

	logger.Debug("Upload-pack response",
		"status", res.StatusCode,
		"statusText", res.Status,
		"responseSize", len(responseBody))
	logger.Debug("Upload-pack raw response", "responseBody", string(responseBody))

	return responseBody, nil
}
