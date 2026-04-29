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
// When the client negotiates side-band-64k, the server delivers the
// report-status multiplexed on channel 1 (each outer packet starts with
// 0x01). Different servers wrap the channel-1 payload differently:
//
//   - Gitea / GitLab (chunked): the channel-1 payload is itself a
//     pkt-line stream of report-status lines, possibly chunked across
//     multiple outer packets.
//   - GitHub / others (raw):    each report-status line is sent in its
//     own channel-1 outer packet as literal text (no inner length
//     prefix).
//
// We accumulate all channel-1 payload, then dispatch on whether the
// accumulated bytes start with a valid pkt-line length (4 hex digits).
// Crucially, channel-1 unwrap is scoped to this function and does not
// leak into the generic detectError path, where channel 1 also carries
// binary packfile data during fetch/clone.
func parseReceivePackResponse(body io.Reader) error {
	parser := protocol.NewParser(body)
	sawUnpackOk := false
	var sideBandBuf bytes.Buffer

	for {
		line, err := parser.Next()
		if err == nil {
			if len(line) > 0 && line[0] == 0x01 {
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

	if sideBandBuf.Len() > 0 {
		ok, err := scanSideBandReportStatus(sideBandBuf.Bytes())
		if err != nil {
			return err
		}
		if ok {
			sawUnpackOk = true
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

// scanSideBandReportStatus inspects accumulated side-band channel-1
// bytes and reports whether an "unpack ok" success sentinel was seen.
// It auto-detects between two on-wire encodings of the channel-1
// payload (nested pkt-line stream vs raw text lines) and returns any
// protocol-level error (ng/unpack failure/ERR) detected by the
// underlying parser.
func scanSideBandReportStatus(payload []byte) (sawUnpackOk bool, err error) {
	if looksLikePktLine(payload) {
		return scanNestedPktLineStream(payload)
	}
	return scanRawReportStatusLines(payload)
}

// scanNestedPktLineStream parses payload as an inner pkt-line stream
// and returns whether an "unpack ok" line was observed. Any protocol
// error detected by the parser is returned wrapped.
func scanNestedPktLineStream(payload []byte) (sawUnpackOk bool, err error) {
	inner := protocol.NewParser(bytes.NewReader(payload))
	for {
		line, parseErr := inner.Next()
		if parseErr == nil {
			if isUnpackOkLine(line) {
				sawUnpackOk = true
			}
			continue
		}
		if parseErr == io.EOF {
			return sawUnpackOk, nil
		}
		return sawUnpackOk, fmt.Errorf("git protocol error: %w", parseErr)
	}
}

// scanRawReportStatusLines treats payload as one or more LF-separated
// raw report-status lines (the form some servers use when sending each
// line in its own channel-1 packet). Each non-empty line is wrapped in
// a synthetic pkt-line and fed to the parser so the same ng / unpack /
// ERR / error: / fatal: detection that runs on the bare channel-0
// stream applies here.
func scanRawReportStatusLines(payload []byte) (sawUnpackOk bool, err error) {
	for raw := range bytes.SplitSeq(payload, []byte("\n")) {
		line := bytes.TrimRight(raw, " \t\r")
		if len(line) == 0 {
			continue
		}
		pkt, marshalErr := protocol.PackLine(line).Marshal()
		if marshalErr != nil {
			return sawUnpackOk, fmt.Errorf("git protocol error: %w", marshalErr)
		}
		inner := protocol.NewParser(bytes.NewReader(pkt))
		parsed, parseErr := inner.Next()
		if parseErr != nil && parseErr != io.EOF {
			return sawUnpackOk, fmt.Errorf("git protocol error: %w", parseErr)
		}
		if isUnpackOkLine(parsed) {
			sawUnpackOk = true
		}
	}
	return sawUnpackOk, nil
}

// looksLikePktLine reports whether the first 4 bytes of data form a
// valid pkt-line length header (4 ASCII hex digits). Used to
// disambiguate the two channel-1 wire encodings of report-status.
func looksLikePktLine(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	for i := range 4 {
		b := data[i]
		switch {
		case b >= '0' && b <= '9':
		case b >= 'a' && b <= 'f':
		case b >= 'A' && b <= 'F':
		default:
			return false
		}
	}
	return true
}

// isUnpackOkLine reports whether a parsed packet line is the
// "unpack ok" report-status sentinel, tolerating any trailing
// whitespace per gitprotocol-common.
func isUnpackOkLine(line []byte) bool {
	return bytes.Equal(bytes.TrimRight(line, " \t\r\n"), unpackOkLine)
}
