package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyDelta(t *testing.T) {
	t.Run("simple insert operation", func(t *testing.T) {
		baseData := []byte("Hello")
		delta := &Delta{
			ExpectedSourceLength: 5,
			Changes: []DeltaChange{
				{DeltaData: []byte("World")},
			},
		}

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "World", string(result))
	})

	t.Run("simple copy operation", func(t *testing.T) {
		baseData := []byte("Hello World")
		delta := &Delta{
			ExpectedSourceLength: 11,
			Changes: []DeltaChange{
				{
					SourceOffset: 0,
					Length:       5,
				},
			},
		}

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "Hello", string(result))
	})

	t.Run("mixed copy and insert operations", func(t *testing.T) {
		baseData := []byte("Hello World")
		delta := &Delta{
			ExpectedSourceLength: 11,
			Changes: []DeltaChange{
				// Copy "Hello"
				{SourceOffset: 0, Length: 5},
				// Insert ", "
				{DeltaData: []byte(", ")},
				// Copy "World"
				{SourceOffset: 6, Length: 5},
				// Insert "!"
				{DeltaData: []byte("!")},
			},
		}

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "Hello, World!", string(result))
	})

	t.Run("copy from multiple locations", func(t *testing.T) {
		baseData := []byte("ABCDEFGH")
		delta := &Delta{
			ExpectedSourceLength: 8,
			Changes: []DeltaChange{
				{SourceOffset: 7, Length: 1}, // H
				{SourceOffset: 4, Length: 1}, // E
				{SourceOffset: 2, Length: 1}, // C
				{SourceOffset: 1, Length: 1}, // B
				{SourceOffset: 0, Length: 1}, // A
			},
		}

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "HECBA", string(result))
	})

	t.Run("empty base data", func(t *testing.T) {
		baseData := []byte("")
		delta := &Delta{
			ExpectedSourceLength: 0,
			Changes: []DeltaChange{
				{DeltaData: []byte("New content")},
			},
		}

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "New content", string(result))
	})

	t.Run("empty delta (no changes)", func(t *testing.T) {
		baseData := []byte("Hello")
		delta := &Delta{
			ExpectedSourceLength: 5,
			Changes:              []DeltaChange{},
		}

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

		delta := &Delta{
			ExpectedSourceLength: 10000,
			Changes: []DeltaChange{
				// Copy first 5000 bytes
				{SourceOffset: 0, Length: 5000},
				// Insert some new data
				{DeltaData: []byte("INSERTED")},
				// Copy last 5000 bytes
				{SourceOffset: 5000, Length: 5000},
			},
		}

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Len(t, result, 10008) // 10000 + 8 inserted bytes

		// Verify the structure
		require.Equal(t, baseData[:5000], result[:5000])
		require.Equal(t, []byte("INSERTED"), result[5000:5008])
		require.Equal(t, baseData[5000:], result[5008:])
	})

	t.Run("error: base size mismatch", func(t *testing.T) {
		baseData := []byte("Hello")
		delta := &Delta{
			ExpectedSourceLength: 10, // Expects 10 but base is 5
			Changes: []DeltaChange{
				{DeltaData: []byte("World")},
			},
		}

		_, err := ApplyDelta(baseData, delta)
		require.Error(t, err)
		require.Contains(t, err.Error(), "base data size mismatch")
	})

	t.Run("error: copy out of bounds", func(t *testing.T) {
		baseData := []byte("Hello")
		delta := &Delta{
			ExpectedSourceLength: 5,
			Changes: []DeltaChange{
				{SourceOffset: 3, Length: 10}, // Tries to copy beyond end
			},
		}

		_, err := ApplyDelta(baseData, delta)
		require.Error(t, err)
		require.Contains(t, err.Error(), "out of bounds")
	})

	t.Run("error: invalid offset", func(t *testing.T) {
		baseData := []byte("Hello")
		delta := &Delta{
			ExpectedSourceLength: 5,
			Changes: []DeltaChange{
				{SourceOffset: 100, Length: 1}, // Offset way beyond base
			},
		}

		_, err := ApplyDelta(baseData, delta)
		require.Error(t, err)
		require.Contains(t, err.Error(), "out of bounds")
	})

	t.Run("error: zero length copy", func(t *testing.T) {
		baseData := []byte("Hello")
		delta := &Delta{
			ExpectedSourceLength: 5,
			Changes: []DeltaChange{
				{SourceOffset: 0, Length: 0}, // Zero-length copy
			},
		}

		_, err := ApplyDelta(baseData, delta)
		require.Error(t, err)
		require.Contains(t, err.Error(), "zero-length copy")
	})

	t.Run("realistic git delta scenario - modify text", func(t *testing.T) {
		// Simulate a realistic scenario: a text file with a line replaced
		baseData := []byte("Line 1\nLine 2\nLine 3\nLine 4\n")

		// Delta that replaces "Line 2" with "Modified Line 2"
		delta := &Delta{
			ExpectedSourceLength: uint64(len(baseData)),
			Changes: []DeltaChange{
				// Copy "Line 1\n"
				{SourceOffset: 0, Length: 7},
				// Insert modified line
				{DeltaData: []byte("Modified Line 2\n")},
				// Copy remaining lines starting from "Line 3"
				{SourceOffset: 14, Length: uint64(len(baseData) - 14)},
			},
		}

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)

		expected := "Line 1\nModified Line 2\nLine 3\nLine 4\n"
		require.Equal(t, expected, string(result))
	})

	t.Run("multiple small inserts and copies", func(t *testing.T) {
		baseData := []byte("The quick brown fox jumps over the lazy dog")
		delta := &Delta{
			ExpectedSourceLength: uint64(len(baseData)),
			Changes: []DeltaChange{
				{SourceOffset: 0, Length: 4},   // "The "
				{DeltaData: []byte("very ")},   // Insert "very "
				{SourceOffset: 4, Length: 6},   // "quick "
				{DeltaData: []byte("and ")},    // Insert "and "
				{SourceOffset: 10, Length: 33}, // rest of string (43 - 10 = 33)
			},
		}

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "The very quick and brown fox jumps over the lazy dog", string(result))
	})

	t.Run("delta duplicates content", func(t *testing.T) {
		baseData := []byte("AB")
		delta := &Delta{
			ExpectedSourceLength: 2,
			Changes: []DeltaChange{
				{SourceOffset: 0, Length: 2}, // AB
				{SourceOffset: 0, Length: 2}, // AB again
				{SourceOffset: 0, Length: 2}, // AB again
			},
		}

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		require.Equal(t, "ABABAB", string(result))
	})

	t.Run("binary data", func(t *testing.T) {
		baseData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
		delta := &Delta{
			ExpectedSourceLength: 7,
			Changes: []DeltaChange{
				{SourceOffset: 4, Length: 3}, // 0xFF, 0xFE, 0xFD
				{DeltaData: []byte{0xAA, 0xBB}},
				{SourceOffset: 0, Length: 4}, // 0x00, 0x01, 0x02, 0x03
			},
		}

		result, err := ApplyDelta(baseData, delta)
		require.NoError(t, err)
		expected := []byte{0xFF, 0xFE, 0xFD, 0xAA, 0xBB, 0x00, 0x01, 0x02, 0x03}
		require.Equal(t, expected, result)
	})

	// When the delta records a target size (parseDelta always does), ApplyDelta
	// enforces it: reconstruction can neither overrun nor fall short of the
	// declared size, and the buffer is preallocated to it rather than to an
	// inflatable estimate. These guard the delta path against decompression
	// bombs (a small delta amplifying its base past the decoded-object cap).
	t.Run("error: output exceeds declared target", func(t *testing.T) {
		delta := &Delta{
			ExpectedSourceLength: 5,
			TargetLength:         3, // declares 3 bytes...
			Changes: []DeltaChange{
				{DeltaData: []byte("world")}, // ...but a change produces 5
			},
		}
		_, err := ApplyDelta([]byte("hello"), delta)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceed declared target")
	})

	t.Run("error: output shorter than declared target", func(t *testing.T) {
		delta := &Delta{
			ExpectedSourceLength: 5,
			TargetLength:         10, // declares 10 bytes...
			Changes: []DeltaChange{
				{DeltaData: []byte("ab")}, // ...but only 2 are produced
			},
		}
		_, err := ApplyDelta([]byte("hello"), delta)
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not match declared target")
	})

	t.Run("preallocation is bounded to the declared target", func(t *testing.T) {
		// A delta made of many tiny changes must not drive a preallocation
		// larger than the declared output. With TargetLength set, capacity is
		// exactly the target no matter how many changes there are.
		const target = 1000
		changes := make([]DeltaChange, target)
		for i := range changes {
			changes[i] = DeltaChange{DeltaData: []byte("y")}
		}
		delta := &Delta{
			ExpectedSourceLength: 1,
			TargetLength:         target,
			Changes:              changes,
		}
		result, err := ApplyDelta([]byte("x"), delta)
		require.NoError(t, err)
		require.Len(t, result, target)
		require.Equal(t, target, cap(result),
			"buffer must be preallocated to the target, not an inflatable estimate")
	})
}

func TestParseDelta_DropsCommandThatWouldOverrunTarget(t *testing.T) {
	t.Parallel()

	// Header: source size 10, target size 10. Then a 7-byte add (fits) and a
	// 5-byte add that would consume more than the 3 bytes still remaining in
	// the target. The overrunning command must be dropped rather than
	// underflowing the unsigned remaining-size counter.
	payload := []byte{
		0x0A,                                    // source size 10
		0x0A,                                    // target size 10
		0x07, 'a', 'b', 'c', 'd', 'e', 'f', 'g', // add 7 bytes
		0x05, 'h', 'i', 'j', 'k', 'l', // add 5 bytes -> would overrun, dropped
	}

	delta, err := parseDelta("parent", payload, 0)
	require.NoError(t, err)
	require.EqualValues(t, 10, delta.TargetLength)
	require.Len(t, delta.Changes, 1, "the overrunning command must be dropped, not underflow the counter")
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
