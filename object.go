package nanogit

import (
	"context"
	"errors"
	"fmt"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/protocol/object"
)

type Object struct {
	Hash hash.Hash
	Type object.Type
	Data []byte
}

func (c *clientImpl) GetObject(ctx context.Context, hash hash.Hash) (*Object, error) {
	pkt, err := protocol.FormatPacks(
		protocol.PackLine("command=fetch\n"),
		protocol.PackLine("object-format=sha1\n"),
		protocol.SpecialPack(protocol.DelimeterPacket),
		protocol.PackLine("no-progress\n"),
		protocol.PackLine("filter blob:none\n"),
		protocol.PackLine(fmt.Sprintf("want %s\n", hash.String())),
		protocol.PackLine("done\n"),
	)
	if err != nil {
		return nil, fmt.Errorf("formatting packets: %w", err)
	}

	out, err := c.UploadPack(ctx, pkt)
	if err != nil {
		return nil, fmt.Errorf("sending commands: %w", err)
	}

	lines, _, err := protocol.ParsePack(out)
	if err != nil {
		return nil, fmt.Errorf("parsing packet: %w", err)
	}

	response, err := protocol.ParseFetchResponse(lines)
	if err != nil {
		return nil, fmt.Errorf("parsing fetch response: %w", err)
	}

	for {
		obj, err := response.Packfile.ReadObject()
		if err != nil {
			return nil, fmt.Errorf("reading object: %w", err)
		}
		if obj.Object == nil {
			break
		}

		if obj.Object.Hash.Is(hash) {
			return &Object{Hash: obj.Object.Hash, Type: obj.Object.Type, Data: obj.Object.Data}, nil
		}
	}

	return nil, errors.New("object not found")
}
