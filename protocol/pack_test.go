package protocol_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol"
)

func TestFormatPackets(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		input    []protocol.Pack
		expected []byte
		wantErr  error
	}{
		"empty": {
			input:    []protocol.Pack{},
			expected: []byte("0000"), // just the flush packet
		},
		"a + LF": {
			input:    []protocol.Pack{protocol.PackLine("a\n")},
			expected: []byte("0006a\n0000"),
		},
		"a": {
			input:    []protocol.Pack{protocol.PackLine("a")},
			expected: []byte("0005a0000"),
		},
		"foobar + \n": {
			input:    []protocol.Pack{protocol.PackLine("foobar\n")},
			expected: []byte("000bfoobar\n0000"),
		},
		"empty line": {
			input:    []protocol.Pack{protocol.PackLine("")},
			expected: []byte("00040000"),
		},
		"special-case: flush packet input": {
			input:    []protocol.Pack{protocol.FlushPacket},
			expected: []byte("0000"),
		},
		"special-case: delimeter packet input": {
			input:    []protocol.Pack{protocol.DelimeterPacket},
			expected: []byte("00010000"),
		},
		"special-case: response end packet input": {
			input:    []protocol.Pack{protocol.ResponseEndPacket},
			expected: []byte("00020000"),
		},
		"data too large": {
			input: []protocol.Pack{
				protocol.PackLine(make([]byte, protocol.MaxPktLineDataSize+1)),
			},
			wantErr: protocol.ErrDataTooLarge,
		},
		"exact max size": {
			input: []protocol.Pack{
				protocol.PackLine(make([]byte, protocol.MaxPktLineDataSize)),
			},
			expected: append(
				[]byte(fmt.Sprintf("%04x", protocol.MaxPktLineDataSize+4)),
				append(make([]byte, protocol.MaxPktLineDataSize), []byte("0000")...)...,
			),
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			actual, err := protocol.FormatPacks(tc.input...)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "expected error from FormatPackets")
			} else {
				require.NoError(t, err, "no error expected from FormatPackets")
			}
			require.Equal(t, tc.expected, actual, "expected and actual byte slices should be equal")
		})
	}
}

