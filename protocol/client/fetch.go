package client

import (
	"bytes"
	"context"
	"crypto"
	"errors"
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

	// NoExtraObjects stops reading the packfile once all wanted objects have been found.
	// This can significantly improve performance when fetching specific objects from large repositories,
	// as it avoids downloading and processing unnecessary objects.
	NoExtraObjects bool

	// MaxResponseBytes caps the upload-pack response body (the packfile
	// stream the server returns) before the parser starts consuming it.
	// 0 disables the cap. High-level callers select an appropriate value
	// from options.Limits based on whether the fetch targets a single
	// object (Limits.SingleObjectFetchMaxBytes) or many
	// (Limits.MultiObjectFetchMaxBytes).
	MaxResponseBytes int64
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

	responseReader, response, err := c.sendFetchRequest(ctx, pkt, pendingOpts.MaxResponseBytes)
	if err != nil {
		return nil, err
	}

	if responseReader != nil {
		defer func() {
			if closeErr := responseReader.Close(); closeErr != nil {
				logger.Error("error closing response reader", "error", closeErr)
			}
		}()
	}

	err = c.processPackfileResponse(ctx, response, objects, storage, pendingOpts)
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
			"noProgress":     opts.NoProgress,
			"noBlobFilter":   opts.NoBlobFilter,
			"deepen":         opts.Deepen,
			"shallow":        opts.Shallow,
			"done":           opts.Done,
			"noExtraObjects": opts.NoExtraObjects,
		})
	logger.Debug("Fetch request raw data", "request", string(pkt))
}

// sendFetchRequest sends the fetch request and parses the response.
// maxBytes caps the response body before parsing; 0 disables the cap.
//
// On a parse error the response body is closed before returning so it
// is not leaked: the caller's "responseReader != nil" defer is skipped
// because we return a nil reader on the error path. (Closing here does
// NOT enable HTTP connection reuse — net/http requires the body to be
// read to EOF for that — but it does release the body's resources and
// any active streaming socket.) The oversize-cap path makes this more
// reachable since truncated-by-cap responses surface as parse errors
// while the underlying body still has unread bytes.
func (c *rawClient) sendFetchRequest(ctx context.Context, pkt []byte, maxBytes int64) (io.ReadCloser, *protocol.FetchResponse, error) {
	logger := log.FromContext(ctx)
	responseReader, err := c.UploadPack(ctx, bytes.NewReader(pkt))
	if err != nil {
		return nil, nil, fmt.Errorf("sending commands: %w", err)
	}

	responseReader = newLimitedReadCloser(responseReader, maxBytes, "fetch")

	parser := protocol.NewParser(responseReader)
	response, err := protocol.ParseFetchResponse(ctx, parser)
	if err != nil {
		if closeErr := responseReader.Close(); closeErr != nil {
			logger.Error("error closing fetch response body after parse failure", "error", closeErr)
		}
		return nil, nil, fmt.Errorf("parsing fetch response stream: %w", err)
	}

	return responseReader, response, nil
}

// allWantedObjectsCollected reports whether the read loop has seen every
// wanted hash. The caller maintains pendingWanted by deleting each hash on
// first sight (see shouldTerminateEarly), so an empty map means "all
// collected". A nil map means the fetch is not NoExtraObjects and the
// early-termination / cap-swallow branches stay disabled.
func allWantedObjectsCollected(pendingWanted map[string]bool) bool {
	return pendingWanted != nil && len(pendingWanted) == 0
}

// classifyReadObjectErr decides what to do with a non-EOF error from
// ReadObject. A non-nil return is the error to propagate (it aborts
// processPackfileResponse); a nil return means the caller should swallow the
// error and break out of the read loop, finalizing whatever was collected.
//
// The default is to propagate, wrapped with the 1-based object position so
// the message points at the offending object — corrupt packfiles, short
// reads, and oversize objects all surface as hard errors. The one exception
// preserves the DoS-protection contract: when a NoExtraObjects fetch has
// already collected every requested object, an *ErrResponseTooLarge fired on
// the trailing bytes is on data the caller never asked for, so surfacing it
// would turn a successful single-object lookup into a false negative against
// a server that over-sends. pendingWanted is the deletion-tracked map
// maintained by the read loop: nil means the fetch isn't NoExtraObjects (so
// the swallow path is disabled), empty means every wanted hash has been seen.
//
// objectsRead is the count of objects successfully read before this error,
// so the reported position is objectsRead+1.
func classifyReadObjectErr(err error, pendingWanted map[string]bool, objectsRead int) error {
	var tooLarge *ErrResponseTooLarge
	if errors.As(err, &tooLarge) && allWantedObjectsCollected(pendingWanted) {
		return nil
	}
	return fmt.Errorf("reading packfile object %d: %w", objectsRead+1, err)
}

