package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/protocol/object"
)

func (c *clientImpl) GetBlob(ctx context.Context, blobID hash.Hash) ([]byte, error) {
	// Format the fetch request
	pkt, err := protocol.FormatPacks(
		protocol.PackLine("command=fetch\n"),
		protocol.PackLine("object-format=sha1\n"),
		protocol.DelimeterPacket,
		protocol.PackLine("no-progress\n"),
		protocol.PackLine(fmt.Sprintf("want %s\n", blobID.String())),
		protocol.PackLine("done\n"),
	)
	if err != nil {
		return nil, fmt.Errorf("formatting packets: %w", err)
	}

	// Send the request
	out, err := c.UploadPack(ctx, pkt)
	if err != nil {
		return nil, fmt.Errorf("sending commands: %w", err)
	}

	// Parse the response
	lines, _, err := protocol.ParsePack(out)
	if err != nil {
		if strings.Contains(err.Error(), "not our ref "+blobID.String()) {
			return nil, fmt.Errorf("blob not found: %w", err)
		}

		return nil, fmt.Errorf("parsing packet: %w", err)
	}

	response, err := protocol.ParseFetchResponse(lines)
	if err != nil {
		return nil, fmt.Errorf("parsing fetch response: %w", err)
	}

	// Find the blob in the packfile
	for {
		obj, err := response.Packfile.ReadObject()
		if err != nil {
			return nil, fmt.Errorf("reading object: %w", err)
		}
		if obj.Object == nil {
			break
		}
		if obj.Object.Type == object.TypeBlob && obj.Object.Hash.Is(blobID) {
			return obj.Object.Data, nil
		}
	}

	return nil, fmt.Errorf("blob not found: %s", blobID)
}