func TestParsePacket(t *testing.T) {
	t.Parallel()

	type expected struct {
		lines [][]byte
		err   error
	}

	toBytesSlice := func(lines ...string) [][]byte {
		out := make([][]byte, len(lines))
		for i, line := range lines {
			out[i] = []byte(line)
		}
		return out
	}

	testcases := map[string]struct {
		input    []byte
		expected expected
	}{
		"flush packet": {
			input: []byte("0000"),
			expected: expected{
				lines: nil,
				err:   nil,
			},
		},
		"delimiter packet": {
			input: []byte("0001"),
			expected: expected{
				lines: nil,
				err:   nil,
			},
		},
		"response end packet": {
			input: []byte("0002"),
			expected: expected{
				lines: nil,
				err:   nil,
			},
		},
		"empty": {
			input: []byte("0004"),
			expected: expected{
				lines: nil,
				err:   nil,
			},
		},
		"single line": {
			input: []byte("0009hello0000"),
			expected: expected{
				lines: toBytesSlice("hello"),
				err:   nil,
			},
		},
		"two lines": {
			input: []byte("0009hello0007bye0000"),
			expected: expected{
				lines: toBytesSlice("hello", "bye"),
				err:   nil,
			},
		},
		"short packet": {
			input: []byte("000"),
			expected: expected{
				lines: nil,
				err:   nil,
			},
		},
		"trailing bytes": {
			input: []byte("0009hello000"),
			expected: expected{
				lines: toBytesSlice("hello"),
				err:   nil,
			},
		},
		"trucated line": {
			// This line says it has 9 bytes, but only has 8.
			input: []byte("0009hell"),
			expected: expected{
				lines: nil,
				err:   new(protocol.PackParseError),
			},
		},
		"invalid length": {
			input: []byte("000Gxxxxxxxxxxxxxxxx"),
			expected: expected{
				lines: nil,
				err:   new(protocol.PackParseError),
			},
		},
		"error packet": {
			input: []byte("000dERR helloF00F"),
			expected: expected{
				lines: nil,
				err:   new(protocol.GitServerError),
			},
		},
		"lines + error packet": {
			input: []byte("0009hello000dERR helloF00F"),
			expected: expected{
				lines: toBytesSlice("hello"),
				err:   new(protocol.GitServerError),
			},
		},
		"git error packet": {
			input: func() []byte {
				message := "error: object 457e2462aee3d41d1a2832f10419213e10091bdc: treeNotSorted: not properly sorted\nfatal: fsck error in packed object\n"
				pkt, _ := protocol.PackLine(message).Marshal()
				return pkt
			}(),
			expected: expected{
				lines: nil,
				err:   new(protocol.GitServerError),
			},
		},
		"fatal error packet": {
			input: func() []byte {
				message := "fatal: fsck error occurred"
				pkt, _ := protocol.PackLine(message).Marshal()
				return pkt
			}(),
			expected: expected{
				lines: nil,
				err:   new(protocol.GitServerError),
			},
		},
		"reference update failure": {
			input: func() []byte {
				message := "ng refs/heads/robertoonboarding failed"
				pkt, _ := protocol.PackLine(message).Marshal()
				return pkt
			}(),
			expected: expected{
				lines: nil,
				err:   new(protocol.GitReferenceUpdateError),
			},
		},
		"user example scenario - original": {
			// This simulates the first example provided by the user
			// Single packet 0083 contains error message with newlines, followed by other packets
			input: func() []byte {
				// First packet: error message with newlines (matches user's 0083 packet)
				message1 := "error: object 457e2462aee3d41d1a2832f10419213e10091bdc: treeNotSorted: not properly sorted\nfatal: fsck error in packed object\n"
				pkt1, _ := protocol.PackLine(message1).Marshal()

				// Remaining packets as separate packets
				message2 := "001dunpack index-pack failed\n"
				pkt2, _ := protocol.PackLine(message2).Marshal()
				message3 := "ng refs/heads/robertoonboarding failed\n"
				pkt3, _ := protocol.PackLine(message3).Marshal()
				pkt4 := []byte("0009000000000000")
				return append(append(append(pkt1, pkt2...), pkt3...), pkt4...)
			}(),
			expected: expected{
				lines: nil,
				err:   new(protocol.GitServerError),
			},
		},
		"user example scenario - ref lock error": {
			// This simulates the second example from user debug output
			// Real-world scenario with ref lock error from receive-pack
			input: func() []byte {
				// First packet: error message (0094 = 148 bytes)
				message1 := "error: cannot lock ref 'refs/heads/main': is at d346cc9cd80dd0bbda023bb29a7ff2d887c75b19 but expected b6ce559b8c2e4834e075696cac5522b379448c13"
				pkt1, _ := protocol.PackLine(message1).Marshal()

				// Subsequent packets
				message2 := "unpack ok"
				pkt2, _ := protocol.PackLine(message2).Marshal()
				message3 := "ng refs/heads/main failed to update ref"
				pkt3, _ := protocol.PackLine(message3).Marshal()
				pkt4 := []byte("0000") // flush packet

				return append(append(append(pkt1, pkt2...), pkt3...), pkt4...)
			}(),
			expected: expected{
				lines: nil,
				err:   new(protocol.GitServerError),
			},
		},
		"multiple error types in sequence": {
			input: func() []byte {
				pkt1, _ := protocol.PackLine("hello").Marshal()
				pkt2, _ := protocol.PackLine("ERR hello").Marshal()
				pkt3, _ := protocol.PackLine("fatal: fsck error occurred").Marshal()
				return append(append(pkt1, pkt2...), pkt3...)
			}(),
			expected: expected{
				lines: toBytesSlice("hello"),
				err:   new(protocol.GitServerError),
			},
		},
		"lines + git error": {
			input: func() []byte {
				pkt1, _ := protocol.PackLine("hello").Marshal()
				pkt2, _ := protocol.PackLine("error: some error").Marshal()
				return append(pkt1, pkt2...)
			}(),
			expected: expected{
				lines: toBytesSlice("hello"),
				err:   new(protocol.GitServerError),
			},
		},
		"lines + reference failure": {
			input: func() []byte {
				pkt1, _ := protocol.PackLine("hello").Marshal()
				pkt2, _ := protocol.PackLine("ng refs/heads/main failed").Marshal()
				return append(pkt1, pkt2...)
			}(),
			expected: expected{
				lines: toBytesSlice("hello"),
				err:   new(protocol.GitReferenceUpdateError),
			},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			parser := protocol.NewParser(io.NopCloser(bytes.NewReader(tc.input)))

			var lines [][]byte
			var err error
			for {
				var line []byte
				line, err = parser.Next()
				if err != nil {
					if err == io.EOF {
						err = nil
						break
					}
					break
				}
				lines = append(lines, line)
			}

			require.Equal(t, tc.expected.lines, lines, "expected and actual lines should be equal")
			if tc.expected.err == nil {
				require.NoError(t, err, "no error expected from ParsePack")
			} else {
				require.Error(t, err, "error expected from ParsePack")
				require.IsType(t, tc.expected.err, err, "expected and actual error types should be equal")
			}
		})
	}
}

