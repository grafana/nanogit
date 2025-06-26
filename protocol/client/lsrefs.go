package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
)

type LsRefsOptions struct {
	Prefix string
}

func (c *rawClient) LsRefs(ctx context.Context, opts LsRefsOptions) ([]protocol.RefLine, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Ls-refs", "prefix", opts.Prefix)

	// Send the ls-refs command directly - Protocol v2 allows this without needing
	// a separate capability advertisement request
	packs := []protocol.Pack{
		protocol.PackLine("command=ls-refs\n"),
		protocol.PackLine("object-format=sha1\n"),
	}

	if opts.Prefix != "" {
		packs = append(packs, protocol.DelimeterPacket)
		packs = append(packs, protocol.PackLine(fmt.Sprintf("ref-prefix %s\n", opts.Prefix)))
	}

	packs = append(packs, protocol.FlushPacket)
	pkt, err := protocol.FormatPacks(packs...)
	if err != nil {
		return nil, fmt.Errorf("format ls-refs command: %w", err)
	}

	logger.Debug("Send Ls-refs request", "requestSize", len(pkt))
	logger.Debug("Ls-refs raw request", "request", string(pkt))

	refsReader, err := c.UploadPack(ctx, bytes.NewReader(pkt))
	if err != nil {
		return nil, fmt.Errorf("send ls-refs command: %w", err)
	}

	logger.Debug("Parsing ls-refs response")
	refs, err := c.parseRefs(refsReader)
	if err != nil {
		return nil, fmt.Errorf("parse refs response: %w", err)
	}

	logger.Debug("Ls-refs completed", "refCount", len(refs))
	return refs, nil
}

// parseRefs parses the ls-refs response one packet at a time.
func (c *rawClient) parseRefs(reader io.ReadCloser) ([]protocol.RefLine, error) {
	defer reader.Close()

	refs := make([]protocol.RefLine, 0)
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
				return nil, fmt.Errorf("reading packet data: %w", err)
			}

			// Parse this packet as a ref line
			refLine, err := protocol.ParseRefLine(packetData)
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
