package protocol_test

import (
	"bytes"
	"crypto"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol"
)

// These tests drive the full ParsePackfile -> ReadObject -> ApplyDelta pipeline
// over hand-crafted packfiles. They cover the tricky delta and header cases a
// real Git server will never emit (unusual varint widths, the size==0 copy
// default, malformed streams, unsupported object types), which the Gitea-backed
// integration suite in tests/ cannot reach because it only replays valid,
// git-produced deltas.

// deltaVarint encodes n as a little-endian 7-bit varint, matching the header
// and copy-size encoding that deltaHeaderSize/parseSize decode.
func deltaVarint(n uint64) []byte {
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

// resolveCraftedRefDelta builds a v2 packfile of [base blob, ref-delta(base)],
// parses it, reads both objects, and returns ApplyDelta's reconstruction of the
// delta against the base. It fails the test on any parse/read/apply error, so
// callers assert only on the reconstructed bytes.
func resolveCraftedRefDelta(t *testing.T, baseData, deltaPayload []byte) []byte {
	t.Helper()

	baseHash, err := protocol.Object(crypto.SHA1, protocol.ObjectTypeBlob, baseData)
	require.NoError(t, err)

	var pack bytes.Buffer
	pack.WriteString("PACK")
	require.NoError(t, binary.Write(&pack, binary.BigEndian, uint32(2))) // version 2
	require.NoError(t, binary.Write(&pack, binary.BigEndian, uint32(2))) // 2 objects
	pack.Write(objectHeader(protocol.ObjectTypeBlob, len(baseData)))
	pack.Write(zlibCompress(t, baseData))
	pack.Write(objectHeader(protocol.ObjectTypeRefDelta, len(deltaPayload)))
	pack.Write(baseHash[:])
	pack.Write(zlibCompress(t, deltaPayload))
	pack.Write(make([]byte, 20)) // trailer checksum (unchecked by the reader)

	pr, err := protocol.ParsePackfile(t.Context(), bytes.NewReader(pack.Bytes()))
	require.NoError(t, err)

	base, err := pr.ReadObject(t.Context())
	require.NoError(t, err)
	require.Equal(t, baseHash, base.Object.Hash)

	entry, err := pr.ReadObject(t.Context())
	require.NoError(t, err)
	require.Equal(t, protocol.ObjectTypeRefDelta, entry.Object.Type)
	require.NotNil(t, entry.Object.Delta)

	resolved, err := protocol.ApplyDelta(baseData, entry.Object.Delta)
	require.NoError(t, err)
	return resolved
}

// patternBytes returns n bytes with a recognizable, position-dependent value so
// copy ranges can be asserted against exact source offsets.
func patternBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i % 251) // 251 is prime: avoids aligning with power-of-two offsets
	}
	return b
}

func TestDeltaReconstruction_MultiByteTargetHeader(t *testing.T) {
	t.Parallel()

	// Target size 200 needs a two-byte header varint (0xC8, 0x01). The output
	// is built from two add instructions because a single add encodes at most
	// 127 literal bytes.
	want := append(bytes.Repeat([]byte("a"), 127), bytes.Repeat([]byte("b"), 73)...)

	var payload []byte
	payload = append(payload, deltaVarint(0)...)   // source size 0 (empty base)
	payload = append(payload, deltaVarint(200)...) // target size 200 -> multi-byte
	payload = append(payload, 0x7F)                // add 127 bytes
	payload = append(payload, bytes.Repeat([]byte("a"), 127)...)
	payload = append(payload, 0x49) // add 73 bytes
	payload = append(payload, bytes.Repeat([]byte("b"), 73)...)

	got := resolveCraftedRefDelta(t, []byte{}, payload)
	require.Equal(t, want, got)
}