func TestPackParseError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *protocol.PackParseError
		expected string
	}{
		{
			name:     "empty error",
			err:      &protocol.PackParseError{},
			expected: "error parsing line \"\"",
		},
		{
			name:     "with line",
			err:      &protocol.PackParseError{Line: []byte("test")},
			expected: "error parsing line \"test\"",
		},
		{
			name:     "with error",
			err:      &protocol.PackParseError{Err: errors.New("test error")},
			expected: "error parsing line \"\": test error",
		},
		{
			name:     "with line and error",
			err:      &protocol.PackParseError{Line: []byte("test"), Err: errors.New("test error")},
			expected: "error parsing line \"test\": test error",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, tt.err.Error())
		})
	}

	// Test error wrapping with errors.Is
	t.Run("errors.Is", func(t *testing.T) {
		baseErr := errors.New("base error")
		err := &protocol.PackParseError{Err: baseErr}

		require.ErrorIs(t, err, baseErr, "errors.Is should find the base error")
		require.NotErrorIs(t, err, errors.New("different error"), "errors.Is should not match different errors")
	})

	// Test error unwrapping with errors.As
	t.Run("errors.As", func(t *testing.T) {
		var parseErr *protocol.PackParseError
		err := fmt.Errorf("wrapped: %w", &protocol.PackParseError{Line: []byte("test"), Err: errors.New("test error")})

		require.ErrorAs(t, err, &parseErr, "should be able to get PackParseError using ErrorAs")
		require.Equal(t, []byte("test"), parseErr.Line)
		require.Equal(t, "test error", parseErr.Err.Error())
	})

	// Test error unwrapping with Unwrap method
	t.Run("Unwrap", func(t *testing.T) {
		baseErr := errors.New("base error")
		err := &protocol.PackParseError{Err: baseErr}

		unwrapped := errors.Unwrap(err)
		require.Equal(t, baseErr, unwrapped, "Unwrap should return the base error")

		// Test with nil error
		nilErr := &protocol.PackParseError{Err: nil}
		require.NoError(t, errors.Unwrap(nilErr), "Unwrap should return nil for nil error")
	})
	// Test IsPackParseError function
	t.Run("IsPackParseError", func(t *testing.T) {
		t.Parallel()

		// Test with a PackParseError
		parseErr := &protocol.PackParseError{Line: []byte("test"), Err: errors.New("test error")}
		require.True(t, protocol.IsPackParseError(parseErr), "IsPackParseError should return true for PackParseError")

		// Test with a wrapped PackParseError
		wrappedErr := fmt.Errorf("wrapped: %w", parseErr)
		require.True(t, protocol.IsPackParseError(wrappedErr), "IsPackParseError should return true for wrapped PackParseError")

		// Test with a different error type
		otherErr := errors.New("different error")
		require.False(t, protocol.IsPackParseError(otherErr), "IsPackParseError should return false for non-PackParseError")

		// Test with nil
		require.False(t, protocol.IsPackParseError(nil), "IsPackParseError should return false for nil")
	})
}

