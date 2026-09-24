package protocol

import (
	"fmt"
	"math"
)

// FileStatus represents the status of a file in a commit
// Git file status codes are documented in the Git documentation:
// https://git-scm.com/docs/git-status#_short_format
// https://git-scm.com/docs/git-diff#_combined_diff_format
type FileStatus string

// Includes only file statuses we have implemented:
// - "M" (Modified): A file was modified.
// - "A" (Added): A file was added.
// - "D" (Deleted): A file was deleted.
// - "T" (Type Changed): A file's type changed (e.g., from regular file to symlink).
// - "R" (Renamed): A file was renamed (requires rename detection to be enabled).
//
// Other Git status codes, such as "C" (Copied), are not currently supported.
const (
	// FileStatusModified indicates a file was modified
	FileStatusModified FileStatus = "M"
	// FileStatusAdded indicates a file was added
	FileStatusAdded FileStatus = "A"
	// FileStatusDeleted indicates a file was deleted
	FileStatusDeleted FileStatus = "D"
	// FileStatusTypeChanged indicates a file's type changed (e.g., from regular file to symlink)
	FileStatusTypeChanged FileStatus = "T"
	// FileStatusRenamed indicates a file was renamed
	FileStatusRenamed FileStatus = "R"
)

var (
	errMissingOffsetByte = strError("missing offset byte")
	errMissingSizeByte   = strError("missing size byte")
	// errMalformedDeltaInstruction is the sentinel wrapped by the concrete
	// errors parseDeltaCommand returns for an instruction that cannot consume
	// any target bytes; match it with errors.Is.
	errMalformedDeltaInstruction = strError("malformed delta instruction")
)

// ErrDeltaSize is the sentinel wrapped by *DeltaSizeError; match it with
// errors.Is. It means the delta is internally inconsistent (malformed or
// truncated), as opposed to ErrObjectTooLarge, which means the declared target
// exceeds the configured cap.
var ErrDeltaSize = strError("delta does not reconstruct its declared target size")

// DeltaSizeError reports that a delta's instructions do not reconstruct an
// object of its declared target size. It wraps ErrDeltaSize.
type DeltaSizeError struct {
	// Declared is the target size, in bytes, from the delta header.
	Declared uint64
	// Actual is the reconstructed output size, in bytes, at the point the
	// mismatch was detected.
	Actual uint64
	// Reason is a short description of which size check failed.
	Reason string
}

func (e *DeltaSizeError) Error() string {
	return fmt.Sprintf("%s: %s (declared target %d bytes, got %d bytes)", ErrDeltaSize, e.Reason, e.Declared, e.Actual)
}

func (e *DeltaSizeError) Unwrap() error { return ErrDeltaSize }

// Delta represents a delta, which is a way to describe the changes to a file between two commits.
//
// A delta is a sequence of instructions that describe how to modify a source file to produce a target file.
// The source file is usually the parent commit, and the target file is the current commit.
//
// Git uses deltas in pack files to efficiently store objects. Instead of storing complete copies
// of files, Git stores the differences between versions. This is particularly useful for large
// files that change little between commits.
//
// For more details about Git's delta format, see:
// https://git-scm.com/docs/pack-format#_deltified_representation
// https://git-scm.com/book/en/v2/Git-Internals-Packfiles#_deltified_storage
type Delta struct {
	Parent               string
	ExpectedSourceLength uint64
	// TargetLength is the declared decoded size, in bytes, of the reconstructed
	// object (the second size in the delta header). ApplyDelta enforces it exactly.
	TargetLength uint64
	// instructions is the raw delta command stream (the payload after the
	// header). ApplyDelta decodes it one instruction at a time; DecodeChanges
	// materializes it into a []DeltaChange on demand.
	instructions []byte
}

// DeltaChange represents a single change to a file.
//
// When iterating, this must be done sequentially, in order.
// No modifications of the source data is necessary.
// The presence of some fields determines how to act; see the documentation of the struct.
type DeltaChange struct {
	// If we should add data from the delta, DeltaData contains the data to add. In this case, ignore the Length & SourceOffset fields.
	DeltaData []byte

	// If we should copy from source (DeltaData == nil), SourceOffset is the starting position in the source, and Length is how much data is to be added.
	Length       uint64
	SourceOffset uint64
}

