package client

import (
	"bytes"
	"context"
	"crypto"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

// deltaSizeVarint encodes n as the little-endian 7-bit varint used by a delta
// header's source/target sizes.
func deltaSizeVarint(n uint64) []byte {
	var out []byte
	for {
		b := byte(n & 0x7f)
		n >>= 7
		if n == 0 {
			return append(out, b)
		}
		out = append(out, b|0x80)
	}
}

// addOnlyDelta builds a delta payload that declares the given source size and
// reconstructs out entirely from add (insert) instructions — no copy from the
// base. Adds are split into <=127-byte chunks (the max a single add encodes).
func addOnlyDelta(sourceLen int, out []byte) []byte {
	payload := deltaSizeVarint(uint64(sourceLen))
	payload = append(payload, deltaSizeVarint(uint64(len(out)))...)
	for len(out) > 0 {
		n := len(out)
		if n > 127 {
			n = 127
		}
		payload = append(payload, byte(n))
		payload = append(payload, out[:n]...)
		out = out[n:]
	}
	return payload
}

// TestFetchResolvesDeltaChain proves the client's multi-pass delta resolution
// handles a chain where one delta's base is itself the reconstruction of
// another delta: base blob <- delta1 (produces mid) <- delta2 (produces final).
// delta2 references mid by mid's reconstructed content hash, so it can only
// resolve after delta1 has been applied — exercising resolveDeltas' iterate-
// until-no-progress loop and the streaming ApplyDelta at each hop.
func TestFetchResolvesDeltaChain(t *testing.T) {
	t.Parallel()

	base := []byte("base object contents")
	mid := []byte("intermediate delta reconstruction output")
	final := []byte("final object reconstructed through a two-hop delta chain")

	baseHash, err := protocol.Object(crypto.SHA1, protocol.ObjectTypeBlob, base)
	require.NoError(t, err)
	midHash, err := protocol.Object(crypto.SHA1, protocol.ObjectTypeBlob, mid)
	require.NoError(t, err)
	finalHash, err := protocol.Object(crypto.SHA1, protocol.ObjectTypeBlob, final)
	require.NoError(t, err)

	delta1 := addOnlyDelta(len(base), mid)  // base -> mid
	delta2 := addOnlyDelta(len(mid), final) // mid  -> final

	// v2 packfile with three objects: base blob, ref-delta(mid), ref-delta(base).
	// delta2 (whose base is mid) is placed BEFORE delta1 (which produces mid) on
	// purpose: the first resolution pass must leave delta2 pending because mid
	// does not exist yet, and only after delta1 is applied in that pass can a
	// second pass resolve delta2 — exercising resolveDeltas' iterate-until-no-
	// progress loop rather than a single in-order sweep.
	pack := []byte("PACK\x00\x00\x00\x02\x00\x00\x00\x03")
	pack = append(pack, encodeObjectHeader(protocol.ObjectTypeBlob, len(base))...)
	pack = append(pack, zlibBytes(t, base)...)
	pack = append(pack, encodeObjectHeader(protocol.ObjectTypeRefDelta, len(delta2))...)
	pack = append(pack, midHash[:]...)
	pack = append(pack, zlibBytes(t, delta2)...)
	pack = append(pack, encodeObjectHeader(protocol.ObjectTypeRefDelta, len(delta1))...)
	pack = append(pack, baseHash[:]...)
	pack = append(pack, zlibBytes(t, delta1)...)

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

	rc, err := NewRawClient(server.URL + "/repo")
	require.NoError(t, err)

	objects, err := rc.Fetch(context.Background(), FetchOptions{
		Want: []hash.Hash{finalHash},
		Done: true,
	})
	require.NoError(t, err)

	// The intermediate reconstruction must be present (delta1 applied to base)...
	midObj, ok := objects[midHash.String()]
	require.True(t, ok, "intermediate delta reconstruction should be resolved")
	require.Equal(t, mid, midObj.Data)

	// ...and the final object reconstructed from it (delta2 applied to mid).
	finalObj, ok := objects[finalHash.String()]
	require.True(t, ok, "final object at the end of the delta chain should be resolved")
	require.Equal(t, protocol.ObjectTypeBlob, finalObj.Type)
	require.Equal(t, final, finalObj.Data)
	require.Equal(t, finalHash, finalObj.Hash)
}
