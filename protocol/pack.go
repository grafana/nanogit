package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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

// EmptyPack is the empty pack file used in Git to represent a non-existent object
// Pack file format: PACK + version(4) + object count(4) + SHA1(20)
var EmptyPack = []byte{
	'P', 'A', 'C', 'K', // PACK signature
	0x00, 0x00, 0x00, 0x02, // version 2
	0x00, 0x00, 0x00, 0x00, // object count 0
	0x2, 0x9d, 0x8, 0x82, 0x3b, 0xd8, 0xa8, 0xea, 0xb5, 0x10, 0xad, 0x6a, 0xc7, 0x5c, 0x82, 0x3c, 0xfd, 0x3e, 0xd3, 0x1e, // SHA1
}

var (
	// ErrDataTooLarge is returned when attempting to create a packet with data larger than MaxPktLineDataSize.
	ErrDataTooLarge = errors.New("the data field is too large")

	// ErrPackParseError is returned when parsing a Git packet fails.
	// This error should only be used with errors.Is() for comparison, not for type assertions.
	ErrPackParseError = errors.New("pack parse error")

	// ErrGitServerError is returned when the Git server reports an error.
	// This error should only be used with errors.Is() for comparison, not for type assertions.
	ErrGitServerError = errors.New("git server error")

	// ErrGitReferenceUpdateError is returned when a Git reference update fails.
	// This error should only be used with errors.Is() for comparison, not for type assertions.
	ErrGitReferenceUpdateError = errors.New("git reference update error")

	// ErrGitUnpackError is returned when Git pack unpacking fails.
	// This error should only be used with errors.Is() for comparison, not for type assertions.
	ErrGitUnpackError = errors.New("git unpack error")
)

// Pack is the interface that wraps the Marshal method.
// All packet types must implement this interface to be used with FormatPackets.
type Pack interface {
	// Marshal converts the packet into its wire format.
	// The returned byte slice should be ready to be sent over the wire.
	Marshal() ([]byte, error)
}

// PackLine represents a regular packet line in Git's protocol.
// It contains arbitrary data that will be prefixed with a length field.
type PackLine []byte

var _ Pack = PackLine{}

// Marshal implements the Pack interface for PackLine.
// It prepends a 4-byte hex length field to the data.
// Returns ErrDataTooLarge if the data exceeds MaxPktLineDataSize.
func (p PackLine) Marshal() ([]byte, error) {
	if len(p) > MaxPktLineDataSize {
		return nil, ErrDataTooLarge
	}
	out := make([]byte, len(p)+4)
	copy(out, []byte(fmt.Sprintf("%04x", len(p)+4)))
	copy(out[4:], p)
	return out, nil
}

// SpecialPack represents a special packet type in Git's protocol.
// These packets have predefined formats and don't need length calculation.
type SpecialPack string

var _ Pack = SpecialPack("")

// Marshal implements the Pack interface for SpecialPack.
// Special packets are pre-defined and known to be valid, so no validation is needed.
func (p SpecialPack) Marshal() ([]byte, error) {
	// We don't need to do anything special here. The special packets are pre-defined, and known to be valid.
	return []byte(p), nil
}

const (
	// FlushPacket is a packet of length '0000'. It is a special-case packet that indicates
	// the end of a message or the need to flush the output buffer.
	// Defined in:
	//   - https://git-scm.com/docs/gitprotocol-common
	//   - https://git-scm.com/docs/protocol-v2
	FlushPacket = SpecialPack("0000")

	// DelimeterPacket is a packet of length '0001'. It is a special-case packet used in
	// protocol v2 to separate sections of a message.
	// Defined in:
	//   - https://git-scm.com/docs/protocol-v2
	DelimeterPacket = SpecialPack("0001")

	// ResponseEndPacket is a packet of length '0002'. It is a special-case packet used in
	// protocol v2 to indicate the end of a response.
	// Defined in:
	//   - https://git-scm.com/docs/protocol-v2
	ResponseEndPacket = SpecialPack("0002")
)

// PackParseError provides structured information about a Git packet parsing error.
type PackParseError struct {
	Line []byte
	Err  error
}

// GitServerError provides structured information about a Git server error.
type GitServerError struct {
	Line      []byte
	ErrorType string // "ERR", "error", "fatal"
	Message   string
}

// GitReferenceUpdateError provides structured information about a Git reference update failure.
type GitReferenceUpdateError struct {
	Line    []byte
	RefName string
	Reason  string
}

