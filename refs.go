package nanogit

import (
	"context"

	"errors"
	"fmt"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

var (
	// ErrRefNotFound is returned when a requested Git reference does not exist in the repository.
	// This error is returned by GetRef when the specified reference name cannot be found.
	ErrRefNotFound = errors.New("ref not found")
)

// Ref represents a Git reference (ref) in the repository.
// A ref is a pointer to a specific commit in the repository's history.
// Common reference types include branches (refs/heads/*), tags (refs/tags/*),
// and remote tracking branches (refs/remotes/*).
type Ref struct {
	// Name is the full reference name (e.g., "refs/heads/main", "refs/tags/v1.0")
	Name string
	// Hash is the commit hash that this reference points to
	Hash hash.Hash
}

// ListRefs retrieves all Git references from the remote repository.
// This includes branches, tags, and other references available on the remote.
// The method uses the Git protocol's ls-refs command to efficiently fetch
// reference information without downloading object data.
//
// Parameters:
//   - ctx: Context for the operation
//
// Returns:
//   - []Ref: Slice of all references found in the repository
//   - error: Error if the request fails or response cannot be parsed
//
// Example:
//
//	refs, err := client.ListRefs(ctx)
//	if err != nil {
//	    return err
//	}
//	for _, ref := range refs {
//	    fmt.Printf("%s -> %s\n", ref.Name, ref.Hash.String())
//	}
func (c *httpClient) ListRefs(ctx context.Context) ([]Ref, error) {
	// Send the ls-refs command directly - Protocol v2 allows this without needing
	// a separate capability advertisement request
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

// GetRef retrieves a specific Git reference by name from the remote repository.
// This method currently fetches all references and filters for the requested one.
// Future optimization could fetch only the specific reference.
//
// Parameters:
//   - ctx: Context for the operation
//   - refName: Full reference name (e.g., "refs/heads/main", "refs/tags/v1.0")
//
// Returns:
//   - Ref: The requested reference with its name and commit hash
//   - error: ErrRefNotFound if the reference doesn't exist, or other errors for request failures
//
// Example:
//
//	ref, err := client.GetRef(ctx, "refs/heads/main")
//	if errors.Is(err, nanogit.ErrRefNotFound) {
//	    fmt.Println("Branch main does not exist")
//	} else if err != nil {
//	    return err
//	} else {
//	    fmt.Printf("main branch points to %s\n", ref.Hash.String())
//	}
//
// TODO: Optimize to fetch only the requested reference instead of all references.
func (c *httpClient) GetRef(ctx context.Context, refName string) (Ref, error) {
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

// CreateRef creates a new Git reference in the remote repository.
// The reference must not already exist, otherwise an error is returned.
// This operation is commonly used to create new branches or tags.
//
// Parameters:
//   - ctx: Context for the operation
//   - ref: The reference to create, containing both name and target commit hash
//
// Returns:
//   - error: Error if the reference already exists, target commit doesn't exist, or operation fails
//
// Example:
//
//	// Create a new branch pointing to a specific commit
//	newRef := nanogit.Ref{
//	    Name: "refs/heads/feature-branch",
//	    Hash: commitHash,
//	}
//	err := client.CreateRef(ctx, newRef)
//	if err != nil {
//	    return fmt.Errorf("failed to create branch: %w", err)
//	}
func (c *httpClient) CreateRef(ctx context.Context, ref Ref) error {
	_, err := c.GetRef(ctx, ref.Name)
	switch {
	case err != nil && !errors.Is(err, ErrRefNotFound):
		return fmt.Errorf("get ref: %w", err)
	case err == nil:
		return fmt.Errorf("ref %s already exists", ref.Name)
	}

	// Create and send the ref update request directly - Protocol v2 allows this
	// without needing a separate capability advertisement request
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

// UpdateRef updates an existing Git reference to point to a new commit.
// The reference must already exist, otherwise an error is returned.
// This operation is commonly used to move branches to new commits.
//
// Parameters:
//   - ctx: Context for the operation
//   - ref: The reference to update, containing both name and new target commit hash
//
// Returns:
//   - error: Error if the reference doesn't exist, target commit doesn't exist, or operation fails
//
// Example:
//
//	// Move main branch to a new commit
//	updatedRef := nanogit.Ref{
//	    Name: "refs/heads/main",
//	    Hash: newCommitHash,
//	}
//	err := client.UpdateRef(ctx, updatedRef)
//	if err != nil {
//	    return fmt.Errorf("failed to update branch: %w", err)
//	}
func (c *httpClient) UpdateRef(ctx context.Context, ref Ref) error {
	// First check if the ref exists
	oldRef, err := c.GetRef(ctx, ref.Name)
	if err != nil {
		if errors.Is(err, ErrRefNotFound) {
			return fmt.Errorf("ref %s does not exist", ref.Name)
		}
		return fmt.Errorf("get ref: %w", err)
	}

	// Create and send the ref update request directly - Protocol v2 allows this
	// without needing a separate capability advertisement request
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

// DeleteRef removes a Git reference from the remote repository.
// The reference must exist, otherwise an error is returned.
// This operation is commonly used to delete branches or tags.
// Note that this only removes the reference itself, not the commits it pointed to.
//
// Parameters:
//   - ctx: Context for the operation
//   - refName: Full reference name to delete (e.g., "refs/heads/feature-branch")
//
// Returns:
//   - error: Error if the reference doesn't exist or deletion fails
//
// Example:
//
//	// Delete a feature branch
//	err := client.DeleteRef(ctx, "refs/heads/feature-branch")
//	if err != nil {
//	    return fmt.Errorf("failed to delete branch: %w", err)
//	}
func (c *httpClient) DeleteRef(ctx context.Context, refName string) error {
	// First check if the ref exists
	oldRef, err := c.GetRef(ctx, refName)
	if err != nil {
		if errors.Is(err, ErrRefNotFound) {
			return fmt.Errorf("ref %s does not exist", refName)
		}
		return fmt.Errorf("get ref: %w", err)
	}

	// Create and send the ref update request directly - Protocol v2 allows this
	// without needing a separate capability advertisement request
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
