package nanogit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

// UploadPack sends a POST request to the git-upload-pack endpoint.
// This endpoint is used to fetch objects and refs from the remote repository.
func (c *rawClient) UploadPack(ctx context.Context, data []byte) ([]byte, error) {
	body := bytes.NewReader(data)

	// NOTE: This path is defined in the protocol-v2 spec as required under $GIT_URL/git-upload-pack.
	// See: https://git-scm.com/docs/protocol-v2#_http_transport
	u := c.base.JoinPath("git-upload-pack").String()

	logger := c.getLogger(ctx)

	logger.Info("UploadPack", "url", u, "requestBody", string(data))
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

	logger.Info("UploadPack", "status", res.StatusCode, "statusText", res.Status, "responseBody", string(responseBody), "requestBody", string(data), "url", u)

	return responseBody, nil
}
