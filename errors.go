package nanogit

import (
	"errors"
	"fmt"

	"github.com/grafana/nanogit/protocol"
)

var (
	// ErrObjectNotFound is returned when a requested Git object cannot be found in the repository.
	// This error should only be used with errors.Is() for comparison, not for type assertions.
	ErrObjectNotFound = errors.New("object not found")

	// ErrObjectAlreadyExists is returned when attempting to create a Git object that already exists.
	// This error should only be used with errors.Is() for comparison, not for type assertions.
	ErrObjectAlreadyExists = errors.New("object already exists")

	// ErrUnexpectedObjectType is returned when a Git object has a different type than expected.
	// This error should only be used with errors.Is() for comparison, not for type assertions.
	ErrUnexpectedObjectType = errors.New("unexpected object type")
)

// ObjectNotFoundError provides structured information about a Git object that was not found.
type ObjectNotFoundError struct {
	ObjectID string
}

func (e *ObjectNotFoundError) Error() string {
	return fmt.Sprintf("object %s not found", e.ObjectID)
}

// Unwrap enables errors.Is() compatibility with ErrObjectNotFound
func (e *ObjectNotFoundError) Unwrap() error {
	return ErrObjectNotFound
}

// NewObjectNotFoundError creates a new ObjectNotFoundError with the specified object ID.
func NewObjectNotFoundError(objectID string) *ObjectNotFoundError {
	return &ObjectNotFoundError{
		ObjectID: objectID,
	}
}

// ObjectAlreadyExistsError provides structured information about a Git object that already exists.
type ObjectAlreadyExistsError struct {
	ObjectID string
}

func (e *ObjectAlreadyExistsError) Error() string {
	return fmt.Sprintf("object %s already exists", e.ObjectID)
}

// Unwrap enables errors.Is() compatibility with ErrObjectAlreadyExists
func (e *ObjectAlreadyExistsError) Unwrap() error {
	return ErrObjectAlreadyExists
}

// NewObjectAlreadyExistsError creates a new ObjectAlreadyExistsError with the specified object ID.
func NewObjectAlreadyExistsError(objectID string) *ObjectAlreadyExistsError {
	return &ObjectAlreadyExistsError{
		ObjectID: objectID,
	}
}

// UnexpectedObjectTypeError provides structured information about a Git object with an unexpected type.
type UnexpectedObjectTypeError struct {
	ObjectID     string
	ExpectedType protocol.ObjectType
	ActualType   protocol.ObjectType
}

func (e *UnexpectedObjectTypeError) Error() string {
	return fmt.Sprintf("object %s has unexpected type %s (expected %s)",
		e.ObjectID, e.ActualType, e.ExpectedType)
}

// Unwrap enables errors.Is() compatibility with ErrUnexpectedObjectType
func (e *UnexpectedObjectTypeError) Unwrap() error {
	return ErrUnexpectedObjectType
}

// NewUnexpectedObjectTypeError creates a new UnexpectedObjectTypeError with the specified details.
func NewUnexpectedObjectTypeError(objectID string, expectedType, actualType protocol.ObjectType) *UnexpectedObjectTypeError {
	return &UnexpectedObjectTypeError{
		ObjectID:     objectID,
		ExpectedType: expectedType,
		ActualType:   actualType,
	}
}
