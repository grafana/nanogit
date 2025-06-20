package protocol_test

import (
	"errors"
	"fmt"
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
		lines     [][]byte
		remainder []byte
		err       error
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
				lines:     nil,
				remainder: []byte{},
				err:       nil,
			},
		},
		"delimiter packet": {
			input: []byte("0001"),
			expected: expected{
				lines:     nil,
				remainder: []byte{},
				err:       nil,
			},
		},
		"response end packet": {
			input: []byte("0002"),
			expected: expected{
				lines:     nil,
				remainder: []byte{},
				err:       nil,
			},
		},
		"empty": {
			input: []byte("0004"),
			expected: expected{
				lines:     nil,
				remainder: []byte{},
				err:       nil,
			},
		},
		"single line": {
			input: []byte("0009hello0000"),
			expected: expected{
				lines:     toBytesSlice("hello"),
				remainder: []byte{},
				err:       nil,
			},
		},
		"two lines": {
			input: []byte("0009hello0007bye0000"),
			expected: expected{
				lines:     toBytesSlice("hello", "bye"),
				remainder: []byte{},
				err:       nil,
			},
		},
		"short packet": {
			input: []byte("000"),
			expected: expected{
				lines:     nil,
				remainder: []byte("000"),
				err:       nil,
			},
		},
		"trailing bytes": {
			input: []byte("0009hello000"),
			expected: expected{
				lines:     toBytesSlice("hello"),
				remainder: []byte("000"),
				err:       nil,
			},
		},
		"trucated line": {
			// This line says it has 9 bytes, but only has 8.
			input: []byte("0009hell"),
			expected: expected{
				lines:     nil,
				remainder: []byte("0009hell"),
				err:       new(protocol.PackParseError),
			},
		},
		"invalid length": {
			input: []byte("000Gxxxxxxxxxxxxxxxx"),
			expected: expected{
				lines:     nil,
				remainder: []byte("000Gxxxxxxxxxxxxxxxx"),
				err:       new(protocol.PackParseError),
			},
		},
		"error packet": {
			input: []byte("000dERR helloF00F"),
			expected: expected{
				lines:     nil,
				remainder: []byte("F00F"),
				err:       errors.New("error packet: hello"),
			},
		},
		"lines + error packet": {
			input: []byte("0009hello000dERR helloF00F"),
			expected: expected{
				lines:     toBytesSlice("hello"),
				remainder: []byte("F00F"),
				err:       errors.New("error packet: hello"),
			},
		},
		"git error packet": {
			input: func() []byte {
				message := "error: object 457e2462aee3d41d1a2832f10419213e10091bdc: treeNotSorted: not properly sorted\nfatal: fsck error in packed object\n"
				pkt, _ := protocol.PackLine(message).Marshal()
				return pkt
			}(),
			expected: expected{
				lines:     nil,
				remainder: []byte{},
				err:       errors.New("git error: error: object 457e2462aee3d41d1a2832f10419213e10091bdc: treeNotSorted: not properly sorted\nfatal: fsck error in packed object\n"),
			},
		},
		"fatal error packet": {
			input: func() []byte {
				message := "fatal: fsck error occurred"
				pkt, _ := protocol.PackLine(message).Marshal()
				return pkt
			}(),
			expected: expected{
				lines:     nil,
				remainder: []byte{},
				err:       errors.New("git error: fatal: fsck error occurred"),
			},
		},
		"reference update failure": {
			input: func() []byte {
				message := "ng refs/heads/robertoonboarding failed"
				pkt, _ := protocol.PackLine(message).Marshal()
				return pkt
			}(),
			expected: expected{
				lines:     nil,
				remainder: []byte{},
				err:       errors.New("reference update failed: refs/heads/robertoonboarding failed"),
			},
		},
		"user example scenario": {
			// This simulates the exact example provided by the user
			input: func() []byte {
				message1 := "error: object 457e2462aee3d41d1a2832f10419213e10091bdc: treeNotSorted: not properly sorted\nfatal: fsck error in packed object\n"
				pkt1, _ := protocol.PackLine(message1).Marshal()
				message2 := "001dunpack index-pack failed\n"
				pkt2, _ := protocol.PackLine(message2).Marshal()
				message3 := "ng refs/heads/robertoonboarding failed\n"
				pkt3, _ := protocol.PackLine(message3).Marshal()
				pkt4 := []byte("0009000000000000")
				return append(append(append(pkt1, pkt2...), pkt3...), pkt4...)
			}(),
			expected: expected{
				lines: nil,
				remainder: func() []byte {
					message2 := "001dunpack index-pack failed\n"
					pkt2, _ := protocol.PackLine(message2).Marshal()
					message3 := "ng refs/heads/robertoonboarding failed\n"
					pkt3, _ := protocol.PackLine(message3).Marshal()
					pkt4 := []byte("0009000000000000")
					return append(append(pkt2, pkt3...), pkt4...)
				}(),
				err: errors.New("git error: error: object 457e2462aee3d41d1a2832f10419213e10091bdc: treeNotSorted: not properly sorted\nfatal: fsck error in packed object\n"),
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
				remainder: func() []byte {
					pkt3, _ := protocol.PackLine("fatal: fsck error occurred").Marshal()
					return pkt3
				}(),
				err: errors.New("error pack: hello"),
			},
		},
		"lines + git error": {
			input: func() []byte {
				pkt1, _ := protocol.PackLine("hello").Marshal()
				pkt2, _ := protocol.PackLine("error: some error").Marshal()
				return append(pkt1, pkt2...)
			}(),
			expected: expected{
				lines:     toBytesSlice("hello"),
				remainder: []byte{},
				err:       errors.New("git error: error: some error"),
			},
		},
		"lines + reference failure": {
			input: func() []byte {
				pkt1, _ := protocol.PackLine("hello").Marshal()
				pkt2, _ := protocol.PackLine("ng refs/heads/main failed").Marshal()
				return append(pkt1, pkt2...)
			}(),
			expected: expected{
				lines:     toBytesSlice("hello"),
				remainder: []byte{},
				err:       errors.New("reference update failed: refs/heads/main failed"),
			},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			lines, remainder, err := protocol.ParsePack(tc.input)
			require.Equal(t, tc.expected.lines, lines, "expected and actual lines should be equal")
			require.Equal(t, tc.expected.remainder, remainder, "expected and actual remainder should be equal")
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
