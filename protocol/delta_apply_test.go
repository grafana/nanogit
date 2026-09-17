package protocol

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// encCopy encodes a copy instruction: the command byte's low nibble selects
// which offset bytes follow and bits 4-6 which size bytes follow, each in
// ascending significance. Only non-zero bytes are emitted, matching git.
func encCopy(offset, size uint64) []byte {
	cmd := byte(0x80)
	var body []byte
	for i := 0; i < 4; i++ {
		if b := byte(offset >> (8 * uint(i))); b != 0 {
			cmd |= 1 << uint(i)
			body = append(body, b)
		}
	}
	for i := 0; i < 3; i++ {
		if b := byte(size >> (8 * uint(i))); b != 0 {
			cmd |= 1 << uint(4+i)
			body = append(body, b)
		}
	}
	return append([]byte{cmd}, body...)
}

// encAdd encodes one or more add (insert) instructions for data, splitting into
// the 127-byte maximum a single add command can carry.
func encAdd(data []byte) []byte {
	var out []byte
	for len(data) > 0 {
		n := len(data)
		if n > 127 {
			n = 127
		}
		out = append(out, byte(n))
		out = append(out, data[:n]...)
		data = data[n:]
	}
	return out
}

// deltaFromChanges builds a *Delta whose instruction stream encodes the given
// changes, exactly as parseDelta would produce it (TargetLength is the sum of
// the change lengths). It sets the unexported instructions field directly so
// ApplyDelta exercises its real streaming path without the 4-byte minimum
// parseDelta imposes on a full payload.
func deltaFromChanges(source uint64, changes []DeltaChange) *Delta {
	instr := []byte{}
	var target uint64
	for _, c := range changes {
		if c.DeltaData != nil {
			instr = append(instr, encAdd(c.DeltaData)...)
			target += uint64(len(c.DeltaData))
		} else {
			instr = append(instr, encCopy(c.SourceOffset, c.Length)...)
			target += c.Length
		}
	}
	return &Delta{ExpectedSourceLength: source, TargetLength: target, instructions: instr}
}

