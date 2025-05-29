package nanogit

import (
	"errors"
	"fmt"
)

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

// ObjectNotFoundError provides structured information about a missing Git object.
// It implements the error interface and supports errors.Is/As for the underlying ErrObjectNotFound.
type ObjectNotFoundError struct {
	ObjectID string
	Err      error
}

func (e *ObjectNotFoundError) Error() string {
	return fmt.Sprintf("object %s not found: %v", e.ObjectID, e.Err)
}

func (e *ObjectNotFoundError) Unwrap() error {
	return e.Err
}

// NewObjectNotFoundError creates a new ObjectNotFoundError with the specified object ID.
func NewObjectNotFoundError(objectID string) *ObjectNotFoundError {
	return &ObjectNotFoundError{
		ObjectID: objectID,
		Err:      ErrObjectNotFound,
	}
}

// ObjectAlreadyExistsError provides structured information about a Git object that already exists.
// It implements the error interface and supports errors.Is/As for the underlying ErrObjectAlreadyExists.
type ObjectAlreadyExistsError struct {
	ObjectID string
	Err      error
}

func (e *ObjectAlreadyExistsError) Error() string {
	return fmt.Sprintf("object %s already exists: %v", e.ObjectID, e.Err)
}

func (e *ObjectAlreadyExistsError) Unwrap() error {
	return e.Err
}

// NewObjectAlreadyExistsError creates a new ObjectAlreadyExistsError with the specified object ID.
func NewObjectAlreadyExistsError(objectID string) *ObjectAlreadyExistsError {
	return &ObjectAlreadyExistsError{
		ObjectID: objectID,
		Err:      ErrObjectAlreadyExists,
	}
}

// UnexpectedObjectTypeError provides structured information about a Git object with an unexpected type.
// It implements the error interface and supports errors.Is/As for the underlying ErrUnexpectedObjectType.
type UnexpectedObjectTypeError struct {
	ObjectID     string
	ExpectedType string
	ActualType   string
	Err          error
}

func (e *UnexpectedObjectTypeError) Error() string {
	return fmt.Sprintf("object %s has unexpected type %s (expected %s): %v",
		e.ObjectID, e.ActualType, e.ExpectedType, e.Err)
}

func (e *UnexpectedObjectTypeError) Unwrap() error {
	return e.Err
}

// NewUnexpectedObjectTypeError creates a new UnexpectedObjectTypeError with the specified details.
func NewUnexpectedObjectTypeError(objectID, expectedType, actualType string) *UnexpectedObjectTypeError {
	return &UnexpectedObjectTypeError{
		ObjectID:     objectID,
		ExpectedType: expectedType,
		ActualType:   actualType,
		Err:          ErrUnexpectedObjectType,
	}
}
