package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
)

// ErrMissingReportStatus is returned when a receive-pack response completes
// without an "unpack ok" report-status line. Historically, nanogit accepted
// such responses as successful pushes, which hid server-side rejections when
// the server silently closed the stream or wrapped the report-status in a
// side-band channel the parser did not recognize. Requiring an explicit
// "unpack ok" ensures push failures surface as errors.
var ErrMissingReportStatus = errors.New("receive-pack response did not contain 'unpack ok' report-status line")

// unpackOkLine is the Git report-status success sentinel (gitprotocol-pack).
var unpackOkLine = []byte("unpack ok")

// ReceivePack sends a POST request to the git-receive-pack endpoint.
// This endpoint is used to send objects to the remote repository.
// The data parameter is streamed to the server, and the response is parsed internally.
// Returns an error if the HTTP request fails or if Git protocol errors are detected.
// Retries on network errors and 429 (Too Many Requests) status codes.
// Note: POST requests do not retry on 5xx errors because the request body is consumed and cannot be re-read.
// However, 429 (Too Many Requests) can be retried even for POST requests.
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

	res, err := c.do(ctx, req)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if closeErr := res.Body.Close(); closeErr != nil {
			logger.Error("error closing response body", "error", closeErr)
		}

		// Check for structured client errors (401, 403, 404)
		if clientErr := CheckHTTPClientError(res); clientErr != nil {
			return clientErr
		}

		// Generic error for other non-2xx codes
		return fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error closing response body: %w", closeErr)
		}
	}()

	logger.Debug("Receive-pack response",
		"status", res.StatusCode,
		"statusText", res.Status)

	return parseReceivePackResponse(res.Body)
}

// parseReceivePackResponse consumes the pkt-line stream from a
// git-receive-pack response and enforces that a successful response
// contains an explicit "unpack ok" report-status line. Any protocol
// error detected by the parser (ERR, ng, unpack failure, error:/fatal:
// messages including side-band-wrapped ones) is surfaced as a wrapped
// git protocol error.
//
// Side-band handling:
//
// When the client negotiates side-band-64k, the server may deliver the
// report-status as a nested pkt-line stream multiplexed on channel 1
// (each packet on the wire starts with a 0x01 byte followed by a
// chunk of the inner stream). We accumulate the channel-1 bytes and
// re-parse them as a secondary pkt-line stream so the positive-
// validation check works for both bare and side-band-wrapped
// report-status responses.
func parseReceivePackResponse(body io.Reader) error {
	parser := protocol.NewParser(body)
	sawUnpackOk := false
	var sideBandBuf bytes.Buffer

	for {
		line, err := parser.Next()
		if err == nil {
			if len(line) > 0 && line[0] == 0x01 {
				// Accumulate channel-1 bytes; they form a nested
				// pkt-line stream that we parse once the outer
				// stream has ended.
				sideBandBuf.Write(line[1:])
				continue
			}
			if isUnpackOkLine(line) {
				sawUnpackOk = true
			}
			continue
		}
		if err == io.EOF {
			break
		}
		return fmt.Errorf("git protocol error: %w", err)
	}

	// Re-parse any side-band-wrapped report-status content so we can
	// detect push failures (ng) or success sentinels that the server
	// delivered on channel 1.
	if sideBandBuf.Len() > 0 {
		innerParser := protocol.NewParser(&sideBandBuf)
		for {
			line, err := innerParser.Next()
			if err == nil {
				if isUnpackOkLine(line) {
					sawUnpackOk = true
				}
				continue
			}
			if err == io.EOF {
				break
			}
			return fmt.Errorf("git protocol error: %w", err)
		}
	}

	if !sawUnpackOk {
		// No report-status was observed. Per gitprotocol-pack, a
		// successful receive-pack response MUST include "unpack ok".
		// Treat the absence as a protocol error rather than silently
		// returning success — otherwise servers that emit a
		// degenerate/empty response, or wrap their report-status in a
		// side-band channel the parser does not recognize, would
		// leave the ref unadvanced with no error reported to the
		// caller.
		return fmt.Errorf("git protocol error: %w", ErrMissingReportStatus)
	}
	return nil
}

// isUnpackOkLine reports whether a parsed packet line is the
// "unpack ok" report-status sentinel, tolerating a leading side-band
// channel-1 byte (0x01) and any trailing whitespace per
// gitprotocol-common.
func isUnpackOkLine(line []byte) bool {
	if len(line) > 0 && line[0] == 0x01 {
		line = line[1:]
	}
	return bytes.Equal(bytes.TrimRight(line, " \t\r\n"), unpackOkLine)
}
