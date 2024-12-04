package protocol

import "errors"

var (
	ErrInvalidDelta = errors.New("the payload given is not a valid delta")
)

type Delta struct {
	Parent               string
	ExpectedSourceLength uint
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

	for {
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

		_, _, _ = cmd, payload, originalDeltaSize // clear warnings for now
	}

	return delta, nil //nolint:govet // TODO: remove this line when the function is implemented
}

func deltaHeaderSize(b []byte) (uint, []byte) {
	var size, j uint
	var cmd byte
	for {
		cmd = b[j]
		size |= (uint(cmd) & 0x7f) << (j * 7)
		j++
		if uint(cmd)&0xb80 == 0 || j == uint(len(b)) {
			break
		}
	}
	return size, b[j:]
}