func TestApplyDelta(t *testing.T) {
	t.Run("simple insert operation", func(t *testing.T) {
		baseData := []byte("Hello")
		delta := deltaFromChanges(5, []DeltaChange{
			{DeltaData: []byte("World")},
		})

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "World", string(result))
	})

	t.Run("simple copy operation", func(t *testing.T) {
		baseData := []byte("Hello World")
		delta := deltaFromChanges(11, []DeltaChange{
			{SourceOffset: 0, Length: 5},
		})

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "Hello", string(result))
	})

	t.Run("mixed copy and insert operations", func(t *testing.T) {
		baseData := []byte("Hello World")
		delta := deltaFromChanges(11, []DeltaChange{
			{SourceOffset: 0, Length: 5}, // Copy "Hello"
			{DeltaData: []byte(", ")},    // Insert ", "
			{SourceOffset: 6, Length: 5}, // Copy "World"
			{DeltaData: []byte("!")},     // Insert "!"
		})

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "Hello, World!", string(result))
	})

	t.Run("copy from multiple locations", func(t *testing.T) {
		baseData := []byte("ABCDEFGH")
		delta := deltaFromChanges(8, []DeltaChange{
			{SourceOffset: 7, Length: 1}, // H
			{SourceOffset: 4, Length: 1}, // E
			{SourceOffset: 2, Length: 1}, // C
			{SourceOffset: 1, Length: 1}, // B
			{SourceOffset: 0, Length: 1}, // A
		})

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "HECBA", string(result))
	})

	t.Run("empty base data", func(t *testing.T) {
		baseData := []byte("")
		delta := deltaFromChanges(0, []DeltaChange{
			{DeltaData: []byte("New content")},
		})

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "New content", string(result))
	})

	t.Run("empty delta (no changes)", func(t *testing.T) {
		baseData := []byte("Hello")
		delta := deltaFromChanges(5, nil)

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("large copy operation", func(t *testing.T) {
		// Create a 10KB base
		baseData := make([]byte, 10000)
		for i := range baseData {
			baseData[i] = byte(i % 256)
		}

		delta := deltaFromChanges(10000, []DeltaChange{
			{SourceOffset: 0, Length: 5000},    // Copy first 5000 bytes
			{DeltaData: []byte("INSERTED")},    // Insert some new data
			{SourceOffset: 5000, Length: 5000}, // Copy last 5000 bytes
		})

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Len(t, result, 10008) // 10000 + 8 inserted bytes

		// Verify the structure
		require.Equal(t, baseData[:5000], result[:5000])
		require.Equal(t, []byte("INSERTED"), result[5000:5008])
		require.Equal(t, baseData[5000:], result[5008:])
	})

	t.Run("error: base size mismatch", func(t *testing.T) {
		baseData := []byte("Hello")                  // 5 bytes...
		delta := deltaFromChanges(10, []DeltaChange{ // ...but the delta expects 10
			{DeltaData: []byte("World")},
		})

		_, err := ApplyDelta(baseData, delta)
		require.Error(t, err)
		require.Contains(t, err.Error(), "base data size mismatch")
	})

	t.Run("realistic git delta scenario - modify text", func(t *testing.T) {
		// Simulate a realistic scenario: a text file with a line replaced
		baseData := []byte("Line 1\nLine 2\nLine 3\nLine 4\n")

		// Delta that replaces "Line 2" with "Modified Line 2"
		delta := deltaFromChanges(uint64(len(baseData)), []DeltaChange{
			{SourceOffset: 0, Length: 7},                           // Copy "Line 1\n"
			{DeltaData: []byte("Modified Line 2\n")},               // Insert modified line
			{SourceOffset: 14, Length: uint64(len(baseData) - 14)}, // Copy from "Line 3"
		})

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)

		expected := "Line 1\nModified Line 2\nLine 3\nLine 4\n"
		require.Equal(t, expected, string(result))
	})

	t.Run("multiple small inserts and copies", func(t *testing.T) {
		baseData := []byte("The quick brown fox jumps over the lazy dog")
		delta := deltaFromChanges(uint64(len(baseData)), []DeltaChange{
			{SourceOffset: 0, Length: 4},   // "The "
			{DeltaData: []byte("very ")},   // Insert "very "
			{SourceOffset: 4, Length: 6},   // "quick "
			{DeltaData: []byte("and ")},    // Insert "and "
			{SourceOffset: 10, Length: 33}, // rest of string (43 - 10 = 33)
		})

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "The very quick and brown fox jumps over the lazy dog", string(result))
	})

	t.Run("delta duplicates content", func(t *testing.T) {
		baseData := []byte("AB")
		delta := deltaFromChanges(2, []DeltaChange{
			{SourceOffset: 0, Length: 2}, // AB
			{SourceOffset: 0, Length: 2}, // AB again
			{SourceOffset: 0, Length: 2}, // AB again
		})

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "ABABAB", string(result))
	})

	t.Run("binary data", func(t *testing.T) {
		baseData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
		delta := deltaFromChanges(7, []DeltaChange{
			{SourceOffset: 4, Length: 3}, // 0xFF, 0xFE, 0xFD
			{DeltaData: []byte{0xAA, 0xBB}},
			{SourceOffset: 0, Length: 4}, // 0x00, 0x01, 0x02, 0x03
		})

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		expected := []byte{0xFF, 0xFE, 0xFD, 0xAA, 0xBB, 0x00, 0x01, 0x02, 0x03}
		require.Equal(t, expected, result)
	})

	t.Run("preallocation is bounded to the declared target", func(t *testing.T) {
		// A delta made of many tiny changes must not drive a preallocation
		// larger than the declared output: the buffer is preallocated to
		// TargetLength no matter how many instructions there are.
		const target = 1000
		changes := make([]DeltaChange, target)
		for i := range changes {
			changes[i] = DeltaChange{DeltaData: []byte("y")}
		}
		delta := deltaFromChanges(1, changes)

		result, err := ApplyDelta([]byte("x"), delta)
		require.NoError(t, err)
		require.Len(t, result, target)
		require.Equal(t, target, cap(result),
			"buffer must be preallocated to the target, not an inflatable estimate")
	})
}

