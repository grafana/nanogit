package client

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/nanogit/options"
)

// ErrServerUnavailable is returned when the Git server is unavailable (HTTP 5xx status codes).
// This error should only be used with errors.Is() for comparison, not for type assertions.
var ErrServerUnavailable = errors.New("server unavailable")

// ErrUnauthorized is returned when authentication fails (HTTP 401).
// This error should only be used with errors.Is() for comparison, not for type assertions.
var ErrUnauthorized = errors.New("unauthorized")

// ErrPermissionDenied is returned when the user lacks permission for the operation (HTTP 403).
// This error should only be used with errors.Is() for comparison, not for type assertions.
var ErrPermissionDenied = errors.New("permission denied")

// ErrRepositoryNotFound is returned when the repository does not exist (HTTP 404).
// This error should only be used with errors.Is() for comparison, not for type assertions.
var ErrRepositoryNotFound = errors.New("repository not found")

// ServerUnavailableError provides structured information about a Git server that is unavailable.
type ServerUnavailableError struct {
	// StatusCode is the HTTP status code (5xx)
	StatusCode int
	// Operation is the HTTP method that failed (e.g., "GET", "POST", "PUT")
	Operation string
	// Underlying is the underlying error
	Underlying error
	// RetryAfter is the delay the server asked for via the Retry-After header.
	// Zero means the server did not ask for a specific delay.
	RetryAfter time.Duration
	// RateLimited reports that the server rejected the request for rate
	// limiting rather than failing it, which makes it retryable for any method
	// whose body can be replayed. Read it through IsRateLimited, which also
	// covers a 429 built without this set.
	RateLimited bool
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

// RetryAfterDelay implements retry.RetryAfter so retriers can honor the delay
// the server asked for instead of their own backoff schedule.
func (e *ServerUnavailableError) RetryAfterDelay() (time.Duration, bool) {
	return e.RetryAfter, e.RetryAfter > 0
}

// IsRateLimited reports whether the server rejected the request for rate
// limiting rather than failing it. A 429 is one by definition, however the
// error was built; any other status needs RateLimited set, since nothing in the
// status alone says so.
func (e *ServerUnavailableError) IsRateLimited() bool {
	return e.RateLimited || e.StatusCode == http.StatusTooManyRequests
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

// CheckServerUnavailable checks if an HTTP response indicates a transient
// failure worth retrying. It checks for:
//   - Server errors (5xx status codes)
//   - Too Many Requests (429 status code)
//   - Forbidden (403) carrying rate-limit signals
//
// If the response is one of those, it returns a ServerUnavailableError carrying
// any Retry-After delay the server asked for.
// The HTTP method is extracted from the response's request.
// The caller is responsible for closing the response body.
func CheckServerUnavailable(res *http.Response) error {
	return checkServerUnavailable(res, nil)
}

// checkServerUnavailable is CheckServerUnavailable with a caller-supplied
// detector merged into the built-in rules. A detector may classify a response
// the built-in rules call permanent, and may supply a delay for one they
// already classified, but may not un-classify one they call transient.
func checkServerUnavailable(res *http.Response, detect options.RateLimitDetector) error {
	var signal options.RateLimitSignal
	if detect != nil {
		signal = detect(res)
	}

	if res.StatusCode < 400 && !signal.RateLimited {
		return nil
	}

	// Both delays are floors, so the longer wins; shortening one is MaxDelay's
	// job, not the detector's.
	retryAfter, hasRetryAfter := parseRetryAfter(res.Header, time.Now())
	retryAfter = max(retryAfter, signal.RetryAfter)

	rateLimited := signal.RateLimited || isRateLimitStatus(res.StatusCode, res.Header, hasRetryAfter)
	if !rateLimited && !isServerErrorStatus(res.StatusCode) {
		return nil
	}

	operation := ""
	if res.Request != nil {
		operation = res.Request.Method
	}
	err := NewServerUnavailableError(operation, res.StatusCode, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status))
	err.RetryAfter = retryAfter
	err.RateLimited = rateLimited
	return err
}

// rateLimitRemainingSuffix matches the canonicalized name of the header that
// reports remaining rate-limit budget, under either convention in use: GitHub's
// x-ratelimit-remaining and the IETF draft's ratelimit-remaining (GitLab).
// Matching the suffix rather than a fixed list means a provider following
// either convention needs no change here.
const rateLimitRemainingSuffix = "Ratelimit-Remaining"

// maxRetryAfterSeconds is the largest Retry-After value representable as a
// time.Duration. Larger values saturate instead of wrapping negative.
const maxRetryAfterSeconds = int64(math.MaxInt64 / int64(time.Second))

// isServerErrorStatus reports whether a status is a server-side failure, which
// is transient but not rate limiting.
func isServerErrorStatus(statusCode int) bool {
	return statusCode >= 500
}

// isRateLimitStatus reports whether a response is a rate-limit rejection under
// the signalling nanogit recognises without a detector.
func isRateLimitStatus(statusCode int, header http.Header, hasRetryAfter bool) bool {
	switch statusCode {
	case http.StatusTooManyRequests:
		return true
	case http.StatusForbidden:
		// GitHub reports secondary rate limits as 403; only the rate-limit
		// signals tell those apart from a real permission denial.
		return hasRetryAfter || rateLimitExhausted(header)
	default:
		return false
	}
}

// rateLimitExhausted reports whether the response says the rate-limit budget is spent.
func rateLimitExhausted(header http.Header) bool {
	for name, values := range header {
		if !strings.HasSuffix(name, rateLimitRemainingSuffix) {
			continue
		}
		for _, value := range values {
			if strings.TrimSpace(value) == "0" {
				return true
			}
		}
	}
	return false
}

// parseRetryAfter reads the Retry-After header in either RFC 9110 form: a delay
// in seconds, or an HTTP date. The bool reports whether the header was present
// and understood, which is a retry signal on its own even when the delay is zero.
func parseRetryAfter(header http.Header, now time.Time) (time.Duration, bool) {
	value := strings.TrimSpace(header.Get("Retry-After"))
	if value == "" {
		return 0, false
	}

	// On ErrRange ParseInt still yields the nearest bound, which is exactly the
	// saturation the switch below wants.
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil || errors.Is(err, strconv.ErrRange) {
		switch {
		case seconds <= 0:
			return 0, true
		case seconds > maxRetryAfterSeconds:
			return time.Duration(math.MaxInt64), true
		default:
			return time.Duration(seconds) * time.Second, true
		}
	}

	// Time.Sub already saturates at the Duration bounds.
	if date, err := http.ParseTime(value); err == nil {
		if delay := date.Sub(now); delay > 0 {
			return delay, true
		}
		return 0, true
	}

	return 0, false
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
	operation := e.Operation
	if operation == "" {
		operation = "unknown"
	}
	endpoint := e.Endpoint
	if endpoint == "" {
		endpoint = "unknown"
	}

	if e.Underlying != nil {
		return fmt.Sprintf("unauthorized (operation %s, endpoint %s, status code %d): %v",
			operation, endpoint, e.StatusCode, e.Underlying)
	}
	return fmt.Sprintf("unauthorized (operation %s, endpoint %s, status code %d)",
		operation, endpoint, e.StatusCode)
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
	// StatusCode is the HTTP status code (403)
	StatusCode int
	// Operation is the HTTP method that failed (e.g., "GET", "POST")
	Operation string
	// Endpoint is the Git protocol endpoint (e.g., "git-receive-pack", "git-upload-pack")
	Endpoint string
	// Underlying is the underlying error
	Underlying error
}

func (e *PermissionDeniedError) Error() string {
	operation := e.Operation
	if operation == "" {
		operation = "unknown"
	}
	endpoint := e.Endpoint
	if endpoint == "" {
		endpoint = "unknown"
	}

	if e.Underlying != nil {
		return fmt.Sprintf("permission denied (operation %s, endpoint %s, status code %d): %v",
			operation, endpoint, e.StatusCode, e.Underlying)
	}
	return fmt.Sprintf("permission denied (operation %s, endpoint %s, status code %d)",
		operation, endpoint, e.StatusCode)
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
	// StatusCode is the HTTP status code (404)
	StatusCode int
	// Operation is the HTTP method that failed (e.g., "GET", "POST")
	Operation string
	// Endpoint is the Git protocol endpoint (e.g., "git-receive-pack", "git-upload-pack")
	Endpoint string
	// Underlying is the underlying error
	Underlying error
}

func (e *RepositoryNotFoundError) Error() string {
	operation := e.Operation
	if operation == "" {
		operation = "unknown"
	}
	endpoint := e.Endpoint
	if endpoint == "" {
		endpoint = "unknown"
	}

	if e.Underlying != nil {
		return fmt.Sprintf("repository not found (operation %s, endpoint %s, status code %d): %v",
			operation, endpoint, e.StatusCode, e.Underlying)
	}
	return fmt.Sprintf("repository not found (operation %s, endpoint %s, status code %d)",
		operation, endpoint, e.StatusCode)
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
// Rate-limited 403s are reported by CheckServerUnavailable instead, so callers
// run that check first and never reach this one for them.
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
		if res.Request.URL != nil {
			endpoint = extractEndpoint(res.Request.URL.Path)
		}
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
