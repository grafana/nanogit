package client

import (
	"errors"
	"fmt"
)

// ErrServerUnavailable is returned when the Git server is unavailable (HTTP 5xx status codes).
// This error should only be used with errors.Is() for comparison, not for type assertions.
var ErrServerUnavailable = errors.New("server unavailable")

// ServerUnavailableError provides structured information about a Git server that is unavailable.
type ServerUnavailableError struct {
	StatusCode int
	Underlying error
}

func (e *ServerUnavailableError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("server unavailable (status code %d): %v", e.StatusCode, e.Underlying)
	}
	return fmt.Sprintf("server unavailable (status code %d)", e.StatusCode)
}

// Unwrap returns the underlying error, preserving the error chain.
func (e *ServerUnavailableError) Unwrap() error {
	return e.Underlying
}

// Is enables errors.Is() compatibility with ErrServerUnavailable.
func (e *ServerUnavailableError) Is(target error) bool {
	return target == ErrServerUnavailable
}

// NewServerUnavailableError creates a new ServerUnavailableError with the specified status code and underlying error.
func NewServerUnavailableError(statusCode int, underlying error) *ServerUnavailableError {
	return &ServerUnavailableError{
		StatusCode: statusCode,
		Underlying: underlying,
	}
}


