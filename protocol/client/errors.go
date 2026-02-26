package client

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ErrServerUnavailable is returned when the Git server is unavailable (HTTP 5xx status codes).
// This error should only be used with errors.Is() for comparison, not for type assertions.
var ErrServerUnavailable = errors.New("server unavailable")

// ErrUnauthorized is returned when authentication fails (HTTP 401).
var ErrUnauthorized = errors.New("unauthorized")

// ErrPermissionDenied is returned when the user lacks permission for the operation (HTTP 403).
var ErrPermissionDenied = errors.New("permission denied")

// ErrRepositoryNotFound is returned when the repository does not exist (HTTP 404).
var ErrRepositoryNotFound = errors.New("repository not found")

// ServerUnavailableError provides structured information about a Git server that is unavailable.
type ServerUnavailableError struct {
	// StatusCode is the HTTP status code (5xx)
	StatusCode int
	// Operation is the HTTP method that failed (e.g., "GET", "POST", "PUT")
	Operation string
	// Underlying is the underlying error
	Underlying error
}

func (e *ServerUnavailableError) Error() string {
	if e.Underlying != nil {
		if e.Operation != "" {
			return fmt.Sprintf("server unavailable (operation %s, status code %d): %v", e.Operation, e.StatusCode, e.Underlying)
		}
		return fmt.Sprintf("server unavailable (status code %d): %v", e.StatusCode, e.Underlying)
	}
	if e.Operation != "" {
		return fmt.Sprintf("server unavailable (operation %s, status code %d)", e.Operation, e.StatusCode)
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

// NewServerUnavailableError creates a new ServerUnavailableError with the specified operation, status code, and underlying error.
// Operation can be empty if the HTTP method is unknown.
func NewServerUnavailableError(operation string, statusCode int, underlying error) *ServerUnavailableError {
	return &ServerUnavailableError{
		Operation:  operation,
		StatusCode: statusCode,
		Underlying: underlying,
	}
}

// CheckServerUnavailable checks if an HTTP response indicates server unavailability.
// It checks for:
//   - Server errors (5xx status codes)
//   - Too Many Requests (429 status code)
//
// If the response is server unavailable, it returns a ServerUnavailableError.
// The HTTP method is extracted from the response's request.
// The caller is responsible for closing the response body.
func CheckServerUnavailable(res *http.Response) error {
	if res.StatusCode >= 500 || res.StatusCode == http.StatusTooManyRequests {
		operation := ""
		if res.Request != nil {
			operation = res.Request.Method
		}
		return NewServerUnavailableError(operation, res.StatusCode, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status))
	}
	return nil
}

// UnauthorizedError provides structured information about an authentication failure.
type UnauthorizedError struct {
	// StatusCode is the HTTP status code (401)
	StatusCode int
	// Operation is the HTTP method that failed (e.g., "GET", "POST")
	Operation string
	// Endpoint is the Git protocol endpoint (e.g., "git-receive-pack", "git-upload-pack")
	Endpoint string
	// Underlying is the underlying error
	Underlying error
}

func (e *UnauthorizedError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("unauthorized (operation %s, endpoint %s, status code %d): %v",
			e.Operation, e.Endpoint, e.StatusCode, e.Underlying)
	}
	return fmt.Sprintf("unauthorized (operation %s, endpoint %s, status code %d)",
		e.Operation, e.Endpoint, e.StatusCode)
}

func (e *UnauthorizedError) Unwrap() error {
	return e.Underlying
}

func (e *UnauthorizedError) Is(target error) bool {
	return target == ErrUnauthorized
}

func NewUnauthorizedError(operation, endpoint string, underlying error) *UnauthorizedError {
	return &UnauthorizedError{
		Operation:  operation,
		Endpoint:   endpoint,
		StatusCode: http.StatusUnauthorized,
		Underlying: underlying,
	}
}

// PermissionDeniedError provides structured information about a permission denial.
type PermissionDeniedError struct {
	StatusCode int
	Operation  string
	Endpoint   string
	Underlying error
}

func (e *PermissionDeniedError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("permission denied (operation %s, endpoint %s, status code %d): %v",
			e.Operation, e.Endpoint, e.StatusCode, e.Underlying)
	}
	return fmt.Sprintf("permission denied (operation %s, endpoint %s, status code %d)",
		e.Operation, e.Endpoint, e.StatusCode)
}

func (e *PermissionDeniedError) Unwrap() error {
	return e.Underlying
}

func (e *PermissionDeniedError) Is(target error) bool {
	return target == ErrPermissionDenied
}

func NewPermissionDeniedError(operation, endpoint string, underlying error) *PermissionDeniedError {
	return &PermissionDeniedError{
		Operation:  operation,
		Endpoint:   endpoint,
		StatusCode: http.StatusForbidden,
		Underlying: underlying,
	}
}

// RepositoryNotFoundError provides structured information about a repository not found error.
type RepositoryNotFoundError struct {
	StatusCode int
	Operation  string
	Endpoint   string
	Underlying error
}

func (e *RepositoryNotFoundError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("repository not found (operation %s, endpoint %s, status code %d): %v",
			e.Operation, e.Endpoint, e.StatusCode, e.Underlying)
	}
	return fmt.Sprintf("repository not found (operation %s, endpoint %s, status code %d)",
		e.Operation, e.Endpoint, e.StatusCode)
}

func (e *RepositoryNotFoundError) Unwrap() error {
	return e.Underlying
}

func (e *RepositoryNotFoundError) Is(target error) bool {
	return target == ErrRepositoryNotFound
}

func NewRepositoryNotFoundError(operation, endpoint string, underlying error) *RepositoryNotFoundError {
	return &RepositoryNotFoundError{
		Operation:  operation,
		Endpoint:   endpoint,
		StatusCode: http.StatusNotFound,
		Underlying: underlying,
	}
}

// CheckHTTPClientError checks if an HTTP response indicates a client error.
// It checks for:
//   - Unauthorized (401 status code)
//   - Permission Denied (403 status code)
//   - Repository Not Found (404 status code)
//
// If the response is a recognized client error, it returns the appropriate error type.
// For other 4xx errors, it returns nil (caller should handle generically).
// The HTTP method and endpoint are extracted from the response's request.
// The caller is responsible for closing the response body.
func CheckHTTPClientError(res *http.Response) error {
	if res.StatusCode < 400 || res.StatusCode >= 500 {
		return nil
	}

	operation := ""
	endpoint := ""
	if res.Request != nil {
		operation = res.Request.Method
		endpoint = extractEndpoint(res.Request.URL.Path)
	}

	underlying := fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)

	switch res.StatusCode {
	case http.StatusUnauthorized:
		return NewUnauthorizedError(operation, endpoint, underlying)
	case http.StatusForbidden:
		return NewPermissionDeniedError(operation, endpoint, underlying)
	case http.StatusNotFound:
		return NewRepositoryNotFoundError(operation, endpoint, underlying)
	default:
		return nil // Other 4xx errors handled generically by callers
	}
}

// extractEndpoint extracts the Git protocol endpoint from a URL path.
// Returns "git-upload-pack", "git-receive-pack", "info/refs", or "unknown".
// Only examines the path component, ignoring query strings.
func extractEndpoint(path string) string {
	// Remove query string if present
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	if strings.Contains(path, "git-receive-pack") {
		return "git-receive-pack"
	}
	if strings.Contains(path, "git-upload-pack") {
		return "git-upload-pack"
	}
	if strings.Contains(path, "info/refs") {
		return "info/refs"
	}
	return "unknown"
}