// processPackfileResponse processes the packfile response and extracts objects
func (c *rawClient) processPackfileResponse(ctx context.Context, response *protocol.FetchResponse, objects map[string]*protocol.PackfileObject, storage storage.PackfileStorage, opts FetchOptions) error {
	logger := log.FromContext(ctx)
	// Build a set of pending wanted object hashes for quick lookup if early termination is enabled
	// Only build this if we have specific objects we want AND NoExtraObjects is enabled
	var pendingWantedHashes map[string]bool
	if opts.NoExtraObjects && len(opts.Want) > 0 {
		pendingWantedHashes = make(map[string]bool, len(opts.Want))
		for _, hash := range opts.Want {
			pendingWantedHashes[hash.String()] = true
		}
		logger.Debug("Early termination enabled", "pendingObjects", len(pendingWantedHashes))
	}

	// Collect delta objects for later resolution
	var deltas []*protocol.PackfileObject

	var count, objectCount, totalDelta int
	for {
		obj, err := response.Packfile.ReadObject(ctx)
		if err != nil {
			// io.EOF is the natural end of the packfile stream.
			if errors.Is(err, io.EOF) {
				logger.Debug("Finished reading objects", "totalObjects", objectCount, "totalDeltas", totalDelta)
				break
			}
			// Anything else propagates — corrupt packfile, short
			// read, oversize object — UNLESS it's a cap breach on
			// trailing bytes after every wanted object was already
			// collected (see classifyReadObjectErr).
			if terr := classifyReadObjectErr(err, pendingWantedHashes, count); terr != nil {
				return terr
			}
			logger.Debug("Cap reached after all wanted objects collected; stopping early",
				"totalObjects", objectCount, "totalDeltas", totalDelta)
			// Drop deltas queued before the swallow for the same
			// reason the early-break path does (bases may be in the
			// unread tail).
			deltas = nil
			break
		}

		if obj.Object == nil {
			break
		}
		count++

		// Collect delta objects for later resolution instead of skipping them
		if obj.Object.Type == protocol.ObjectTypeRefDelta {
			totalDelta++
			logger.Debug("Found delta object", "parent", obj.Object.Delta.Parent)
			deltas = append(deltas, obj.Object)
			continue
		}

		if storage != nil {
			storage.Add(obj.Object)
		}

		objects[obj.Object.Hash.String()] = obj.Object
		objectCount++

		if shouldTerminateEarly(pendingWantedHashes, obj.Object.Hash.String()) {
			logger.Debug("All wanted objects found, stopping early",
				"totalObjectsRead", objectCount, "queuedDeltas", totalDelta)
			// Drop any deltas queued before the early break.
			// Their bases may live in the unread tail of the
			// stream, so resolveDeltas would spuriously report
			// missing-base errors against an over-sending or
			// adversarial server. The wanted set is already
			// satisfied by the non-delta objects collected above.
			deltas = nil
			break
		}
	}

	// Resolve deltas if any were found
	if len(deltas) > 0 {
		logger.Debug("Resolving deltas", "deltaCount", len(deltas), "baseObjectCount", len(objects))
		err := c.resolveDeltas(ctx, deltas, objects, storage)
		if err != nil {
			return fmt.Errorf("failed to resolve deltas: %w", err)
		}
	}

	return nil
}

// shouldTerminateEarly deletes the just-read hash from the pending wanted
// set and reports whether the set is now empty. Returning true tells the
// caller to break out of the read loop — both to avoid draining the rest of
// the packfile (wasteful) and to keep the new tight per-operation caps from
// tripping ErrResponseTooLarge on bytes the caller never asked for.
//
// Tracking by deletion rather than a counter is what makes this safe
// against a malicious or buggy server that sends the same wanted object
// more than once: each hash decrements the pending set exactly once, so
// duplicates can't trigger early termination before every distinct hash
// has been seen.
//
// Splitting this out is mostly housekeeping: it keeps
// processPackfileResponse under the gocyclo-15 ceiling and makes the
// early-termination contract testable in isolation.
func shouldTerminateEarly(pendingWanted map[string]bool, objHash string) bool {
	if pendingWanted == nil {
		return false
	}
	if !pendingWanted[objHash] {
		return false
	}
	delete(pendingWanted, objHash)
	return len(pendingWanted) == 0
}

// resolveDeltas resolves delta objects by applying them to their base objects.
// This function handles delta chains where a delta's base might itself be a delta.
// Resolution is done iteratively until all deltas are resolved or we detect unresolvable deltas.
func (c *rawClient) resolveDeltas(ctx context.Context, deltas []*protocol.PackfileObject, objects map[string]*protocol.PackfileObject, storage storage.PackfileStorage) error {
	logger := log.FromContext(ctx)

	remaining := make([]*protocol.PackfileObject, len(deltas))
	copy(remaining, deltas)

	maxIterations := len(deltas) + 1
	for iteration := 1; len(remaining) > 0 && iteration <= maxIterations; iteration++ {
		resolvedCount, stillPending := c.resolveDeltaIteration(ctx, remaining, objects, storage)
		remaining = stillPending

		if resolvedCount == 0 && len(remaining) > 0 {
			return c.createMissingBasesError(remaining)
		}

		logger.Debug("Delta resolution iteration complete", "iteration", iteration, "resolved", resolvedCount, "remaining", len(remaining))
	}

	if len(remaining) > 0 {
		return fmt.Errorf("failed to resolve all deltas after %d iterations: %d deltas remaining", maxIterations, len(remaining))
	}

	logger.Debug("All deltas resolved successfully", "totalDeltas", len(deltas))
	return nil
}

