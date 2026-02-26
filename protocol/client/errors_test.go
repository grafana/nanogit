package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerUnavailableError(t *testing.T) {
	t.Parallel()

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := errors.New("got status code 500: 500 Internal Server Error")
		err := NewServerUnavailableError("", 500, underlying)

		unwrapped := errors.Unwrap(err)
		require.Equal(t, underlying, unwrapped, "Unwrap should return the underlying error")
	})

	t.Run("Is enables errors.Is compatibility", func(t *testing.T) {
		t.Parallel()
		underlying := errors.New("got status code 500: 500 Internal Server Error")
		err := NewServerUnavailableError("", 500, underlying)

		require.True(t, errors.Is(err, ErrServerUnavailable), "errors.Is should find ErrServerUnavailable")
		require.False(t, errors.Is(err, errors.New("different error")), "errors.Is should not match different errors")
	})

	t.Run("Unwrap preserves error chain", func(t *testing.T) {
		t.Parallel()
		underlying := fmt.Errorf("got status code 500: %w", errors.New("Internal Server Error"))
		err := NewServerUnavailableError("", 500, underlying)

		// Unwrap should return the underlying error
		unwrapped := errors.Unwrap(err)
		require.Equal(t, underlying, unwrapped)

		// Should still be able to check for ErrServerUnavailable
		require.True(t, errors.Is(err, ErrServerUnavailable))

		// Should be able to unwrap further to get the original error
		originalErr := errors.Unwrap(unwrapped)
		require.NotNil(t, originalErr)
		require.Contains(t, originalErr.Error(), "Internal Server Error")
	})

	t.Run("Error message includes status code and underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := errors.New("got status code 500: 500 Internal Server Error")
		err := NewServerUnavailableError("", 500, underlying)

		msg := err.Error()
		require.Contains(t, msg, "server unavailable")
		require.Contains(t, msg, "status code 500")
		require.Contains(t, msg, underlying.Error())
	})

	t.Run("Error message works with nil underlying error", func(t *testing.T) {
		t.Parallel()
		err := NewServerUnavailableError("", 503, nil)

		msg := err.Error()
		require.Contains(t, msg, "server unavailable")
		require.Contains(t, msg, "status code 503")
	})
}

