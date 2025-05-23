package nanogit

import (
	"context"

	"errors"
	"fmt"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

var (
	ErrRefNotFound = errors.New("ref not found")
)

type Ref struct {
	Name string
	Hash hash.Hash
}

// ListRefs sends a request to list all references in the repository.
// It returns a map of reference names to their commit hashes.
func (c *clientImpl) ListRefs(ctx context.Context) ([]Ref, error) {
	// First get the initial capability advertisement
	_, err := c.smartInfo(ctx, "git-upload-pack")
	if err != nil {
		return nil, fmt.Errorf("get repository info: %w", err)
	}

	// Now send the ls-refs command
	pkt, err := protocol.FormatPacks(
		protocol.PackLine("command=ls-refs\n"),
		protocol.PackLine("object-format=sha1\n"),
		protocol.FlushPacket,
	)
	if err != nil {
		return nil, fmt.Errorf("format ls-refs command: %w", err)
	}

	refsData, err := c.uploadPack(ctx, pkt)
	if err != nil {
		return nil, fmt.Errorf("send ls-refs command: %w", err)
	}

	refs := make([]Ref, 0)
	lines, _, err := protocol.ParsePack(refsData)
	if err != nil {
		return nil, fmt.Errorf("parse refs response: %w", err)
	}

	for _, line := range lines {
		ref, h, err := protocol.ParseRefLine(line)
		if err != nil {
			return nil, fmt.Errorf("parse ref line: %w", err)
		}

		parsedHash, err := hash.FromHex(h)
		if err != nil {
			return nil, fmt.Errorf("parse ref hash: %w", err)
		}

		if ref != "" {
			refs = append(refs, Ref{Name: ref, Hash: parsedHash})
		}
	}

	return refs, nil
}

// GetRef sends a request to get a single reference in the repository.
// It returns the reference name, hash, and any error encountered.
// FIXME: Fetch only the requested ref, not all refs.
func (c *clientImpl) GetRef(ctx context.Context, refName string) (Ref, error) {
	refs, err := c.ListRefs(ctx)
	if err != nil {
		return Ref{}, fmt.Errorf("list refs: %w", err)
	}

	for _, r := range refs {
		if r.Name == refName {
			return r, nil
		}
	}

	return Ref{}, ErrRefNotFound
}

// CreateRef creates a new reference in the repository.
// It returns any error encountered.
func (c *clientImpl) CreateRef(ctx context.Context, ref Ref) error {
	_, err := c.GetRef(ctx, ref.Name)
	switch {
	case err != nil && !errors.Is(err, ErrRefNotFound):
		return fmt.Errorf("get ref: %w", err)
	case err == nil:
		return fmt.Errorf("ref %s already exists", ref.Name)
	}

	// First get the initial capability advertisement
	_, err = c.smartInfo(ctx, "git-receive-pack")
	if err != nil {
		return fmt.Errorf("get receive-pack capability: %w", err)
	}

	pkt, err := protocol.NewCreateRefRequest(ref.Name, ref.Hash).Format()
	if err != nil {
		return fmt.Errorf("format ref update request: %w", err)
	}

	// Send the ref update
	_, err = c.receivePack(ctx, pkt)
	if err != nil {
		return fmt.Errorf("send ref update: %w", err)
	}

	return nil
}

// UpdateRef updates an existing reference in the repository.
// It returns any error encountered.
func (c *clientImpl) UpdateRef(ctx context.Context, ref Ref) error {
	// First check if the ref exists
	oldRef, err := c.GetRef(ctx, ref.Name)
	if err != nil {
		if errors.Is(err, ErrRefNotFound) {
			return fmt.Errorf("ref %s does not exist", ref.Name)
		}
		return fmt.Errorf("get ref: %w", err)
	}

	// First get the initial capability advertisement
	_, err = c.smartInfo(ctx, "git-receive-pack")
	if err != nil {
		return fmt.Errorf("get receive-pack capability: %w", err)
	}

	// Create the ref update request
	pkt, err := protocol.NewUpdateRefRequest(oldRef.Hash, ref.Hash, ref.Name).Format()
	if err != nil {
		return fmt.Errorf("format ref update request: %w", err)
	}

	// Send the ref update
	_, err = c.receivePack(ctx, pkt)
	if err != nil {
		return fmt.Errorf("update ref: %w", err)
	}

	return nil
}

// DeleteRef deletes a reference from the repository.
// It returns ErrRefNotFound if the reference does not exist,
// or any other error encountered during the deletion process.
func (c *clientImpl) DeleteRef(ctx context.Context, refName string) error {
	// First check if the ref exists
	oldRef, err := c.GetRef(ctx, refName)
	if err != nil {
		if errors.Is(err, ErrRefNotFound) {
			return fmt.Errorf("ref %s does not exist", refName)
		}
		return fmt.Errorf("get ref: %w", err)
	}

	// First get the initial capability advertisement
	_, err = c.smartInfo(ctx, "git-receive-pack")
	if err != nil {
		return fmt.Errorf("get receive-pack capability: %w", err)
	}

	// Create the ref update request
	pkt, err := protocol.NewDeleteRefRequest(oldRef.Hash, refName).Format()
	if err != nil {
		return fmt.Errorf("format ref update request: %w", err)
	}

	// Send the ref update
	_, err = c.receivePack(ctx, pkt)
	if err != nil {
		return fmt.Errorf("delete ref: %w", err)
	}

	return nil
}
