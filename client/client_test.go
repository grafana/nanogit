package client

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

	"github.com/grafana/nanogit/protocol"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		options []Option
		wantErr error
	}{
		{
			name:    "valid repo without options",
			repo:    "https://github.com/owner/repo",
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
				func(c *clientImpl) error {
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
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := New(tt.repo, tt.options...)
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

func TestWithGitHub(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		token    string
		wantPath string
		wantAuth string
	}{
		{
			name:     "clean URL",
			repo:     "https://github.com/owner/repo",
			token:    "token123",
			wantPath: "/owner/repo",
			wantAuth: "token token123",
		},
		{
			name:     "URL with .git suffix",
			repo:     "https://github.com/owner/repo.git",
			token:    "token123",
			wantPath: "/owner/repo",
			wantAuth: "token token123",
		},
		{
			name:     "URL with trailing slash",
			repo:     "https://github.com/owner/repo/",
			token:    "token123",
			wantPath: "/owner/repo",
			wantAuth: "token token123",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := New(tt.repo, WithTokenAuth(tt.token), WithGitHub())
			require.NoError(t, err)

			c, ok := client.(*clientImpl)
			require.True(t, ok, "client should be of type *client")

			require.Equal(t, tt.wantPath, c.base.Path)
			require.NotNil(t, c.tokenAuth)
			require.Equal(t, tt.wantAuth, *c.tokenAuth)
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
			client, err := New("https://github.com/owner/repo", WithHTTPClient(tt.httpClient))
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr.Error(), err.Error())
				return
			}

			require.NoError(t, err)

			c, ok := client.(*clientImpl)
			require.True(t, ok, "client should be of type *client")

			if tt.httpClient == nil {
				require.NotNil(t, c.client, "client should not be nil even when nil is provided")
			} else {
				require.Equal(t, tt.httpClient, c.client, "http client should match the provided client")
			}
		})
	}
}

func TestSendCommands(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  string
		expectedResult string
		setupClient    func(*clientImpl)
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
			setupClient: func(c *clientImpl) {
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
			setupClient: func(c *clientImpl) {
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

			client, err := New(url)
			require.NoError(t, err)
			if tt.setupClient != nil {
				c, ok := client.(*clientImpl)
				require.True(t, ok, "client should be of type *client")
				tt.setupClient(c)
			}

			response, err := client.SendCommands(context.Background(), []byte("test data"))
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

func TestSmartInfoRequest(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  string
		expectedResult string
		setupClient    func(*clientImpl)
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
			setupClient: func(c *clientImpl) {
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
			setupClient: func(c *clientImpl) {
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
					if r.URL.Query().Get("service") != "git-upload-pack" {
						t.Errorf("expected service=git-upload-pack, got %s", r.URL.Query().Get("service"))
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

			client, err := New(url)
			require.NoError(t, err)
			if tt.setupClient != nil {
				c, ok := client.(*clientImpl)
				require.True(t, ok, "client should be of type *client")
				tt.setupClient(c)
			}

			response, err := client.SmartInfoRequest(context.Background())
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
				if _, err := w.Write([]byte("ok")); err != nil {
					t.Errorf("failed to write response: %v", err)
					return
				}
			}))
			defer server.Close()

			client, err := New(server.URL, tt.authOption)
			require.NoError(t, err)

			_, err = client.SendCommands(context.Background(), []byte("test"))
			require.NoError(t, err)
		})
	}
}

func TestListRefs(t *testing.T) {
	tests := []struct {
		name          string
		infoRefsResp  string
		lsRefsResp    string
		expectedRefs  []Ref
		expectedError string
		setupClient   func(*clientImpl)
	}{
		{
			name:         "successful response with multiple refs",
			infoRefsResp: "001e# service=git-upload-pack\n0000",
			lsRefsResp: func() string {
				pkt, _ := protocol.FormatPacks(
					protocol.PackLine("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d refs/heads/master\n"),
					protocol.PackLine("8fd1a60b01f91b314f59955a4e4d4e80d8edf11e refs/heads/develop\n"),
					protocol.PackLine("9fd1a60b01f91b314f59955a4e4d4e80d8edf11f refs/tags/v1.0.0\n"),
				)
				return string(pkt)
			}(),
			expectedRefs: []Ref{
				{Name: "refs/heads/master", Hash: "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d"},
				{Name: "refs/heads/develop", Hash: "8fd1a60b01f91b314f59955a4e4d4e80d8edf11e"},
				{Name: "refs/tags/v1.0.0", Hash: "9fd1a60b01f91b314f59955a4e4d4e80d8edf11f"},
			},
			expectedError: "",
		},
		{
			name:         "HEAD reference with symref",
			infoRefsResp: "001e# service=git-upload-pack\n0000",
			lsRefsResp: func() string {
				pkt, _ := protocol.FormatPacks(
					protocol.PackLine("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d HEAD symref=HEAD:refs/heads/master\n"),
				)
				return string(pkt)
			}(),
			expectedRefs: []Ref{
				{Name: "refs/heads/master", Hash: "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d"},
			},
			expectedError: "",
		},
		{
			name:          "empty response",
			infoRefsResp:  "001e# service=git-upload-pack\n0000",
			lsRefsResp:    "0000",
			expectedRefs:  []Ref{},
			expectedError: "",
		},
		{
			name:         "invalid hash length",
			infoRefsResp: "001e# service=git-upload-pack\n0000",
			lsRefsResp: `003f7fd1a60b01f91b314f59955a4e4d4e80d8ed refs/heads/master
0000`,
			expectedRefs:  nil,
			expectedError: "invalid hash length: got 36, want 40",
		},
		{
			name:         "invalid ref format",
			infoRefsResp: "001e# service=git-upload-pack\n0000",
			lsRefsResp: `003f7fd1a60b01f91b314f59955a4e4d4e80d8edf11d
0000`,
			expectedRefs:  nil,
			expectedError: "error parsing line \"003f7fd1a60b01f91b314f59955a4e4d4e80d8edf11d\\n0000\": line declared 63 bytes, but only 49 are available",
		},
		{
			name:          "info/refs request fails",
			infoRefsResp:  "",
			lsRefsResp:    "",
			expectedRefs:  nil,
			expectedError: "get repository info",
			setupClient: func(c *clientImpl) {
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
		{
			name:          "ls-refs request fails",
			infoRefsResp:  "001e# service=git-upload-pack\n0000",
			lsRefsResp:    "",
			expectedRefs:  nil,
			expectedError: "get repository info: Get \"http://127.0.0.1:0/info/refs?service=git-upload-pack\": dial tcp 127.0.0.1:0: i/o timeout",
			setupClient: func(c *clientImpl) {
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
					if strings.HasPrefix(r.URL.Path, "/info/refs") {
						w.Write([]byte(tt.infoRefsResp))
						return
					}
					if r.URL.Path == "/git-upload-pack" {
						w.Write([]byte(tt.lsRefsResp))
						return
					}
					t.Errorf("unexpected request path: %s", r.URL.Path)
				}))
				defer server.Close()
			}

			url := "http://127.0.0.1:0"
			if server != nil {
				url = server.URL
			}

			client, err := New(url)
			require.NoError(t, err)
			if tt.setupClient != nil {
				c, ok := client.(*clientImpl)
				require.True(t, ok, "client should be of type *client")
				tt.setupClient(c)
			}

			refs, err := client.ListRefs(context.Background())
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				require.Nil(t, refs)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedRefs, refs)
			}
		})
	}
}
