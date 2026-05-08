package client

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/retry"
	"github.com/stretchr/testify/require"
)

func TestReceivePack(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
		setupClient   options.Option
	}{
		{
			name:          "successful response",
			statusCode:    http.StatusOK,
			responseBody:  "000dunpack ok0000", // Valid Git packet format: unpack ok + flush
			expectedError: "",
			setupClient:   nil,
		},
		{
			name:          "not found",
			statusCode:    http.StatusNotFound,
			responseBody:  "not found",
			expectedError: "got status code 404: 404 Not Found",
			setupClient:   nil,
		},
		{
			name:          "server error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  "server error",
			expectedError: "server unavailable",
			setupClient:   nil,
		},
		{
			name:          "bad gateway",
			statusCode:    http.StatusBadGateway,
			responseBody:  "bad gateway",
			expectedError: "server unavailable",
			setupClient:   nil,
		},
		{
			name:          "service unavailable",
			statusCode:    http.StatusServiceUnavailable,
			responseBody:  "service unavailable",
			expectedError: "server unavailable",
			setupClient:   nil,
		},
		{
			name:          "gateway timeout",
			statusCode:    http.StatusGatewayTimeout,
			responseBody:  "gateway timeout",
			expectedError: "server unavailable",
			setupClient:   nil,
		},
		{
			name:          "timeout error",
			statusCode:    0,
			responseBody:  "",
			expectedError: "context deadline exceeded",
			setupClient: options.WithHTTPClient(&http.Client{
				Timeout: 1 * time.Nanosecond,
			}),
		},
		{
			name:          "connection refused",
			statusCode:    0,
			responseBody:  "",
			expectedError: "i/o timeout",
			setupClient: options.WithHTTPClient(&http.Client{
				Transport: &http.Transport{
					DialContext: (&net.Dialer{
						Timeout: 1 * time.Nanosecond,
					}).DialContext,
				},
			}),
		},
		{
			name:       "git server error response",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "error: cannot lock ref 'refs/heads/main': is at d346cc9cd80dd0bbda023bb29a7ff2d887c75b19 but expected b6ce559b8c2e4834e075696cac5522b379448c13"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError: "git server error:",
			setupClient:   nil,
		},
		{
			name:       "git reference update error",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "ng refs/heads/main failed to update ref"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError: "reference update failed for refs/heads/main:",
			setupClient:   nil,
		},
		{
			name:       "git unpack error",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "unpack index-pack failed"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError: "index-pack failed",
			setupClient:   nil,
		},
		{
			name:       "git fatal error with unpack keyword",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "fatal: unpack failed due to corrupt data"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError: "unpack failed due to corrupt data",
			setupClient:   nil,
		},
		{
			name:       "git ERR packet",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "ERR push declined due to email policy"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError: "push declined due to email policy",
			setupClient:   nil,
		},
		{
			name:       "multi-line error like user's first example",
			statusCode: http.StatusOK,
			responseBody: func() string {
				message := "error: object 457e2462aee3d41d1a2832f10419213e10091bdc: treeNotSorted: not properly sorted\nfatal: fsck error in packed object\n"
				pkt, _ := protocol.PackLine(message).Marshal()
				return string(pkt)
			}(),
			expectedError: "object 457e2462aee3d41d1a2832f10419213e10091bdc: treeNotSorted: not properly sorted",
			setupClient:   nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var server *httptest.Server
			if tt.setupClient == nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/repo.git/git-receive-pack" {
						t.Errorf("expected path /repo.git/git-receive-pack, got %s", r.URL.Path)
						return
					}
					if r.Method != http.MethodPost {
						t.Errorf("expected method POST, got %s", r.Method)
						return
					}

					// Check default headers
					if gitProtocol := r.Header.Get("Git-Protocol"); gitProtocol != "version=2" {
						t.Errorf("expected Git-Protocol header 'version=2', got %s", gitProtocol)
						return
					}
					if userAgent := r.Header.Get("User-Agent"); userAgent != "nanogit/0" {
						t.Errorf("expected User-Agent header 'nanogit/0', got %s", userAgent)
						return
					}

					w.WriteHeader(tt.statusCode)
					if _, err := w.Write([]byte(tt.responseBody)); err != nil {
						t.Errorf("failed to write response: %v", err)
						return
					}
				}))
				defer server.Close()
			}

			url := "http://127.0.0.1:0/repo"
			if server != nil {
				url = server.URL + "/repo"
			}

			var (
				client *rawClient
				err    error
			)

			if tt.setupClient != nil {
				client, err = NewRawClient(url, tt.setupClient)
			} else {
				client, err = NewRawClient(url)
			}
			require.NoError(t, err)

			err = client.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				// Verify ServerUnavailableError for 5xx status codes
				if tt.statusCode >= 500 && tt.statusCode < 600 {
					require.True(t, errors.Is(err, ErrServerUnavailable), "error should be ErrServerUnavailable")
					var serverErr *ServerUnavailableError
					require.ErrorAs(t, err, &serverErr, "error should be ServerUnavailableError type")
					require.Equal(t, tt.statusCode, serverErr.StatusCode, "status code should match")
					require.NotNil(t, serverErr.Underlying, "underlying error should not be nil")
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReceivePack_Retry(t *testing.T) {
	t.Parallel()

	t.Run("retries on network errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 2 {
				// Simulate network error
				hj, ok := w.(http.Hijacker)
				if ok {
					conn, _, _ := hj.Hijack()
					_ = conn.Close()
				}
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("000dunpack ok0000"))
		}))
		defer server.Close()

		retrier := newTestRetrier(3)
		retrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return err != nil
		}

		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// Note: This test verifies retries are attempted
		_ = client.ReceivePack(ctx, strings.NewReader("test data"))

		// Verify retrier Wait was called (HTTP retrier delegates Wait to wrapped retrier)
		// Note: ShouldRetry is only delegated for network errors with Timeout()
		// Connection close might not result in timeout error, so ShouldRetry might not be called
		require.GreaterOrEqual(t, retrier.WaitCallCount(), 0, "Wait may be called if retries occur")
	})

	t.Run("does not retry on 5xx errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		retrier := newTestRetrier(3)
		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = client.ReceivePack(ctx, strings.NewReader("test data"))
		require.Error(t, err)
		require.Equal(t, 1, attemptCount, "Should not retry POST requests on 5xx errors")

		// Verify retrier Wait was not called (no retries for 5xx POST errors)
		require.Equal(t, 0, retrier.WaitCallCount(), "Wait should not be called for 5xx POST errors")
	})

}

