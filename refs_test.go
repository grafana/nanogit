package nanogit

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

			client, err := NewClient(url)
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
						// Simulate refs list for GetRef
						var refsResp string
						switch tt.name {
						case "successful get of existing ref":
							// Ref exists
							pkt, _ := protocol.FormatPacks(
								protocol.PackLine("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d refs/heads/master\n"),
							)
							refsResp = string(pkt)
						case "get non-existent ref":
							// Ref does not exist
							refsResp = "0000"
						default:
							refsResp = "0000"
						}
						w.WriteHeader(http.StatusOK)
						if _, err := w.Write([]byte(refsResp)); err != nil {
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

			client, err := NewClient(url)
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

func TestCreateRef(t *testing.T) {
	tests := []struct {
		name          string
		infoRefsResp  string
		refToCreate   Ref
		expectedError string
		setupClient   func(*clientImpl)
	}{
		{
			name:         "successful ref creation",
			infoRefsResp: "001e# service=git-receive-pack\n0000",
			refToCreate: Ref{
				Name: "refs/heads/main",
				Hash: "1234567890123456789012345678901234567890",
			},
			expectedError: "",
		},
		{
			name:         "create ref that already exists",
			infoRefsResp: "001e# service=git-receive-pack\n0000",
			refToCreate: Ref{
				Name: "refs/heads/main",
				Hash: "1234567890123456789012345678901234567890",
			},
			expectedError: "ref refs/heads/main already exists",
		},
		{
			name:         "create ref with invalid hash length",
			infoRefsResp: "001e# service=git-receive-pack\n0000",
			refToCreate: Ref{
				Name: "refs/heads/main",
				Hash: "1234", // too short
			},
			expectedError: "format ref update request: invalid new ref hash length",
		},
		{
			name:         "info/refs request fails",
			infoRefsResp: "",
			refToCreate: Ref{
				Name: "refs/heads/main",
				Hash: "1234567890123456789012345678901234567890",
			},
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
			name:         "receive-pack request fails",
			infoRefsResp: "001e# service=git-receive-pack\n0000",
			refToCreate: Ref{
				Name: "refs/heads/main",
				Hash: "1234567890123456789012345678901234567890",
			},
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
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var server *httptest.Server
			shouldCheckBody := tt.expectedError == "" || strings.Contains(tt.expectedError, "send ref update: got status code 500")
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
						// Simulate refs list for GetRef in CreateRef tests
						var refsResp string
						switch tt.name {
						case "successful ref creation":
							// Ref does not exist
							refsResp = "0000"
						case "create ref that already exists":
							// Ref exists
							pkt, _ := protocol.FormatPacks(
								protocol.PackLine(fmt.Sprintf("%s %s\n", tt.refToCreate.Hash, tt.refToCreate.Name)),
							)
							refsResp = string(pkt)
						default:
							refsResp = "0000"
						}
						w.WriteHeader(http.StatusOK)
						if _, err := w.Write([]byte(refsResp)); err != nil {
							t.Errorf("failed to write response: %v", err)
							return
						}
						return
					}
					if r.URL.Path == "/git-receive-pack" {
						if tt.expectedError == "send ref update: got status code 500" {
							w.WriteHeader(http.StatusInternalServerError)
							if _, err := w.Write([]byte("error: refs/heads/main already exists")); err != nil {
								t.Errorf("failed to write response: %v", err)
								return
							}
							return
						}
						if shouldCheckBody {
							body, err := io.ReadAll(r.Body)
							if err != nil {
								t.Errorf("failed to read request body: %v", err)
								return
							}
							expectedRefLine := fmt.Sprintf("%s %s %s\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n0000",
								protocol.ZeroHash, // old value is zero hash for new refs
								tt.refToCreate.Hash,
								tt.refToCreate.Name,
							)
							refLine := string(body[4 : len(body)-len(protocol.EmptyPack)-4])
							if refLine != expectedRefLine {
								t.Errorf("unexpected ref line:\ngot:  %q\nwant: %q", refLine, expectedRefLine)
								return
							}
							if !bytes.Equal(body[len(body)-len(protocol.EmptyPack)-4:len(body)-4], protocol.EmptyPack) {
								t.Error("empty pack file not found in request")
								return
							}
							if !bytes.Equal(body[len(body)-4:], []byte(protocol.FlushPacket)) {
								t.Error("flush packet not found in request")
								return
							}
						}
						w.WriteHeader(http.StatusOK)
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

			client, err := NewClient(url)
			require.NoError(t, err)
			if tt.setupClient != nil {
				c, ok := client.(*clientImpl)
				require.True(t, ok, "client should be of type *client")
				tt.setupClient(c)
			}

			err = client.CreateRef(context.Background(), tt.refToCreate)
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdateRef(t *testing.T) {
	tests := []struct {
		name          string
		infoRefsResp  string
		refToUpdate   Ref
		expectedError string
		setupClient   func(*clientImpl)
	}{
		{
			name:         "successful ref update",
			infoRefsResp: "001e# service=git-receive-pack\n0000",
			refToUpdate: Ref{
				Name: "refs/heads/main",
				Hash: "1234567890123456789012345678901234567890",
			},
			expectedError: "",
		},
		{
			name:         "update non-existent ref",
			infoRefsResp: "001e# service=git-receive-pack\n0000",
			refToUpdate: Ref{
				Name: "refs/heads/non-existent",
				Hash: "1234567890123456789012345678901234567890",
			},
			expectedError: "ref refs/heads/non-existent does not exist",
		},
		{
			name:         "update ref with invalid ref name",
			infoRefsResp: "001e# service=git-receive-pack\n0000",
			refToUpdate: Ref{
				Name: "invalid-ref", // missing refs/ prefix
				Hash: "1234567890123456789012345678901234567890",
			},
			expectedError: "ref invalid-ref does not exist",
		},
		{
			name:         "update ref with invalid hash length",
			infoRefsResp: "001e# service=git-receive-pack\n0000",
			refToUpdate: Ref{
				Name: "refs/heads/main",
				Hash: "1234", // too short
			},
			expectedError: "ref refs/heads/main does not exist",
		},
		{
			name:         "info/refs request fails",
			infoRefsResp: "",
			refToUpdate: Ref{
				Name: "refs/heads/main",
				Hash: "1234567890123456789012345678901234567890",
			},
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
			name:         "receive-pack request fails",
			infoRefsResp: "001e# service=git-receive-pack\n0000",
			refToUpdate: Ref{
				Name: "refs/heads/main",
				Hash: "1234567890123456789012345678901234567890",
			},
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
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var server *httptest.Server
			shouldCheckBody := tt.expectedError == "" || strings.Contains(tt.expectedError, "send ref update: got status code 500")
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
						// Simulate refs list for GetRef in UpdateRef tests
						var refsResp string
						switch tt.name {
						case "successful ref update":
							// Ref exists
							pkt, _ := protocol.FormatPacks(
								protocol.PackLine(fmt.Sprintf("%s %s\n", tt.refToUpdate.Hash, tt.refToUpdate.Name)),
							)
							refsResp = string(pkt)
						case "update non-existent ref":
							// Ref does not exist
							refsResp = "0000"
						default:
							refsResp = "0000"
						}
						w.WriteHeader(http.StatusOK)
						if _, err := w.Write([]byte(refsResp)); err != nil {
							t.Errorf("failed to write response: %v", err)
							return
						}
						return
					}
					if r.URL.Path == "/git-receive-pack" {
						if tt.expectedError == "ref refs/heads/non-existent does not exist" {
							w.WriteHeader(http.StatusInternalServerError)
							if _, err := w.Write([]byte("error: ref refs/heads/non-existent does not exist")); err != nil {
								t.Errorf("failed to write response: %v", err)
								return
							}
							return
						}
						if shouldCheckBody {
							body, err := io.ReadAll(r.Body)
							if err != nil {
								t.Errorf("failed to read request body: %v", err)
								return
							}
							expectedRefLine := fmt.Sprintf("%s %s %s\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n0000",
								tt.refToUpdate.Hash, // old value is the current hash
								tt.refToUpdate.Hash, // new value is the same hash
								tt.refToUpdate.Name,
							)
							refLine := string(body[4 : len(body)-len(protocol.EmptyPack)-4])
							if refLine != expectedRefLine {
								t.Errorf("unexpected ref line:\ngot:  %q\nwant: %q", refLine, expectedRefLine)
								return
							}
							if !bytes.Equal(body[len(body)-len(protocol.EmptyPack)-4:len(body)-4], protocol.EmptyPack) {
								t.Error("empty pack file not found in request")
								return
							}
							if !bytes.Equal(body[len(body)-4:], []byte(protocol.FlushPacket)) {
								t.Error("flush packet not found in request")
								return
							}
						}
						w.WriteHeader(http.StatusOK)
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

			client, err := NewClient(url)
			require.NoError(t, err)
			if tt.setupClient != nil {
				c, ok := client.(*clientImpl)
				require.True(t, ok, "client should be of type *client")
				tt.setupClient(c)
			}

			err = client.UpdateRef(context.Background(), tt.refToUpdate)
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDeleteRef(t *testing.T) {
	tests := []struct {
		name          string
		infoRefsResp  string
		refToDelete   string
		expectedError string
		setupClient   func(*clientImpl)
	}{
		{
			name:          "successful ref deletion",
			infoRefsResp:  "001e# service=git-receive-pack\n0000",
			refToDelete:   "refs/heads/main",
			expectedError: "",
		},
		{
			name:          "delete non-existent ref",
			infoRefsResp:  "001e# service=git-receive-pack\n0000",
			refToDelete:   "refs/heads/non-existent",
			expectedError: "ref refs/heads/non-existent does not exist",
		},
		{
			name:          "delete ref with invalid ref name",
			infoRefsResp:  "001e# service=git-receive-pack\n0000",
			refToDelete:   "invalid-ref", // missing refs/ prefix
			expectedError: "ref invalid-ref does not exist",
		},
		{
			name:          "info/refs request fails",
			infoRefsResp:  "",
			refToDelete:   "refs/heads/main",
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
			name:          "receive-pack request fails",
			infoRefsResp:  "001e# service=git-receive-pack\n0000",
			refToDelete:   "refs/heads/main",
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
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var server *httptest.Server
			shouldCheckBody := tt.expectedError == "" || strings.Contains(tt.expectedError, "send ref update: got status code 500")
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
						// Simulate refs list for GetRef in DeleteRef tests
						var refsResp string
						switch tt.name {
						case "successful ref deletion":
							// Ref exists
							pkt, _ := protocol.FormatPacks(
								protocol.PackLine(fmt.Sprintf("%s %s\n", "1234567890123456789012345678901234567890", tt.refToDelete)),
							)
							refsResp = string(pkt)
						case "delete non-existent ref":
							// Ref does not exist
							refsResp = "0000"
						default:
							refsResp = "0000"
						}
						w.WriteHeader(http.StatusOK)
						if _, err := w.Write([]byte(refsResp)); err != nil {
							t.Errorf("failed to write response: %v", err)
							return
						}
						return
					}
					if r.URL.Path == "/git-receive-pack" {
						if tt.expectedError == "ref refs/heads/non-existent does not exist" {
							w.WriteHeader(http.StatusInternalServerError)
							if _, err := w.Write([]byte("error: ref refs/heads/non-existent does not exist")); err != nil {
								t.Errorf("failed to write response: %v", err)
								return
							}
							return
						}
						if shouldCheckBody {
							body, err := io.ReadAll(r.Body)
							if err != nil {
								t.Errorf("failed to read request body: %v", err)
								return
							}
							expectedRefLine := fmt.Sprintf("%s %s %s\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n0000",
								"1234567890123456789012345678901234567890", // old value is the current hash
								protocol.ZeroHash,                          // new value is zero hash for deletion
								tt.refToDelete,
							)
							refLine := string(body[4 : len(body)-len(protocol.EmptyPack)-4])
							if refLine != expectedRefLine {
								t.Errorf("unexpected ref line:\ngot:  %q\nwant: %q", refLine, expectedRefLine)
								return
							}
							if !bytes.Equal(body[len(body)-len(protocol.EmptyPack)-4:len(body)-4], protocol.EmptyPack) {
								t.Error("empty pack file not found in request")
								return
							}
							if !bytes.Equal(body[len(body)-4:], []byte(protocol.FlushPacket)) {
								t.Error("flush packet not found in request")
								return
							}
						}
						w.WriteHeader(http.StatusOK)
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

			client, err := NewClient(url)
			require.NoError(t, err)
			if tt.setupClient != nil {
				c, ok := client.(*clientImpl)
				require.True(t, ok, "client should be of type *client")
				tt.setupClient(c)
			}

			err = client.DeleteRef(context.Background(), tt.refToDelete)
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
