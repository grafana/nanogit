package protocol

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidDelta = errors.New("the payload given is not a valid delta")
)

type Delta struct {
	Parent               string
	ExpectedSourceLength uint64
	// Changes contains all the modifications to do in order.
	//
	// When iterating, this must be done sequentially, in order.
	// No modifications of the source data is necessary.
	// The presence of some fields determines how to act; see the documentation of the struct.
	Changes []DeltaChange
}

type DeltaChange struct {
	// If we should add data from the delta, DeltaData contains the data to add. In this case, ignore the Length & SourceOffset fields.
	DeltaData []byte

	// If we should copy from source (DeltaData == nil), SourceOffset is the starting position in the source, and Length is how much data is to be added.
	Length       uint64
	SourceOffset uint64
}

func parseDelta(parent string, payload []byte) (*Delta, error) {
	delta := &Delta{Parent: parent}

	const minDeltaSize = 4
	if len(payload) < minDeltaSize {
		return nil, ErrInvalidDelta
	}
	delta.ExpectedSourceLength, payload = deltaHeaderSize(payload)
	deltaSize, payload := deltaHeaderSize(payload)
	originalDeltaSize := deltaSize

	for deltaSize > 0 &&
		// Protect against underflows.
		deltaSize <= originalDeltaSize {
		// The command and its data depends on the bits in it.
		//
		// The following explanation uses diagrams from RFC 1951 (section 3.1): https://www.ietf.org/rfc/rfc1951.txt
		//
		// If the last bit ((cmd >> 7) & 1) is unset (zero), this is an instruction to add new data FROM the delta TO the patched source (ie patched parent).
		// The format is:
		//	+----------+============+
		//	| 0xxxxxxx |    data    |
		//	+----------+============+
		// The x's define the size of the data to come. It must not be zero.
		//
		// If the last bit is set, however, we are instructed to copy data FROM the source TO the patched source.
		// The format is:
		//	+----------+---------+---------+---------+---------+-------+-------+-------+
		//	| 1xxxxxxx | offset1 | offset2 | offset3 | offset4 | size1 | size2 | size3 |
		//	+----------+---------+---------+---------+---------+-------+-------+-------+
		// The x's define which of the offsets and sizes are set. offset1 is represented by bit 0 (ie the right-most bit), offset2 by bit 1, etc.
		// If all size bits are unset or size == 0, size should be set to 0x10000.
		// If all offset bits are unset, it is defaulted to 0.
		// If offset bits that aren't next to each other are set (e.g. offset1 and offset3 are set), they are still treated as their appropriate positions. I.e. offset1 would represent bits 0-7, and offset3 bits 16-23.
		//
		// If the entire cmd is 0x0, it is reserved and MUST return an error.
		cmd := payload[0]
		payload = payload[1:]
		// FIXME: We need to safely check if a byte exists, and return err if not.
		if cmd&0b1000_0000 != 0 { // Copy data instruction
			var offset, size uint64
			if cmd&1 != 0 {
				offset = uint64(payload[0])
				payload = payload[1:]
			}
			if cmd&0b0000_0010 != 0 {
				offset = uint64(payload[0]) << 8
				payload = payload[1:]
			}
			if cmd&0b0000_0100 != 0 {
				offset = uint64(payload[0]) << 16
				payload = payload[1:]
			}
			if cmd&0b0000_1000 != 0 {
				offset = uint64(payload[0]) << 24
				payload = payload[1:]
			}

			if cmd&0b0001_0000 != 0 {
				size = uint64(payload[0])
				payload = payload[1:]
			}
			if cmd&0b0010_0000 != 0 {
				size = uint64(payload[0]) << 8
				payload = payload[1:]
			}
			if cmd&0b0100_0000 != 0 {
				size = uint64(payload[0]) << 16
				payload = payload[1:]
			}
			if size == 0 { // documented exception
				size = 0x10000
			}

			if size > originalDeltaSize ||
				offset+size > delta.ExpectedSourceLength ||
				offset+size < offset {
				break
			}

			delta.Changes = append(delta.Changes, DeltaChange{
				SourceOffset: offset,
				Length:       size,
			})
			deltaSize -= size
		} else if cmd != 0 { // Add data instruction
			if uint64(cmd) > originalDeltaSize {
				break
			}

			delta.Changes = append(delta.Changes, DeltaChange{
				// We don't have to do anything about cmd's top bit here. It is 0; we only need the 7 others which act as a 7-bit integer size.
				DeltaData: payload[:cmd],
			})
			deltaSize -= uint64(cmd)
			payload = payload[cmd:]
		} else { // Cmd == 0; reserved.
			return nil, fmt.Errorf("%w: payload included a cmd 0x0 (reserved) instruction", ErrInvalidDelta)
		}
	}

	return delta, nil
}

func deltaHeaderSize(b []byte) (uint64, []byte) {
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