func TestGitServerError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		line        []byte
		errorType   string
		message     string
		expectedErr string
	}{
		{
			name:        "ERR packet",
			line:        []byte("000dERR hello"),
			errorType:   "ERR",
			message:     "hello",
			expectedErr: "git server ERR: hello",
		},
		{
			name:        "error packet",
			line:        []byte("0012error: some error"),
			errorType:   "error",
			message:     " some error",
			expectedErr: "git server error:  some error",
		},
		{
			name:        "fatal packet",
			line:        []byte("0011fatal: fatal error"),
			errorType:   "fatal",
			message:     " fatal error",
			expectedErr: "git server fatal:  fatal error",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := protocol.NewGitServerError(tt.line, tt.errorType, tt.message)
			require.Equal(t, tt.expectedErr, err.Error())
			require.Equal(t, tt.line, err.Line)
			require.Equal(t, tt.errorType, err.ErrorType)
			require.Equal(t, tt.message, err.Message)

			// Test that it's a GitServerError
			require.True(t, protocol.IsGitServerError(err))
		})
	}

	// Test IsGitServerError function
	t.Run("IsGitServerError", func(t *testing.T) {
		t.Parallel()

		// Test with a GitServerError
		serverErr := protocol.NewGitServerError([]byte("test"), "ERR", "test message")
		require.True(t, protocol.IsGitServerError(serverErr))

		// Test with a wrapped GitServerError
		wrappedErr := fmt.Errorf("wrapped: %w", serverErr)
		require.True(t, protocol.IsGitServerError(wrappedErr))

		// Test with a different error type
		otherErr := errors.New("different error")
		require.False(t, protocol.IsGitServerError(otherErr))

		// Test with nil
		require.False(t, protocol.IsGitServerError(nil))
	})
}

func TestGitReferenceUpdateError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		line        []byte
		refName     string
		reason      string
		expectedErr string
	}{
		{
			name:        "reference update failed",
			line:        []byte("0020ng refs/heads/main failed"),
			refName:     "refs/heads/main",
			reason:      "failed",
			expectedErr: "reference update failed for refs/heads/main: failed",
		},
		{
			name:        "reference update with detailed reason",
			line:        []byte("0030ng refs/heads/feature non-fast-forward"),
			refName:     "refs/heads/feature",
			reason:      "non-fast-forward",
			expectedErr: "reference update failed for refs/heads/feature: non-fast-forward",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := protocol.NewGitReferenceUpdateError(tt.line, tt.refName, tt.reason)
			require.Equal(t, tt.expectedErr, err.Error())
			require.Equal(t, tt.line, err.Line)
			require.Equal(t, tt.refName, err.RefName)
			require.Equal(t, tt.reason, err.Reason)

			// Test that it's a GitReferenceUpdateError
			require.True(t, protocol.IsGitReferenceUpdateError(err))
		})
	}

	// Test IsGitReferenceUpdateError function
	t.Run("IsGitReferenceUpdateError", func(t *testing.T) {
		t.Parallel()

		// Test with a GitReferenceUpdateError
		refErr := protocol.NewGitReferenceUpdateError([]byte("test"), "refs/heads/main", "failed")
		require.True(t, protocol.IsGitReferenceUpdateError(refErr))

		// Test with a wrapped GitReferenceUpdateError
		wrappedErr := fmt.Errorf("wrapped: %w", refErr)
		require.True(t, protocol.IsGitReferenceUpdateError(wrappedErr))

		// Test with a different error type
		otherErr := errors.New("different error")
		require.False(t, protocol.IsGitReferenceUpdateError(otherErr))

		// Test with nil
		require.False(t, protocol.IsGitReferenceUpdateError(nil))
	})
}

func TestGitUnpackError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		line        []byte
		message     string
		expectedErr string
	}{
		{
			name:        "unpack failed",
			line:        []byte("0015unpack failed"),
			message:     "failed",
			expectedErr: "pack unpack failed: failed",
		},
		{
			name:        "unpack with detailed message",
			line:        []byte("0025unpack index-pack failed"),
			message:     "index-pack failed",
			expectedErr: "pack unpack failed: index-pack failed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := protocol.NewGitUnpackError(tt.line, tt.message)
			require.Equal(t, tt.expectedErr, err.Error())
			require.Equal(t, tt.line, err.Line)
			require.Equal(t, tt.message, err.Message)

			// Test that it's a GitUnpackError
			require.True(t, protocol.IsGitUnpackError(err))
		})
	}

	// Test IsGitUnpackError function
	t.Run("IsGitUnpackError", func(t *testing.T) {
		t.Parallel()

		// Test with a GitUnpackError
		unpackErr := protocol.NewGitUnpackError([]byte("test"), "failed")
		require.True(t, protocol.IsGitUnpackError(unpackErr))

		// Test with a wrapped GitUnpackError
		wrappedErr := fmt.Errorf("wrapped: %w", unpackErr)
		require.True(t, protocol.IsGitUnpackError(wrappedErr))

		// Test with a different error type
		otherErr := errors.New("different error")
		require.False(t, protocol.IsGitUnpackError(otherErr))

		// Test with nil
		require.False(t, protocol.IsGitUnpackError(nil))
	})
}

