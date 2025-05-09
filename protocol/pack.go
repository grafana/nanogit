package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

// Package protocol implements Git's packet format used in various Git protocols.
// Git uses a packet-based protocol for communication between clients and servers.
// This package provides types and functions for working with Git's packet format.
//
// The packet format is used in several Git protocols:
//   - Git Protocol v1 (pack protocol)
//   - Git Protocol v2
//   - Smart HTTP protocol
//
// For more details about Git's packet format, see:
//   - https://git-scm.com/docs/gitprotocol-common
//   - https://git-scm.com/docs/gitprotocol-pack
//   - https://git-scm.com/docs/protocol-v2

// A non-binary line SHOULD BE terminated by an LF, which if present MUST be included in the total length.
// Receivers MUST treat pkt-lines with non-binary data the same whether or not they contain the trailing LF (stripping the LF if present, and not complaining when it is missing).
//
// The maximum length of a pkt-line's data component is 65516 bytes.
// Implementations MUST NOT send pkt-line whose length exceeds 65520 (65516 bytes of payload + 4 bytes of length data).
//
// A pkt-line with a length field of 0 ("0000"), called a flush-pkt, is a special case and MUST be handled differently than an empty pkt-line ("0004").
const (
	// PktLineLengthSize is the size of the length field in a packet (4 ASCII hex digits).
	// The length field is part of the value, i.e. the data is the value - 4.
	PktLineLengthSize = 4

	// MaxPktLineDataSize is the maximum size of the data field in a packet (65516 bytes).
	// This is the maximum payload size that can be sent in a single packet.
	MaxPktLineDataSize = 65516

	// MaxPktLineSize is the maximum total size of a packet (65520 bytes).
	// This includes both the length field (4 bytes) and the data field (65516 bytes).
	MaxPktLineSize = MaxPktLineDataSize + PktLineLengthSize
)

var (
	// ErrDataTooLarge is returned when attempting to create a packet with data larger than MaxPktLineDataSize.
	ErrDataTooLarge = errors.New("the data field is too large")
)

// Packet is the interface that wraps the Marshal method.
// All packet types must implement this interface to be used with FormatPackets.
type Packet interface {
	// Marshal converts the packet into its wire format.
	// The returned byte slice should be ready to be sent over the wire.
	Marshal() ([]byte, error)
}

// PacketLine represents a regular packet line in Git's protocol.
// It contains arbitrary data that will be prefixed with a length field.
type PacketLine []byte

var _ Packet = PacketLine{}

// Marshal implements the Packet interface for PacketLine.
// It prepends a 4-byte hex length field to the data.
// Returns ErrDataTooLarge if the data exceeds MaxPktLineDataSize.
func (p PacketLine) Marshal() ([]byte, error) {
	if len(p) > MaxPktLineDataSize {
		return nil, ErrDataTooLarge
	}
	out := make([]byte, len(p)+4)
	copy(out, []byte(fmt.Sprintf("%04x", len(p)+4)))
	copy(out[4:], p)
	return out, nil
}

// SpecialPacket represents a special packet type in Git's protocol.
// These packets have predefined formats and don't need length calculation.
type SpecialPacket string

var _ Packet = SpecialPacket("")

// Marshal implements the Packet interface for SpecialPacket.
// Special packets are pre-defined and known to be valid, so no validation is needed.
func (p SpecialPacket) Marshal() ([]byte, error) {
	// We don't need to do anything special here. The special packets are pre-defined, and known to be valid.
	return []byte(p), nil
}

const (
	// FlushPacket is a packet of length '0000'. It is a special-case packet that indicates
	// the end of a message or the need to flush the output buffer.
	// Defined in:
	//   - https://git-scm.com/docs/gitprotocol-common
	//   - https://git-scm.com/docs/protocol-v2
	FlushPacket = SpecialPacket("0000")

	// DelimeterPacket is a packet of length '0001'. It is a special-case packet used in
	// protocol v2 to separate sections of a message.
	// Defined in:
	//   - https://git-scm.com/docs/protocol-v2
	DelimeterPacket = SpecialPacket("0001")

	// ResponseEndPacket is a packet of length '0002'. It is a special-case packet used in
	// protocol v2 to indicate the end of a response.
	// Defined in:
	//   - https://git-scm.com/docs/protocol-v2
	ResponseEndPacket = SpecialPacket("0002")
)

