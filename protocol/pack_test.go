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
