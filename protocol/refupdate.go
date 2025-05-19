package protocol

import (
	"fmt"
)

// ZeroHash represents the all-zeros SHA-1 hash used in Git to represent a non-existent object
const ZeroHash = "0000000000000000000000000000000000000000"

type RefUpdateRequest struct {
	OldRef  string
	NewRef  string
	RefName string
}

// Format formats the ref update request into a byte slice that can be sent over the wire.
// The format follows Git's receive-pack protocol:
//   - A pkt-line containing the ref update command
//   - An empty pack file (required by the protocol)
//   - A flush packet to indicate the end of the request
//
// The ref update command format is:
//
//	<old-value> <new-value> <ref-name>\000<capabilities>\n
//
// The old-value and new-value fields have specific meanings depending on the operation:
//   - Create: old-value is ZeroHash, new-value is the target hash
//   - Update: old-value is the current hash, new-value is the target hash
//   - Delete: old-value is the current hash, new-value is ZeroHash
//
// Returns:
//   - A byte slice containing the formatted request
//   - Any error that occurred during formatting
//
// Examples:
//
//	Create refs/heads/main pointing to 1234...:
//	"0000... 1234... refs/heads/main\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n"
//
//	Update refs/heads/main from 1234... to 5678...:
//	"1234... 5678... refs/heads/main\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n"
//
//	Delete refs/heads/main:
//	"1234... 0000... refs/heads/main\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n"
func (r RefUpdateRequest) Format() ([]byte, error) {
	// Validate hash lengths
	if len(r.OldRef) != 40 && r.OldRef != ZeroHash {
		return nil, fmt.Errorf("invalid old ref hash length: got %d, want 40", len(r.OldRef))
	}
	if len(r.NewRef) != 40 && r.NewRef != ZeroHash {
		return nil, fmt.Errorf("invalid new ref hash length: got %d, want 40", len(r.NewRef))
	}

	// Create the ref using receive-pack
	// Format: <old-value> <new-value> <ref-name>\000<capabilities>\n
	refLine := fmt.Sprintf("%s %s %s\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n", r.OldRef, r.NewRef, r.RefName)

	// Calculate the correct length (including the 4 bytes of the length field)
	lineLen := len(refLine) + 4
	pkt := make([]byte, 0, lineLen+4)
	pkt = fmt.Appendf(pkt, "%04x%s0000", lineLen, refLine)

	// Send pack file as raw data (not as a pkt-line)
	// It seems we need to send the empty pack even if it's not needed.
	pkt = append(pkt, EmptyPack...)

	// Add final flush packet
	pkt = append(pkt, FlushPacket...)

	return pkt, nil
}