func TestRemoteRejectionError(t *testing.T) {
	t.Parallel()

	t.Run("Error passes through underlying when no remote messages", func(t *testing.T) {
		t.Parallel()
		underlying := protocol.NewGitReferenceUpdateError(
			[]byte("ng refs/heads/main pre-receive hook declined"),
			"refs/heads/main", "pre-receive hook declined",
		)
		wrapped := &protocol.RemoteRejectionError{Err: underlying}
		require.Equal(t, underlying.Error(), wrapped.Error())
	})

	t.Run("Error passes through underlying when RemoteMessages is nil", func(t *testing.T) {
		t.Parallel()
		underlying := errors.New("boom")
		wrapped := &protocol.RemoteRejectionError{Err: underlying, RemoteMessages: nil}
		require.Equal(t, "boom", wrapped.Error())
	})

	t.Run("Error returns fallback when Err is nil and no remote messages", func(t *testing.T) {
		t.Parallel()
		// Defensive zero-value behaviour: the type is exported, so
		// external callers can construct a bare RemoteRejectionError.
		// .Error() must not panic.
		wrapped := &protocol.RemoteRejectionError{}
		require.NotPanics(t, func() { _ = wrapped.Error() })
		require.NotEmpty(t, wrapped.Error())
	})

	t.Run("Error returns fallback prefix when Err is nil but remote messages set", func(t *testing.T) {
		t.Parallel()
		wrapped := &protocol.RemoteRejectionError{
			RemoteMessages: []string{"line 1", "line 2"},
		}
		require.NotPanics(t, func() { _ = wrapped.Error() })
		got := wrapped.Error()
		require.Contains(t, got, "remote: line 1")
		require.Contains(t, got, "remote: line 2")
	})

	t.Run("Unwrap returns nil when Err is nil and does not panic", func(t *testing.T) {
		t.Parallel()
		wrapped := &protocol.RemoteRejectionError{}
		require.NotPanics(t, func() { _ = errors.Unwrap(wrapped) })
		require.Nil(t, errors.Unwrap(wrapped))
	})

	t.Run("Error appends single remote message on its own line", func(t *testing.T) {
		t.Parallel()
		underlying := protocol.NewGitReferenceUpdateError(
			[]byte("ng refs/heads/main pre-receive hook declined"),
			"refs/heads/main", "pre-receive hook declined",
		)
		wrapped := &protocol.RemoteRejectionError{
			Err: underlying,
			RemoteMessages: []string{
				"GitLab: You are not allowed to push code to protected branches on this project.",
			},
		}
		require.Equal(t,
			"reference update failed for refs/heads/main: pre-receive hook declined\n"+
				"remote: GitLab: You are not allowed to push code to protected branches on this project.",
			wrapped.Error())
	})

	t.Run("Error appends each remote message on its own line preserving order", func(t *testing.T) {
		t.Parallel()
		underlying := protocol.NewGitUnpackError([]byte("unpack failed"), "failed")
		wrapped := &protocol.RemoteRejectionError{
			Err: underlying,
			RemoteMessages: []string{
				"line 1",
				"line 2",
				"line 3",
			},
		}
		require.Equal(t,
			"pack unpack failed: failed\n"+
				"remote: line 1\n"+
				"remote: line 2\n"+
				"remote: line 3",
			wrapped.Error())
	})

	t.Run("Unwrap returns the underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := errors.New("underlying")
		wrapped := &protocol.RemoteRejectionError{Err: underlying}
		require.Same(t, underlying, errors.Unwrap(wrapped))
	})

	t.Run("errors.As finds the wrapper through chain", func(t *testing.T) {
		t.Parallel()
		underlying := protocol.NewGitReferenceUpdateError(nil, "refs/heads/main", "denied")
		wrapped := &protocol.RemoteRejectionError{Err: underlying, RemoteMessages: []string{"hello"}}
		// Wrap once more like callers would.
		outer := fmt.Errorf("git protocol error: %w", wrapped)

		var got *protocol.RemoteRejectionError
		require.True(t, errors.As(outer, &got))
		require.Equal(t, []string{"hello"}, got.RemoteMessages)
	})

	t.Run("errors.As finds underlying typed error through wrapper", func(t *testing.T) {
		t.Parallel()
		underlying := protocol.NewGitReferenceUpdateError(nil, "refs/heads/main", "denied")
		wrapped := &protocol.RemoteRejectionError{Err: underlying, RemoteMessages: []string{"hello"}}

		var refErr *protocol.GitReferenceUpdateError
		require.True(t, errors.As(wrapped, &refErr))
		require.Equal(t, "refs/heads/main", refErr.RefName)
		require.Equal(t, "denied", refErr.Reason)
	})

	t.Run("errors.Is finds underlying sentinel through wrapper", func(t *testing.T) {
		t.Parallel()
		underlying := protocol.NewGitReferenceUpdateError(nil, "refs/heads/main", "denied")
		wrapped := &protocol.RemoteRejectionError{Err: underlying, RemoteMessages: []string{"hello"}}
		require.True(t, errors.Is(wrapped, protocol.ErrGitReferenceUpdateError))
	})

	t.Run("errors.Is finds GitUnpackError sentinel through wrapper", func(t *testing.T) {
		t.Parallel()
		underlying := protocol.NewGitUnpackError(nil, "boom")
		wrapped := &protocol.RemoteRejectionError{Err: underlying, RemoteMessages: []string{"hello"}}
		require.True(t, errors.Is(wrapped, protocol.ErrGitUnpackError))
	})
}