// GitUnpackError provides structured information about a Git pack unpacking error.
type GitUnpackError struct {
	Line    []byte
	Message string
}

func (e *PackParseError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("error parsing line %q", e.Line)
	}
	return fmt.Sprintf("error parsing line %q: %s", e.Line, e.Err.Error())
}

// Unwrap enables errors.Is() compatibility with ErrPackParseError
func (e *PackParseError) Unwrap() error {
	return e.Err
}

func (e *GitServerError) Error() string {
	return fmt.Sprintf("git server %s: %s", e.ErrorType, e.Message)
}

// Unwrap enables errors.Is() compatibility with ErrGitServerError
func (e *GitServerError) Unwrap() error {
	return ErrGitServerError
}

func (e *GitReferenceUpdateError) Error() string {
	return fmt.Sprintf("reference update failed for %s: %s", e.RefName, e.Reason)
}

// Unwrap enables errors.Is() compatibility with ErrGitReferenceUpdateError
func (e *GitReferenceUpdateError) Unwrap() error {
	return ErrGitReferenceUpdateError
}

func (e *GitUnpackError) Error() string {
	return "pack unpack failed: " + e.Message
}

// Unwrap enables errors.Is() compatibility with ErrGitUnpackError
func (e *GitUnpackError) Unwrap() error {
	return ErrGitUnpackError
}

// NewPackParseError creates a new PackParseError with the given line and error.
func NewPackParseError(line []byte, err error) *PackParseError {
	return &PackParseError{
		Line: line,
		Err:  err,
	}
}

// NewGitServerError creates a new GitServerError with the specified details.
func NewGitServerError(line []byte, errorType, message string) *GitServerError {
	return &GitServerError{
		Line:      line,
		ErrorType: errorType,
		Message:   message,
	}
}

// NewGitReferenceUpdateError creates a new GitReferenceUpdateError with the specified details.
func NewGitReferenceUpdateError(line []byte, refName, reason string) *GitReferenceUpdateError {
	return &GitReferenceUpdateError{
		Line:    line,
		RefName: refName,
		Reason:  reason,
	}
}

// NewGitUnpackError creates a new GitUnpackError with the specified details.
func NewGitUnpackError(line []byte, message string) *GitUnpackError {
	return &GitUnpackError{
		Line:    line,
		Message: message,
	}
}

// IsPackParseError checks if an error is a PackParseError.
func IsPackParseError(err error) bool {
	return errors.As(err, new(*PackParseError))
}

// IsGitServerError checks if an error is a GitServerError.
func IsGitServerError(err error) bool {
	return errors.As(err, new(*GitServerError))
}

// IsGitReferenceUpdateError checks if an error is a GitReferenceUpdateError.
func IsGitReferenceUpdateError(err error) bool {
	return errors.As(err, new(*GitReferenceUpdateError))
}

// IsGitUnpackError checks if an error is a GitUnpackError.
func IsGitUnpackError(err error) bool {
	return errors.As(err, new(*GitUnpackError))
}

// FormatPacks converts a sequence of packets into their wire format.
// It automatically appends a FlushPacket if none is present in the sequence.
// Returns an error if any packet fails to marshal.
func FormatPacks(packs ...Pack) ([]byte, error) {
	var out bytes.Buffer
	flushed := false
	for _, pl := range packs {
		marshalled, err := pl.Marshal()
		if err != nil {
			return nil, err
		}
		out.Write(marshalled)

		if sp, ok := pl.(SpecialPack); ok && sp == FlushPacket {
			flushed = true
		}
	}
	if !flushed {
		out.Write([]byte(FlushPacket))
	}
	return out.Bytes(), nil
}

