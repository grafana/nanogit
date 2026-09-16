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
)

// ErrDeltaSize is the sentinel wrapped by *DeltaSizeError. Match it with
// errors.Is(err, ErrDeltaSize) to detect any delta whose instructions do not
// reconstruct exactly its declared target size.
//
// It is distinct from ErrObjectTooLarge: ErrObjectTooLarge means a delta's
// declared target exceeds the configured decoded-object cap (a size-limit
// rejection), whereas ErrDeltaSize means the delta is internally inconsistent
// — malformed or truncated — regardless of any limit.
var ErrDeltaSize = strError("delta does not reconstruct its declared target size")

// DeltaSizeError reports that a delta's instructions do not reconstruct an
// object of its declared target size, so the delta is malformed or truncated.
// nanogit returns it from more than one place: while parsing a delta (an
// instruction that would consume past the declared target) and while applying
// one (a change that would overrun the target, or a final reconstructed length
// that does not match). It wraps ErrDeltaSize so errors.Is(err, ErrDeltaSize)
// matches, while exposing the declared target and the offending size.
type DeltaSizeError struct {
	// Declared is the target size, in bytes, declared in the delta header.
	Declared uint64
	// Actual is the reconstructed output size, in bytes, observed at the point
	// the mismatch was detected. For an overrun it is the size the offending
	// instruction would have produced; for a final-length check it is the
	// completed output length.
	Actual uint64
	// Reason is a short, stable description of which size check failed.
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
	// TargetLength is the declared decoded size, in bytes, of the object the
	// delta reconstructs (the second size in the delta header). ApplyDelta
	// enforces it exactly (rejecting an over- or under-length result), so a
	// caller can reject an over-large reconstruction up front by checking
	// TargetLength against the decoded-object cap before applying the delta.
	TargetLength uint64
	// Changes contains pre-decoded modifications to apply, in order. It is an
	// input to ApplyDelta for callers that build a Delta by hand (e.g. tests).
	//
	// NOTE: a Delta produced by the packfile reader now leaves this nil, whereas
	// earlier versions populated it. Decoding a delta's whole instruction stream
	// into a []DeltaChange up front lets a small, highly compressible payload
	// amplify into hundreds of MiB of metadata (~40 bytes per instruction, and
	// an instruction can be as small as one payload byte), so a parsed Delta
	// instead retains the raw instruction bytes (see instructions) and ApplyDelta
	// streams them one at a time. Callers that inspected Changes on a parsed
	// Delta should call DecodeChanges instead, which decodes on demand.
	//
	// When iterating, this must be done sequentially, in order. No modification
	// of the source data is necessary. The presence of some fields determines
	// how to act; see the documentation of the struct.
	Changes []DeltaChange
	// instructions holds the raw delta command stream (the payload after the
	// header) for a parsed Delta. ApplyDelta decodes it one instruction at a
	// time so reconstruction memory stays bounded by TargetLength regardless of
	// how many instructions a (possibly hostile) delta declares. It is nil for a
	// hand-built Delta, which supplies its work through Changes instead.
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

// DecodeChanges returns the delta's instructions decoded into a slice, in
// order. It restores the pre-streaming ability to inspect a parsed delta's
// changes: parseDelta no longer populates the Changes field (doing so risked a
// metadata bomb), so a caller that needs the decoded instructions materializes
// them explicitly here and pays the O(number-of-instructions) cost only when it
// asks for it.
//
// For a hand-built Delta (no retained instruction bytes) it returns the Changes
// field as-is. Prefer ApplyDelta when you only need the reconstructed object;
// it streams the instructions without building this slice.
func (d *Delta) DecodeChanges() ([]DeltaChange, error) {
	if d.instructions == nil {
		return d.Changes, nil
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
// maxDecodedObjectBytes caps the reconstructed object's declared decoded size
// (the delta's target size). A value <= 0 disables the check. It is enforced
// immediately after the header is decoded — before the command stream is even
// walked — so an oversized target is rejected up front, and the cap applies to
// every consumer (not just those that resolve deltas). Oversized targets
// surface as *ObjectTooLargeError (which wraps ErrObjectTooLarge).
//
// parseDelta does NOT materialize the instructions into delta.Changes: doing so
// would let a small, highly compressible payload amplify into hundreds of MiB
// of []DeltaChange metadata even when both its payload and target sit under the
// cap. It retains the raw instruction bytes on the Delta and validates them
// with a single streaming walk (holding one instruction at a time); ApplyDelta
// later re-walks them to reconstruct the object, bounded by TargetLength.
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

	// Validate the instruction stream without retaining it. A nil callback means
	// walkDeltaCommands only checks structure (and rejects an instruction that
	// overruns the declared target), keeping memory O(1) in the instruction
	// count.
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

// walkDeltaCommands decodes the instructions in a delta command stream in order,
// invoking fn (when non-nil) with each decoded change. It intentionally keeps
// only one DeltaChange live at a time: callers that need to act on every
// instruction (ApplyDelta) do so through fn rather than receiving a slice, so a
// hostile delta cannot amplify a small, in-limit payload into hundreds of MiB
// of []DeltaChange metadata (~40 bytes per instruction). A nil fn validates the
// stream without acting on it.
//
// It validates the whole stream: an instruction that consumes past the target
// yields a *DeltaSizeError, a malformed (zero-consumption) instruction and a
// stream that ends early ("missing cmd byte") or late (trailing bytes) are all
// rejected, so a well-formed stream fills exactly targetLength bytes and is
// consumed exactly. ApplyDelta keeps its own final-length check as a guard for
// hand-built Deltas, which supply Changes directly and never pass through here.
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

		if consumedSize == 0 {
			// parseDeltaCommand reports zero consumption only for a malformed
			// instruction — an out-of-bounds copy, or an add whose length
			// exceeds the target (the size==0 copy is normalized to 64 KiB
			// earlier, so it never lands here). Reject it at parse time rather
			// than stopping and leaving a short delta: otherwise ApplyDelta would
			// preallocate the full declared target before failing the final
			// length check, and resolveDeltas would repeat that cap-sized
			// allocation on every pass while other deltas make progress.
			return strError("malformed delta instruction (out-of-bounds copy or oversized add)")
		}

		// A command consuming more than the target has left is malformed: a
		// well-formed delta's instructions sum to exactly the declared target.
		// Reject it here rather than underflowing the unsigned counter below.
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

	// A well-formed delta's instructions end exactly when the target is filled.
	// Any bytes left over (trailing commands after remaining hit zero, or a
	// non-empty stream on a zero-target delta) are malformed — reject them
	// rather than silently ignoring them and reconstructing a truncated object.
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
		return DeltaChange{}, payload, 0, nil
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
		return DeltaChange{}, payload, 0, nil
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