// resolveDeltaIteration processes one iteration of delta resolution
func (c *rawClient) resolveDeltaIteration(ctx context.Context, deltas []*protocol.PackfileObject, objects map[string]*protocol.PackfileObject, storage storage.PackfileStorage) (int, []*protocol.PackfileObject) {
	logger := log.FromContext(ctx)
	var stillPending []*protocol.PackfileObject
	resolvedCount := 0

	for _, delta := range deltas {
		if delta.Delta == nil {
			logger.Debug("Skipping object without delta", "type", delta.Type)
			continue
		}

		baseObj, found := c.findBaseObject(ctx, delta.Delta.Parent, objects, storage)
		if !found {
			stillPending = append(stillPending, delta)
			continue
		}

		if err := c.resolveSingleDelta(ctx, delta, baseObj, objects, storage); err != nil {
			logger.Debug("Failed to resolve delta", "parent", delta.Delta.Parent, "error", err)
			stillPending = append(stillPending, delta)
			continue
		}

		resolvedCount++
	}

	return resolvedCount, stillPending
}

// findBaseObject finds the base object for a delta, checking both objects map and storage
func (c *rawClient) findBaseObject(ctx context.Context, parentHash string, objects map[string]*protocol.PackfileObject, storage storage.PackfileStorage) (*protocol.PackfileObject, bool) {
	logger := log.FromContext(ctx)

	// First check in-memory objects
	if baseObj, exists := objects[parentHash]; exists {
		return baseObj, true
	}

	// Then check storage if available
	if storage != nil {
		baseHash, err := hash.FromHex(parentHash)
		if err != nil {
			logger.Debug("Invalid parent hash in delta", "parent", parentHash, "error", err)
			return nil, false
		}

		if baseObj, exists := storage.Get(baseHash); exists {
			return baseObj, true
		}
	}

	logger.Debug("Base object not found", "parent", parentHash)
	return nil, false
}

// resolveSingleDelta resolves a single delta object and adds it to the objects map
func (c *rawClient) resolveSingleDelta(ctx context.Context, delta *protocol.PackfileObject, baseObj *protocol.PackfileObject, objects map[string]*protocol.PackfileObject, storage storage.PackfileStorage) error {
	logger := log.FromContext(ctx)

	// Apply the delta to the base object
	resolvedData, err := protocol.ApplyDelta(baseObj.Data, delta.Delta)
	if err != nil {
		return fmt.Errorf("failed to apply delta: %w", err)
	}

	// Calculate the hash and type of resolved object
	resolvedHash, resolvedType, err := c.inferResolvedObjectType(ctx, resolvedData, baseObj)
	if err != nil {
		return fmt.Errorf("failed to infer object type: %w", err)
	}

	// Create and parse the resolved object
	resolvedObj := &protocol.PackfileObject{
		Type: resolvedType,
		Data: resolvedData,
		Hash: resolvedHash,
	}

	if err := c.parseResolvedObject(ctx, resolvedObj); err != nil {
		logger.Debug("Warning: failed to parse resolved object", "hash", resolvedHash.String(), "error", err)
	}

	// Add to objects map and storage
	objects[resolvedHash.String()] = resolvedObj
	if storage != nil {
		storage.Add(resolvedObj)
	}

	logger.Debug("Resolved delta", "hash", resolvedHash.String(), "parent", delta.Delta.Parent, "type", resolvedType)
	return nil
}

// createMissingBasesError creates an error for unresolvable deltas
func (c *rawClient) createMissingBasesError(remaining []*protocol.PackfileObject) error {
	missingBases := make([]string, 0, len(remaining))
	for _, delta := range remaining {
		if delta.Delta != nil {
			missingBases = append(missingBases, delta.Delta.Parent)
		}
	}
	return fmt.Errorf("unable to resolve %d deltas: missing base objects %v", len(remaining), missingBases)
}

// inferResolvedObjectType infers the object type of a resolved delta object
// and calculates its hash. Since deltas don't store the target object type explicitly,
// we infer it from the base object type (deltas typically maintain the same type).
func (c *rawClient) inferResolvedObjectType(ctx context.Context, data []byte, baseObj *protocol.PackfileObject) (hash.Hash, protocol.ObjectType, error) {
	// In most cases, delta objects preserve the type of their base
	targetType := baseObj.Type

	// Calculate the hash with the inferred type
	resolvedHash, err := protocol.Object(crypto.SHA1, targetType, data)
	if err != nil {
		return hash.Zero, protocol.ObjectTypeInvalid, fmt.Errorf("failed to calculate object hash: %w", err)
	}

	return resolvedHash, targetType, nil
}

// parseResolvedObject parses the content of resolved delta objects (trees and commits)
// to ensure their fields are properly populated for consumers.
func (c *rawClient) parseResolvedObject(ctx context.Context, obj *protocol.PackfileObject) error {
	// Parse the object using the public Parse() method
	return obj.Parse()
}
