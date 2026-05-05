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

// maxRemoteProgressBytes caps the total bytes of side-band channel 2
// (progress) buffered per receive-pack response. The buffer is only
// read on the error path, but is populated speculatively from every
// channel-2 packet — without a cap, a successful push that streams
// verbose progress would retain it all until EOF and then drop it,
// regressing memory usage versus the pre-channel-2-capture code path.
//
// 64 KiB is an order of magnitude larger than realistic hook output
// (typical GitLab push-rule violations and pre-receive hook messages
// are well under 1 KiB) yet small enough that worst-case retention is
// trivial even under high concurrency. When exceeded, further
// channel-2 packets are silently dropped — the surfaced error message
// gets clipped, which is far better than an unbounded allocation.
const maxRemoteProgressBytes = 64 * 1024

// remoteProgressBuffer captures side-band channel 2 payloads up to a
// fixed total byte budget. Once the budget is exhausted further
// payloads are dropped on the floor; the truncation is silent because
// the only consumer (decodeRemoteProgress) is purely diagnostic and
// the alternative — a "(truncated)" sentinel line — adds noise to the
// surfaced error string for negligible debugging value.
type remoteProgressBuffer struct {
	payloads [][]byte
	used     int
}

func (b *remoteProgressBuffer) append(payload []byte) {
	if len(payload) == 0 || b.used >= maxRemoteProgressBytes {
		return
	}
	remain := maxRemoteProgressBytes - b.used
	if len(payload) > remain {
		payload = payload[:remain]
	}
	b.payloads = append(b.payloads, payload)
	b.used += len(payload)
}

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
// report-status multiplexed on channel 1 (each outer packet starts
// with 0x01). Side-band framing is a byte stream, so outer packet
// boundaries do NOT carry semantic meaning — inner pkt-lines or raw
// text lines may be split arbitrarily across outer packets. Channel-1
// payloads are accumulated and reassembled before line-level analysis;
// see scanSideBandReportStatus for the reassembly + format-detection
// strategy. Channel-1 unwrap is scoped to this function and does not
// leak into the generic detectError path, where channel 1 also carries
// binary packfile data during fetch/clone.
func parseReceivePackResponse(body io.Reader) (err error) {
	parser := protocol.NewParser(body)
	sawUnpackOk := false
	// sawReportStatusContent tracks whether the server sent anything
	// that could legitimately carry a report-status sentinel — i.e. a
	// bare channel-0 line or a non-empty channel-1 packet. Channel-2
	// (progress) does NOT arm the unpack-ok requirement, so callers
	// that omit report-status from their advertised capabilities are
	// not misreported as failed pushes when the server still streams
	// progress.
	sawReportStatusContent := false
	var sideBandPackets [][]byte
	// progress accumulates side-band channel 2 payloads up to
	// maxRemoteProgressBytes. Channel 2 carries the human-readable
	// output of pre-receive hooks and push rules (visible to git CLI
	// users as `remote: …` lines), which is the actionable detail
	// behind a bare "pre-receive hook declined" or "push rule
	// violation" reason. We surface it on the error path only —
	// successful pushes drop it after EOF — and the cap keeps a
	// successful, verbose push from holding onto unbounded progress.
	var progress remoteProgressBuffer
	defer func() {
		if err == nil {
			return
		}
		msgs := decodeRemoteProgress(progress.payloads)
		if len(msgs) == 0 {
			return
		}
		err = &protocol.RemoteRejectionError{Err: err, RemoteMessages: msgs}
	}()

	for {
		line, parseErr := parser.Next()
		if parseErr == io.EOF {
			break
		}
		if parseErr != nil {
			return fmt.Errorf("git protocol error: %w", parseErr)
		}
		ok, content, classifyErr := classifyReceivePackLine(line, &sideBandPackets, &progress)
		if classifyErr != nil {
			return classifyErr
		}
		if ok {
			sawUnpackOk = true
		}
		if content {
			sawReportStatusContent = true
		}
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

// classifyReceivePackLine inspects a single parsed pkt-line from the
// receive-pack response and either appends a channel-1 payload to the
// running side-band buffer, captures a channel-2 progress payload (up
// to the buffer's byte budget), returns a fatal channel-3 error, or
// signals whether the line counts as report-status content / contains
// an "unpack ok" sentinel for the bare channel-0 case.
func classifyReceivePackLine(line []byte, sideBandPackets *[][]byte, progress *remoteProgressBuffer) (sawUnpackOk, sawReportStatusContent bool, err error) {
	if len(line) == 0 {
		return false, false, nil
	}
	switch line[0] {
	case 0x02:
		// Channel 2 is progress; it never carries report-status and
		// does not arm any requirement. Capture the payload so
		// parseReceivePackResponse can attach it to any error
		// returned from this response — pre-receive hook stdout and
		// push-rule violation messages are emitted here on GitLab,
		// and a bare "pre-receive hook declined" is rarely actionable
		// without that context. Capture is bounded; see
		// remoteProgressBuffer.
		progress.append(line[1:])
		return false, false, nil
	case 0x03:
		// Channel 3 is fatal. detectError already converts the
		// well-formed "fatal:"/"error:" prefixed shape into a
		// GitServerError; an arbitrary non-empty channel-3 payload
		// that fell through must still be surfaced rather than
		// silently dropped, otherwise a server that closes the stream
		// with a fatal note in a shape the parser does not recognize
		// would look like a successful push.
		payload := line[1:]
		if len(payload) == 0 {
			return false, false, nil
		}
		return false, false, fmt.Errorf("git protocol error: %w",
			protocol.NewGitServerError(line, "fatal", string(payload)))
	case 0x01:
		// Parser.Next() already returns a freshly allocated slice (see
		// protocol.readPacketData), so a sub-slice of it is safe to
		// retain across iterations.
		payload := line[1:]
		*sideBandPackets = append(*sideBandPackets, payload)
		return false, len(payload) > 0, nil
	default:
		// Bare channel-0 report-status line.
		return isUnpackOkLine(line), true, nil
	}
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
//
// Trade-off: in the raw path, distinct report-status lines without an
// LF separator between them merge into a single line. The exact error
// surfaced depends on what that merged line looks like to detectError
// (typically GitUnpackError when it begins with "unpack ", or
// ErrMissingReportStatus when it does not match any sentinel). The
// alternative — preserving outer packet boundaries as line boundaries
// — would handle no-LF-separated cases but would also misclassify
// legitimate mid-token splits (e.g. "unpack o" + "k\n") as failures,
// which is the path real chunked servers use. We optimise for the
// byte-stream interpretation that side-band actually specifies and
// rely on the gitprotocol-common SHOULD that report-status lines are
// LF-terminated.
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

// decodeRemoteProgress concatenates side-band channel 2 payloads,
// splits them on LF or CR (servers commonly use \r for spinner
// updates), trims trailing whitespace, and returns the non-empty
// lines in arrival order. Leading whitespace is preserved so indented
// hook output (bullet lists, sub-items emitted by GitLab push rules,
// etc.) renders correctly when re-surfaced as `remote: …` lines.
// Channel 2 is a byte stream, so an individual progress line may be
// split across outer packets; reassembly before splitting is
// essential.
func decodeRemoteProgress(packets [][]byte) []string {
	if len(packets) == 0 {
		return nil
	}
	var buf bytes.Buffer
	for _, p := range packets {
		buf.Write(p)
	}
	lines := bytes.FieldsFunc(buf.Bytes(), func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	if len(lines) == 0 {
		return nil
	}
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		// TrimRight strips trailing CR/whitespace from the kept
		// content; TrimSpace is used only to decide emptiness so
		// lines like "    - bullet" keep their indentation.
		kept := bytes.TrimRight(l, " \t\r\n")
		if len(bytes.TrimSpace(kept)) == 0 {
			continue
		}
		out = append(out, string(kept))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
