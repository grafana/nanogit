package protocol

import (
	"fmt"
)

// ApplyDelta applies delta changes to a base object's data to reconstruct the full object.
// This function processes delta instructions sequentially to produce the target object.
//
// The delta format consists of two types of instructions:
//  1. Copy from source: Copy a range of bytes from the base data
//  2. Insert new data: Insert bytes directly from the delta
//
// A parsed Delta (from parseDelta) carries its raw instruction bytes and is
// streamed one instruction at a time, so reconstruction never holds more than
// the output buffer plus a single decoded instruction — a delta cannot amplify
// a small payload into a large []DeltaChange. A hand-built Delta supplies its
// work through Changes instead, which are applied the same way.
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

	// Pre-allocate the result buffer to the delta's declared target size and
	// enforce that exact size below. This bounds the allocation to the output —
	// which parseDelta already validated against the decoded-object cap — rather
	// than to the (possibly much larger) base.
	//
	// A parsed delta always knows its target, even when it is legitimately zero,
	// so it is always bounded: TargetLength == 0 must NOT be read as "unknown",
	// or a tiny zero-target delta against a large base would preallocate the
	// base size. Only a hand-built Delta with no retained instructions may leave
	// the target unknown (zero), in which case we fall back to the source length
	// as a hint and skip the output-length guards.
	bounded := delta.instructions != nil || delta.TargetLength > 0
	initialCap := delta.TargetLength
	if !bounded {
		initialCap = delta.ExpectedSourceLength
	}
	result := make([]byte, 0, initialCap)

	// appendChange resolves one decoded change to its output bytes and appends
	// them, rejecting any change that would push the reconstruction past the
	// declared target. That bound is what keeps a delta from amplifying its base
	// beyond the cap TargetLength was validated against. The index is only used
	// for diagnostics.
	idx := 0
	appendChange := func(change DeltaChange) error {
		chunk, err := deltaChunk(idx, change, baseData)
		if err != nil {
			return err
		}
		if bounded && uint64(len(result))+uint64(len(chunk)) > delta.TargetLength {
			return &DeltaSizeError{
				Declared: delta.TargetLength,
				Actual:   uint64(len(result)) + uint64(len(chunk)),
				Reason:   fmt.Sprintf("change %d would push output past declared target", idx),
			}
		}
		result = append(result, chunk...)
		idx++
		return nil
	}

	if delta.instructions != nil {
		// Preferred path: stream straight from the raw delta payload so we never
		// hold more than one decoded instruction at a time.
		if err := walkDeltaCommands(delta.ExpectedSourceLength, delta.TargetLength, delta.instructions, appendChange); err != nil {
			return nil, err
		}
	} else {
		// Legacy path: a hand-built Delta carrying pre-decoded Changes.
		for _, change := range delta.Changes {
			if err := appendChange(change); err != nil {
				return nil, err
			}
		}
	}

	// A well-formed delta reconstructs exactly TargetLength bytes; anything
	// else is a malformed (or truncated) delta.
	if bounded && uint64(len(result)) != delta.TargetLength {
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
