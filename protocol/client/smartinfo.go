package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/grafana/nanogit/log"
)

// SmartInfo retrieves reference and capability information from the remote Git repository
// using the Smart HTTP protocol.
//
// It sends a GET request to the $GIT_URL/info/refs endpoint with the specified service
// (e.g., "git-upload-pack" or "git-receive-pack") as a query parameter. This is required
// by the Git Smart Protocol v2 specification for repository discovery and capability
// negotiation.
//
// The response contains a list of references and advertised server capabilities in the
// format expected by Git clients.
//
// See:
//   - https://git-scm.com/docs/http-protocol#_smart_clients
//   - https://git-scm.com/docs/protocol-v2#_http_transport
//
// Parameters:
//
//	ctx     - Context for request cancellation and deadlines.
//	service - The Git service to query ("git-upload-pack" or "git-receive-pack").
//
// Returns:
//
//	The raw response body from the server, or an error if the request fails.
//
// Errors:
//
//	Returns an error if the HTTP request fails, the server returns a non-2xx status code,
//	or the response body cannot be read.
func (c *rawClient) SmartInfo(ctx context.Context, service string) ([]byte, error) {
	u := c.base.JoinPath("info/refs")

	query := make(url.Values)
	query.Set("service", service)
	u.RawQuery = query.Encode()

	logger := log.FromContext(ctx)
	logger.Debug("SmartInfo", "url", u.String(), "service", service)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	c.addDefaultHeaders(req)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	logger.Debug("SmartInfo response",
		"status", res.StatusCode,
		"statusText", res.Status,
		"responseSize", len(body))
	logger.Debug("SmartInfo raw response", "body", string(body))

	return body, nil
}
