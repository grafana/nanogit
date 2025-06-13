package nanogit

import (
	"context"

	"errors"
	"fmt"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/client"
	"github.com/grafana/nanogit/protocol/hash"
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
	lines, err := c.LsRefs(ctx, client.LsRefsOptions{})
	if err != nil {
		return nil, fmt.Errorf("list refs: %w", err)
	}

	refs := make([]Ref, 0)
	for _, line := range lines {
		refs = append(refs, Ref{Name: line.RefName, Hash: line.Hash})
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
func (c *httpClient) GetRef(ctx context.Context, refName string) (Ref, error) {
	lines, err := c.LsRefs(ctx, client.LsRefsOptions{Prefix: refName})
	if err != nil {
		return Ref{}, fmt.Errorf("list refs: %w", err)
	}

	if len(lines) == 0 {
		return Ref{}, NewRefNotFoundError(refName)
	}

	if len(lines) > 1 {
		return Ref{}, fmt.Errorf("multiple refs found for %s", refName)
	}

	refLine := lines[0]
	if refLine.RefName != refName {
		return Ref{}, NewRefNotFoundError(refName)
	}

	return Ref{Name: refLine.RefName, Hash: refLine.Hash}, nil
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
	if err == nil {
		return NewRefAlreadyExistsError(ref.Name)
	}
	if !errors.Is(err, ErrObjectNotFound) {
		return fmt.Errorf("get ref: %w", err)
	}
	// Ref doesn't exist, we can create it (this is the expected case)

	// Create and send the ref update request directly - Protocol v2 allows this
	// without needing a separate capability advertisement request
	pkt, err := protocol.NewCreateRefRequest(ref.Name, ref.Hash).Format()
	if err != nil {
		return fmt.Errorf("format ref update request: %w", err)
	}

	// Send the ref update
	_, err = c.ReceivePack(ctx, pkt)
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
		if errors.Is(err, ErrObjectNotFound) {
			return err
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
	_, err = c.ReceivePack(ctx, pkt)
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
		if errors.Is(err, ErrObjectNotFound) {
			return err
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
	_, err = c.ReceivePack(ctx, pkt)
	if err != nil {
		return fmt.Errorf("delete ref: %w", err)
	}

	return nil
}
