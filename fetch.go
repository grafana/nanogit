package nanogit

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

type FetchOptions struct {
	NoCache      bool
	NoProgress   bool
	NoBlobFilter bool
	Want         []hash.Hash
	Done         bool // not sure why we need this one
	Deepen       int
	Shallow      bool
}

func (c *rawClient) Fetch(ctx context.Context, opts FetchOptions) (map[string]*protocol.PackfileObject, error) {
	logger := log.FromContext(ctx)
	objects := make(map[string]*protocol.PackfileObject)

	storage := c.getPackfileStorage(ctx)
	if storage != nil && !opts.NoCache {
		pending := make([]hash.Hash, 0, len(opts.Want))
		for _, want := range opts.Want {
			obj, ok := storage.Get(want)
			if !ok {
				pending = append(pending, want)
			} else {
				objects[want.String()] = obj
			}
		}

		if len(objects) == len(opts.Want) {
			return objects, nil
		}

		opts.Want = pending
	}

	packs := []protocol.Pack{
		protocol.PackLine("command=fetch\n"),
		protocol.PackLine("object-format=sha1\n"),
		protocol.SpecialPack(protocol.DelimeterPacket),
	}

	if opts.NoProgress {
		packs = append(packs, protocol.PackLine("no-progress\n"))
	}

	if opts.NoBlobFilter {
		packs = append(packs, protocol.PackLine("filter blob:none\n"))
	}

	for _, want := range opts.Want {
		packs = append(packs, protocol.PackLine(fmt.Sprintf("want %s\n", want.String())))
		// FIXME: does this make sense for multiple wants? or only for a single one?
		if opts.Shallow {
			packs = append(packs, protocol.PackLine(fmt.Sprintf("shallow %s\n", want.String())))
		}
	}

	if opts.Deepen > 0 {
		packs = append(packs, protocol.PackLine(fmt.Sprintf("deepen %d\n", opts.Deepen)))
	}

	if opts.Shallow {
		// FIXME: does this make sense for multiple wants? or only for a single one?
		for _, want := range opts.Want {
			packs = append(packs, protocol.PackLine(fmt.Sprintf("shallow %s\n", want.String())))
		}
	}

	if opts.Done {
		packs = append(packs, protocol.PackLine("done\n"))
	}
	packs = append(packs, protocol.FlushPacket)

	pkt, err := protocol.FormatPacks(packs...)
	if err != nil {
		return nil, fmt.Errorf("formatting packets: %w", err)
	}

	out, err := c.UploadPack(ctx, pkt)
	if err != nil {
		return nil, fmt.Errorf("sending commands: %w", err)
	}

	logger.Debug("Raw server response", "response", hex.EncodeToString(out))
	logger.Debug("Specific fetch request", "request", string(pkt))

	lines, _, err := protocol.ParsePack(out)
	if err != nil {
		logger.Debug("ParsePack error", "error", err)
		return nil, fmt.Errorf("parsing packet: %w", err)
	}

	logger.Debug("Parsed lines", "lines", lines)
	response, err := protocol.ParseFetchResponse(lines)
	if err != nil {
		logger.Debug("ParseFetchResponse error", "error", err)
		return nil, fmt.Errorf("parsing fetch response: %w", err)
	}

	for {
		obj, err := response.Packfile.ReadObject()
		if err != nil {
			logger.Debug("ReadObject error", "error", err)
			break
		}
		if obj.Object == nil {
			break
		}

		storage := c.getPackfileStorage(ctx)
		if storage != nil {
			storage.Add(obj.Object)
		}

		objects[obj.Object.Hash.String()] = obj.Object
	}

	if len(objects) == 0 {
		return nil, ErrObjectNotFound
	}

	return objects, nil
}
