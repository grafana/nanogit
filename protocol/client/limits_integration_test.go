package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol/hash"
)

// hugePktLineBody returns a body whose first pkt-line declares 65280 bytes
// of data but is followed by enough payload that the limit reader trips
// inside the readPacketData ReadFull call. The exact payload bytes don't
// matter; only that there are far more than the cap allows.
func hugePktLineBody() []byte {
	// Length 0xFF00 (65280) is below MaxPktLineSize (65520), so the
	// per-packet validation accepts it. We then write 65276 payload bytes
	// (length minus the 4-byte header) so a well-behaved client reads
	// the whole packet — tests with a low cap will trip on the way.
	const declaredLen = 0xFF00
	var b strings.Builder
	b.WriteString("ff00")
	b.WriteString(strings.Repeat("a", declaredLen-4))
	return []byte(b.String())
}

func TestLsRefsHonorsRefsMetadataLimit(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(hugePktLineBody())
	}))
	t.Cleanup(server.Close)

	rc, err := NewRawClient(server.URL+"/repo",
		options.WithLimits(options.Limits{RefsMetadata: 128}))
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
