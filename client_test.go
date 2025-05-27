package nanogit

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		options []Option
		wantErr error
	}{
		{
			name:    "valid HTTPS repo without options",
			repo:    "https://github.com/owner/repo",
			options: nil,
			wantErr: nil,
		},
		{
			name:    "valid HTTP repo without options",
			repo:    "http://github.com/owner/repo",
			options: nil,
			wantErr: nil,
		},
		{
			name:    "invalid repo URL",
			repo:    "://invalid-url-with-no-scheme",
			options: nil,
			wantErr: errors.New("parsing url: parse \"://invalid-url-with-no-scheme\": missing protocol scheme"),
		},
		{
			name: "valid repo with basic auth",
			repo: "https://github.com/owner/repo",
			options: []Option{
				WithBasicAuth("user", "pass"),
			},
			wantErr: nil,
		},
		{
			name: "valid repo with token auth",
			repo: "https://github.com/owner/repo",
			options: []Option{
				WithTokenAuth("token123"),
			},
			wantErr: nil,
		},
		{
			name: "valid repo with custom user agent",
			repo: "https://github.com/owner/repo",
			options: []Option{
				WithUserAgent("custom-agent/1.0"),
			},
			wantErr: nil,
		},
		{
			name: "option returns error",
			repo: "https://github.com/owner/repo",
			options: []Option{
				func(c *httpClient) error {
					return errors.New("option application failed")
				},
			},
			wantErr: errors.New("option application failed"),
		},
		{
			name: "nil option is skipped",
			repo: "https://github.com/owner/repo",
			options: []Option{
				nil,
				WithUserAgent("custom-agent/1.0"),
			},
			wantErr: nil,
		},
		{
			name:    "empty repo URL",
			repo:    "",
			options: nil,
			wantErr: errors.New("repository URL cannot be empty"),
		},
		{
			name:    "git protocol URL",
			repo:    "git://github.com/owner/repo",
			options: nil,
			wantErr: errors.New("only HTTP and HTTPS URLs are supported"),
		},
		{
			name:    "ssh protocol URL",
			repo:    "ssh://git@github.com/owner/repo",
			options: nil,
			wantErr: errors.New("only HTTP and HTTPS URLs are supported"),
		},
		{
			name: "multiple auth options",
			repo: "https://github.com/owner/repo",
			options: []Option{
				WithBasicAuth("user", "pass"),
				WithTokenAuth("token123"),
			},
			wantErr: errors.New("cannot use both basic auth and token auth"),
		},
		{
			name: "invalid basic auth",
			repo: "https://github.com/owner/repo",
			options: []Option{
				WithBasicAuth("", "pass"),
			},
			wantErr: errors.New("username cannot be empty"),
		},
		{
			name: "invalid token auth",
			repo: "https://github.com/owner/repo",
			options: []Option{
				WithTokenAuth(""),
			},
			wantErr: errors.New("token cannot be empty"),
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewHTTPClient(tt.repo, tt.options...)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}

func TestWithHTTPClient(t *testing.T) {
	tests := []struct {
		name       string
		httpClient *http.Client
		wantErr    error
	}{
		{
			name: "valid http client",
			httpClient: &http.Client{
				Timeout: 5 * time.Second,
			},
			wantErr: nil,
		},
		{
			name:       "nil http client",
			httpClient: nil,
			wantErr:    errors.New("httpClient is nil"),
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewHTTPClient("https://github.com/owner/repo", WithHTTPClient(tt.httpClient))
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr.Error(), err.Error())
				return
			}

			require.NoError(t, err)

			c, ok := client.(*httpClient)
			require.True(t, ok, "client should be of type *client")

			if tt.httpClient == nil {
				require.NotNil(t, c.client, "client should not be nil even when nil is provided")
			} else {
				require.Equal(t, tt.httpClient, c.client, "http client should match the provided client")
			}
		})
	}
}

type logEntry struct {
	level string
	msg   string
	args  []any
}

type testLogger struct {
	entries []logEntry
}

func (l *testLogger) Debug(msg string, keysAndValues ...any) {
	l.entries = append(l.entries, logEntry{"Debug", msg, keysAndValues})
}
func (l *testLogger) Info(msg string, keysAndValues ...any) {
	l.entries = append(l.entries, logEntry{"Info", msg, keysAndValues})
}
func (l *testLogger) Warn(msg string, keysAndValues ...any) {
	l.entries = append(l.entries, logEntry{"Warn", msg, keysAndValues})
}
func (l *testLogger) Error(msg string, keysAndValues ...any) {
	l.entries = append(l.entries, logEntry{"Error", msg, keysAndValues})
}