func TestParsePackNewErrorTypes(t *testing.T) {
	t.Parallel()

	t.Run("unpack ok", func(t *testing.T) {
		input := func() []byte {
			message := "unpack ok"
			pkt, _ := protocol.PackLine(message).Marshal()
			return pkt
		}()
		parser := protocol.NewParser(bytes.NewReader(input))
		lines := [][]byte{}
		var err error
		for {
			var line []byte
			line, err = parser.Next()
			if err != nil {
				break
			}
			lines = append(lines, line)
		}

		require.Equal(t, io.EOF, err)
		require.Equal(t, [][]byte{[]byte("unpack ok")}, lines)
	})

	t.Run("unpack failed", func(t *testing.T) {
		input := func() []byte {
			message := "unpack index-pack failed"
			pkt, _ := protocol.PackLine(message).Marshal()
			return pkt
		}()

		parser := protocol.NewParser(bytes.NewReader(input))

		lines := [][]byte{}
		var err error
		for {
			var line []byte
			line, err = parser.Next()
			if err != nil {
				break
			}
			lines = append(lines, line)
		}
		require.Empty(t, lines)
		require.Error(t, err)
		require.True(t, protocol.IsGitUnpackError(err))

		var unpackErr *protocol.GitUnpackError
		require.ErrorAs(t, err, &unpackErr)
		require.Equal(t, "index-pack failed", unpackErr.Message)
	})

	t.Run("fatal with unpack keyword", func(t *testing.T) {
		input := func() []byte {
			message := "fatal: unpack failed"
			pkt, _ := protocol.PackLine(message).Marshal()
			return pkt
		}()

		parser := protocol.NewParser(bytes.NewReader(input))

		lines := [][]byte{}
		var err error
		for {
			var line []byte
			line, err = parser.Next()
			if err != nil {
				break
			}
			lines = append(lines, line)
		}
		require.Empty(t, lines)
		require.Error(t, err)
		require.True(t, protocol.IsGitUnpackError(err))

		var unpackErr *protocol.GitUnpackError
		require.ErrorAs(t, err, &unpackErr)
		require.Equal(t, " unpack failed", unpackErr.Message)
	})
}

