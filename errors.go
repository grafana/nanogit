package nanogit

import "errors"

var (
	// ErrObjectNotFound is returned when a requested Git object does not exist in the repository.
	// This error is returned by GetRef when the specified reference name cannot be found.
	ErrObjectNotFound = errors.New("git object not found")

	// ErrObjectAlreadyExists is returned when a requested Git object already exists in the repository.
	// This error is returned by CreateRef when the specified reference name already exists.
	ErrObjectAlreadyExists = errors.New("git object already exists")

	// ErrUnexpectedObjectType is returned when a requested Git object is not of the expected type.
	ErrUnexpectedObjectType = errors.New("unexpected git object type")
)