func TestUploadPack(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  string
		expectedResult string
		setupClient    func(*httpClient)
	}{
		{
			name:           "successful response",
			statusCode:     http.StatusOK,
			responseBody:   "response data",
			expectedError:  "",
			expectedResult: "response data",
			setupClient:    nil,
		},
		{
			name:           "not found",
			statusCode:     http.StatusNotFound,
			responseBody:   "not found",
			expectedError:  "got status code 404: 404 Not Found",
			expectedResult: "",
			setupClient:    nil,
		},
		{
			name:           "server error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   "server error",
			expectedError:  "got status code 500: 500 Internal Server Error",
			expectedResult: "",
			setupClient:    nil,
		},
		{
			name:           "timeout error",
			statusCode:     0,
			responseBody:   "",
			expectedError:  "context deadline exceeded",
			expectedResult: "",
			setupClient: func(c *httpClient) {
				c.client = &http.Client{
					Timeout: 1 * time.Nanosecond,
				}
			},
		},
		{
			name:           "connection refused",
			statusCode:     0,
			responseBody:   "",
			expectedError:  "i/o timeout",
			expectedResult: "",
			setupClient: func(c *httpClient) {
				c.base, _ = url.Parse("http://127.0.0.1:0")
				c.client = &http.Client{
					Transport: &http.Transport{
						DialContext: (&net.Dialer{
							Timeout: 1 * time.Nanosecond,
						}).DialContext,
					},
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var server *httptest.Server
			if tt.setupClient == nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/git-upload-pack" {
						t.Errorf("expected path /git-upload-pack, got %s", r.URL.Path)
						return
					}
					if r.Method != http.MethodPost {
						t.Errorf("expected method POST, got %s", r.Method)
						return
					}

					// Check default headers
					if gitProtocol := r.Header.Get("Git-Protocol"); gitProtocol != "version=2" {
						t.Errorf("expected Git-Protocol header 'version=2', got %s", gitProtocol)
						return
					}
					if userAgent := r.Header.Get("User-Agent"); userAgent != "nanogit/0" {
						t.Errorf("expected User-Agent header 'nanogit/0', got %s", userAgent)
						return
					}

					w.WriteHeader(tt.statusCode)
					if _, err := w.Write([]byte(tt.responseBody)); err != nil {
						t.Errorf("failed to write response: %v", err)
						return
					}
				}))
				defer server.Close()
			}

			url := "http://127.0.0.1:0"
			if server != nil {
				url = server.URL
			}

			client, err := NewHTTPClient(url)
			require.NoError(t, err)
			c, ok := client.(*httpClient)
			require.True(t, ok, "client should be of type *client")
			if tt.setupClient != nil {
				tt.setupClient(c)
			}

			response, err := c.uploadPack(context.Background(), []byte("test data"))
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				require.Empty(t, response)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, string(response))
			}
		})
	}
}

func TestReceivePack(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  string
		expectedResult string
		setupClient    func(*httpClient)
	}{
		{
			name:           "successful response",
			statusCode:     http.StatusOK,
			responseBody:   "refs data",
			expectedError:  "",
			expectedResult: "refs data",
			setupClient:    nil,
		},
		{
			name:           "not found",
			statusCode:     http.StatusNotFound,
			responseBody:   "not found",
			expectedError:  "got status code 404: 404 Not Found",
			expectedResult: "",
			setupClient:    nil,
		},
		{
			name:           "server error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   "server error",
			expectedError:  "got status code 500: 500 Internal Server Error",
			expectedResult: "",
			setupClient:    nil,
		},
		{
			name:           "timeout error",
			statusCode:     0,
			responseBody:   "",
			expectedError:  "context deadline exceeded",
			expectedResult: "",
			setupClient: func(c *httpClient) {
				c.client = &http.Client{
					Timeout: 1 * time.Nanosecond,
				}
			},
		},
		{
			name:           "connection refused",
			statusCode:     0,
			responseBody:   "",
			expectedError:  "i/o timeout",
			expectedResult: "",
			setupClient: func(c *httpClient) {
				c.base, _ = url.Parse("http://127.0.0.1:0")
				c.client = &http.Client{
					Transport: &http.Transport{
						DialContext: (&net.Dialer{
							Timeout: 1 * time.Nanosecond,
						}).DialContext,
					},
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var server *httptest.Server
			if tt.setupClient == nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/git-receive-pack" {
						t.Errorf("expected path /git-receive-pack, got %s", r.URL.Path)
						return
					}
					if r.Method != http.MethodPost {
						t.Errorf("expected method POST, got %s", r.Method)
						return
					}

					// Check default headers
					if gitProtocol := r.Header.Get("Git-Protocol"); gitProtocol != "version=2" {
						t.Errorf("expected Git-Protocol header 'version=2', got %s", gitProtocol)
						return
					}
					if userAgent := r.Header.Get("User-Agent"); userAgent != "nanogit/0" {
						t.Errorf("expected User-Agent header 'nanogit/0', got %s", userAgent)
						return
					}

					w.WriteHeader(tt.statusCode)
					if _, err := w.Write([]byte(tt.responseBody)); err != nil {
						t.Errorf("failed to write response: %v", err)
						return
					}
				}))
				defer server.Close()
			}

			url := "http://127.0.0.1:0"
			if server != nil {
				url = server.URL
			}

			client, err := NewHTTPClient(url)
			require.NoError(t, err)
			c, ok := client.(*httpClient)
			require.True(t, ok, "client should be of type *client")
			if tt.setupClient != nil {
				tt.setupClient(c)
			}

			response, err := c.receivePack(context.Background(), []byte("test data"))
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				require.Empty(t, response)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, string(response))
			}
		})
	}
}