func TestCheckHTTPClientError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		statusCode     int
		method         string
		path           string
		expectedError  error
		expectedType   interface{}
		expectedNil    bool
		checkOperation string
		checkEndpoint  string
	}{
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			method:         "POST",
			path:           "/repo.git/git-receive-pack",
			expectedError:  ErrUnauthorized,
			expectedType:   &UnauthorizedError{},
			checkOperation: "POST",
			checkEndpoint:  "git-receive-pack",
		},
		{
			name:           "403 Forbidden on receive-pack",
			statusCode:     http.StatusForbidden,
			method:         "POST",
			path:           "/repo.git/git-receive-pack",
			expectedError:  ErrPermissionDenied,
			expectedType:   &PermissionDeniedError{},
			checkOperation: "POST",
			checkEndpoint:  "git-receive-pack",
		},
		{
			name:           "403 Forbidden on info/refs",
			statusCode:     http.StatusForbidden,
			method:         "GET",
			path:           "/repo.git/info/refs?service=git-receive-pack",
			expectedError:  ErrPermissionDenied,
			expectedType:   &PermissionDeniedError{},
			checkOperation: "GET",
			checkEndpoint:  "info/refs",
		},
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			method:         "GET",
			path:           "/repo.git/info/refs",
			expectedError:  ErrRepositoryNotFound,
			expectedType:   &RepositoryNotFoundError{},
			checkOperation: "GET",
			checkEndpoint:  "info/refs",
		},
		{
			name:        "200 OK returns nil",
			statusCode:  http.StatusOK,
			method:      "GET",
			path:        "/repo.git/info/refs",
			expectedNil: true,
		},
		{
			name:        "301 Redirect returns nil",
			statusCode:  http.StatusMovedPermanently,
			method:      "GET",
			path:        "/repo.git/info/refs",
			expectedNil: true,
		},
		{
			name:        "400 Bad Request returns nil (other 4xx)",
			statusCode:  http.StatusBadRequest,
			method:      "POST",
			path:        "/repo.git/git-receive-pack",
			expectedNil: true,
		},
		{
			name:        "500 Internal Server Error returns nil (5xx handled elsewhere)",
			statusCode:  http.StatusInternalServerError,
			method:      "GET",
			path:        "/repo.git/info/refs",
			expectedNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a test HTTP request
			req := httptest.NewRequest(tt.method, "http://example.com"+tt.path, nil)

			// Create a test HTTP response
			res := &http.Response{
				StatusCode: tt.statusCode,
				Status:     http.StatusText(tt.statusCode),
				Request:    req,
			}

			err := CheckHTTPClientError(res)

			if tt.expectedNil {
				assert.Nil(t, err)
				return
			}

			require.NotNil(t, err)

			// Check errors.Is() compatibility
			assert.True(t, errors.Is(err, tt.expectedError),
				"expected errors.Is to match %v", tt.expectedError)

			// Check errors.As() compatibility and structured fields
			switch tt.expectedType.(type) {
			case *UnauthorizedError:
				var unauthErr *UnauthorizedError
				require.True(t, errors.As(err, &unauthErr))
				assert.Equal(t, http.StatusUnauthorized, unauthErr.StatusCode)
				assert.Equal(t, tt.checkOperation, unauthErr.Operation)
				assert.Equal(t, tt.checkEndpoint, unauthErr.Endpoint)
				assert.NotNil(t, unauthErr.Underlying)
			case *PermissionDeniedError:
				var permErr *PermissionDeniedError
				require.True(t, errors.As(err, &permErr))
				assert.Equal(t, http.StatusForbidden, permErr.StatusCode)
				assert.Equal(t, tt.checkOperation, permErr.Operation)
				assert.Equal(t, tt.checkEndpoint, permErr.Endpoint)
				assert.NotNil(t, permErr.Underlying)
			case *RepositoryNotFoundError:
				var notFoundErr *RepositoryNotFoundError
				require.True(t, errors.As(err, &notFoundErr))
				assert.Equal(t, http.StatusNotFound, notFoundErr.StatusCode)
				assert.Equal(t, tt.checkOperation, notFoundErr.Operation)
				assert.Equal(t, tt.checkEndpoint, notFoundErr.Endpoint)
				assert.NotNil(t, notFoundErr.Underlying)
			default:
				t.Fatalf("unexpected type: %T", tt.expectedType)
			}
		})
	}
}

func TestExtractEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path     string
		expected string
	}{
		{
			path:     "/repo.git/git-receive-pack",
			expected: "git-receive-pack",
		},
		{
			path:     "/org/repo.git/git-upload-pack",
			expected: "git-upload-pack",
		},
		{
			path:     "/repo.git/info/refs",
			expected: "info/refs",
		},
		{
			path:     "/repo.git/info/refs?service=git-receive-pack",
			expected: "info/refs", // Query string is ignored
		},
		{
			path:     "/some/unknown/path",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			result := extractEndpoint(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnauthorizedError(t *testing.T) {
	t.Parallel()

	t.Run("Error message with underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := fmt.Errorf("got status code 401: 401 Unauthorized")
		err := NewUnauthorizedError("POST", "git-receive-pack", underlying)

		expected := "unauthorized (operation POST, endpoint git-receive-pack, status code 401): got status code 401: 401 Unauthorized"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Error message without underlying error", func(t *testing.T) {
		t.Parallel()
		err := NewUnauthorizedError("GET", "info/refs", nil)

		expected := "unauthorized (operation GET, endpoint info/refs, status code 401)"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := fmt.Errorf("wrapped error")
		err := NewUnauthorizedError("POST", "git-receive-pack", underlying)

		assert.Equal(t, underlying, err.Unwrap())
	})

	t.Run("Is enables errors.Is compatibility", func(t *testing.T) {
		t.Parallel()
		err := NewUnauthorizedError("POST", "git-receive-pack", nil)

		assert.True(t, errors.Is(err, ErrUnauthorized))
		assert.False(t, errors.Is(err, ErrPermissionDenied))
		assert.False(t, errors.Is(err, ErrRepositoryNotFound))
	})
}

func TestPermissionDeniedError(t *testing.T) {
	t.Parallel()

	t.Run("Error message with underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := fmt.Errorf("got status code 403: 403 Forbidden")
		err := NewPermissionDeniedError("POST", "git-receive-pack", underlying)

		expected := "permission denied (operation POST, endpoint git-receive-pack, status code 403): got status code 403: 403 Forbidden"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Error message without underlying error", func(t *testing.T) {
		t.Parallel()
		err := NewPermissionDeniedError("GET", "info/refs", nil)

		expected := "permission denied (operation GET, endpoint info/refs, status code 403)"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := fmt.Errorf("wrapped error")
		err := NewPermissionDeniedError("POST", "git-receive-pack", underlying)

		assert.Equal(t, underlying, err.Unwrap())
	})

	t.Run("Is enables errors.Is compatibility", func(t *testing.T) {
		t.Parallel()
		err := NewPermissionDeniedError("POST", "git-receive-pack", nil)

		assert.True(t, errors.Is(err, ErrPermissionDenied))
		assert.False(t, errors.Is(err, ErrUnauthorized))
		assert.False(t, errors.Is(err, ErrRepositoryNotFound))
	})
}

func TestRepositoryNotFoundError(t *testing.T) {
	t.Parallel()

	t.Run("Error message with underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := fmt.Errorf("got status code 404: 404 Not Found")
		err := NewRepositoryNotFoundError("GET", "info/refs", underlying)

		expected := "repository not found (operation GET, endpoint info/refs, status code 404): got status code 404: 404 Not Found"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Error message without underlying error", func(t *testing.T) {
		t.Parallel()
		err := NewRepositoryNotFoundError("GET", "info/refs", nil)

		expected := "repository not found (operation GET, endpoint info/refs, status code 404)"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := fmt.Errorf("wrapped error")
		err := NewRepositoryNotFoundError("GET", "info/refs", underlying)

		assert.Equal(t, underlying, err.Unwrap())
	})

	t.Run("Is enables errors.Is compatibility", func(t *testing.T) {
		t.Parallel()
		err := NewRepositoryNotFoundError("GET", "info/refs", nil)

		assert.True(t, errors.Is(err, ErrRepositoryNotFound))
		assert.False(t, errors.Is(err, ErrUnauthorized))
		assert.False(t, errors.Is(err, ErrPermissionDenied))
	})
}

func TestCheckHTTPClientError_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles responses with nil Request", func(t *testing.T) {
		t.Parallel()
		res := &http.Response{
			StatusCode: http.StatusUnauthorized,
			Status:     "401 Unauthorized",
			Request:    nil,
		}

		err := CheckHTTPClientError(res)
		require.NotNil(t, err)

		var unauthErr *UnauthorizedError
		require.True(t, errors.As(err, &unauthErr))
		assert.Equal(t, "", unauthErr.Operation)
		assert.Equal(t, "", unauthErr.Endpoint)
	})

	t.Run("handles request with empty URL path", func(t *testing.T) {
		t.Parallel()
		req := &http.Request{
			Method: "POST",
			URL:    &url.URL{Path: ""},
		}

		res := &http.Response{
			StatusCode: http.StatusForbidden,
			Status:     "403 Forbidden",
			Request:    req,
		}

		err := CheckHTTPClientError(res)
		require.NotNil(t, err)

		var permErr *PermissionDeniedError
		require.True(t, errors.As(err, &permErr))
		assert.Equal(t, "POST", permErr.Operation)
		assert.Equal(t, "unknown", permErr.Endpoint)
	})
}
