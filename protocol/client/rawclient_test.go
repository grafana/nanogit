package client

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/nanogit/options"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {

	errOption := func(o *options.Options) error {
		return errors.New("option application failed")
	}

	tests := []struct {
		name    string
		repo    string
		options []options.Option
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
			options: []options.Option{
				options.WithBasicAuth("user", "pass"),
			},
			wantErr: nil,
		},
		{
			name: "valid repo with token auth",
			repo: "https://github.com/owner/repo",
			options: []options.Option{
				options.WithTokenAuth("token123"),
			},
			wantErr: nil,
		},
		{
			name: "valid repo with custom user agent",
			repo: "https://github.com/owner/repo",
			options: []options.Option{
				options.WithUserAgent("custom-agent/1.0"),
			},
			wantErr: nil,
		},
		{
			name: "option returns error",
			repo: "https://github.com/owner/repo",
			options: []options.Option{
				errOption,
			},
			wantErr: errors.New("option application failed"),
		},
		{
			name: "nil option is skipped",
			repo: "https://github.com/owner/repo",
			options: []options.Option{
				nil,
				options.WithUserAgent("custom-agent/1.0"),
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
			options: []options.Option{
				options.WithBasicAuth("user", "pass"),
				options.WithTokenAuth("token123"),
			},
			wantErr: errors.New("cannot use both basic auth and token auth"),
		},
		{
			name: "invalid basic auth",
			repo: "https://github.com/owner/repo",
			options: []options.Option{
				options.WithBasicAuth("", "pass"),
			},
			wantErr: errors.New("username cannot be empty"),
		},
		{
			name: "invalid token auth",
			repo: "https://github.com/owner/repo",
			options: []options.Option{
				options.WithTokenAuth(""),
			},
			wantErr: errors.New("token cannot be empty"),
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewRawClient(tt.repo, tt.options...)
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
			client, err := NewRawClient("https://github.com/owner/repo", options.WithHTTPClient(tt.httpClient))
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr.Error(), err.Error())
				return
			}

			require.NoError(t, err)

			if tt.httpClient == nil {
				require.NotNil(t, client.client, "client should not be nil even when nil is provided")
			} else {
				require.Equal(t, tt.httpClient, client.client, "http client should match the provided client")
			}
		})
	}
}