// ParsePack parses a sequence of Git protocol packets from a byte slice according to the
// Git Smart HTTP protocol specification (https://git-scm.com/docs/gitprotocol-pack).
//
// Git uses a packet-line format where each packet is prefixed with a 4-byte hex length field.
// The length includes the 4-byte length field itself, so the actual data is (length - 4) bytes.
//
// Returns:
//   - lines: Successfully parsed packet data (without length prefixes)
//   - remainder: Unparsed bytes remaining in the input (may be incomplete packets)
//   - err: Error encountered during parsing (if any)
//
// Packet Types Handled:
//
// Regular Data Packets:
//   - Format: 4-byte hex length + data
//   - Example: "0009hello" (length=9, data="hello")
//   - Returned in lines slice
//
// Special Control Packets:
//   - Flush packet: "0000" - indicates end of message
//   - Delimiter packet: "0001" - separates message sections (protocol v2)
//   - Response end packet: "0002" - indicates end of response (protocol v2)
//   - Empty packet: "0004" - should not be sent but is handled gracefully
//
// Side-band Multiplexing:
//
//	Git protocol v2 uses side-band multiplexing to allow multiple communication channels
//	within a single stream. Each packet may be prefixed with a single byte indicating
//	the side-band channel:
//	- Channel 1 (0x01): Packfile data
//	- Channel 2 (0x02): Progress messages and non-fatal errors (displayed on stderr)
//	- Channel 3 (0x03): Fatal error messages that terminate the connection
//
//	When side-band multiplexing is active, error messages may appear as:
//	- "0015\x02error: message" (channel 2 - progress/error info)
//	- "0015\x03fatal: message" (channel 3 - fatal errors)
//
//	Per Git protocol v2 spec: https://git-scm.com/docs/protocol-v2#_packfile_negotiation
//
// Server Error Packets (terminate parsing and return structured errors):
//
//  1. ERR Packets:
//     - Format: length + "ERR " + message
//     - Example: "000dERR hello"
//     - Returns: GitServerError with ErrorType="ERR"
//     - Spec: RFC gitprotocol-pack error-line format
//
//  2. Git Error/Fatal Messages:
//     - Direct format: length + "error:" + message or length + "fatal:" + message
//     - Side-band format: length + 0x02 + "error:" + message or length + 0x02 + "fatal:" + message
//     - Examples: "0015error: bad object", "0016\x02error: bad object"
//     - Returns: GitServerError with ErrorType="error" or "fatal"
//     - Special case: Messages containing "unpack" return GitUnpackError
//     - Source: Direct error messages or Git side-band channel 2 (progress/error messages)
//     - Note: Side-band channel 2 is used for progress info and error messages that should
//     be displayed to the user (typically on stderr). The leading 0x02 byte indicates
//     side-band channel 2 per Git protocol v2 specification.
//
//  3. Reference Update Failures:
//     - Format: length + "ng " + refname + " " + reason
//     - Example: "0020ng refs/heads/main failed"
//     - Returns: GitReferenceUpdateError with parsed refname and reason
//     - Spec: Git report-status protocol "ng" (no good) responses
//
//  4. Unpack Status Messages:
//     - Format: length + "unpack " + status
//     - Examples: "000bunpack ok", "0019unpack index-pack failed"
//     - Success: Continues parsing (adds to lines)
//     - Failure: Returns GitUnpackError
//     - Spec: Git report-status protocol unpack status
//
// Error Conditions:
//   - Invalid hex length field: Returns PackParseError
//   - Truncated packets: Returns PackParseError
//   - Malformed packet data: Returns PackParseError
//
// Protocol Compliance:
//   - Implements Git packet-line format per gitprotocol-common
//   - Handles error reporting per gitprotocol-pack
//   - Supports Git protocol v1 and v2 control packets
//   - Compatible with Git Smart HTTP protocol error handling
//
// Example Usage:
//
//	data := []byte("0009hello000dERR failed0000")
//	lines, err := ParsePack(bytes.NewReader(data))
//	// Returns: lines=["hello"], err=GitServerError
//
// ParsePack parses Git protocol packets from an io.Reader for streaming scenarios.
// This is the streaming equivalent of ParsePack that processes packets as they arrive
// without buffering the entire response in memory.
//
// Returns:
//   - lines: Successfully parsed packet data (without length prefixes)
//   - err: Error encountered during parsing (if any)
//
// Unlike ParsePack, this function processes the stream until EOF or an error occurs.
// It handles the same packet types but doesn't return remainder bytes since the stream
// is consumed as it's read.
func ParsePack(reader io.Reader) (lines [][]byte, err error) {
	for {
		// Read length header (4 hex bytes)
		lengthBytes := make([]byte, 4)
		_, err := io.ReadFull(reader, lengthBytes)
		if err != nil {
			if err == io.EOF {
				return lines, nil // Normal end of stream
			}
			return lines, NewPackParseError(lengthBytes, fmt.Errorf("reading packet length: %w", err))
		}

		length, err := strconv.ParseUint(string(lengthBytes), 16, 16)
		if err != nil {
			return lines, NewPackParseError(lengthBytes, fmt.Errorf("parsing line length: %w", err))
		}

		// Handle different packet types
		switch {
		case length < 4:
			// Special packets: flush (0000), delimiter (0001), response-end (0002)
			if length == 2 { // ResponseEndPacket
				return lines, nil
			}
			// Continue for other special packets (flush, delimiter)

		case length == 4:
			// Empty packet - nothing more to read for this packet
			continue

		default:
			// Read packet data
			dataLength := length - 4
			packetData := make([]byte, dataLength)
			if _, err := io.ReadFull(reader, packetData); err != nil {
				return lines, NewPackParseError(lengthBytes, fmt.Errorf("reading packet data: %w", err))
			}

			// Process packet content using helper functions
			switch {
			case bytes.HasPrefix(packetData, []byte("ERR ")):
				return lines, handleERRPacket(append(lengthBytes, packetData...), packetData)

			case isErrorOrFatalMessage(packetData):
				return lines, handleErrorFatalMessage(append(lengthBytes, packetData...), packetData)

			case bytes.HasPrefix(packetData, []byte("ng ")):
				return lines, handleReferenceUpdateFailure(append(lengthBytes, packetData...), packetData)

			case bytes.HasPrefix(packetData, []byte("unpack ")):
				unpackContent := string(packetData[7:]) // Skip "unpack "
				if unpackContent != "ok" {
					return lines, NewGitUnpackError(append(lengthBytes, packetData...), unpackContent)
				}
				// If unpack ok, add to lines and continue
				lines = append(lines, packetData)

			default:
				// Regular data packet
				lines = append(lines, packetData)
			}
		}
	}
}

