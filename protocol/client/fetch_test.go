package client

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol/hash"
)

func TestFetch_CorruptPackfile(t *testing.T) {
	t.Parallel()

	var body bytes.Buffer
	writePkt := func(b []byte) {
		fmt.Fprintf(&body, "%04x", len(b)+4)
		body.Write(b)
	}
	writePkt([]byte("packfile\n"))
	pack := []byte("PACK" +
		"\x00\x00\x00\x02" + // version 2
		"\x00\x00\x00\x01" + // 1 object
		"\x33" + // blob, size 3
		"\xff\xff") // invalid zlib stream
	writePkt(append([]byte{1}, pack...))
	body.WriteString("0000")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(body.Bytes()); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewRawClient(server.URL + "/repo")
	require.NoError(t, err)

	want, err := hash.FromHex("0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)

	_, err = client.Fetch(t.Context(), FetchOptions{Want: []hash.Hash{want}, Done: true})
	require.ErrorContains(t, err, "reading packfile object 1")
	require.ErrorContains(t, err, "zlib: invalid header")
}
