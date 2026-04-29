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

// ErrMissingReportStatus is returned when a receive-pack response
// contains report-status content (a bare channel-0 line or a non-empty
// channel-1 packet) but no "unpack ok" sentinel. Channel-2 progress
// packets and an empty/flush-only body do NOT arm the requirement —
// callers that legitimately omit report-status / report-status-v2 from
// their advertised capabilities (via WithReceivePackCapabilities) can
// still see progress messages from a server, and the server is not
// required to reply with report-status in that configuration. The
// requirement kicks in precisely when the server sent something the
// parser could not interpret as a successful sentinel, which is the
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
	// sawReportStatusContent tracks whether the server sent anything
	// that could legitimately carry a report-status sentinel — i.e. a
	// bare channel-0 line or a non-empty channel-1 packet. Channel-2
	// (progress) and channel-3 (fatal — already converted to an error
	// by detectError) do NOT arm the unpack-ok requirement, so callers
	// that omit report-status from their advertised capabilities are
	// not misreported as failed pushes when the server still streams
	// progress.
	sawReportStatusContent := false
	var sideBandPackets [][]byte

	for {
		line, err := parser.Next()
		if err == nil {
			if len(line) == 0 {
				continue
			}
			switch line[0] {
			case 0x02, 0x03:
				// Progress / fatal channels carry no report-status.
				continue
			case 0x01:
				// Parser.Next() already returns a freshly allocated
				// slice (see protocol.readPacketData), so a sub-slice
				// of it is safe to retain across iterations.
				payload := line[1:]
				sideBandPackets = append(sideBandPackets, payload)
				if len(payload) > 0 {
					sawReportStatusContent = true
				}
			default:
				// Bare channel-0 report-status line.
				sawReportStatusContent = true
				if isUnpackOkLine(line) {
					sawUnpackOk = true
				}
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

	if !sawUnpackOk && sawReportStatusContent {
		// The server sent report-status-shaped content but no
		// "unpack ok" was observed. This is the silent-failure shape:
		// a rejection wrapped in a channel the parser could not
		// interpret would otherwise leave the ref unadvanced with no
		// error to the caller.
		return fmt.Errorf("git protocol error: %w", ErrMissingReportStatus)
	}
	return nil
}

// scanSideBandReportStatus inspects channel-1 packets in arrival order
// and reports whether an "unpack ok" success sentinel was seen. The
// encoding is auto-detected by peeking the first 4 bytes of the
// channel-1 byte stream — accumulating across packets when the inner
// pkt-line length header is fragmented — so servers that split the
// header across outer side-band packets are still classified as
// nested. Otherwise we fall back to raw text parsing.
//
// Both the nested and the raw paths concatenate the channel-1 byte
// stream before running line-level analysis. Side-band framing is a
// byte stream, so packet boundaries do NOT carry semantic meaning —
// inner pkt-lines or raw text lines may be split arbitrarily across
// outer packets, including in the middle of "unpack ok" or "ng <ref>".
// Per gitprotocol-common, non-binary report-status lines SHOULD be
// terminated by an LF; a non-conformant server that omits LF
// separators will produce a single merged line, which surfaces as
// ErrMissingReportStatus rather than a silent misclassification.
func scanSideBandReportStatus(packets [][]byte) (sawUnpackOk bool, err error) {
	// Format detection: peek the first 4 bytes of the channel-1 byte
	// stream, accumulating across packets so a fragmented inner
	// pkt-line length header is still classified as nested.
	var prefix []byte
	for _, p := range packets {
		if len(prefix) >= 4 {
			break
		}
		need := 4 - len(prefix)
		if len(p) < need {
			prefix = append(prefix, p...)
		} else {
			prefix = append(prefix, p[:need]...)
		}
	}
	if len(prefix) == 0 {
		return false, nil
	}

	var buf bytes.Buffer
	for _, p := range packets {
		buf.Write(p)
	}

	if looksLikePktLine(prefix) {
		return scanNestedPktLineStream(buf.Bytes())
	}
	return scanRawReportStatusStream(buf.Bytes())
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

// scanRawReportStatusStream treats the concatenated channel-1 byte
// stream as one or more LF-separated raw report-status lines. Each
// non-empty line is wrapped in a synthetic pkt-line and fed to Parser
// so the same ng / unpack failure / ERR / error: / fatal: detection
// that runs on the bare channel-0 stream applies here.
//
// Reassembling across packet boundaries before classification matters:
// side-band is a byte-stream framing layer, so a status line can be
// split mid-token (e.g. "unpack o" then "k\n"). Running detectError on
// the per-packet fragment "unpack o" would match the "unpack " prefix
// and emit a spurious GitUnpackError on a successful push, so analysis
// only fires on complete lines.
func scanRawReportStatusStream(payload []byte) (sawUnpackOk bool, err error) {
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
