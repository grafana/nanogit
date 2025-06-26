package protocol

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/grafana/nanogit/log"
)

// ParseLsRefsResponse parses the ls-refs response one packet at a time.
func ParseLsRefsResponse(ctx context.Context, reader io.ReadCloser) ([]RefLine, error) {
	logger := log.FromContext(ctx)

	defer reader.Close()

	refs := make([]RefLine, 0)
	for {
		// Read packet length (4 hex bytes)
		lengthBytes := make([]byte, 4)
		_, err := io.ReadFull(reader, lengthBytes)
		if err != nil {
			if err == io.EOF {
				// End of stream
				return refs, nil
			}

			return nil, fmt.Errorf("reading packet length: %w", err)
		}

		length, err := strconv.ParseUint(string(lengthBytes), 16, 16)
		if err != nil {
			return nil, fmt.Errorf("parsing packet length: %w", err)
		}

		// Handle different packet types
		switch {
		case length < 4:
			// Special packets (flush, delimiter, response-end)
			if length == 0 {
				// Flush packet (0000) - end of response
				return refs, nil
			}
			// Other special packets - continue reading
			continue

		case length == 4:
			// Empty packet - continue
			continue
		default:
			// Read packet data
			dataLength := length - 4
			packetData := make([]byte, dataLength)
			if _, err := io.ReadFull(reader, packetData); err != nil {
				if err == io.ErrUnexpectedEOF {
					return nil, fmt.Errorf("line declared %d bytes but unexpected EOF occurred", dataLength)
				}

				return nil, fmt.Errorf("reading packet data: %w", err)
			}

			logger.Debug("Parsing ls-refs packet",
				"packet_data", string(packetData),
				"data_length", dataLength)

			// Parse this packet as a ref line
			refLine, err := ParseRefLine(packetData)
			if err != nil {
				return nil, fmt.Errorf("parse ref line: %w", err)
			}

			// Only add non-empty ref names
			if refLine.RefName != "" {
				refs = append(refs, refLine)
			}
		}
	}
}
