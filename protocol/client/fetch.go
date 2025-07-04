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

	cachedObjects, pendingOpts := c.checkCacheForObjects(ctx, opts, objects, storage)
	if cachedObjects {
		return objects, nil
	}

	pkt, err := c.buildFetchRequest(pendingOpts)
	if err != nil {
		return nil, err
	}

	c.logFetchRequest(logger, pkt, pendingOpts)

	response, err := c.sendFetchRequest(ctx, pkt)
	if err != nil {
		return nil, err
	}

	err = c.processPackfileResponse(ctx, response, objects, storage)
	if err != nil {
		return nil, err
	}

	logger.Debug("Fetch completed", "totalObjects", len(objects))
	return objects, nil
}

// checkCacheForObjects checks if objects are available in cache and returns cached objects
func (c *rawClient) checkCacheForObjects(ctx context.Context, opts FetchOptions, objects map[string]*protocol.PackfileObject, storage storage.PackfileStorage) (bool, FetchOptions) {
	logger := log.FromContext(ctx)
	
	if storage == nil || opts.NoCache {
		return false, opts
	}

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
		return true, opts
	}

	logger.Debug("Some objects not found in cache", "foundInCache", len(objects), "pendingFetch", len(pending))
	opts.Want = pending
	return false, opts
}

// buildFetchRequest constructs the fetch request packet
func (c *rawClient) buildFetchRequest(opts FetchOptions) ([]byte, error) {
	packs := c.buildBasicPacks(opts)
	packs = c.addWantPacks(packs, opts)
	packs = c.addOptionalPacks(packs, opts)

	return protocol.FormatPacks(packs...)
}

// buildBasicPacks creates the basic pack structure
func (c *rawClient) buildBasicPacks(opts FetchOptions) []protocol.Pack {
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

	return packs
}

// addWantPacks adds want and shallow pack lines
func (c *rawClient) addWantPacks(packs []protocol.Pack, opts FetchOptions) []protocol.Pack {
	for _, want := range opts.Want {
		packs = append(packs, protocol.PackLine(fmt.Sprintf("want %s\n", want.String())))
		if opts.Shallow {
			packs = append(packs, protocol.PackLine(fmt.Sprintf("shallow %s\n", want.String())))
		}
	}
	return packs
}

// addOptionalPacks adds optional pack lines and finishes the pack
func (c *rawClient) addOptionalPacks(packs []protocol.Pack, opts FetchOptions) []protocol.Pack {
	if opts.Deepen > 0 {
		packs = append(packs, protocol.PackLine(fmt.Sprintf("deepen %d\n", opts.Deepen)))
	}

	if opts.Shallow {
		for _, want := range opts.Want {
			packs = append(packs, protocol.PackLine(fmt.Sprintf("shallow %s\n", want.String())))
		}
	}

	if opts.Done {
		packs = append(packs, protocol.PackLine("done\n"))
	}
	
	return append(packs, protocol.FlushPacket)
}

// logFetchRequest logs the fetch request details
func (c *rawClient) logFetchRequest(logger log.Logger, pkt []byte, opts FetchOptions) {
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
}

// sendFetchRequest sends the fetch request and parses the response
func (c *rawClient) sendFetchRequest(ctx context.Context, pkt []byte) (*protocol.FetchResponse, error) {
	logger := log.FromContext(ctx)
	
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

	return response, nil
}

// processPackfileResponse processes the packfile response and extracts objects
func (c *rawClient) processPackfileResponse(ctx context.Context, response *protocol.FetchResponse, objects map[string]*protocol.PackfileObject, storage storage.PackfileStorage) error {
	logger := log.FromContext(ctx)
	
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

	return nil
}
