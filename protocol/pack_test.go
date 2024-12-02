package protocol_test

import (
	"testing"

	"github.com/grafana/hackathon-2024-12-nanogit/protocol"
	"github.com/stretchr/testify/require"
)

func TestFormatPackets(t *testing.T) {
	testcases := map[string]struct {
		input    []protocol.Packet
		expected []byte
	}{
		"empty": {
			input:    []protocol.Packet{},
			expected: []byte("0000"), // just the flush packet
		},
		"a + LF": {
			input:    []protocol.Packet{protocol.PacketLine("a\n")},
			expected: []byte("0006a\n0000"),
		},
		"a": {
			input:    []protocol.Packet{protocol.PacketLine("a")},
			expected: []byte("0005a0000"),
		},
		"foobar + \n": {
			input:    []protocol.Packet{protocol.PacketLine("foobar\n")},
			expected: []byte("000bfoobar\n0000"),
		},
		"empty line": {
			input:    []protocol.Packet{protocol.PacketLine("")},
			expected: []byte("00040000"),
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			actual, err := protocol.FormatPackets(tc.input...)
			require.NoError(t, err, "no error expected from FormatPackets")
			require.Equal(t, tc.expected, actual, "expected and actual byte slices should be equal")
		})
	}
}

func TestParsePacket(t *testing.T) {
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
		"empty": {
			input: []byte("0000"),
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
				err:       new(protocol.ParseError),
			},
		},
		"invalid length": {
			input: []byte("000Gxxxxxxxxxxxxxxxx"),
			expected: expected{
				lines:     nil,
				remainder: []byte("000Gxxxxxxxxxxxxxxxx"),
				err:       new(protocol.ParseError),
			},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			lines, remainder, err := protocol.ParsePacket(tc.input)
			require.Equal(t, tc.expected.lines, lines, "expected and actual lines should be equal")
			require.Equal(t, tc.expected.remainder, remainder, "expected and actual remainder should be equal")
			if tc.expected.err == nil {
				require.NoError(t, err, "no error expected from ParsePacket")
			} else {
				require.Error(t, err, "error expected from ParsePacket")
				require.IsType(t, tc.expected.err, err, "expected and actual error types should be equal")
			}
		})
	}
}
