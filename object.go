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

func (c *httpClient) getObject(ctx context.Context, hash hash.Hash) (*protocol.PackfileObject, error) {
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

	c.logger.Debug("Fetch request", "object", hash.String(), "request", string(pkt))

	out, err := c.uploadPack(ctx, pkt)
	if err != nil {
		c.logger.Debug("UploadPack error", "object", hash.String(), "error", err)
		if strings.Contains(err.Error(), "not our ref") {
			return nil, errors.New("object not found")
		}
		return nil, fmt.Errorf("sending commands: %w", err)
	}

	c.logger.Debug("Raw server response", "object", hash.String(), "response", hex.EncodeToString(out))

	lines, _, err := protocol.ParsePack(out)
	if err != nil {
		c.logger.Debug("ParsePack error", "object", hash.String(), "error", err)
		return nil, fmt.Errorf("parsing packet: %w", err)
	}

	c.logger.Debug("Parsed lines", "object", hash.String(), "lines", lines)

	response, err := protocol.ParseFetchResponse(lines)
	if err != nil {
		c.logger.Debug("ParseFetchResponse error", "object", hash.String(), "error", err)
		return nil, fmt.Errorf("parsing fetch response: %w", err)
	}

	found := false
	var foundObj *protocol.PackfileObject
	var allObjs []string
	for {
		obj, err := response.Packfile.ReadObject()
		if err != nil {
			c.logger.Debug("ReadObject error", "object", hash.String(), "error", err)
			break
		}
		if obj.Object == nil {
			break
		}

		allObjs = append(allObjs, fmt.Sprintf("%s (%v)", obj.Object.Hash.String(), obj.Object.Type))

		if obj.Object.Hash.Is(hash) {
			found = true
			foundObj = obj.Object
			c.logger.Debug("Found object", "object", hash.String(), "type", obj.Object.Type)
			break
		}
	}

	if !found {
		c.logger.Debug("Object not found in packfile. Objects in packfile:", "object", hash.String(), "objects", allObjs)
		return nil, errors.New("object not found")
	}

	return foundObj, nil
}
