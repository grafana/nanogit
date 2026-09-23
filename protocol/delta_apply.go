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

	// Preallocate to the declared target (parseDelta validated it against the
	// decoded-object cap), not to the possibly-larger base. A target above
	// math.MaxInt cannot be a slice capacity (int is 32-bit on armv7) and would
	// panic in make, so reject it with the same ObjectTooLargeError the
	// standard-object path uses, saturating Size to int64.
	if delta.TargetLength > math.MaxInt {
		size := int64(delta.TargetLength)
		if delta.TargetLength > math.MaxInt64 {
			size = math.MaxInt64
		}
		return nil, &ObjectTooLargeError{Size: size, Limit: math.MaxInt}
	}

	result := make([]byte, 0, delta.TargetLength)

	// walkDeltaCommands guarantees the instructions fill exactly TargetLength, so
	// appendChange just appends each chunk. idx is only for diagnostics.
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

	// Defense in depth: walkDeltaCommands already guarantees this.
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
