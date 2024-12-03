package protocol

import (
	"errors"
	"log/slog"
	"strings"
)

// Acknowledgements contains whether a nack ("NAK") was received, or a list of ACKs, and for which objects those apply.
// If Nack is true, Acks is always empty. If Nack is false, Acks may be non-empty.
// The objects returned in Acks are always requested. Not all requested objects are necessarily listed.
// Not all sent objects are included in the list, and it may even be empty even if a cut point is found. This is an optimisation by the Git server.
//
// Git documentation defines the format as:
//
//	acknowledgments = PKT-LINE("acknowledgments" LF)
//	    (nak | *ack)
//	    (ready)
//	ready = PKT-LINE("ready" LF)
//	nak = PKT-LINE("NAK" LF)
//	ack = PKT-LINE("ACK" SP obj-id LF)
type Acknowledgements struct {
	// Invariant: Nack == true => Acks == nil
	//            Nack == false => len(Acks) >= 0

	Nack bool
	// FIXME: Are obj-ids fine as strings? Do we want a more proper type for them?
	//    obj-id    =  40*(HEXDIGIT)
	Acks []string
}

// TODO: Do we want to parse the acknowledgements here?

type FetchResponse struct {
	// These fields are in order.
	// TODO: Do we want a session ID field? It might be useful for OTel tracing?

	Acks Acknowledgements
	// mariell: Intentionally excluding shallow-info because we don't need them right now. Maybe later?
	// mariell: Intentionally excluding wanted-refs because we don't need them right now. Maybe later?
	// mariell: Intentionally excluding packfile-uris because I can't see us needing them.

	// The packfile contains the majority of the information we want.
	//
	//	packfile section
	//	* This section is only included if the client has sent 'want'
	//	  lines in its request and either requested that no more
	//	  negotiation be done by sending 'done' or if the server has
	//	  decided it has found a sufficient cut point to produce a
	//	  packfile.
	//
	//	Always begins with the section header "packfile".
	//
	//	The transmission of the packfile begins immediately after the section header.
	//
	//	The data transfer of the packfile is always multiplexed, using the same semantics of the side-band-64k capability from protocol version 1.
	//	This means that each packet, during the packfile data stream, is made up of a leading 4-byte pkt-line length (typical of the pkt-line format), followed by a 1-byte stream code, followed by the actual data.
	//
	//	The stream code can be one of:
	//	1 - pack data
	//	2 - progress messages
	//	3 - fatal error message just before stream aborts
	Packfile *Packfile
	// When encoded, a flush-pkt is presented here.
}

type FatalFetchError string

func (e FatalFetchError) Error() string {
	return string(e)
}

var (
	ErrInvalidFetchStatus       = errors.New("invalid status in fetch packfile")
	_                     error = FatalFetchError("")
)

func ParseFetchResponse(lines [][]byte) (*FetchResponse, error) {
	fr := &FetchResponse{}
	for i, line := range lines {
		if len(line) > 30 {
			// Too long to be a section header
			continue
		}

		// We SHOULD NOT require a \n.
		switch strings.TrimSpace(string(line)) {
		case "acknowledgements":
			// TODO: Parse!
			slog.Info("next part", "part", lines[i+1])
		case "packfile":
			// These are the final pktlines. That means they're all parts of the packfile.
			// Because of this, we can just join them! We already know we don't multiplex, so they're all just streamed in multiple lines (due to the pktline size limit).
			var joined []byte
			for _, next := range lines[i+1:] {
				status := next[0]
				switch status {
				case 1: // This is the pack data.
					joined = append(joined, next[1:]...)
				case 2: // This is progress status. We don't want it.
					continue
				case 3: // This is a fatal error message.
					return nil, FatalFetchError(string(next[1:]))
				default:
					return nil, ErrInvalidFetchStatus
				}
			}

			var err error
			fr.Packfile, err = ParsePackfile(joined)
			if err != nil {
				return nil, nil
			}

			break // there is no more actionable data, so we don't need to iterate more.

		case "shallow-info", "wanted-refs":
			// Ignore.
		default:
			// Log??
		}
	}
	return fr, nil
}