func TestParserRejectsOversizedPktLine(t *testing.T) {
	t.Parallel()

	// trackingReader records how many bytes were actually read so we can
	// assert that the parser did NOT pull the (fake) 65535-byte payload
	// into memory after rejecting the length header.
	type trackingReader struct {
		buf  *bytes.Buffer
		read int
	}

	readWith := func(payloadSize int, hexLen string) (int, error) {
		t.Helper()
		// 4-byte hex length, no body — readPacketLength only reads the
		// header, and we want to confirm the parser bails before trying
		// to read any payload bytes.
		buf := bytes.NewBuffer([]byte(hexLen))
		// Pad with arbitrary garbage to simulate a body the parser must
		// NOT consume.
		buf.Write(bytes.Repeat([]byte{'x'}, payloadSize))
		tr := &trackingReader{buf: buf}
		reader := readerFunc(func(p []byte) (int, error) {
			n, err := tr.buf.Read(p)
			tr.read += n
			return n, err
		})
		parser := protocol.NewParser(reader)
		_, err := parser.Next()
		return tr.read, err
	}

	t.Run("length 65521 is rejected (just over the cap)", func(t *testing.T) {
		// 0xFFF1 = 65521, one above MaxPktLineSize (65520).
		bytesRead, err := readWith(100, "fff1")
		require.Error(t, err)

		var tooLarge *protocol.ErrPktLineTooLarge
		require.ErrorAs(t, err, &tooLarge)
		require.Equal(t, uint64(65521), tooLarge.Length)
		require.Equal(t, uint64(protocol.MaxPktLineSize), tooLarge.Max)

		// errors.Is should match the sentinel.
		require.True(t, errors.Is(err, protocol.ErrPktLineTooLargeSentinel))

		// Crucially, the parser must not have read any payload bytes.
		require.Equal(t, 4, bytesRead, "parser should bail after the 4-byte length header")
	})

	t.Run("length 65535 is rejected (max representable)", func(t *testing.T) {
		bytesRead, err := readWith(100, "ffff")
		require.Error(t, err)

		var tooLarge *protocol.ErrPktLineTooLarge
		require.ErrorAs(t, err, &tooLarge)
		require.Equal(t, uint64(65535), tooLarge.Length)

		require.Equal(t, 4, bytesRead)
	})

	t.Run("length 65520 is accepted (exactly at the cap)", func(t *testing.T) {
		// Build a real well-formed packet of length 65520 and ensure it
		// parses without triggering the new check.
		payload := bytes.Repeat([]byte{'a'}, protocol.MaxPktLineDataSize)
		hdr := fmt.Sprintf("%04x", protocol.MaxPktLineSize)
		buf := bytes.NewBuffer(nil)
		buf.WriteString(hdr)
		buf.Write(payload)

		parser := protocol.NewParser(buf)
		got, err := parser.Next()
		require.NoError(t, err)
		require.Equal(t, payload, got)
	})
}

