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
