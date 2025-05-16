package client

import (
	"context"
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
						if _, err := w.Write([]byte(tt.infoRefsResp)); err != nil {
							t.Errorf("failed to write response: %v", err)
							return
						}
						return
					}
					if r.URL.Path == "/git-upload-pack" {
						if _, err := w.Write([]byte(tt.lsRefsResp)); err != nil {
							t.Errorf("failed to write response: %v", err)
							return
						}
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

func TestGetRef(t *testing.T) {
	tests := []struct {
		name          string
		infoRefsResp  string
		lsRefsResp    string
		refToGet      string
		expectedRef   Ref
		expectedError error
		setupClient   func(*clientImpl)
	}{
		{
			name:         "successful get of existing ref",
			infoRefsResp: "001e# service=git-upload-pack\n0000",
			lsRefsResp: func() string {
				pkt, _ := protocol.FormatPacks(
					protocol.PackLine("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d refs/heads/master\n"),
				)
				return string(pkt)
			}(),
			refToGet: "refs/heads/master",
			expectedRef: Ref{
				Name: "refs/heads/master",
				Hash: "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
			},
			expectedError: nil,
		},
		{
			name:         "get non-existent ref",
			infoRefsResp: "001e# service=git-upload-pack\n0000",
			lsRefsResp: func() string {
				pkt, _ := protocol.FormatPacks(
					protocol.PackLine("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d refs/heads/master\n"),
				)
				return string(pkt)
			}(),
			refToGet:      "refs/heads/non-existent",
			expectedRef:   Ref{},
			expectedError: ErrRefNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var server *httptest.Server
			if tt.setupClient == nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, "/info/refs") {
						if _, err := w.Write([]byte(tt.infoRefsResp)); err != nil {
							t.Errorf("failed to write response: %v", err)
							return
						}
						return
					}
					if r.URL.Path == "/git-upload-pack" {
						if _, err := w.Write([]byte(tt.lsRefsResp)); err != nil {
							t.Errorf("failed to write response: %v", err)
							return
						}
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

			ref, err := client.GetRef(context.Background(), tt.refToGet)
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError, err)
				require.Equal(t, Ref{}, ref)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedRef, ref)
			}
		})
	}
}