// readerFunc adapts a function into an io.Reader for use in tests.
type readerFunc func(p []byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

func TestErrPktLineTooLarge(t *testing.T) {
	t.Parallel()

	t.Run("Error formats length and max", func(t *testing.T) {
		// Pin the exact stringification: operators read this in logs
		// and pre-existing alerting rules may match the prefix.
		e := &protocol.ErrPktLineTooLarge{Length: 70000, Max: protocol.MaxPktLineSize}
		require.Equal(t,
			"pkt-line length 70000 exceeds protocol max 65520",
			e.Error())
	})

	t.Run("errors.Is matches the sentinel", func(t *testing.T) {
		e := &protocol.ErrPktLineTooLarge{Length: 1, Max: 0}
		require.True(t, errors.Is(e, protocol.ErrPktLineTooLargeSentinel))
		// Unwrap returns the sentinel.
		require.Same(t, protocol.ErrPktLineTooLargeSentinel, e.Unwrap())
	})

	t.Run("errors.As recovers the typed error", func(t *testing.T) {
		// Wrap once with fmt.Errorf to simulate the path taken when
		// the parser returns the error and a higher layer adds
		// context (e.g. fmt.Errorf("parse: %w", err)).
		wrapped := fmt.Errorf("parse: %w", &protocol.ErrPktLineTooLarge{Length: 99999, Max: 65520})
		var typed *protocol.ErrPktLineTooLarge
		require.True(t, errors.As(wrapped, &typed))
		require.Equal(t, uint64(99999), typed.Length)
		require.Equal(t, uint64(65520), typed.Max)
	})
}

// TestParsePack_UnpackTrailingWhitespace covers bug-fix: "unpack ok\n"
// and "unpack ok\r\n" (with trailing whitespace permitted by
// gitprotocol-common) must be accepted as success, not rejected as
// GitUnpackError{Message: "ok\n"}. Real-world servers (Gitea, GitHub)
// include a trailing LF on the report-status sentinel, especially when
// side-band is disabled.
func TestParsePack_UnpackTrailingWhitespace(t *testing.T) {
	t.Parallel()

	// parseAll drains parser.Next() until EOF or a non-EOF error.
	parseAll := func(data []byte) ([][]byte, error) {
		parser := protocol.NewParser(bytes.NewReader(data))
		var lines [][]byte
		for {
			line, err := parser.Next()
			if err != nil {
				if err == io.EOF {
					return lines, nil
				}
				return lines, err
			}
			lines = append(lines, line)
		}
	}

	pkt := func(s string) []byte {
		b, err := protocol.PackLine(s).Marshal()
		require.NoError(t, err)
		return b
	}

	tests := []struct {
		name        string
		input       []byte
		wantErr     bool
		wantMessage string // for GitUnpackError cases
	}{
		{
			name:    "unpack ok no trailing newline",
			input:   pkt("unpack ok"),
			wantErr: false,
		},
		{
			name:    "unpack ok with LF",
			input:   pkt("unpack ok\n"),
			wantErr: false,
		},
		{
			name:    "unpack ok with CRLF",
			input:   pkt("unpack ok\r\n"),
			wantErr: false,
		},
		{
			name:    "unpack ok with trailing spaces",
			input:   pkt("unpack ok  \n"),
			wantErr: false,
		},
		{
			name:        "unpack failure with trailing LF",
			input:       pkt("unpack index-pack failed\n"),
			wantErr:     true,
			wantMessage: "index-pack failed\n",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseAll(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				require.True(t, protocol.IsGitUnpackError(err),
					"expected GitUnpackError, got %T: %v", err, err)
				var unpackErr *protocol.GitUnpackError
				require.ErrorAs(t, err, &unpackErr)
				require.Equal(t, tt.wantMessage, unpackErr.Message)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestParsePack_Channel1NotMisclassified guards the rule that the
// generic Parser does NOT unwrap side-band channel 1 (0x01) before
// matching report-status prefixes. Channel 1 also carries arbitrary
// binary packfile data during fetch/clone (via MultiplexedReader); a
// pack chunk that happens to start with bytes spelling "ng " or
// "unpack " must not be misclassified as a reference-update or unpack
// failure. Channel-1 unwrapping for receive-pack is performed in
// protocol/client.parseReceivePackResponse, where it is correctly
// scoped to the response-parsing context.
func TestParsePack_Channel1NotMisclassified(t *testing.T) {
	t.Parallel()

	parseAll := func(data []byte) ([][]byte, error) {
		parser := protocol.NewParser(bytes.NewReader(data))
		var lines [][]byte
		for {
			line, err := parser.Next()
			if err != nil {
				if err == io.EOF {
					return lines, nil
				}
				return lines, err
			}
			lines = append(lines, line)
		}
	}

	wrapChannel1 := func(payload []byte) []byte {
		body := append([]byte{0x01}, payload...)
		b, err := protocol.PackLine(body).Marshal()
		require.NoError(t, err)
		return b
	}

	tests := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "channel 1 chunk starting with 'ng ' is opaque data",
			payload: []byte("ng refs/heads/main pre-receive hook declined\n"),
		},
		{
			name:    "channel 1 chunk starting with 'unpack ' is opaque data",
			payload: []byte("unpack index-pack failed\n"),
		},
		{
			name:    "channel 1 chunk starting with 'ERR ' is opaque data",
			payload: []byte("ERR push declined"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lines, err := parseAll(wrapChannel1(tt.payload))
			require.NoError(t, err, "channel 1 binary payload must not produce a report-status error")
			require.Len(t, lines, 1)
			require.Equal(t, byte(0x01), lines[0][0], "channel byte must be preserved on the returned line")
		})
	}

	t.Run("channel 2 progress still treated as regular", func(t *testing.T) {
		t.Parallel()
		// Progress messages (channel 2) that are NOT error:/fatal: are
		// returned as regular data lines, not errors. This guards the
		// existing behavior for side-band channel 2 progress output.
		body := append([]byte{0x02}, []byte("Counting objects: 100% (5/5), done.\n")...)
		b, err := protocol.PackLine(body).Marshal()
		require.NoError(t, err)
		_, err = parseAll(b)
		require.NoError(t, err)
	})
}