// isErrorOrFatalMessage checks if packet contains error/fatal messages (direct or side-band)
func isErrorOrFatalMessage(data []byte) bool {
	// Direct format: "error:" or "fatal:"
	if bytes.HasPrefix(data, []byte("error:")) || bytes.HasPrefix(data, []byte("fatal:")) {
		return true
	}

	// Side-band format: check for channels 2 or 3 with error/fatal messages
	if len(data) > 1 && (data[0] == 0x02 || data[0] == 0x03) {
		return bytes.HasPrefix(data[1:], []byte("error:")) || bytes.HasPrefix(data[1:], []byte("fatal:"))
	}

	return false
}

// handleERRPacket processes ERR packets per gitprotocol-pack specification
func handleERRPacket(fullPacket, packetData []byte) error {
	// ERR packet format: "ERR " + message
	message := string(packetData[4:]) // Skip "ERR "
	return NewGitServerError(fullPacket, "ERR", message)
}

// handleErrorFatalMessage processes error/fatal messages (direct or side-band format)
func handleErrorFatalMessage(fullPacket, packetData []byte) error {
	var messageStart int
	var fullMessage string

	// Determine message format and extract content
	if bytes.HasPrefix(packetData, []byte("error:")) || bytes.HasPrefix(packetData, []byte("fatal:")) {
		// Direct format: no side-band prefix
		messageStart = 0
		fullMessage = string(packetData)
	} else {
		// Side-band format: skip the side-band channel byte (0x02 or 0x03)
		messageStart = 1
		fullMessage = string(packetData[1:])
	}

	// Parse error type and message
	var errorType, message string
	if bytes.HasPrefix(packetData[messageStart:], []byte("error:")) {
		errorType = "error"
		message = fullMessage[6:] // Remove "error:" prefix
	} else {
		errorType = "fatal"
		message = fullMessage[6:] // Remove "fatal:" prefix
	}

	// Check if this is an unpack error
	if bytes.Contains(packetData, []byte("unpack")) {
		return NewGitUnpackError(fullPacket, message)
	}

	return NewGitServerError(fullPacket, errorType, message)
}

// handleReferenceUpdateFailure processes "ng" (no good) reference update failure packets
func handleReferenceUpdateFailure(fullPacket, packetData []byte) error {
	// Format: "ng <refname> <error-msg>"
	parts := bytes.SplitN(packetData[3:], []byte(" "), 2) // Skip "ng "

	var refName, reason string
	if len(parts) >= 1 {
		refName = string(parts[0])
	}
	if len(parts) >= 2 {
		reason = string(parts[1])
	} else {
		reason = "update failed"
	}

	return NewGitReferenceUpdateError(fullPacket, refName, reason)
}
