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

// ErrMissingReportStatus is returned when a receive-pack response contains
// non-trivial content but no "unpack ok" report-status sentinel. Empty or
// flush-only responses are accepted silently — they correspond to clients
// that legitimately omitted report-status / report-status-v2 from their
// advertised capabilities (via WithReceivePackCapabilities), in which case
// the server is not required to send a report-status reply. The
// requirement only kicks in when the server sent something the parser
// could not interpret as a successful sentinel, which is precisely the
// silent-failure mode the side-band channel-1 wrapping bug exhibited.
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
// git-receive-pack response. When the response contains a recognizable
// report-status (either bare or side-band-wrapped) it must include an
// explicit "unpack ok" sentinel; otherwise ErrMissingReportStatus is
// returned. Empty or flush-only bodies are accepted silently because
// callers may legitimately omit report-status from their negotiated
// capabilities. Any protocol error detected by the parser
// (ERR / ng / unpack failure / error: / fatal:) is surfaced as a wrapped
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
// Channel-1 packet boundaries are preserved: each outer packet's
// payload is stored separately so that two raw packets without
// trailing LF do not concatenate into a single misclassified line.
// Format detection runs on the first non-empty payload; if it begins
// with a pkt-line length header we treat the whole channel-1 stream as
// nested pkt-lines, otherwise we process each packet's payload as
// LF-separated raw lines. Channel-1 unwrap is scoped to this function
// and does not leak into the generic detectError path, where channel 1
// also carries binary packfile data during fetch/clone.
func parseReceivePackResponse(body io.Reader) error {
	parser := protocol.NewParser(body)
	sawUnpackOk := false
	sawAnyContent := false
	var sideBandPackets [][]byte

	for {
		line, err := parser.Next()
		if err == nil {
			sawAnyContent = true
			if len(line) > 0 && line[0] == 0x01 {
				// Parser.Next() already returns a freshly allocated
				// slice (see protocol.readPacketData), so a sub-slice
				// of it is safe to retain across iterations.
				sideBandPackets = append(sideBandPackets, line[1:])
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

	if len(sideBandPackets) > 0 {
		ok, err := scanSideBandReportStatus(sideBandPackets)
		if err != nil {
			return err
		}
		if ok {
			sawUnpackOk = true
		}
	}

	if !sawUnpackOk && sawAnyContent {
		// The server sent something but no "unpack ok" was observed.
		// This is the silent-failure shape: a rejection wrapped in a
		// channel the parser could not interpret would otherwise
		// leave the ref unadvanced with no error to the caller.
		return fmt.Errorf("git protocol error: %w", ErrMissingReportStatus)
	}
	return nil
}

// scanSideBandReportStatus inspects channel-1 packets in arrival order
// and reports whether an "unpack ok" success sentinel was seen. The
// encoding is auto-detected from the first non-empty packet payload: a
// leading pkt-line length header selects the nested-pkt-line path,
// otherwise we fall back to raw text parsing.
//
// Raw mode is hybrid by design. Side-band framing is in principle a
// chunked byte stream, so a status line can be split across packet
// boundaries; conversely, observed-in-the-wild servers send each
// report-status line in its own channel-1 packet without a trailing
// LF, so naively concatenating would merge distinct lines. We therefore
// run two passes:
//
//  1. Per-packet pass — drives error detection (ng / unpack failure /
//     ERR / error: / fatal:) and recognizes "unpack ok" when each line
//     fits in a single packet. Fragment payloads that do not match any
//     prefix do not produce false positives.
//  2. Concatenated-stream fallback — only if the per-packet pass did
//     not see "unpack ok" we re-scan the joined byte stream split on LF
//     so a server that fragments "unpack ok\n" across packets is still
//     recognized. Errors are intentionally not re-evaluated here to
//     avoid spurious matches from cross-packet fragment merging.
func scanSideBandReportStatus(packets [][]byte) (sawUnpackOk bool, err error) {
	var first []byte
	for _, p := range packets {
		if len(p) > 0 {
			first = p
			break
		}
	}
	if first == nil {
		return false, nil
	}
	if looksLikePktLine(first) {
		var buf bytes.Buffer
		for _, p := range packets {
			buf.Write(p)
		}
		return scanNestedPktLineStream(buf.Bytes())
	}

	sawUnpackOk, err = scanRawReportStatusPackets(packets)
	if err != nil {
		return sawUnpackOk, err
	}
	if !sawUnpackOk && containsUnpackOkLine(packets) {
		sawUnpackOk = true
	}
	return sawUnpackOk, nil
}

// containsUnpackOkLine concatenates the channel-1 payloads and reports
// whether the resulting byte stream contains an "unpack ok" line
// (LF-terminated or end-of-stream). Used as a fallback when per-packet
// scanning misses a status line that was split across packet
// boundaries.
func containsUnpackOkLine(packets [][]byte) bool {
	var buf bytes.Buffer
	for _, p := range packets {
		buf.Write(p)
	}
	for raw := range bytes.SplitSeq(buf.Bytes(), []byte("\n")) {
		if bytes.Equal(bytes.TrimRight(raw, " \t\r"), unpackOkLine) {
			return true
		}
	}
	return false
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

// scanRawReportStatusPackets walks each channel-1 packet payload
// independently. Within a single packet, lines are LF-separated; across
// packets, the boundaries provided by the outer pkt-line length act as
// implicit line terminators so two raw packets without trailing LF
// (e.g. "unpack ok" + "ok refs/...") are not merged into a single
// misclassified line.
func scanRawReportStatusPackets(packets [][]byte) (sawUnpackOk bool, err error) {
	for _, payload := range packets {
		ok, e := scanRawReportStatusLines(payload)
		if e != nil {
			return sawUnpackOk, e
		}
		if ok {
			sawUnpackOk = true
		}
	}
	return sawUnpackOk, nil
}

// scanRawReportStatusLines splits a single packet payload on LF and
// feeds each non-empty line through Parser so the same ng / unpack /
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
