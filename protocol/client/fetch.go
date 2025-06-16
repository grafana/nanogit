package client

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/storage"
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
	logger.Debug("Fetch", "wantCount", len(opts.Want), "noCache", opts.NoCache)
	objects := make(map[string]*protocol.PackfileObject)

	storage := storage.FromContext(ctx)
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
			logger.Debug("All objects found in cache", "objectCount", len(objects))
			return objects, nil
		}

		logger.Debug("Some objects not found in cache", "foundInCache", len(objects), "pendingFetch", len(pending))
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

	logger.Debug("Send fetch request",
		"requestSize", len(pkt),
		"options", map[string]interface{}{
			"noProgress":   opts.NoProgress,
			"noBlobFilter": opts.NoBlobFilter,
			"deepen":       opts.Deepen,
			"shallow":      opts.Shallow,
			"done":         opts.Done,
		})
	logger.Debug("Fetch request raw data", "request", string(pkt))

	out, err := c.UploadPack(ctx, pkt)
	if err != nil {
		return nil, fmt.Errorf("sending commands: %w", err)
	}

	logger.Debug("Received server response", "responseSize", len(out))
	logger.Debug("Server response raw data", "response", hex.EncodeToString(out))

	lines, _, err := protocol.ParsePack(out)
	if err != nil {
		logger.Debug("Failed to parse pack", "error", err)
		return nil, fmt.Errorf("parsing packet: %w", err)
	}

	logger.Debug("Parse pack lines", "lineCount", len(lines))
	response, err := protocol.ParseFetchResponse(lines)
	if err != nil {
		return nil, fmt.Errorf("parsing fetch response: %w", err)
	}

	objectCount := 0
	for {
		obj, err := response.Packfile.ReadObject()
		if err != nil {
			logger.Debug("Finished reading objects", "error", err, "totalObjects", objectCount)
			break
		}
		if obj.Object == nil {
			break
		}

		if storage != nil {
			storage.Add(obj.Object)
		}

		objects[obj.Object.Hash.String()] = obj.Object
		objectCount++
	}

	logger.Debug("Fetch completed", "totalObjects", len(objects))
	return objects, nil
}
