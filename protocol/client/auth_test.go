package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthentication(t *testing.T) {
	tests := []struct {
		name           string
		authOption     Option
		expectedHeader string
	}{
		{
			name:           "basic auth",
			authOption:     WithBasicAuth("user", "pass"),
			expectedHeader: "Basic dXNlcjpwYXNz",
		},
		{
			name:           "token auth",
			authOption:     WithTokenAuth("token123"),
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

			c, err := NewRawClient(server.URL, tt.authOption)
			require.NoError(t, err)

			_, err = c.UploadPack(context.Background(), []byte("test"))
			require.NoError(t, err)
		})
	}
}

func TestWithBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		wantErr  error
	}{
		{
			name:     "valid credentials",
			username: "user",
			password: "pass",
			wantErr:  nil,
		},
		{
			name:     "empty username",
			username: "",
			password: "pass",
			wantErr:  errors.New("username cannot be empty"),
		},
		{
			name:     "empty password allowed",
			username: "user",
			password: "",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewRawClient("https://github.com/owner/repo", WithBasicAuth(tt.username, tt.password))
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)

			require.NotNil(t, client.basicAuth)
			require.Equal(t, tt.username, client.basicAuth.Username)
			require.Equal(t, tt.password, client.basicAuth.Password)
		})
	}
}

func TestWithTokenAuth(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:    "valid token",
			token:   "token123",
			wantErr: nil,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: errors.New("token cannot be empty"),
		},
		{
			name:    "token with bearer prefix",
			token:   "Bearer token123",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewRawClient("https://github.com/owner/repo", WithTokenAuth(tt.token))
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)

			require.NotNil(t, client.tokenAuth)
			require.Equal(t, tt.token, *client.tokenAuth)
		})
	}
}

func TestAuthConflict(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		wantErr error
	}{
		{
			name: "basic auth then token auth",
			options: []Option{
				WithBasicAuth("user", "pass"),
				WithTokenAuth("token123"),
			},
			wantErr: errors.New("cannot use both basic auth and token auth"),
		},
		{
			name: "token auth then basic auth",
			options: []Option{
				WithTokenAuth("token123"),
				WithBasicAuth("user", "pass"),
			},
			wantErr: errors.New("cannot use both basic auth and token auth"),
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewRawClient("https://github.com/owner/repo", tt.options...)
			require.Error(t, err)
			require.Equal(t, tt.wantErr.Error(), err.Error())
			require.Nil(t, client)
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
		setupAuth     func(*RawClient)
	}{
		{
			name:          "authorized with basic auth",
			statusCode:    http.StatusOK,
			responseBody:  "capabilities",
			expectedAuth:  true,
			expectedError: "",
			setupAuth: func(c *RawClient) {
				c.basicAuth = &struct{ Username, Password string }{"user", "pass"}
			},
		},
		{
			name:          "authorized with token auth",
			statusCode:    http.StatusOK,
			responseBody:  "capabilities",
			expectedAuth:  true,
			expectedError: "",
			setupAuth: func(c *RawClient) {
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
			setupAuth: func(c *RawClient) {
				c.basicAuth = &struct{ Username, Password string }{"user", "wrong"}
			},
		},
		{
			name:          "server error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  "server error",
			expectedAuth:  false,
			expectedError: "get repository info: got status code 500: 500 Internal Server Error",
			setupAuth: func(c *RawClient) {
				c.basicAuth = &struct{ Username, Password string }{"user", "pass"}
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/info/refs" {
					t.Errorf("expected path /info/refs, got %s", r.URL.Path)
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

			client, err := NewRawClient(server.URL)
			require.NoError(t, err)

			tt.setupAuth(client)

			authorized, err := client.IsAuthorized(context.Background())
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Equal(t, tt.expectedError, err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedAuth, authorized)
		})
	}
}