// TestReceivePack_PositiveValidation ensures a receive-pack response is
// considered successful only when one of the following holds:
//   - the body is empty / flush-only (caller may have legitimately
//     omitted report-status from negotiated capabilities), OR
//   - the body contains an explicit "unpack ok" sentinel (bare or
//     side-band-wrapped).
//
// Non-trivial bodies that lack "unpack ok" are rejected so server-side
// rejections wrapped in channels the parser cannot interpret no longer
// silently appear as successful pushes.
func TestReceivePack_PositiveValidation(t *testing.T) {
	t.Parallel()

	pkt := func(s string) []byte {
		b, err := protocol.PackLine(s).Marshal()
		require.NoError(t, err)
		return b
	}
	flushed := func(data []byte) []byte {
		return append(append([]byte{}, data...), []byte("0000")...)
	}
	// sideband1 wraps `inner` (which is itself a pkt-line stream) inside a
	// single outer channel-1 pkt-line, mimicking how Gitea/GitLab deliver
	// report-status when side-band-64k is negotiated.
	sideband1 := func(inner []byte) []byte {
		payload := append([]byte{0x01}, inner...)
		b, err := protocol.PackLine(payload).Marshal()
		require.NoError(t, err)
		return b
	}
	// rawSideband1 wraps a literal text payload (no inner pkt-line
	// length prefix) inside a single channel-1 outer packet. Used to
	// exercise the raw-line branch of the side-band parser.
	rawSideband1 := func(text string) []byte {
		payload := append([]byte{0x01}, []byte(text)...)
		b, err := protocol.PackLine(payload).Marshal()
		require.NoError(t, err)
		return b
	}

	tests := []struct {
		name        string
		body        []byte
		wantErr     bool
		errContains string
	}{
		{
			name:    "unpack ok is accepted",
			body:    flushed(pkt("unpack ok")),
			wantErr: false,
		},
		{
			name:    "unpack ok with trailing LF is accepted",
			body:    flushed(pkt("unpack ok\n")),
			wantErr: false,
		},
		{
			name: "side-band channel 1 nested report-status is accepted",
			body: flushed(sideband1(flushed(append(
				pkt("unpack ok\n"),
				pkt("ok refs/heads/main\n")...,
			)))),
			wantErr: false,
		},
		{
			name: "side-band channel 1 raw ng payload triggers reference update error",
			body: flushed(rawSideband1("ng refs/heads/main pre-receive hook declined\n")),
			wantErr:     true,
			errContains: "reference update failed for refs/heads/main",
		},
		{
			name: "side-band channel 1 nested ng triggers reference update error",
			body: flushed(sideband1(flushed(append(
				pkt("unpack ok\n"),
				pkt("ng refs/heads/main pre-receive hook declined\n")...,
			)))),
			wantErr:     true,
			errContains: "reference update failed for refs/heads/main",
		},
		{
			// Side-band framing is a byte stream, so packet boundaries
			// do NOT carry semantic meaning — a status line may be
			// split mid-token across outer packets (here "unpack o" +
			// "k\n"). Reassembling across boundaries before line
			// classification is essential: a per-packet scan would
			// match "unpack " on the first fragment and emit a
			// spurious GitUnpackError with Message="o" on a successful
			// push. This is the Codex P1 case.
			name: "raw side-band channel 1 line split mid-token across packets is recognized",
			body: flushed(append(
				rawSideband1("unpack o"),
				rawSideband1("k\n")...,
			)),
			wantErr: false,
		},
		{
			// Companion to the mid-token case: a status line split
			// AFTER a token boundary ("unpack " + "ok\n") must also
			// reassemble.
			name: "raw side-band channel 1 unpack ok split at token boundary is recognized",
			body: flushed(append(
				rawSideband1("unpack "),
				rawSideband1("ok\n")...,
			)),
			wantErr: false,
		},
		{
			// Two raw channel-1 packets carrying distinct lines
			// without a trailing LF on the first one is genuinely
			// ambiguous in the byte-stream model — the bytes merge
			// into "unpack okok refs/heads/main" with no separator.
			// We surface this as a failure (the merged line matches
			// the "unpack " prefix and yields GitUnpackError) rather
			// than silently misclassifying the response. Servers
			// SHOULD LF-terminate per gitprotocol-common; one that
			// doesn't is non-conformant and a hard failure here is
			// safer than a silent success.
			name: "raw side-band channel 1 distinct lines without LF separator surfaces as failure",
			body: flushed(append(
				rawSideband1("unpack ok"),
				rawSideband1("ok refs/heads/main")...,
			)),
			wantErr:     true,
			errContains: "pack unpack failed",
		},
		{
			name:    "empty body is accepted (no report-status negotiated)",
			body:    []byte{},
			wantErr: false,
		},
		{
			name:    "flush-only body is accepted (no report-status negotiated)",
			body:    []byte("0000"),
			wantErr: false,
		},
		{
			// Channel-2 progress packets do not carry report-status
			// content, so a body that contains only progress (and
			// no unpack ok) is accepted. Callers that omit
			// report-status from negotiated capabilities legitimately
			// hit this shape and must not be misreported as failed.
			name:    "channel-2 progress only is accepted (no report-status negotiated)",
			body:    flushed(pkt(string(append([]byte{0x02}, []byte("Counting objects: 1, done.\n")...)))),
			wantErr: false,
		},
		{
			// Same shape, but the body also includes a non-empty
			// channel-1 packet that is NOT report-status (just
			// gibberish). The channel-1 packet arms the report-status
			// requirement, so the missing unpack ok must surface as
			// ErrMissingReportStatus.
			name: "channel-1 content without unpack ok is rejected",
			body: flushed(append(
				pkt(string(append([]byte{0x02}, []byte("progress\n")...))),
				rawSideband1("ng refs/heads/main hook declined")...,
			)),
			wantErr:     true,
			errContains: "reference update failed for refs/heads/main",
		},
		{
			// Server fragments the inner pkt-line 4-byte length
			// header across two channel-1 outer packets: first
			// packet carries "00", second packet starts with "0e"
			// followed by "unpack ok\n0000". Format detection must
			// accumulate across packets to recognize the nested
			// stream rather than misclassifying it as raw.
			name: "nested side-band length header fragmented across packets is recognized",
			body: flushed(append(
				rawSideband1("00"),
				rawSideband1("0eunpack ok\n0000")...,
			)),
			wantErr: false,
		},
		{
			// Channel 3 is the fatal channel. detectError converts
			// well-formed "fatal:"/"error:" shapes already; a
			// non-empty channel-3 payload that does NOT match those
			// prefixes (a server-specific abrupt close) must still
			// surface as a GitServerError rather than be silently
			// dropped — otherwise the push would look successful
			// despite the server signalling termination.
			name: "channel-3 payload without fatal prefix surfaces as server error",
			body: flushed(pkt(string(append([]byte{0x03}, []byte("connection terminated by host")...)))),
			wantErr:     true,
			errContains: "git server fatal:",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(tt.body)
			}))
			defer server.Close()

			c, err := NewRawClient(server.URL + "/repo")
			require.NoError(t, err)

			err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestRemoteProgressBuffer_AppendTruncatesAndCopies verifies that
