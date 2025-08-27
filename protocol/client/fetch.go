package client

import (
	"bytes"
	"context"
	"fmt"
	"io"

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
	// OnObjectFetched is called for each object fetched.
	OnObjectFetched func(ctx context.Context, obj *protocol.PackfileObject) (stop bool, err error)
}

func (c *rawClient) Fetch(ctx context.Context, opts FetchOptions) error {
	logger := log.FromContext(ctx)
	logger.Debug("Fetch", "wantCount", len(opts.Want), "noCache", opts.NoCache)

	storage := storage.FromContext(ctx)
	cachedObjects, pendingOpts, err := c.checkCacheForObjects(ctx, opts, storage)
	if err != nil {
		return fmt.Errorf("check cache for objects: %w", err)
	} else if cachedObjects {
		return nil
	}

	pkt, err := c.buildFetchRequest(pendingOpts)
	if err != nil {
		return err
	}

	c.logFetchRequest(logger, pkt, pendingOpts)

	responseReader, response, err := c.sendFetchRequest(ctx, pkt)
	if err != nil {
		return err
	}

	if responseReader != nil {
		defer func() {
			if closeErr := responseReader.Close(); closeErr != nil {
				logger.Error("error closing response reader", "error", closeErr)
			}
		}()
	}

	err = c.processPackfileResponse(ctx, response, storage, pendingOpts)
	if err != nil {
		return err
	}

	return nil
}

// checkCacheForObjects checks if objects are available in cache and returns cached objects
func (c *rawClient) checkCacheForObjects(ctx context.Context, opts FetchOptions, storage storage.PackfileStorage) (bool, FetchOptions, error) {
	logger := log.FromContext(ctx)
	if storage == nil || opts.NoCache {
		return false, opts, nil
	}

	pending := make([]hash.Hash, 0, len(opts.Want))
	var found int
	for _, want := range opts.Want {
		obj, ok := storage.Get(want)
		if !ok {
			pending = append(pending, want)
		} else {
			found++
			if opts.OnObjectFetched != nil {
				stop, err := opts.OnObjectFetched(ctx, obj)
				if err != nil {
					logger.Error("fetch callback error", "error", err, "object", want.String())
					return true, opts, fmt.Errorf("fetch callback: %w", err)
				}

				if stop {
					logger.Debug("Fetch callback requested to stop early", "object", want.String())
					return true, opts, nil
				}
			}
		}
	}

	if found == len(opts.Want) {
		logger.Debug("All objects found in cache", "objectCount", len(opts.Want))
		return true, opts, nil
	}

	logger.Debug("Some objects not found in cache", "foundInCache", found, "pendingFetch", len(pending))
	opts.Want = pending
	return false, opts, nil
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
func (c *rawClient) sendFetchRequest(ctx context.Context, pkt []byte) (io.ReadCloser, *protocol.FetchResponse, error) {
	responseReader, err := c.UploadPack(ctx, bytes.NewReader(pkt))
	if err != nil {
		return nil, nil, fmt.Errorf("sending commands: %w", err)
	}

	parser := protocol.NewParser(responseReader)
	response, err := protocol.ParseFetchResponse(ctx, parser)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing fetch response stream: %w", err)
	}

	return responseReader, response, nil
}

// processPackfileResponse processes the packfile response and extracts objects
func (c *rawClient) processPackfileResponse(ctx context.Context, response *protocol.FetchResponse, storage storage.PackfileStorage, opts FetchOptions) error {
	logger := log.FromContext(ctx)
	var count, objectCount, foundWantedCount, totalDelta int
	wanted := make(map[string]struct{}, len(opts.Want))
	for _, w := range opts.Want {
		wanted[w.String()] = struct{}{}
	}

	for {
		obj, err := response.Packfile.ReadObject(ctx)
		if err != nil {
			logger.Debug("Finished reading objects", "error", err, "totalObjects", objectCount, "foundWanted", foundWantedCount, "totalDeltas", totalDelta)
			break
		}

		if obj.Object == nil {
			break
		}
		count++

		// Skip delta objects as we only want to process full objects and we cannot apply them
		if obj.Object.Type == protocol.ObjectTypeRefDelta {
			totalDelta++
			logger.Debug("Skipping delta object", "hash", obj.Object.Hash.String())
			continue
		}

		if storage != nil {
			storage.Add(obj.Object)
		}

		objectCount++
		if _, ok := wanted[obj.Object.Hash.String()]; ok {
			foundWantedCount++
			delete(wanted, obj.Object.Hash.String())
		}

		if opts.OnObjectFetched != nil {
			stop, err := opts.OnObjectFetched(ctx, obj.Object)
			if err != nil {
				return fmt.Errorf("fetch callback: %w", err)
			}

			if stop {
				break
			}
		}
	}

	logger.Debug("Finished processing fetch response", "totalRead", count, "totalObjects", objectCount, "foundWanted", foundWantedCount, "totalDeltas", totalDelta)

	return nil
}
