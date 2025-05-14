package protocol_test

import (
	"io"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/object"
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

func TestGolden(t *testing.T) {
	testcases := map[string]struct {
		expectedObjects []object.Type
	}{
		"simple.dat": {
			expectedObjects: []object.Type{
				object.TypeTree,
				object.TypeCommit,
			},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data := loadGolden(t, name)
			pr, err := protocol.ParsePackfile(data)
			require.NoError(t, err)

			for _, obj := range tc.expectedObjects {
				entry, err := pr.ReadObject()
				require.NoError(t, err)

				require.NotNil(t, entry.Object)
				require.Nil(t, entry.Trailer)

				require.Equal(t, obj, entry.Object.Type)

			}

			// There should be a trailer here.
			entry, err := pr.ReadObject()
			require.NoError(t, err)
			require.Nil(t, entry.Object)
			require.NotNil(t, entry.Trailer)

			_, err = pr.ReadObject()
			require.ErrorIs(t, err, io.EOF)
		})
	}
}

func loadGolden(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(path.Join("testdata", name))
	require.NoError(t, err)

	return data
}
