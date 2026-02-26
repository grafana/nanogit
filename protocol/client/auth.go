package client

import (
	"context"
	"errors"
	"fmt"
)

// CanRead checks if the client has read access to the Git server.
// It queries the git-upload-pack service capabilities to verify read permissions.
//
// Returns:
//   - true if the client can read from the repository (can access git-upload-pack)
//   - false if the server returns 401 Unauthorized or 403 Forbidden (no access)
//   - error if there are any other connection or protocol issues
func (c *rawClient) CanRead(ctx context.Context) (bool, error) {
	err := c.SmartInfo(ctx, "git-upload-pack")
	if err != nil {
		// Check for authentication and permission errors
		if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrPermissionDenied) {
			return false, nil
		}
		return false, fmt.Errorf("check read permission: %w", err)
	}
	return true, nil
}

// IsAuthorized is deprecated. Use CanRead instead.
// It checks if the client can successfully communicate with the Git server.
func (c *rawClient) IsAuthorized(ctx context.Context) (bool, error) {
	return c.CanRead(ctx)
}

// CanWrite checks if the client has repository-level write access to the Git server.
// It queries the git-receive-pack service capabilities to verify write permissions.
//
// IMPORTANT LIMITATIONS:
// - This method checks REPOSITORY-LEVEL write access only (git-receive-pack service availability)
// - It CANNOT detect branch-specific protections (e.g., protected main branch)
// - It CANNOT detect path-based restrictions or fine-grained permissions
// - A true result means the user has SOME write capability, but specific operations may still fail
// - A false result definitively means the user has NO write access to the repository
//
// This method performs a read-only capability query and does NOT attempt to write any data.
// For definitive permission checks, attempt the actual operation (Push, CreateRef, etc.) and
// handle the resulting PermissionDeniedError or UnauthorizedError.
//
// Returns:
//   - true if the client has repository-level write permission (can access git-receive-pack)
//   - false if the server returns 401 Unauthorized or 403 Forbidden (read-only or no access)
//   - error if there are any other connection or protocol issues
//
// Use case: Pre-check to determine if user is read-only before attempting time-consuming operations.
func (c *rawClient) CanWrite(ctx context.Context) (bool, error) {
	err := c.SmartInfo(ctx, "git-receive-pack")
	if err != nil {
		// Check for both authentication and permission errors
		if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrPermissionDenied) {
			return false, nil
		}
		return false, fmt.Errorf("check write permission: %w", err)
	}
	return true, nil
}
