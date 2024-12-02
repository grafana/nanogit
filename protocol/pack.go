package protocol

import (
	"errors"
	"fmt"
	"strconv"
)

// Packfiles are the packet type of Git.
// They are described in:
//   * https://git-scm.com/docs/gitprotocol-common
//   * https://git-scm.com/docs/gitprotocol-pack

// A non-binary line SHOULD BE terminated by an LF, which if present MUST be included in the total length.
// Receivers MUST treat pkt-lines with non-binary data the same whether or not they contain the trailing LF (stripping the LF if present, and not complaining when it is missing).
//
// The maximum length of a pkt-line’s data component is 65516 bytes.
// Implementations MUST NOT send pkt-line whose length exceeds 65520 (65516 bytes of payload + 4 bytes of length data).
//
// A pkt-line with a length field of 0 ("0000"), called a flush-pkt, is a special case and MUST be handled differently than an empty pkt-line ("0004").
const (
	// The length field of a packet includes 4 ASCII digits for the length.
	// The length field is part of the value, i.e. the data is the value - 4.
	PktLineLengthSize = 4
	// The data field can be at most 65516 bytes long, making the whole packet 65520 bytes at most.
	MaxPktLineDataSize = 65516
	// The maximum packet size with the largest packet possible is 65520 bytes.
	MaxPktLineSize = MaxPktLineDataSize + PktLineLengthSize
)

var (
	ErrDataTooLarge = errors.New("the data field is too large")
)

type specialPacket []byte

var (
	// FlushPacket is a packet of length '0000'. It is a special-case, defined by two docs:
	//   - https://git-scm.com/docs/gitprotocol-common
	//   - https://git-scm.com/docs/protocol-v2 or https://git-scm.com/docs/gitprotocol-v2
	FlushPacket = specialPacket("0000")

	// DelimeterPacket is a packet of length '0001'. It is a special-case, defined by the v2 document:
	// https://git-scm.com/docs/protocol-v2 or https://git-scm.com/docs/gitprotocol-v2
	DelimeterPacket = specialPacket("0001")

	// ResponseEndPacket is a packet of length '0002'. It is a special-case, defined by the v2 document:
	// https://git-scm.com/docs/protocol-v2 or https://git-scm.com/docs/gitprotocol-v2
	ResponseEndPacket = specialPacket("0002")
)

func FormatPacket(packetLines ...[]byte) []byte {
	var out []byte
	for _, pl := range packetLines {
		n := fmt.Sprintf("%04x", len(pl)+4)
		out = append(out, []byte(n)...)
		out = append(out, pl...)
	}
	out = append(out, FlushPacket...)
	return out
}

func ParsePacket(b []byte) (lines [][]byte, remainder []byte, err error) {
	for len(b) > 0 {
		length, err := strconv.ParseInt(string(b[:4]), 16, 32)
		if err != nil {
			return nil, b, err
		}
		if length < 4 {
			// This is a special-case packet.
			// For now, we don't really have a good solution to handle special-case packets.
			b = b[4:]
			if length == 2 { // ResponseEndPacket
				return lines, b, nil
			}
			continue
		}
		// The length includes the first 4 bytes as well, so we should be good with this.
		line := b[4:length]
		lines = append(lines, line)
		if int(length) > len(line)+4 {
			return lines, b, fmt.Errorf("packet declared %d bytes, but only had %d", length, len(line)+4)
		}
		b = b[length:]
	}
	return lines, b, nil
}