func TestDeltaReconstruction_MultiByteCopyOffsetAndSize(t *testing.T) {
	t.Parallel()

	// Copy 40 bytes starting at source offset 260. Offset 260 needs two offset
	// bytes (0x04, 0x01); the size fits in one (0x28). The command byte selects
	// which offset/size bytes follow: 0x80 | bits0,1 (offset) | bit4 (size).
	base := patternBytes(300)
	want := base[260:300]

	var payload []byte
	payload = append(payload, deltaVarint(300)...) // source size 300
	payload = append(payload, deltaVarint(40)...)  // target size 40
	payload = append(payload, 0x93, 0x04, 0x01, 0x28)

	got := resolveCraftedRefDelta(t, base, payload)
	require.Equal(t, want, got)
}

func TestDeltaReconstruction_CopySizeZeroMeans64KiB(t *testing.T) {
	t.Parallel()

	// A copy command with no size bytes has size 0, which the format defines as
	// 0x10000 (65536). Exercise that default with a base large enough to serve
	// the copy.
	base := patternBytes(70000)
	const copyLen = 0x10000
	want := base[:copyLen]

	var payload []byte
	payload = append(payload, deltaVarint(70000)...)   // source size
	payload = append(payload, deltaVarint(copyLen)...) // target size 65536
	payload = append(payload, 0x80)                    // copy: offset 0, size 0 -> 65536

	got := resolveCraftedRefDelta(t, base, payload)
	require.Len(t, got, copyLen)
	require.Equal(t, want, got)
}

func TestDeltaReconstruction_MixedCopyAndAdd(t *testing.T) {
	t.Parallel()

	// A realistic delta shape: copy a prefix from the base, insert new bytes,
	// then copy a suffix. base = "Hello, world!" -> "Hello, DELTA world!".
	base := []byte("Hello, world!")
	want := []byte("Hello, DELTA world!")

	var payload []byte
	payload = append(payload, deltaVarint(uint64(len(base)))...) // source size 13
	payload = append(payload, deltaVarint(uint64(len(want)))...) // target size 19
	payload = append(payload, 0x90, 0x07)                        // copy base[0:7] "Hello, "
	payload = append(payload, 0x06)                              // add 6 bytes
	payload = append(payload, []byte("DELTA ")...)
	payload = append(payload, 0x91, 0x07, 0x06) // copy base[7:13] "world!"

	got := resolveCraftedRefDelta(t, base, payload)
	require.Equal(t, want, got)
}

func TestDeltaReconstruction_ManyTinyInstructionsWithinCap(t *testing.T) {
	t.Parallel()

	// The metadata-bomb shape, but legitimate and in-cap: a target built from
	// thousands of one-byte adds. It must reconstruct correctly while the reader
	// streams the instructions rather than materializing a []DeltaChange.
	const target = 20000
	want := bytes.Repeat([]byte("z"), target)

	payload := append([]byte{}, deltaVarint(0)...)
	payload = append(payload, deltaVarint(target)...)
	for i := 0; i < target; i++ {
		payload = append(payload, 0x01, 'z') // add one byte
	}

	got := resolveCraftedRefDelta(t, []byte{}, payload)
	require.Equal(t, want, got)
}

func TestReadObject_OfsDeltaUnsupported(t *testing.T) {
	t.Parallel()

	// nanogit negotiates protocol v2 and does not accept offset deltas; an
	// ofs-delta object in a pack must surface ErrUnsupportedObjectType rather
	// than being silently mishandled.
	var pack bytes.Buffer
	pack.WriteString("PACK")
	require.NoError(t, binary.Write(&pack, binary.BigEndian, uint32(2)))
	require.NoError(t, binary.Write(&pack, binary.BigEndian, uint32(1)))
	pack.Write(objectHeader(protocol.ObjectTypeOfsDelta, 8))

	pr, err := protocol.ParsePackfile(t.Context(), bytes.NewReader(pack.Bytes()))
	require.NoError(t, err)

	_, err = pr.ReadObject(t.Context())
	require.ErrorIs(t, err, protocol.ErrUnsupportedObjectType)
}