func TestParseDelta_StreamsInsteadOfMaterializingChanges(t *testing.T) {
	t.Parallel()

	// A delta made of many tiny add instructions, all within a small target and
	// a small payload. The metadata-bomb concern is that decoding these into a
	// []DeltaChange up front costs ~40 bytes each; parseDelta must instead retain
	// only the raw instruction bytes, so its footprint is independent of the
	// instruction count.
	const target = 4096
	source := []byte("x")
	payload := []byte{
		byte(len(source)), // source size 1
		0x80, 0x20,        // target size 4096 (7-bit little-endian varint)
	}
	for i := 0; i < target; i++ {
		payload = append(payload, 0x01, 'a') // add one byte
	}

	delta, err := parseDelta("parent", payload, 0)
	require.NoError(t, err)
	require.EqualValues(t, target, delta.TargetLength)
	require.NotNil(t, delta.instructions,
		"a parsed delta must retain its raw instruction bytes for streaming apply")

	// And the retained instructions must still reconstruct the object exactly.
	result, err := ApplyDelta(source, delta)
	require.NoError(t, err)
	require.Len(t, result, target)
	require.Equal(t, []byte(strings.Repeat("a", target)), result)
}

func TestApplyDelta_ParsedZeroTargetDoesNotAllocateBase(t *testing.T) {
	t.Parallel()

	// A parsed delta that legitimately declares a zero-byte target against a
	// non-trivial base must reconstruct an empty object WITHOUT preallocating
	// the base size. Regression guard: reading TargetLength==0 as "unknown"
	// used to fall back to ExpectedSourceLength, turning repeated tiny deltas
	// against a large base into an allocation/GC DoS.
	const baseLen = 16384
	base := bytes.Repeat([]byte("x"), baseLen)

	// Header only: source size 16384 (0x80,0x80,0x01), target size 0 (0x00).
	// The source varint is three bytes, so the header alone meets the 4-byte
	// minimum without any (now-rejected) trailing instruction padding.
	payload := []byte{0x80, 0x80, 0x01, 0x00}

	delta, err := parseDelta("parent", payload, 0)
	require.NoError(t, err)
	require.EqualValues(t, baseLen, delta.ExpectedSourceLength)
	require.EqualValues(t, 0, delta.TargetLength)

	got, err := ApplyDelta(base, delta)
	require.NoError(t, err)
	require.Empty(t, got)
	require.Equal(t, 0, cap(got),
		"a parsed zero-target delta must not preallocate the base size")
}

func TestApplyDelta_TargetExceedingPlatformMaxErrorsNotPanics(t *testing.T) {
	t.Parallel()

	// A declared target above the platform's int range must surface an error
	// rather than panic inside make (slice capacities are int, 32-bit on armv7).
	// math.MaxUint64 exceeds MaxInt on every supported platform.
	delta := &Delta{
		ExpectedSourceLength: 0,
		TargetLength:         math.MaxUint64,
	}

	require.NotPanics(t, func() {
		_, err := ApplyDelta([]byte{}, delta)
		require.Error(t, err)
		var sizeErr *DeltaSizeError
		require.ErrorAs(t, err, &sizeErr)
		require.Contains(t, sizeErr.Reason, "platform")
	})
}

func TestDelta_DecodeChanges(t *testing.T) {
	t.Parallel()

	t.Run("decodes a parsed delta's instructions on demand", func(t *testing.T) {
		t.Parallel()

		// base "Hello, world!" -> copy "Hello, ", add "DELTA ", copy "world!".
		base := []byte("Hello, world!")
		payload := []byte{
			0x0D,       // source size 13
			0x13,       // target size 19
			0x90, 0x07, // copy base[0:7]
			0x06, 'D', 'E', 'L', 'T', 'A', ' ', // add 6 bytes
			0x91, 0x07, 0x06, // copy base[7:13]
		}

		delta, err := parseDelta("parent", payload, 0)
		require.NoError(t, err)

		changes, err := delta.DecodeChanges()
		require.NoError(t, err)
		require.Len(t, changes, 3)
		require.Equal(t, DeltaChange{SourceOffset: 0, Length: 7}, changes[0])
		require.Equal(t, []byte("DELTA "), changes[2-1].DeltaData)
		require.Equal(t, DeltaChange{SourceOffset: 7, Length: 6}, changes[2])

		// Decoded changes reconstruct the same object ApplyDelta produces.
		resolved, err := ApplyDelta(base, delta)
		require.NoError(t, err)
		require.Equal(t, []byte("Hello, DELTA world!"), resolved)
	})

	t.Run("returns nil for a delta with no instruction stream", func(t *testing.T) {
		t.Parallel()

		got, err := (&Delta{}).DecodeChanges()
		require.NoError(t, err)
		require.Nil(t, got)
	})
}

