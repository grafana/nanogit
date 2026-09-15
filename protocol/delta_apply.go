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

	// Pre-allocate the result buffer to the delta's declared target size when
	// it is known (parseDelta always records it). This both avoids repeated
	// growth and, crucially, prevents a malformed delta from driving a huge
	// preallocation: the previous "ExpectedSourceLength + 64*len(Changes)"
	// estimate could be inflated far past the real output by a delta made of
	// many tiny changes. When TargetLength is 0 (e.g. a hand-built Delta) we
	// fall back to the source length as a conservative hint and skip the
	// output-length guards below.
	//
	// Callers that decode untrusted packs (see resolveSingleDelta) reject a
	// TargetLength over the decoded-object cap before calling ApplyDelta, so
	// the preallocation here is bounded by that cap.
	bounded := delta.TargetLength > 0
	initialCap := delta.ExpectedSourceLength
	if bounded {
		initialCap = delta.TargetLength
	}
	result := make([]byte, 0, initialCap)

	// Apply each delta change sequentially
	for i, change := range delta.Changes {
		var chunk []byte
		if change.DeltaData != nil {
			// Instruction type 1: Insert new data from the delta
			chunk = change.DeltaData
		} else {
			// Instruction type 2: Copy data from the base object
			// Validate that the copy operation is within bounds
			if change.SourceOffset+change.Length > uint64(len(baseData)) {
				return nil, fmt.Errorf("delta change %d: copy operation out of bounds (offset=%d, length=%d, base_size=%d)",
					i, change.SourceOffset, change.Length, len(baseData))
			}

			if change.Length == 0 {
				return nil, fmt.Errorf("delta change %d: invalid zero-length copy operation", i)
			}

			// Copy the specified range from base data
			chunk = baseData[change.SourceOffset : change.SourceOffset+change.Length]
		}

		// Reject any change that would push the output past the declared
		// target. This keeps a delta from amplifying its base beyond the
		// bound that was validated against the decoded-object cap.
		if bounded && uint64(len(result))+uint64(len(chunk)) > delta.TargetLength {
			return nil, fmt.Errorf("delta change %d: output would exceed declared target size %d bytes", i, delta.TargetLength)
		}

		result = append(result, chunk...)
	}

	// A well-formed delta reconstructs exactly TargetLength bytes; anything
	// else is a malformed (or truncated) delta.
	if bounded && uint64(len(result)) != delta.TargetLength {
		return nil, fmt.Errorf("delta output size %d does not match declared target size %d", len(result), delta.TargetLength)
	}

	return result, nil
}