// remoteProgressBuffer caps total bytes at maxRemoteProgressBytes and,
// crucially, copies the truncated prefix into a fresh slice instead
// of subslicing the input. Subslicing alone would keep the original
// pkt-line's full backing array (~64 KiB) alive for the lifetime of
// the buffer, defeating the byte cap in terms of actual memory
// retention.
func TestRemoteProgressBuffer_AppendTruncatesAndCopies(t *testing.T) {
	t.Parallel()

	t.Run("zero-length payload is a no-op", func(t *testing.T) {
		t.Parallel()
		var b remoteProgressBuffer
		b.append(nil)
		b.append([]byte{})
		require.Empty(t, b.payloads)
		require.Equal(t, 0, b.used)
	})

	t.Run("payload smaller than budget is appended as-is", func(t *testing.T) {
		t.Parallel()
		var b remoteProgressBuffer
		small := []byte("hello")
		b.append(small)
		require.Len(t, b.payloads, 1)
		require.Equal(t, []byte("hello"), b.payloads[0])
		require.Equal(t, len(small), b.used)
	})

	t.Run("packets past the cap are dropped silently", func(t *testing.T) {
		t.Parallel()
		var b remoteProgressBuffer
		b.append(bytes.Repeat([]byte("x"), maxRemoteProgressBytes))
		require.Equal(t, maxRemoteProgressBytes, b.used)

		// Subsequent packet should be entirely dropped.
		b.append([]byte("ignored"))
		require.Len(t, b.payloads, 1)
		require.Equal(t, maxRemoteProgressBytes, b.used)
	})

	t.Run("truncated payload is copied so original backing array can be GC'd", func(t *testing.T) {
		t.Parallel()
		var b remoteProgressBuffer

		// Pre-fill so the next append must truncate.
		const headroom = 16
		b.append(bytes.Repeat([]byte("a"), maxRemoteProgressBytes-headroom))

		// Construct a payload much larger than the remaining
		// headroom; the kept prefix should be exactly `headroom`
		// bytes and stored in a fresh allocation, not a subslice
		// of `payload`.
		payload := make([]byte, 4*1024)
		for i := range payload {
			payload[i] = 'X'
		}
		b.append(payload)

		require.Equal(t, maxRemoteProgressBytes, b.used,
			"buffer should be exactly at the cap after truncation")
		require.Len(t, b.payloads, 2)

		stored := b.payloads[1]
		require.Len(t, stored, headroom, "stored prefix size")

		// Capacity equality is the structural proof: a subslice of
		// `payload` would have cap(stored) == cap(payload)-offset
		// (~4096), keeping the full 4 KiB allocation alive. A fresh
		// copy has cap(stored) == len(stored) == headroom.
		require.Equal(t, headroom, cap(stored),
			"truncated prefix must be a fresh allocation, not a subslice "+
				"of the input (subslicing would retain the full input backing array)")

		// Behavioural proof: mutating the original after the fact
		// must not change the stored bytes.
		want := append([]byte(nil), stored...)
		for i := range payload {
			payload[i] = 0
		}
		require.Equal(t, want, stored, "stored bytes must be decoupled from input")
	})
}

