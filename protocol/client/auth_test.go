package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafana/nanogit/options"
	"github.com/stretchr/testify/require"
)

func TestAuthentication(t *testing.T) {
	tests := []struct {
		name           string
		authOption     options.Option
		expectedHeader string
	}{
		{
			name:           "basic auth",
			authOption:     options.WithBasicAuth("user", "pass"),
			expectedHeader: "Basic dXNlcjpwYXNz",
		},
		{
			name:           "token auth",
			authOption:     options.WithTokenAuth("token123"),
			expectedHeader: "token123",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check default headers
				if gitProtocol := r.Header.Get("Git-Protocol"); gitProtocol != "version=2" {
					t.Errorf("expected Git-Protocol header 'version=2', got %s", gitProtocol)
					return
				}
				if userAgent := r.Header.Get("User-Agent"); userAgent != "nanogit/0" {
					t.Errorf("expected User-Agent header 'nanogit/0', got %s", userAgent)
					return
				}

				auth := r.Header.Get("Authorization")
				if auth != tt.expectedHeader {
					t.Errorf("expected Authorization header %s, got %s", tt.expectedHeader, auth)
					return
				}

				if contentType := r.Header.Get("Content-Type"); contentType != "application/x-git-upload-pack-request" {
					t.Errorf("expected Content-Type header 'application/x-git-upload-pack-request', got %s", contentType)
					return
				}

				if _, err := w.Write([]byte("ok")); err != nil {
					t.Errorf("failed to write response: %v", err)
					return
				}
			}))
			defer server.Close()

			c, err := NewRawClient(server.URL + "/repo", tt.authOption)
			require.NoError(t, err)

			responseReader, err := c.UploadPack(context.Background(), strings.NewReader("test"))
			require.NoError(t, err)
			_ = responseReader.Close()
		})
	}
}

func TestIsAuthorized(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedAuth  bool
		expectedError string
		setupAuth     func(*rawClient)
	}{
		{
			name:          "authorized with basic auth",
			statusCode:    http.StatusOK,
			responseBody:  "capabilities",
			expectedAuth:  true,
			expectedError: "",
			setupAuth: func(c *rawClient) {
				c.basicAuth = &struct{ Username, Password string }{"user", "pass"}
			},
		},
		{
			name:          "authorized with token auth",
			statusCode:    http.StatusOK,
			responseBody:  "capabilities",
			expectedAuth:  true,
			expectedError: "",
			setupAuth: func(c *rawClient) {
				token := "token123"
				c.tokenAuth = &token
			},
		},
		{
			name:          "unauthorized",
			statusCode:    http.StatusUnauthorized,
			responseBody:  "unauthorized",
			expectedAuth:  false,
			expectedError: "",
			setupAuth: func(c *rawClient) {
				c.basicAuth = &struct{ Username, Password string }{"user", "wrong"}
			},
		},
		{
			name:          "server error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  "server error",
			expectedAuth:  false,
			expectedError: "server unavailable",
			setupAuth: func(c *rawClient) {
				c.basicAuth = &struct{ Username, Password string }{"user", "pass"}
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/repo.git/info/refs" {
					t.Errorf("expected path /repo.git/info/refs, got %s", r.URL.Path)
					return
				}
				if r.URL.Query().Get("service") != "git-upload-pack" {
					t.Errorf("expected service=git-upload-pack, got %s", r.URL.Query().Get("service"))
					return
				}

				w.WriteHeader(tt.statusCode)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("failed to write response: %v", err)
					return
				}
			}))
			defer server.Close()

			client, err := NewRawClient(server.URL + "/repo")
			require.NoError(t, err)

			tt.setupAuth(client)

			authorized, err := client.IsAuthorized(context.Background())
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedAuth, authorized)
		})
	}
}
