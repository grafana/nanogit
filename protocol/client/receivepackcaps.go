package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
)

// FetchReceivePackCapabilities issues GET info/refs?service=git-receive-pack and
// returns the capabilities the server advertised after the NUL byte on the
// first ref line. Unlike SmartInfo (which only validates the HTTP transport),
// this method consumes and parses the response body so the caller can use the
// advertised set for capability negotiation on subsequent ReceivePack calls.
//
// 4xx responses are mapped through CheckHTTPClientError so callers see the
// usual ErrRepositoryNotFound / authentication errors. 5xx and network errors
// follow the same retry path as SmartInfo via the shared do() helper.
func (c *rawClient) FetchReceivePackCapabilities(ctx context.Context) (caps []protocol.Capability, err error) {
	u := c.base.JoinPath("info/refs")

	query := make(url.Values)
	query.Set("service", "git-receive-pack")
	u.RawQuery = query.Encode()

	logger := log.FromContext(ctx)
	logger.Debug("FetchReceivePackCapabilities", "url", u.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	c.addDefaultHeaders(req)

	res, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error closing response body: %w", closeErr)
		}
	}()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// Map 401/403/404 to the structured client errors used elsewhere so
		// users see ErrRepositoryNotFound / auth failures consistently.
		if clientErr := CheckHTTPClientError(res); clientErr != nil {
			return nil, clientErr
		}
		return nil, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	logger.Debug("FetchReceivePackCapabilities response",
		"status", res.StatusCode,
		"statusText", res.Status)

	caps, err = protocol.ParseReceivePackInfoRefs(res.Body)
	if err != nil {
		return nil, fmt.Errorf("parse receive-pack info/refs: %w", err)
	}

	logger.Debug("FetchReceivePackCapabilities parsed",
		"capability_count", len(caps))
	return caps, nil
}