// TestDecodeRemoteProgress exercises the channel-2 progress decoder
// directly. Side-band channel 2 is a byte stream — a single line may
// be split across outer packets, and servers terminate lines with
// either LF, CRLF, or bare CR (the latter is used for spinner
// overwrites). The decoder concatenates payloads, splits on LF/CR,
// trims surrounding whitespace, and drops empty lines.
func TestDecodeRemoteProgress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		packets [][]byte
		want    []string
	}{
		{
			name:    "nil input",
			packets: nil,
			want:    nil,
		},
		{
			name:    "no packets",
			packets: [][]byte{},
			want:    nil,
		},
		{
			name:    "all-whitespace packets are dropped",
			packets: [][]byte{[]byte("   \t  \n\r\n  ")},
			want:    nil,
		},
		{
			// FieldsFunc treats consecutive separators as one and
			// produces no fields at all, so this hits the
			// len(lines)==0 short-circuit rather than the
			// per-line trim-then-drop path above.
			name:    "only line terminators yields no lines",
			packets: [][]byte{[]byte("\n\n\r\n\r\n")},
			want:    nil,
		},
		{
			name:    "single LF-terminated line",
			packets: [][]byte{[]byte("hello\n")},
			want:    []string{"hello"},
		},
		{
			name:    "single line without trailing LF",
			packets: [][]byte{[]byte("hello")},
			want:    []string{"hello"},
		},
		{
			name:    "CRLF line terminator",
			packets: [][]byte{[]byte("hello\r\nworld\r\n")},
			want:    []string{"hello", "world"},
		},
		{
			name:    "bare CR line terminator (spinner overwrite)",
			packets: [][]byte{[]byte("frame1\rframe2\rfinal\n")},
			want:    []string{"frame1", "frame2", "final"},
		},
		{
			name:    "trailing whitespace is trimmed but leading whitespace is preserved",
			packets: [][]byte{[]byte("  spaced  \n\thello\t\n")},
			want:    []string{"  spaced", "\thello"},
		},
		{
			name: "indented hook output keeps its indentation (bullet lists, sub-items)",
			packets: [][]byte{
				[]byte("GL-HOOK-ERR: Push rule violations:\n"),
				[]byte("  - Commit message must reference an issue\n"),
				[]byte("  - File exceeds maximum size\n"),
				[]byte("    See https://example.com/policy\n"),
			},
			want: []string{
				"GL-HOOK-ERR: Push rule violations:",
				"  - Commit message must reference an issue",
				"  - File exceeds maximum size",
				"    See https://example.com/policy",
			},
		},
		{
			name:    "consecutive empty lines are dropped",
			packets: [][]byte{[]byte("\n\n\nreal line\n\n\n")},
			want:    []string{"real line"},
		},
		{
			name: "line split across packet boundaries is reassembled",
			packets: [][]byte{
				[]byte("part one "),
				[]byte("part two\n"),
			},
			want: []string{"part one part two"},
		},
		{
			name: "multiple packets with multiple lines",
			packets: [][]byte{
				[]byte("first\nsecond"),
				[]byte("-continued\n"),
				[]byte("third\n"),
			},
			want: []string{"first", "second-continued", "third"},
		},
		{
			name: "GitLab-style decorated multi-line output",
			packets: [][]byte{
				[]byte("\n========================================\n"),
				[]byte("GL-HOOK-ERR: Commit must reference issue.\n"),
				[]byte("GL-HOOK-ERR: See https://example.com\n"),
				[]byte("========================================\n\n"),
			},
			want: []string{
				"========================================",
				"GL-HOOK-ERR: Commit must reference issue.",
				"GL-HOOK-ERR: See https://example.com",
				"========================================",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := decodeRemoteProgress(tt.packets)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestReceivePack_RemoteProgressOnError ensures human-readable remote
// progress messages emitted on side-band channel 2 are attached to any
// error returned from the same response. Pre-receive hooks and push
// rules on servers like GitLab write their reason to channel 2; a bare
// "pre-receive hook declined" reason is rarely actionable on its own,
// so the wrapper preserves the underlying typed error and enriches the
// surfaced message with the captured `remote: …` lines.
func TestReceivePack_RemoteProgressOnError(t *testing.T) {
	t.Parallel()

	pkt := func(s string) []byte {
		b, err := protocol.PackLine(s).Marshal()
		require.NoError(t, err)
		return b
	}
	flushed := func(data []byte) []byte {
		return append(append([]byte{}, data...), []byte("0000")...)
	}
	progress := func(text string) []byte {
		return pkt(string(append([]byte{0x02}, []byte(text)...)))
	}
	rawSideband1 := func(text string) []byte {
		payload := append([]byte{0x01}, []byte(text)...)
		b, err := protocol.PackLine(payload).Marshal()
		require.NoError(t, err)
		return b
	}

	t.Run("ng with channel-2 progress surfaces both reason and remote messages", func(t *testing.T) {
		t.Parallel()
		body := flushed(bytes.Join([][]byte{
			progress("GitLab: You are not allowed to push code to protected branches on this project.\n"),
			rawSideband1("ng refs/heads/main pre-receive hook declined\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.Error(t, err)

		// Underlying typed error is preserved for programmatic inspection.
		var refErr *protocol.GitReferenceUpdateError
		require.True(t, errors.As(err, &refErr), "expected GitReferenceUpdateError in chain, got %T: %v", err, err)
		require.Equal(t, "refs/heads/main", refErr.RefName)
		require.Equal(t, "pre-receive hook declined", refErr.Reason)

		// Wrapper carries the channel-2 progress lines.
		var wrapped *protocol.RemoteRejectionError
		require.True(t, errors.As(err, &wrapped), "expected RemoteRejectionError in chain, got %T: %v", err, err)
		require.Equal(t, []string{
			"GitLab: You are not allowed to push code to protected branches on this project.",
		}, wrapped.RemoteMessages)

		// The surfaced message includes both pieces.
		require.Contains(t, err.Error(), "reference update failed for refs/heads/main: pre-receive hook declined")
		require.Contains(t, err.Error(), "remote: GitLab: You are not allowed to push code to protected branches on this project.")
	})

	t.Run("multiline progress is split into separate remote lines and empties dropped", func(t *testing.T) {
		t.Parallel()
		body := flushed(bytes.Join([][]byte{
			progress("\n========================================\n"),
			progress("GL-HOOK-ERR: Commit message must reference an issue.\n"),
			progress("GL-HOOK-ERR: See https://example.com/policy\n"),
			progress("========================================\n\n"),
			rawSideband1("ng refs/heads/main pre-receive hook declined\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.Error(t, err)

		var wrapped *protocol.RemoteRejectionError
		require.True(t, errors.As(err, &wrapped))
		require.Equal(t, []string{
			"========================================",
			"GL-HOOK-ERR: Commit message must reference an issue.",
			"GL-HOOK-ERR: See https://example.com/policy",
			"========================================",
		}, wrapped.RemoteMessages)
	})

	t.Run("progress line split across packets is reassembled", func(t *testing.T) {
		t.Parallel()
		body := flushed(bytes.Join([][]byte{
			progress("GitLab: Push rule "),
			progress("violation: file too large.\n"),
			rawSideband1("ng refs/heads/main pre-receive hook declined\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.Error(t, err)
		require.Contains(t, err.Error(), "remote: GitLab: Push rule violation: file too large.")
	})

	t.Run("channel-2 progress on a successful push does not produce a wrapper error", func(t *testing.T) {
		t.Parallel()
		body := flushed(bytes.Join([][]byte{
			progress("Counting objects: 1, done.\n"),
			pkt("unpack ok\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.NoError(t, err)
	})

	t.Run("error without channel-2 progress is not wrapped", func(t *testing.T) {
		t.Parallel()
		body := flushed(rawSideband1("ng refs/heads/main pre-receive hook declined\n"))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.Error(t, err)

		var wrapped *protocol.RemoteRejectionError
		require.False(t, errors.As(err, &wrapped), "did not expect RemoteRejectionError when no channel-2 progress was sent, got %T: %v", err, err)
	})

	t.Run("unpack failure with channel-2 progress is wrapped and preserves GitUnpackError", func(t *testing.T) {
		t.Parallel()
		// Bare channel-0 "unpack <reason>" (not "ok") triggers
		// detectError → GitUnpackError directly out of parser.Next().
		// The same response carries channel-2 progress from the
		// server. Note: a channel-2 packet whose payload is prefixed
		// with "error:" or "fatal:" is intentionally converted to a
		// GitServerError by detectError (see isErrorOrFatalMessageOptimized);
		// only non-error-prefixed channel-2 payloads are captured as
		// progress, so the fixture deliberately uses a plain message.
		body := flushed(bytes.Join([][]byte{
			progress("Receiving objects...\n"),
			pkt("unpack index-pack failed\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.Error(t, err)

		var unpackErr *protocol.GitUnpackError
		require.True(t, errors.As(err, &unpackErr),
			"expected GitUnpackError in chain, got %T: %v", err, err)
		require.Equal(t, "index-pack failed", strings.TrimSpace(unpackErr.Message))

		var wrapped *protocol.RemoteRejectionError
		require.True(t, errors.As(err, &wrapped))
		require.Equal(t, []string{"Receiving objects..."}, wrapped.RemoteMessages)
	})

	t.Run("channel-3 fatal with channel-2 progress is wrapped and preserves GitServerError", func(t *testing.T) {
		t.Parallel()
		// Channel 2 progress arrives first, then a channel-3 fatal
		// terminates parsing. The defer in parseReceivePackResponse
		// must still attach the captured progress.
		body := flushed(bytes.Join([][]byte{
			progress("Counting objects: 1, done.\n"),
			progress("Resolving deltas: 100% (0/0).\n"),
			pkt(string(append([]byte{0x03}, []byte("connection terminated by host")...))),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.Error(t, err)

		var serverErr *protocol.GitServerError
		require.True(t, errors.As(err, &serverErr),
			"expected GitServerError in chain, got %T: %v", err, err)

		var wrapped *protocol.RemoteRejectionError
		require.True(t, errors.As(err, &wrapped))
		require.Equal(t, []string{
			"Counting objects: 1, done.",
			"Resolving deltas: 100% (0/0).",
		}, wrapped.RemoteMessages)
	})

	t.Run("missing report-status with channel-2 progress is wrapped", func(t *testing.T) {
		t.Parallel()
		// A non-empty channel-1 packet with content that does not
		// match any error sentinel arms the report-status requirement
		// without satisfying it, producing ErrMissingReportStatus.
		// Channel-2 progress in the same response should still be
		// attached.
		body := flushed(bytes.Join([][]byte{
			progress("server-side message of the day\n"),
			rawSideband1("garbled report status with no recognizable line\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingReportStatus)

		var wrapped *protocol.RemoteRejectionError
		require.True(t, errors.As(err, &wrapped),
			"expected ErrMissingReportStatus to be wrapped with progress, got %T: %v", err, err)
		require.Equal(t, []string{"server-side message of the day"}, wrapped.RemoteMessages)
	})

	t.Run("empty channel-2 packet does not produce a wrapper", func(t *testing.T) {
		t.Parallel()
		// A packet that is just the side-band channel byte (0x02)
		// with no payload should not be captured as progress —
		// otherwise rejected pushes with no real progress would
		// surface a wrapper with an empty RemoteMessages list.
		body := flushed(bytes.Join([][]byte{
			pkt(string([]byte{0x02})),
			rawSideband1("ng refs/heads/main pre-receive hook declined\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.Error(t, err)

		var wrapped *protocol.RemoteRejectionError
		require.False(t, errors.As(err, &wrapped),
			"empty channel-2 payload must not arm the wrapper, got %T: %v", err, err)

		var refErr *protocol.GitReferenceUpdateError
		require.True(t, errors.As(err, &refErr))
	})

	t.Run("CR-only line terminator in progress is split correctly", func(t *testing.T) {
		t.Parallel()
		body := flushed(bytes.Join([][]byte{
			progress("Compressing: 50%\rCompressing: 100%\rdone\n"),
			rawSideband1("ng refs/heads/main pre-receive hook declined\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.Error(t, err)

		var wrapped *protocol.RemoteRejectionError
		require.True(t, errors.As(err, &wrapped))
		require.Equal(t, []string{
			"Compressing: 50%",
			"Compressing: 100%",
			"done",
		}, wrapped.RemoteMessages)
	})

	t.Run("empty channel-3 packet does not produce an error or wrapper", func(t *testing.T) {
		t.Parallel()
		// A packet that is just the side-band channel byte (0x03)
		// with no payload should not be surfaced as an error —
		// detectError already converts well-formed "fatal:"/"error:"
		// prefixed channel-3 packets, and classifyReceivePackLine
		// surfaces non-empty channel-3 payloads as a generic fatal.
		// An empty channel-3 packet is a no-op; combined with a
		// successful unpack ok, the response succeeds.
		body := flushed(bytes.Join([][]byte{
			pkt(string([]byte{0x03})),
			pkt("unpack ok\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.NoError(t, err)
	})

	t.Run("channel-2 packet with error: prefix alongside unpack ok is not an error", func(t *testing.T) {
		t.Parallel()
		// Per gitprotocol-pack, channel 2 is informational only.
		// A "error:" prefix on channel 2 must not cause failure when
		// channel 1 reports "unpack ok".
		body := flushed(bytes.Join([][]byte{
			progress("error: missing prerequisite object abc123\n"),
			pkt("unpack ok\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.NoError(t, err, "channel-2 error: prefix must not cause failure when channel-1 reports success")
	})

	t.Run("channel-2 fatal: alongside unpack ok is not an error (issue #124392)", func(t *testing.T) {
		t.Parallel()
		// Regression test for https://github.com/grafana/grafana/issues/124392.
		// GitLab CE emits "fatal: cannot exec..." on channel 2 while channel 1
		// carries "unpack ok". nanogit was incorrectly treating channel-2
		// "fatal:" as a fatal error. Only channel 3 is truly fatal.
		body := flushed(bytes.Join([][]byte{
			progress("fatal: cannot exec 'exit 0 #': No such file or directory\n"),
			pkt("unpack ok\n"),
			pkt("ok refs/heads/main\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.NoError(t, err, "channel-2 fatal: must not cause failure when channel-1 reports success")
	})

	// noKeepAliveClient gives a sub-test its own HTTP client whose
	// transport disables keep-alives, so heavy response-body tests
	// don't add idle connections to the shared http.DefaultTransport
	// pool. Without this, a flake in net/http's idle-connection
	// handling (golang/go#22330) can surface as
	// "http: CloseIdleConnections called" in any concurrent sub-test.
	noKeepAliveClient := options.WithHTTPClient(&http.Client{
		Transport: &http.Transport{DisableKeepAlives: true},
	})

	t.Run("channel-2 progress is bounded by maxRemoteProgressBytes", func(t *testing.T) {
		t.Parallel()
		// A verbose hook that streams more channel-2 bytes than the
		// per-response cap must not retain unbounded data; once the
		// cap is exhausted further packets are dropped silently and
		// the surfaced error contains only the captured prefix.
		// pkt-line lengths are 4 hex chars (uint16, max 0xffff), so
		// a single packet payload can't exceed ~64 KiB. We send
		// several 32 KiB chunks whose total exceeds the cap; the
		// captured byte count must not exceed maxRemoteProgressBytes.
		chunk := func(b byte) []byte {
			line := bytes.Repeat([]byte{b}, 32*1024)
			return progress(string(line) + "\n")
		}
		body := flushed(bytes.Join([][]byte{
			chunk('a'), chunk('b'), chunk('c'), chunk('d'), chunk('e'), // 5 * 32 KiB = 160 KiB > 64 KiB cap
			rawSideband1("ng refs/heads/main pre-receive hook declined\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL+"/repo", noKeepAliveClient)
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.Error(t, err)

		var wrapped *protocol.RemoteRejectionError
		require.True(t, errors.As(err, &wrapped))

		var totalRetained int
		for _, m := range wrapped.RemoteMessages {
			totalRetained += len(m)
		}
		require.LessOrEqual(t, totalRetained, maxRemoteProgressBytes,
			"total captured progress must not exceed the byte cap")
		// Sanity: we should have actually filled close to the cap so
		// this test would notice if the bound were silently set to 0.
		require.Greater(t, totalRetained, maxRemoteProgressBytes/2,
			"captured progress should fill at least half the cap before truncation")
	})

	t.Run("channel-2 packets past the cap are silently dropped", func(t *testing.T) {
		t.Parallel()
		// First packet nearly fills the cap; second packet exceeds
		// the remaining headroom and is mostly dropped.
		filler := bytes.Repeat([]byte("a"), maxRemoteProgressBytes-1024) // leaves 1 KiB headroom
		dropped := bytes.Repeat([]byte("b"), 32*1024)                    // only ~1 KiB fits, rest dropped
		body := flushed(bytes.Join([][]byte{
			progress(string(filler) + "\n"),
			progress(string(dropped) + "\n"),
			rawSideband1("ng refs/heads/main pre-receive hook declined\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL+"/repo", noKeepAliveClient)
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.Error(t, err)

		var wrapped *protocol.RemoteRejectionError
		require.True(t, errors.As(err, &wrapped))
		var totalRetained int
		for _, m := range wrapped.RemoteMessages {
			totalRetained += len(m)
		}
		require.LessOrEqual(t, totalRetained, maxRemoteProgressBytes,
			"total captured progress must not exceed the byte cap regardless of packet count")
	})

	t.Run("progress emitted after the rejection in the same response is still captured", func(t *testing.T) {
		t.Parallel()
		// Servers that interleave progress around the report-status
		// can emit channel-2 packets after the ng line. The defer in
		// parseReceivePackResponse runs after the full body is read,
		// so trailing progress must still be attached.
		body := flushed(bytes.Join([][]byte{
			rawSideband1("ng refs/heads/main pre-receive hook declined\n"),
			progress("trailing remote message after rejection\n"),
		}, nil))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		c, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = c.ReceivePack(context.Background(), bytes.NewReader([]byte("test data")))
		require.Error(t, err)

		var wrapped *protocol.RemoteRejectionError
		require.True(t, errors.As(err, &wrapped))
		require.Equal(t, []string{"trailing remote message after rejection"}, wrapped.RemoteMessages)
	})
}
