package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

// hugePktLineBody returns a body whose first pkt-line declares a total
// length of 65280 bytes (0xFF00) — i.e. 65276 payload bytes plus the
// 4-byte length header. We then write all 65276 payload bytes so a
// well-behaved client reads the whole packet; tests with a low cap will
// trip inside the readPacketData ReadFull call before the payload ends.
func hugePktLineBody() []byte {
	// 0xFF00 is below MaxPktLineSize (65520), so the per-packet
	// validation accepts it. The payload is the declared length minus
	// the 4-byte header.
	const declaredLen = 0xFF00
	var b strings.Builder
	b.WriteString("ff00")
	b.WriteString(strings.Repeat("a", declaredLen-4))
	return []byte(b.String())
}

func TestLsRefsHonorsRefsMetadataMaxBytesLimit(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(hugePktLineBody())
	}))
	t.Cleanup(server.Close)

	rc, err := NewRawClient(server.URL+"/repo",
		options.WithLimits(options.Limits{RefsMetadataMaxBytes: 128}))
	require.NoError(t, err)

	_, err = rc.LsRefs(context.Background(), LsRefsOptions{})
	require.Error(t, err)

	var tooLarge *ErrResponseTooLarge
	require.True(t, errors.As(err, &tooLarge), "expected *ErrResponseTooLarge, got %T: %v", err, err)
	require.Equal(t, "ls-refs", tooLarge.Op)
	require.Equal(t, int64(128), tooLarge.Limit)
}

func TestFetchHonorsSingleObjectLimit(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(hugePktLineBody())
	}))
	t.Cleanup(server.Close)

	rc, err := NewRawClient(server.URL + "/repo")
	require.NoError(t, err)

	wantHash, err := hash.FromHex("0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)

	_, err = rc.Fetch(context.Background(), FetchOptions{
		Want:             []hash.Hash{wantHash},
		MaxResponseBytes: 256,
	})
	require.Error(t, err)

	var tooLarge *ErrResponseTooLarge
	require.True(t, errors.As(err, &tooLarge), "expected *ErrResponseTooLarge, got %T: %v", err, err)
	require.Equal(t, "fetch", tooLarge.Op)
	require.Equal(t, int64(256), tooLarge.Limit)
}

// encodeObjectHeader encodes a packfile object's type + size varint the same
// way a real pack does, so a test object can declare an arbitrary decoded size.
func encodeObjectHeader(objType protocol.ObjectType, size int) []byte {
	b := byte(objType)<<4 | byte(size&0xF)
	size >>= 4
	var out []byte
	for size > 0 {
		out = append(out, b|0x80)
		b = byte(size & 0x7F)
		size >>= 7
	}
	return append(out, b)
}

// TestFetchHonorsMaxObjectDecodedBytes proves the decoded-object cap threads
// from options.Limits through Fetch to the packfile parser and rejects an
// object whose declared decoded size exceeds the cap — before it is inflated
// or allocated, so the compressed payload need not even be valid.
func TestFetchHonorsMaxObjectDecodedBytes(t *testing.T) {
	t.Parallel()

	const declaredSize = 1 << 20 // 1 MiB declared decoded size, over the cap below

	pack := []byte("PACK\x00\x00\x00\x02\x00\x00\x00\x01") // v2, 1 object
	pack = append(pack, encodeObjectHeader(protocol.ObjectTypeBlob, declaredSize)...)

	var body bytes.Buffer
	writePkt := func(b []byte) {
		fmt.Fprintf(&body, "%04x", len(b)+4)
		body.Write(b)
	}
	writePkt([]byte("packfile\n"))
	writePkt(append([]byte{1}, pack...)) // sideband channel 1 = pack data
	body.WriteString("0000")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body.Bytes())
	}))
	t.Cleanup(server.Close)

	rc, err := NewRawClient(server.URL+"/repo",
		options.WithLimits(options.Limits{MaxObjectDecodedBytes: 4096}))
	require.NoError(t, err)

	wantHash, err := hash.FromHex("0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)

	_, err = rc.Fetch(context.Background(), FetchOptions{Want: []hash.Hash{wantHash}, Done: true})
	require.Error(t, err)

	var tooLarge *protocol.ObjectTooLargeError
	require.True(t, errors.As(err, &tooLarge), "expected *protocol.ObjectTooLargeError, got %T: %v", err, err)
	require.Equal(t, declaredSize, tooLarge.Size)
	require.Equal(t, int64(4096), tooLarge.Limit)
	// The sentinel is preserved through the wrap for errors.Is callers.
	require.ErrorIs(t, err, protocol.ErrObjectTooLarge)
}

func TestFetchUnboundedByDefault(t *testing.T) {
	t.Parallel()

	// Sanity check: without WithLimits and without MaxResponseBytes set on
	// the request, the limit reader is a no-op and we reach the parser as
	// before. The parser will fail because the body isn't a valid Git
	// fetch response — but the failure must NOT be *ErrResponseTooLarge.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(hugePktLineBody())
	}))
	t.Cleanup(server.Close)

	rc, err := NewRawClient(server.URL + "/repo")
	require.NoError(t, err)

	wantHash, err := hash.FromHex("0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)

	_, err = rc.Fetch(context.Background(), FetchOptions{Want: []hash.Hash{wantHash}})
	if err != nil {
		var tooLarge *ErrResponseTooLarge
		require.False(t, errors.As(err, &tooLarge), "no limit was set, but got ErrResponseTooLarge: %v", err)
	}
}