// PacketParseError represents an error that occurred while parsing a packet.
// It includes the problematic line and the underlying error.
type PacketParseError struct {
	Line []byte
	Err  error
}

// Error implements the error interface for ParseError.
func (e *PacketParseError) Error() string {
	return fmt.Sprintf("error parsing line %q: %s", e.Line, e.Err.Error())
}

// Unwrap returns the underlying error.
func (e *PacketParseError) Unwrap() error {
	return e.Err
}

// NewPacketParseError creates a new PacketParseError with the given line and error.
func NewPacketParseError(line []byte, err error) *PacketParseError {
	return &PacketParseError{
		Line: line,
		Err:  err,
	}
}

// IsPacketParseError checks if an error is a PacketParseError.
func IsPacketParseError(err error) bool {
	return errors.As(err, new(*PacketParseError))
}

// FormatPackets converts a sequence of packets into their wire format.
// It automatically appends a FlushPacket if none is present in the sequence.
// Returns an error if any packet fails to marshal.
func FormatPackets(packets ...Packet) ([]byte, error) {
	var out bytes.Buffer
	flushed := false
	for _, pl := range packets {
		marshalled, err := pl.Marshal()
		if err != nil {
			return nil, err
		}
		out.Write(marshalled)

		if sp, ok := pl.(SpecialPacket); ok && sp == FlushPacket {
			flushed = true
		}
	}
	if !flushed {
		out.Write([]byte(FlushPacket))
	}
	return out.Bytes(), nil
}

// ParsePacket parses a sequence of packets from a byte slice.
// It returns:
//   - lines: The parsed packet lines
//   - remainder: Any remaining bytes that couldn't be parsed
//   - err: Any error that occurred during parsing
//
// The function handles:
//   - Regular packets with data
//   - Special packets (flush, delimiter, response end)
//   - Error packets (starting with "ERR ")
//   - Empty packets
//
// TODO: Accept an io.Reader to the function, and return a new kind of reader.
func ParsePacket(b []byte) (lines [][]byte, remainder []byte, err error) {
	// There should be at least 4 bytes in the packet.
	for len(b) >= 4 {
		length, err := strconv.ParseUint(string(b[:4]), 16, 16)

		switch {
		case err != nil:
			return nil, b, NewPacketParseError(b, fmt.Errorf("parsing line length: %w", err))

		case length < 4:
			// This is a special-case packet.
			// For now, we don't really have a good solution to handle special-case packets.
			b = b[4:]
			if length == 2 { // ResponseEndPacket
				return lines, b, nil
			}
			continue

		case length == 4:
			// This is an empty packet; it SHOULD not be sent.
			// https://git-scm.com/docs/gitprotocol-common#_pkt_line_format
			b = b[4:]
			continue

		case uint64(len(b)) < length:
			return lines, b, NewPacketParseError(b, fmt.Errorf("line declared %d bytes, but only %d are avaiable", length, len(b)))

		case bytes.HasPrefix(b[4:], []byte("ERR ")):
			// This is an error packet.
			// https://git-scm.com/docs/gitprotocol-pack#_pkt_line_format

			// An error packet is a special pkt-line that contains
			// an error string.
			//
			//    error-line     =  PKT-LINE("ERR" SP explanation-text)
			//
			// Throughout the protocol, where PKT-LINE(...) is
			// expected, an error packet MAY be sent. Once this
			// packet is sent by a client or a server, the data
			// transfer process defined in this protocol is
			// terminated.

			return lines, b[length:], fmt.Errorf("error packet: %s", b[8:length])
		}

		// The length includes the first 4 bytes as well, so we should be good with this.
		lines = append(lines, b[4:length])
		b = b[length:]
	}

	return lines, b, nil
}
