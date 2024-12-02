package protocol

import (
	"bytes"
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

type Packet interface {
	Marshal() ([]byte, error)
}

type PacketLine []byte

func (p PacketLine) Marshal() ([]byte, error) {
	if len(p) > MaxPktLineDataSize {
		return nil, ErrDataTooLarge
	}
	out := make([]byte, len(p)+4)
	copy(out, []byte(fmt.Sprintf("%04x", len(p)+4)))
	copy(out[4:], p)
	return out, nil
}

type SpecialPacket []byte

func (p SpecialPacket) Marshal() ([]byte, error) {
	// We don't need to do anything special here. The special packets are pre-defined, and known to be valid.
	return []byte(p), nil
}

var (
	// FlushPacket is a packet of length '0000'. It is a special-case, defined by two docs:
	//   - https://git-scm.com/docs/gitprotocol-common
	//   - https://git-scm.com/docs/protocol-v2 or https://git-scm.com/docs/gitprotocol-v2
	FlushPacket = SpecialPacket("0000")

	// DelimeterPacket is a packet of length '0001'. It is a special-case, defined by the v2 document:
	// https://git-scm.com/docs/protocol-v2 or https://git-scm.com/docs/gitprotocol-v2
	DelimeterPacket = SpecialPacket("0001")

	// ResponseEndPacket is a packet of length '0002'. It is a special-case, defined by the v2 document:
	// https://git-scm.com/docs/protocol-v2 or https://git-scm.com/docs/gitprotocol-v2
	ResponseEndPacket = SpecialPacket("0002")
)

type ParseError struct {
	Line []byte
	Err  error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("error parsing line %q: %s", e.Line, e.Err.Error())
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

func NewParseError(line []byte, err error) *ParseError {
	return &ParseError{
		Line: line,
		Err:  err,
	}
}

func IsParseError(err error) bool {
	return errors.As(err, new(*ParseError))
}

func FormatPackets(packets ...Packet) ([]byte, error) {
	var out bytes.Buffer
	for _, pl := range packets {
		marshalled, err := pl.Marshal()
		if err != nil {
			return nil, err
		}
		out.Write(marshalled)
	}
	out.Write(FlushPacket)
	return out.Bytes(), nil
}

func ParsePacket(b []byte) (lines [][]byte, remainder []byte, err error) {
	// FIXME: We need to parse error packets: https://git-scm.com/docs/gitprotocol-pack#_pkt_line_format
	//
	// An error packet is a special pkt-line that contains an error string.
	//    error-line     =  PKT-LINE("ERR" SP explanation-text)
	// Throughout the protocol, where PKT-LINE(...) is expected, an error packet MAY be sent. Once this packet is sent by a client or a server, the data transfer process defined in this protocol is terminated.

	// There should be at least 4 bytes in the packet.
	for len(b) >= 4 {
		length, err := strconv.ParseUint(string(b[:4]), 16, 16)

		switch {
		case err != nil:
			return nil, b, NewParseError(b, fmt.Errorf("parsing line length: %w", err))

		case length < 4:
			// This is a special-case packet.
			// For now, we don't really have a good solution to handle special-case packets.
			b = b[4:]
			if length == 2 { // ResponseEndPacket
				return lines, b, nil
			}
			continue

		case len(b) < int(length): //nolint:gosec // length is expected to be at most 2^16.
			return lines, b, NewParseError(b, fmt.Errorf("line declared %d bytes, but only %d are avaiable", length, len(b)))
		}

		// The length includes the first 4 bytes as well, so we should be good with this.
		lines = append(lines, b[4:length])
		b = b[length:]
	}

	return lines, b, nil
}
