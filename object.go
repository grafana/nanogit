package nanogit

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

func (c *httpClient) getTree(ctx context.Context, want hash.Hash) (*protocol.PackfileObject, error) {
	packs := []protocol.Pack{
		protocol.PackLine("command=fetch\n"),
		protocol.PackLine("object-format=sha1\n"),
		protocol.SpecialPack(protocol.DelimeterPacket),
		protocol.PackLine("no-progress\n"),
		protocol.PackLine("filter blob:none\n"),
		protocol.PackLine(fmt.Sprintf("want %s\n", want.String())),
		protocol.PackLine("done\n"),
	}

	pkt, err := protocol.FormatPacks(packs...)
	if err != nil {
		return nil, fmt.Errorf("formatting packets: %w", err)
	}

	c.logger.Debug("Specific fetch request", "want", want, "request", string(pkt))

	out, err := c.uploadPack(ctx, pkt)
	if err != nil {
		c.logger.Debug("UploadPack error", "want", want, "error", err)
		if strings.Contains(err.Error(), "not our ref") {
			return nil, NewObjectNotFoundError(want)
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

		if obj.Object.Type != protocol.ObjectTypeTree {
			return nil, NewUnexpectedObjectTypeError(want, protocol.ObjectTypeTree, obj.Object.Type)
		}

		objects[obj.Object.Hash.String()] = obj.Object
	}

	// Due to Git protocol limitations, when fetching a tree object, we receive all tree objects
	// in the path. We must filter the response to extract only the requested tree.
	if obj, ok := objects[want.String()]; ok {
		return obj, nil
	}

	return nil, NewObjectNotFoundError(want)
}

func (c *httpClient) getCommit(ctx context.Context, want hash.Hash) (*protocol.PackfileObject, error) {
	packs := []protocol.Pack{
		protocol.PackLine("command=fetch\n"),
		protocol.PackLine("object-format=sha1\n"),
		protocol.SpecialPack(protocol.DelimeterPacket),
		protocol.PackLine("no-progress\n"),
		protocol.PackLine("filter blob:none\n"),
		protocol.PackLine(fmt.Sprintf("want %s\n", want.String())),
		protocol.PackLine(fmt.Sprintf("shallow %s\n", want.String())),
		protocol.PackLine("deepen 1\n"),
		protocol.PackLine("done\n"),
	}

	pkt, err := protocol.FormatPacks(packs...)
	if err != nil {
		return nil, fmt.Errorf("formatting packets: %w", err)
	}

	c.logger.Debug("Specific fetch request", "want", want, "request", string(pkt))

	out, err := c.uploadPack(ctx, pkt)
	if err != nil {
		c.logger.Debug("UploadPack error", "want", want, "error", err)
		if strings.Contains(err.Error(), "not our ref") {
			return nil, NewObjectNotFoundError(want)
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

	objects := make([]*protocol.PackfileObject, 0)
	var foundObj *protocol.PackfileObject
	for {
		obj, err := response.Packfile.ReadObject()
		if err != nil {
			c.logger.Debug("ReadObject error", "want", want, "error", err)
			break
		}
		if obj.Object == nil {
			break
		}

		// Skip tree objects that are included in the response despite the blob:none filter.
		// Most Git servers don't support tree:0 filter specification, so we may receive
		// recursive tree objects that we need to filter out.
		if obj.Object.Type == protocol.ObjectTypeTree {
			continue
		}

		if obj.Object.Type != protocol.ObjectTypeCommit {
			return nil, NewUnexpectedObjectTypeError(want, protocol.ObjectTypeCommit, obj.Object.Type)
		}

		objects = append(objects, obj.Object)
		if obj.Object.Hash.Is(want) {
			foundObj = obj.Object
		}
	}

	// we got more commits than expected
	if len(objects) > 1 {
		return nil, NewUnexpectedObjectCountError(1, objects)
	}

	if foundObj != nil {
		return foundObj, nil
	}

	return nil, NewObjectNotFoundError(want)
}

func (c *httpClient) getBlob(ctx context.Context, want hash.Hash) (*protocol.PackfileObject, error) {
	packs := []protocol.Pack{
		protocol.PackLine("command=fetch\n"),
		protocol.PackLine("object-format=sha1\n"),
		protocol.SpecialPack(protocol.DelimeterPacket),
		protocol.PackLine("no-progress\n"),
		protocol.PackLine(fmt.Sprintf("want %s\n", want.String())),
		protocol.PackLine("done\n"),
	}

	pkt, err := protocol.FormatPacks(packs...)
	if err != nil {
		return nil, fmt.Errorf("formatting packets: %w", err)
	}

	c.logger.Debug("Specific fetch request", "want", want, "request", string(pkt))

	out, err := c.uploadPack(ctx, pkt)
	if err != nil {
		c.logger.Debug("UploadPack error", "want", want, "error", err)
		if strings.Contains(err.Error(), "not our ref") {
			return nil, NewObjectNotFoundError(want)
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

	objects := make([]*protocol.PackfileObject, 0)
	var foundObj *protocol.PackfileObject
	for {
		obj, err := response.Packfile.ReadObject()
		if err != nil {
			c.logger.Debug("ReadObject error", "want", want, "error", err)
			break
		}
		if obj.Object == nil {
			break
		}

		if obj.Object.Type != protocol.ObjectTypeBlob {
			return nil, NewUnexpectedObjectTypeError(want, protocol.ObjectTypeBlob, obj.Object.Type)
		}

		objects = append(objects, obj.Object)
		if obj.Object.Hash.Is(want) {
			foundObj = obj.Object
		}
	}

	// we got more commits than expected
	if len(objects) > 1 {
		return nil, NewUnexpectedObjectCountError(1, objects)
	}

	if foundObj != nil {
		return foundObj, nil
	}

	return nil, NewObjectNotFoundError(want)
}

// getRootTree fetches the root tree of the repository.
func (c *httpClient) getCommitTree(ctx context.Context, commitHash hash.Hash) (map[string]*protocol.PackfileObject, error) {
	packs := []protocol.Pack{
		protocol.PackLine("command=fetch\n"),
		protocol.PackLine("object-format=sha1\n"),
		protocol.SpecialPack(protocol.DelimeterPacket),
		protocol.PackLine("no-progress\n"),
		protocol.PackLine("filter blob:none\n"),
		protocol.PackLine(fmt.Sprintf("want %s\n", commitHash.String())),
		protocol.PackLine(fmt.Sprintf("shallow %s\n", commitHash.String())),
		protocol.PackLine("done\n"),
	}

	pkt, err := protocol.FormatPacks(packs...)
	if err != nil {
		return nil, fmt.Errorf("formatting packets: %w", err)
	}

	c.logger.Debug("Specific fetch request", "want", commitHash, "request", string(pkt))

	out, err := c.uploadPack(ctx, pkt)
	if err != nil {
		c.logger.Debug("UploadPack error", "want", commitHash, "error", err)
		if strings.Contains(err.Error(), "not our ref") {
			return nil, NewObjectNotFoundError(commitHash)
		}
		return nil, fmt.Errorf("sending commands: %w", err)
	}

	c.logger.Debug("Raw server response", "want", commitHash, "response", hex.EncodeToString(out))

	lines, _, err := protocol.ParsePack(out)
	if err != nil {
		c.logger.Debug("ParsePack error", "want", commitHash, "error", err)
		return nil, fmt.Errorf("parsing packet: %w", err)
	}

	c.logger.Debug("Parsed lines", "want", commitHash, "lines", lines)

	response, err := protocol.ParseFetchResponse(lines)
	if err != nil {
		c.logger.Debug("ParseFetchResponse error", "want", commitHash, "error", err)
		return nil, fmt.Errorf("parsing fetch response: %w", err)
	}

	objects := make(map[string]*protocol.PackfileObject)
	for {
		obj, err := response.Packfile.ReadObject()
		if err != nil {
			c.logger.Debug("ReadObject error", "want", commitHash, "error", err)
			break
		}
		if obj.Object == nil {
			break
		}

		objects[obj.Object.Hash.String()] = obj.Object
	}

	return objects, nil
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
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("send commands: %w", err)
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