func TestSmartInfo(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  string
		expectedResult string
		setupClient    func(*httpClient)
	}{
		{
			name:           "successful response",
			statusCode:     http.StatusOK,
			responseBody:   "refs data",
			expectedError:  "",
			expectedResult: "refs data",
			setupClient:    nil,
		},
		{
			name:           "not found",
			statusCode:     http.StatusNotFound,
			responseBody:   "not found",
			expectedError:  "got status code 404: 404 Not Found",
			expectedResult: "",
			setupClient:    nil,
		},
		{
			name:           "server error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   "server error",
			expectedError:  "got status code 500: 500 Internal Server Error",
			expectedResult: "",
			setupClient:    nil,
		},
		{
			name:           "timeout error",
			statusCode:     0,
			responseBody:   "",
			expectedError:  "context deadline exceeded",
			expectedResult: "",
			setupClient: func(c *httpClient) {
				c.client = &http.Client{
					Timeout: 1 * time.Nanosecond,
				}
			},
		},
		{
			name:           "connection refused",
			statusCode:     0,
			responseBody:   "",
			expectedError:  "i/o timeout",
			expectedResult: "",
			setupClient: func(c *httpClient) {
				c.base, _ = url.Parse("http://127.0.0.1:0")
				c.client = &http.Client{
					Transport: &http.Transport{
						DialContext: (&net.Dialer{
							Timeout: 1 * time.Nanosecond,
						}).DialContext,
					},
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var server *httptest.Server
			if tt.setupClient == nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if !strings.HasPrefix(r.URL.Path, "/info/refs") {
						t.Errorf("expected path starting with /info/refs, got %s", r.URL.Path)
						return
					}
					if r.URL.Query().Get("service") != "custom-service" {
						t.Errorf("expected service=custom-service, got %s", r.URL.Query().Get("service"))
						return
					}
					if r.Method != http.MethodGet {
						t.Errorf("expected method GET, got %s", r.Method)
						return
					}

					// Check default headers
					if gitProtocol := r.Header.Get("Git-Protocol"); gitProtocol != "version=2" {
						t.Errorf("expected Git-Protocol header 'version=2', got %s", gitProtocol)
						return
					}
					if userAgent := r.Header.Get("User-Agent"); userAgent != "nanogit/0" {
						t.Errorf("expected User-Agent header 'nanogit/0', got %s", userAgent)
						return
					}

					w.WriteHeader(tt.statusCode)
					if _, err := w.Write([]byte(tt.responseBody)); err != nil {
						t.Errorf("failed to write response: %v", err)
						return
					}
				}))
				defer server.Close()
			}

			url := "http://127.0.0.1:0"
			if server != nil {
				url = server.URL
			}

			client, err := NewHTTPClient(url)
			require.NoError(t, err)
			c, ok := client.(*httpClient)
			require.True(t, ok, "client should be of type *client")
			if tt.setupClient != nil {
				tt.setupClient(c)
			}

			response, err := c.smartInfo(context.Background(), "custom-service")
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				require.Empty(t, response)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, string(response))
			}
		})
	}
}