func TestParseDelta_RejectsCommandThatWouldOverrunTarget(t *testing.T) {
	t.Parallel()

	// Header: source size 10, target size 10. Then a 7-byte add (fits) and a
	// 5-byte add that would consume more than the 3 bytes still remaining in
	// the target. The overrunning command is unambiguously malformed (a
	// well-formed delta's instructions sum to exactly the target), so parseDelta
	// must reject it up front rather than silently dropping it and leaning on
	// ApplyDelta's downstream length check — and rather than underflowing the
	// unsigned remaining-size counter.
	payload := []byte{
		0x0A,                                    // source size 10
		0x0A,                                    // target size 10
		0x07, 'a', 'b', 'c', 'd', 'e', 'f', 'g', // add 7 bytes
		0x05, 'h', 'i', 'j', 'k', 'l', // add 5 bytes -> would overrun, rejected
	}

	delta, err := parseDelta("parent", payload, 0)
	require.Nil(t, delta)
	require.Error(t, err)
	var sizeErr *DeltaSizeError
	require.ErrorAs(t, err, &sizeErr)
	require.EqualValues(t, 10, sizeErr.Declared)
	// 7 bytes already consumed + the 5-byte command = 12, past the 10 target.
	require.EqualValues(t, 12, sizeErr.Actual)
	require.ErrorIs(t, err, ErrDeltaSize)
}

func TestParseDelta_RejectsOutOfBoundsInstruction(t *testing.T) {
	t.Parallel()

	// A copy whose size (20) exceeds the declared target (10) is malformed;
	// parseDeltaCommand reports zero consumption for it. parseDelta must reject
	// it up front rather than tolerating it and leaving a short delta — which
	// would push the full cap-sized allocation and rejection down into
	// ApplyDelta, where resolveDeltas retries it on every pass (a DoS vector).
	payload := []byte{
		0x04,       // source size 4
		0x0A,       // target size 10
		0x90, 0x14, // copy offset 0, size 20 (> target) -> zero consumption
	}

	delta, err := parseDelta("parent", payload, 0)
	require.Nil(t, delta)
	require.Error(t, err)
	require.Contains(t, err.Error(), "malformed delta instruction")
}

func TestParseDelta_RejectsTrailingCommands(t *testing.T) {
	t.Parallel()

	// A target of 1 followed by two one-byte adds: the first add fills the
	// target, and Git requires the stream to end there. The trailing second add
	// must be rejected, not silently ignored (which would reconstruct only the
	// first byte).
	payload := []byte{
		0x00,      // source size 0
		0x01,      // target size 1
		0x01, 'a', // add 1 byte -> fills the target
		0x01, 'b', // trailing add -> malformed
	}

	delta, err := parseDelta("parent", payload, 0)
	require.Nil(t, delta)
	require.Error(t, err)
	require.Contains(t, err.Error(), "trailing bytes")
}

func TestParseDelta_RejectsOversizedTargetBeforeCommandLoop(t *testing.T) {
	t.Parallel()

	// Header: source size 5, target size 1 MiB, then a long run of 1-byte add
	// commands. With a small cap, parseDelta must reject on the declared target
	// immediately — before materializing any DeltaChange for those commands.
	payload := []byte{
		0x05,             // source size 5
		0x80, 0x80, 0x40, // target size 1<<20 (1 MiB)
	}
	for i := 0; i < 100; i++ {
		payload = append(payload, 0x01, 'a') // add 1 byte
	}

	const cap = 4096
	delta, err := parseDelta("parent", payload, cap)
	require.Nil(t, delta)
	require.ErrorIs(t, err, ErrObjectTooLarge)

	var tooLarge *ObjectTooLargeError
	require.ErrorAs(t, err, &tooLarge)
	require.Equal(t, int64(1<<20), tooLarge.Size)
	require.Equal(t, int64(cap), tooLarge.Limit)

	// A cap of 0 disables the oversized-target check (used by direct parseDelta
	// callers/tests). The payload here is truncated relative to its huge
	// declared target, so parsing still fails — but not with ErrObjectTooLarge.
	_, err = parseDelta("parent", payload, 0)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrObjectTooLarge)
}
