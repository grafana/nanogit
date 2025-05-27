package nanogit

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

func (c *httpClient) getSingleObject(ctx context.Context, want hash.Hash) (*protocol.PackfileObject, error) {
	objects, err := c.getObjects(ctx, want)
	if err != nil {
		return nil, err
	}

	if obj, ok := objects[want.String()]; ok {
		return obj, nil
	}

	return nil, fmt.Errorf("object %s not found: %w", want.String(), ErrRefNotFound)
}

func (c *httpClient) getObjects(ctx context.Context, want ...hash.Hash) (map[string]*protocol.PackfileObject, error) {
	packs := []protocol.Pack{
		protocol.PackLine("command=fetch\n"),
		protocol.PackLine("object-format=sha1\n"),
		protocol.SpecialPack(protocol.DelimeterPacket),
		protocol.PackLine("no-progress\n"),
		protocol.PackLine("filter blob:none\n"),
	}

	for _, w := range want {
		packs = append(packs, protocol.PackLine(fmt.Sprintf("want %s\n", w.String())))
	}
	packs = append(packs, protocol.PackLine("done\n"))

	pkt, err := protocol.FormatPacks(packs...)
	if err != nil {
		return nil, fmt.Errorf("formatting packets: %w", err)
	}

	c.logger.Debug("Fetch request", "want", want, "request", string(pkt))

	out, err := c.uploadPack(ctx, pkt)
	if err != nil {
		c.logger.Debug("UploadPack error", "want", want, "error", err)
		if strings.Contains(err.Error(), "not our ref") {
			return nil, errors.New("object not found")
		}
		return nil, fmt.Errorf("sending commands: %w", err)
	}

	c.logger.Debug("Raw server response", "want", want, "response", hex.EncodeToString(out))

	lines, _, err := protocol.ParsePack(out)
	if err != nil {
		c.logger.Debug("ParsePack error", "want", want, "error", err)
		return nil, fmt.Errorf("parsing packet: %w", err)
	}

	c.logger.Debug("Parsed lines", "want", want, "lines", lines)

	response, err := protocol.ParseFetchResponse(lines)
	if err != nil {
		c.logger.Debug("ParseFetchResponse error", "want", want, "error", err)
		return nil, fmt.Errorf("parsing fetch response: %w", err)
	}

	objects := make(map[string]*protocol.PackfileObject)
	for {
		obj, err := response.Packfile.ReadObject()
		if err != nil {
			c.logger.Debug("ReadObject error", "want", want, "error", err)
			break
		}
		if obj.Object == nil {
			break
		}

		objects[obj.Object.Hash.String()] = obj.Object
	}

	return objects, nil
}
