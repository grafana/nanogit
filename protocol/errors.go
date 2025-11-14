package protocol

import (
	"errors"
	"fmt"
	"io"
)

// strError is a simple string-based error type that implements the error interface.
// It allows creating lightweight error values from string constants without
// allocating a new error for each instance.
type strError string

// Error implements the error interface by returning the string value of the error.
func (e strError) Error() string {
	return string(e)
}

// eofIsUnexpected checks if the error is an io.EOF.
// If it is, we return io.ErrUnexpectedEOF.
// If not, we return the input error verbatim.
func eofIsUnexpected(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	} else {
		return err
	}
}

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
