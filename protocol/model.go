package protocol

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/grafana/nanogit/log"
)

// Acknowledgements contains whether a nack ("NAK") was received, or a list of ACKs, and for which objects those apply.
// If Nack is true, Acks is always empty. If Nack is false, Acks may be non-empty.
// The objects returned in Acks are always requested. Not all requested objects are necessarily listed.
// Not all sent objects are included in the list, and it may even be empty even if a cut point is found. This is an optimisation by the Git server.
//
// [Git documentation][protocol_fetch] defines the format as:
//
//	acknowledgments = PKT-LINE("acknowledgments" LF)
//	    (nak | *ack)
//	    (ready)
//	ready = PKT-LINE("ready" LF)
//	nak = PKT-LINE("NAK" LF)
//	ack = PKT-LINE("ACK" SP obj-id LF)
//
// [protocol_fetch]: https://git-scm.com/docs/protocol-v2#_fetch
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
	Packfile *PackfileReader
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

func ParseFetchResponse(ctx context.Context, parser *Parser) (response *FetchResponse, err error) {
	logger := log.FromContext(ctx)
	logger.Debug("Starting fetch response parsing")

	fr := &FetchResponse{}
	sectionCount := 0

outer:
	for {
		sectionCount++
		logger.Debug("Reading next section", "section_number", sectionCount)

		line, err := parser.Next()
		if err != nil {
			if err == io.EOF {
				logger.Debug("Reached end of response", "total_sections", sectionCount-1)
				break
			}

			logger.Debug("Error reading next line", "error", err, "section_number", sectionCount)
			return nil, err
		}

		logger.Debug("Received line", "line_content", strings.TrimSpace(string(line)), "line_length", len(line), "section_number", sectionCount)

		if len(line) > 30 {
			// Too long to be a section header
			logger.Debug("Line too long to be section header, skipping", "line_length", len(line))
			continue
		}

		// We SHOULD NOT require a \n.
		sectionType := strings.TrimSpace(string(line))
		logger.Debug("Processing section", "section_type", sectionType, "section_number", sectionCount)

		switch sectionType {
		case "acknowledgements":
			logger.Debug("Processing acknowledgements section")
			// TODO: Parse!
		case "packfile":
			logger.Debug("Processing packfile section")
			var err error
			fr.Packfile, err = ParsePackfileFromParser(parser)
			if err != nil {
				logger.Debug("Error parsing packfile", "error", err)
				return nil, err
			}

			if fr.Packfile == nil {
				logger.Debug("No packfile data collected, returning empty response")
				return fr, nil
			}

			logger.Debug("Successfully parsed packfile")
			break outer // break out of the outer loop since we've processed the packfile
		case "shallow-info", "wanted-refs":
			logger.Debug("Ignoring section", "section_type", sectionType)
			// Ignore.
		default:
			logger.Debug("Unknown section type encountered", "section_type", sectionType, "section_number", sectionCount)
			// TODO: what do we do here? log?
		}
	}

	logger.Debug("Completed fetch response parsing", "total_sections_processed", sectionCount-1)

	return fr, nil
}
