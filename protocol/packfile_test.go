package protocol_test

import (
	"testing"

	"github.com/grafana/hackathon-2024-12-nanogit/protocol"
	"github.com/stretchr/testify/require"
)

func TestParsePackfile(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		input         []byte
		expectedError error
	}{
		"empty": {
			input:         []byte{},
			expectedError: protocol.ErrNoPackfileSignature,
		},
		"no signature": {
			input:         []byte("HELO"),
			expectedError: protocol.ErrNoPackfileSignature,
		},
		"truncated": {
			input:         []byte("PA"),
			expectedError: protocol.ErrNoPackfileSignature,
		},
		"empty version 2": {
			input: []byte("PACK" +
				"\x00\x00\x00\x02" +
				"\x00\x00\x00\x00"),
		},
		"empty version 3": {
			input: []byte("PACK" +
				"\x00\x00\x00\x03" +
				"\x00\x00\x00\x00"),
		},
		"invalid version": {
			input: []byte("PACK" +
				"\x00\x00\x00\x04" +
				"\x00\x00\x00\x00"),
			expectedError: protocol.ErrUnsupportedPackfileVersion,
		},
		"valid": {
			input: []byte("PACK" +
				"\x00\x00\x00\x02" +
				"\x00\x00\x00\x01"),
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := protocol.ParsePackfile(tc.input)
			require.ErrorIs(t, err, tc.expectedError)

			// We don't really have a way to validate that the
			// number of objects field was read correctly.
		})
	}
}
