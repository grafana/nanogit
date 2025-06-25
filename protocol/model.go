package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
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
	Packfile PackfileObjectReader
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

// PackfileObjectReader defines the interface for reading objects from a packfile.
// This interface is implemented by both PackfileReader and streamingPackfileReader.
type PackfileObjectReader interface {
	ReadObject() (PackfileEntry, error)
}

// ParseFetchResponseStream parses a fetch response from a streaming reader.
// This avoids loading the entire response into memory, which is especially
// important for large packfiles. Unlike ParseFetchResponse, this function
// creates a PackfileReader directly from the stream without buffering all data.
func ParseFetchResponseStream(reader io.ReadCloser) (*FetchResponse, error) {
	fr := &FetchResponse{}

	// Create a streaming packfile reader that will handle side-band demultiplexing
	packfileReader := &streamingPackfileReader{
		reader:        reader,
		closer:        reader,
		foundPackfile: false,
	}

	// The PackfileReader will be created lazily when we encounter the packfile section
	fr.Packfile = packfileReader

	return fr, nil
}

// streamingPackfileReader implements the PackfileReader interface but works
// with a streaming reader that includes Git protocol packets and side-band data.
type streamingPackfileReader struct {
	reader        io.Reader
	closer        io.Closer
	foundPackfile bool
	packReader    *PackfileReader
	err           error
	closed        bool
}

// ReadObject implements the PackfileReader interface for streaming.
func (s *streamingPackfileReader) ReadObject() (PackfileEntry, error) {
	if s.err != nil {
		return PackfileEntry{}, s.err
	}

	// Lazily initialize the packfile reader when first called
	if !s.foundPackfile {
		s.err = s.findAndInitializePackfile()
		if s.err != nil {
			return PackfileEntry{}, s.err
		}
	}

	if s.packReader == nil {
		return PackfileEntry{}, ErrInvalidFetchStatus
	}

	entry, err := s.packReader.ReadObject()

	// Close the response body when we're done reading (on EOF or error)
	if err == io.EOF || err != nil {
		s.closeOnce()
	}

	return entry, err
}

// closeOnce ensures the response body is closed only once
func (s *streamingPackfileReader) closeOnce() {
	if !s.closed && s.closer != nil {
		s.closer.Close()
		s.closed = true
	}
}

// findAndInitializePackfile parses the protocol stream to find the packfile section
// and initializes the PackfileReader with the packfile data stream.
func (s *streamingPackfileReader) findAndInitializePackfile() error {
	for {
		// Read length header (4 hex bytes)
		lengthBytes := make([]byte, 4)
		_, err := io.ReadFull(s.reader, lengthBytes)
		if err != nil {
			if err == io.EOF {
				return err // End of stream without finding packfile
			}
			return fmt.Errorf("reading packet length: %w", err)
		}

		length, err := strconv.ParseUint(string(lengthBytes), 16, 16)
		if err != nil {
			return fmt.Errorf("parsing packet length: %w", err)
		}

		// Handle different packet types
		switch {
		case length < 4:
			// Special packets: flush (0000), delimiter (0001), response-end (0002)
			if length == 2 { // ResponseEndPacket
				return ErrInvalidFetchStatus
			}
			// Continue for other special packets

		case length == 4:
			// Empty packet - continue
			continue

		default:
			// Read packet data
			dataLength := length - 4
			packetData := make([]byte, dataLength)
			if _, err := io.ReadFull(s.reader, packetData); err != nil {
				return fmt.Errorf("reading packet data: %w", err)
			}

			// Check if this is a section header
			if len(packetData) <= 30 { // Section headers are short
				sectionName := strings.TrimSpace(string(packetData))
				if sectionName == "packfile" {
					// Found packfile section - create a side-band demultiplexing reader
					demuxReader := &sideBandReader{reader: s.reader}
					s.packReader, err = ParsePackfile(demuxReader)
					if err != nil {
						return fmt.Errorf("creating packfile reader: %w", err)
					}
					s.foundPackfile = true
					return nil
				}
				// Skip other sections (acknowledgements, shallow-info, etc.)
			}
		}
	}
}

// sideBandReader implements io.Reader and handles Git side-band demultiplexing.
// It filters out progress messages and extracts pack data from the stream.
type sideBandReader struct {
	reader io.Reader
	buffer []byte // Buffer for partial packet data
}

func (s *sideBandReader) Read(p []byte) (n int, err error) {
	// If we have buffered data from a previous call, return it first
	if len(s.buffer) > 0 {
		n = copy(p, s.buffer)
		s.buffer = s.buffer[n:]
		return n, nil
	}

	for {
		// Read packet length
		lengthBytes := make([]byte, 4)
		_, err := io.ReadFull(s.reader, lengthBytes)
		if err != nil {
			return 0, err
		}

		length, err := strconv.ParseUint(string(lengthBytes), 16, 16)
		if err != nil {
			return 0, fmt.Errorf("parsing packet length: %w", err)
		}

		// Handle packet types
		switch {
		case length < 4:
			// Special packets: continue reading
			continue
		case length == 4:
			// Empty packet: continue
			continue
		default:
			// Read packet data
			dataLength := length - 4
			packetData := make([]byte, dataLength)
			if _, err := io.ReadFull(s.reader, packetData); err != nil {
				return 0, fmt.Errorf("reading packet data: %w", err)
			}

			if len(packetData) == 0 {
				continue
			}

			// Handle side-band multiplexing
			status := packetData[0]
			switch status {
			case 1: // Pack data
				data := packetData[1:]
				if len(data) <= len(p) {
					// Data fits in the provided buffer
					copy(p, data)
					return len(data), nil
				} else {
					// Data is larger than buffer - copy what fits and save the rest
					copy(p, data[:len(p)])
					s.buffer = append(s.buffer, data[len(p):]...)
					return len(p), nil
				}
			case 2: // Progress messages - skip
				continue
			case 3: // Fatal error
				return 0, FatalFetchError(string(packetData[1:]))
			default:
				return 0, ErrInvalidFetchStatus
			}
		}
	}
}

func ParseFetchResponse(lines [][]byte) (*FetchResponse, error) {
	fr := &FetchResponse{}
outer:
	for i, line := range lines {
		if len(line) > 30 {
			// Too long to be a section header
			continue
		}

		// We SHOULD NOT require a \n.
		switch strings.TrimSpace(string(line)) {
		case "acknowledgements":
			// TODO: Parse!
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

			if len(joined) == 0 {
				return fr, nil
			}

			var err error
			fr.Packfile, err = ParsePackfile(bytes.NewReader(joined))
			if err != nil {
				return nil, err
			}

			break outer // break out of the outer loop since we've processed the packfile

		case "shallow-info", "wanted-refs":
			// Ignore.
		default:
			// TODO: what do we do here? log?
		}
	}
	return fr, nil
}
