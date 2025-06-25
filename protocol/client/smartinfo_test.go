package client

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/nanogit/options"
	"github.com/stretchr/testify/require"
)

func TestSmartInfo(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  string
		expectedResult string
		setupClient    options.Option
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
			setupClient: options.WithHTTPClient(&http.Client{
				Timeout: 1 * time.Nanosecond,
			}),
		},
		{
			name:           "connection refused",
			statusCode:     0,
			responseBody:   "",
			expectedError:  "i/o timeout",
			expectedResult: "",
			setupClient: options.WithHTTPClient(&http.Client{
				Transport: &http.Transport{
					DialContext: (&net.Dialer{
						Timeout: 1 * time.Nanosecond,
					}).DialContext,
				},
			}),
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

			var (
				client *rawClient
				err    error
			)

			if tt.setupClient != nil {
				client, err = NewRawClient(url, tt.setupClient)
			} else {
				client, err = NewRawClient(url)
			}
			require.NoError(t, err)

			responseReader, err := client.SmartInfo(context.Background(), "custom-service")
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				require.Nil(t, responseReader)
			} else {
				require.NoError(t, err)
				defer responseReader.Close()
				responseData, err := io.ReadAll(responseReader)
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, string(responseData))
			}
		})
	}
}
