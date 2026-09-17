package protocol

import (
	"fmt"
	"math"
)

// ApplyDelta applies delta changes to a base object's data to reconstruct the full object.
// This function processes delta instructions sequentially to produce the target object.
//
// The delta format consists of two types of instructions:
//  1. Copy from source: Copy a range of bytes from the base data
//  2. Insert new data: Insert bytes directly from the delta
//
// The Delta carries its raw instruction bytes (see parseDelta) and ApplyDelta
// streams them one at a time, so reconstruction never holds more than the
// output buffer plus a single decoded instruction — a delta cannot amplify a
// small payload into a large []DeltaChange.
//
// Parameters:
//   - baseData: The source/base object data to apply the delta to
//   - delta: The Delta object containing the changes to apply
//
// Returns:
//   - []byte: The reconstructed target object data
//   - error: Error if delta application fails (e.g., base size mismatch, invalid offsets)
//
// For more details about Git's delta format, see:
// https://git-scm.com/docs/pack-format#_deltified_representation
func ApplyDelta(baseData []byte, delta *Delta) ([]byte, error) {
	// Validate base data size matches delta's expectation
	if uint64(len(baseData)) != delta.ExpectedSourceLength {
		return nil, fmt.Errorf("base data size mismatch: got %d bytes, delta expects %d bytes",
			len(baseData), delta.ExpectedSourceLength)
	}

	// Pre-allocate the result buffer to the delta's declared target size. This
	// bounds the allocation to the output — which parseDelta already validated
	// against the decoded-object cap — rather than to the (possibly much larger)
	// base. TargetLength is authoritative even when zero.
	//
	// Slice capacities are limited to int, which is 32-bit on the supported
	// armv7 builds. A target above that (a parsed delta whose configured decoded
	// cap exceeds the platform word) would panic in make rather than surface an
	// error, so reject it explicitly first.
	if delta.TargetLength > math.MaxInt {
		return nil, &DeltaSizeError{
			Declared: delta.TargetLength,
			Actual:   delta.TargetLength,
			Reason:   fmt.Sprintf("declared target exceeds this platform's maximum allocatable size (%d bytes)", math.MaxInt),
		}
	}

	result := make([]byte, 0, delta.TargetLength)

	// walkDeltaCommands streams one decoded instruction at a time and guarantees
	// the instructions fill exactly TargetLength (it rejects short, long, and
	// overrunning streams), so appendChange simply appends each chunk. The index
	// is only used for diagnostics.
	idx := 0
	appendChange := func(change DeltaChange) error {
		chunk, err := deltaChunk(idx, change, baseData)
		if err != nil {
			return err
		}
		result = append(result, chunk...)
		idx++
		return nil
	}

	if err := walkDeltaCommands(delta.ExpectedSourceLength, delta.TargetLength, delta.instructions, appendChange); err != nil {
		return nil, err
	}

	// Defense in depth: a well-formed stream reconstructs exactly TargetLength
	// bytes. walkDeltaCommands already enforces this, so a mismatch here would be
	// an internal invariant violation rather than bad input.
	if uint64(len(result)) != delta.TargetLength {
		return nil, &DeltaSizeError{
			Declared: delta.TargetLength,
			Actual:   uint64(len(result)),
			Reason:   "reconstructed output size does not match declared target",
		}
	}

	return result, nil
}

// deltaChunk resolves a single decoded change to the bytes it contributes to
// the reconstruction: literal data carried in the delta, or a bounds-checked
// range copied from the base object. idx is used only for diagnostics.
func deltaChunk(idx int, change DeltaChange, baseData []byte) ([]byte, error) {
	// Instruction type 1: insert new data carried in the delta.
	if change.DeltaData != nil {
		return change.DeltaData, nil
	}

	// Instruction type 2: copy a range from the base object.
	if change.SourceOffset+change.Length > uint64(len(baseData)) {
		return nil, fmt.Errorf("delta change %d: copy operation out of bounds (offset=%d, length=%d, base_size=%d)",
			idx, change.SourceOffset, change.Length, len(baseData))
	}
	if change.Length == 0 {
		return nil, fmt.Errorf("delta change %d: invalid zero-length copy operation", idx)
	}
	return baseData[change.SourceOffset : change.SourceOffset+change.Length], nil
}