// DecodeChanges materializes the delta's instruction stream into a slice of
// DeltaChange, in order. Prefer ApplyDelta, which streams the instructions
// without allocating the slice; use DecodeChanges only to inspect individual
// changes. Returns nil when there is no instruction stream.
func (d *Delta) DecodeChanges() ([]DeltaChange, error) {
	if d.instructions == nil {
		return nil, nil
	}

	var changes []DeltaChange
	err := walkDeltaCommands(d.ExpectedSourceLength, d.TargetLength, d.instructions, func(c DeltaChange) error {
		changes = append(changes, c)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return changes, nil
}

// parseDelta parses a delta payload into a Delta struct.
//
// The delta format consists of:
// 1. A header containing the source and target sizes
// 2. A sequence of instructions, each starting with a command byte
//
// The command byte determines the type of instruction:
// - If the high bit is 0: Add new data from the delta
// - If the high bit is 1: Copy data from the source
//
// For more details about the delta format, see:
// https://git-scm.com/docs/pack-format#_deltified_representation
// FIXME: This logic is pretty hard to follow and test. So it's missing coverage for now
// Review it once we have some more integration testing so that we don't break things unintentionally.
//
// A declared target above maxDecodedObjectBytes (when > 0) is rejected with
// *ObjectTooLargeError before the instruction stream is walked. The instructions
// are validated but not materialized: the raw bytes are retained for ApplyDelta
// to stream, so a compressible delta cannot amplify into a large metadata slice.
func parseDelta(parent string, payload []byte, maxDecodedObjectBytes int64) (*Delta, error) {
	const minDeltaSize = 4
	if len(payload) < minDeltaSize {
		return nil, strError("payload too short")
	}

	expectedSourceLength, payload := deltaHeaderSize(payload)
	targetLength, payload := deltaHeaderSize(payload)

	// Reject an oversized target up front, before walking the command stream.
	if maxDecodedObjectBytes > 0 && targetLength > uint64(maxDecodedObjectBytes) {
		reported := int64(targetLength)
		if targetLength > math.MaxInt64 {
			reported = math.MaxInt64
		}
		return nil, &ObjectTooLargeError{Size: reported, Limit: maxDecodedObjectBytes}
	}

	// Validate the instruction stream without retaining it (nil callback).
	if err := walkDeltaCommands(expectedSourceLength, targetLength, payload, nil); err != nil {
		return nil, err
	}

	return &Delta{
		Parent:               parent,
		ExpectedSourceLength: expectedSourceLength,
		TargetLength:         targetLength,
		instructions:         payload,
	}, nil
}

// walkDeltaCommands decodes the instruction stream in order, invoking fn (when
// non-nil) with each decoded change. It keeps only one DeltaChange live at a
// time, so a hostile delta cannot amplify a small payload into a large
// []DeltaChange; a nil fn validates without acting.
//
// It validates the whole stream — an instruction consuming past the target, a
// malformed instruction (reported by parseDeltaCommand), and a stream that ends
// early or late are all rejected — so a well-formed stream fills exactly
// targetLength bytes and is consumed exactly.
func walkDeltaCommands(expectedSourceLength, targetLength uint64, instructions []byte, fn func(DeltaChange) error) error {
	remaining := targetLength
	payload := instructions
	for remaining > 0 && remaining <= targetLength {
		if len(payload) == 0 {
			return strError("missing cmd byte")
		}

		cmd := payload[0]
		payload = payload[1:]

		change, newPayload, consumedSize, err := parseDeltaCommand(cmd, payload, expectedSourceLength, targetLength)
		if err != nil {
			return err
		}

		// Reject an instruction that consumes past the target (also avoids
		// underflowing remaining below).
		if consumedSize > remaining {
			return &DeltaSizeError{
				Declared: targetLength,
				Actual:   (targetLength - remaining) + consumedSize,
				Reason:   "instruction consumes past declared target",
			}
		}

		if fn != nil {
			if err := fn(change); err != nil {
				return err
			}
		}

		remaining -= consumedSize
		payload = newPayload
	}

	// Trailing bytes after the target is filled are malformed; reject rather
	// than silently ignore them.
	if len(payload) > 0 {
		return strError("delta has trailing bytes after the declared target was reached")
	}

	return nil
}

// parseDeltaCommand parses a single delta command and returns the resulting change,
// remaining payload, consumed size, and any error.
func parseDeltaCommand(cmd byte, payload []byte, expectedSourceLength, originalDeltaSize uint64) (DeltaChange, []byte, uint64, error) {
	if cmd&0x80 != 0 {
		return parseCopyCommand(cmd, payload, expectedSourceLength, originalDeltaSize)
	} else if cmd != 0 {
		return parseAddCommand(cmd, payload, originalDeltaSize)
	} else {
		return DeltaChange{}, payload, 0, strError("payload included a cmd 0x0 (reserved) instruction")
	}
}

// parseCopyCommand parses a copy data instruction from delta command
func parseCopyCommand(cmd byte, payload []byte, expectedSourceLength, originalDeltaSize uint64) (DeltaChange, []byte, uint64, error) {
	offset, newPayload, err := parseOffset(cmd, payload)
	if err != nil {
		return DeltaChange{}, payload, 0, err
	}

	size, finalPayload, err := parseSize(cmd, newPayload)
	if err != nil {
		return DeltaChange{}, payload, 0, err
	}

	if size == 0 {
		size = 0x10000
	}

	if size > originalDeltaSize || offset+size > expectedSourceLength || offset+size < offset {
		return DeltaChange{}, payload, 0, fmt.Errorf("%w: copy offset=%d size=%d out of bounds for source length %d and target size %d",
			errMalformedDeltaInstruction, offset, size, expectedSourceLength, originalDeltaSize)
	}

	change := DeltaChange{
		SourceOffset: offset,
		Length:       size,
	}

	return change, finalPayload, size, nil
}

// parseAddCommand parses an add data instruction from delta command
func parseAddCommand(cmd byte, payload []byte, originalDeltaSize uint64) (DeltaChange, []byte, uint64, error) {
	if uint64(cmd) > originalDeltaSize {
		return DeltaChange{}, payload, 0, fmt.Errorf("%w: add length %d exceeds target size %d",
			errMalformedDeltaInstruction, cmd, originalDeltaSize)
	}
	if len(payload) < int(cmd) {
		return DeltaChange{}, payload, 0, strError("missing data bytes")
	}

	change := DeltaChange{
		DeltaData: payload[:cmd],
	}

	return change, payload[cmd:], uint64(cmd), nil
}

// parseOffset extracts offset value from copy command
func parseOffset(cmd byte, payload []byte) (uint64, []byte, error) {
	var offset uint64

	if (cmd & 0b1) != 0 {
		if len(payload) == 0 {
			return 0, payload, errMissingOffsetByte
		}
		offset |= uint64(payload[0])
		payload = payload[1:]
	}
	if (cmd & 0b10) != 0 {
		if len(payload) == 0 {
			return 0, payload, errMissingOffsetByte
		}
		offset |= uint64(payload[0]) << 8
		payload = payload[1:]
	}
	if (cmd & 0b100) != 0 {
		if len(payload) == 0 {
			return 0, payload, errMissingOffsetByte
		}
		offset |= uint64(payload[0]) << 16
		payload = payload[1:]
	}
	if (cmd & 0b1000) != 0 {
		if len(payload) == 0 {
			return 0, payload, errMissingOffsetByte
		}
		offset |= uint64(payload[0]) << 24
		payload = payload[1:]
	}

	return offset, payload, nil
}

// parseSize extracts size value from copy command
func parseSize(cmd byte, payload []byte) (uint64, []byte, error) {
	var size uint64

	if (cmd & 0b10000) != 0 {
		if len(payload) == 0 {
			return 0, payload, errMissingSizeByte
		}
		size |= uint64(payload[0])
		payload = payload[1:]
	}
	if (cmd & 0b100000) != 0 {
		if len(payload) == 0 {
			return 0, payload, errMissingSizeByte
		}
		size |= uint64(payload[0]) << 8
		payload = payload[1:]
	}
	if (cmd & 0b1000000) != 0 {
		if len(payload) == 0 {
			return 0, payload, errMissingSizeByte
		}
		size |= uint64(payload[0]) << 16
		payload = payload[1:]
	}

	return size, payload, nil
}

// deltaHeaderSize parses the header of a delta.
// It returns the size of the delta and the remaining payload.
//
// The header is a sequence of 7-bit integers, terminated by a byte with the most significant bit set.
// The first byte has the least significant 7 bits set.
//
// For more details about the delta header format, see:
// https://git-scm.com/docs/pack-format#_deltified_representation
// FIXME: This logic is pretty hard to follow and test. So it's missing coverage for now
// Review it once we have some more integration testing so that we don't break things unintentionally.
func deltaHeaderSize(b []byte) (uint64, []byte) {
	// TODO: This is a bit of a hack. We should probably have a better way to handle this.
	if len(b) == 0 {
		return 0, b
	}

	var size, j uint64
	var cmd byte
	for {
		cmd = b[j]
		size |= (uint64(cmd) & 0x7f) << (j * 7)
		j++
		if uint64(cmd)&0xb80 == 0 || j == uint64(len(b)) {
			break
		}
	}
	return size, b[j:]
}
