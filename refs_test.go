package nanogit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/require"
)

func TestListRefs(t *testing.T) {
	hashify := func(h string) hash.Hash {
		parsedHex, err := hash.FromHex(h)
		require.NoError(t, err)
		return parsedHex
	}

	tests := []struct {
		name          string
		lsRefsResp    string
		expectedRefs  []Ref
		expectedError string
		setupClient   options.Option
	}{
		{
			name: "successful response with multiple refs",
			lsRefsResp: func() string {
				pkt, _ := protocol.FormatPacks(
					protocol.PackLine("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d refs/heads/master\n"),
					protocol.PackLine("8fd1a60b01f91b314f59955a4e4d4e80d8edf11e refs/heads/develop\n"),
					protocol.PackLine("9fd1a60b01f91b314f59955a4e4d4e80d8edf11f refs/tags/v1.0.0\n"),
				)
				return string(pkt)
			}(),
			expectedRefs: []Ref{
				{Name: "refs/heads/master", Hash: hashify("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d")},
				{Name: "refs/heads/develop", Hash: hashify("8fd1a60b01f91b314f59955a4e4d4e80d8edf11e")},
				{Name: "refs/tags/v1.0.0", Hash: hashify("9fd1a60b01f91b314f59955a4e4d4e80d8edf11f")},
			},
			expectedError: "",
		},
		{
			name: "HEAD reference with symref",
			lsRefsResp: func() string {
				pkt, _ := protocol.FormatPacks(
					protocol.PackLine("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d HEAD symref=HEAD:refs/heads/master\n"),
				)
				return string(pkt)
			}(),
			expectedRefs: []Ref{
				{Name: "refs/heads/master", Hash: hashify("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d")},
			},
			expectedError: "",
		},
		{
			name:          "empty response",
			lsRefsResp:    "0000",
			expectedRefs:  []Ref{},
			expectedError: "",
		},
		{
			name: "invalid hash length",
			lsRefsResp: `003f7fd1a60b01f91b314f59955a4e4d4e80d8ed refs/heads/master
0000`,
			expectedRefs:  nil,
			expectedError: "invalid hash length: got 36, want 40",
		},
		{
			name: "invalid ref format",
			lsRefsResp: `003f7fd1a60b01f91b314f59955a4e4d4e80d8edf11d
0000`,
			expectedRefs:  nil,
			expectedError: "error parsing line \"003f7fd1a60b01f91b314f59955a4e4d4e80d8edf11d\\n0000\": line declared 63 bytes, but only 49 are available",
		},
		{
			name:          "ls-refs request fails",
			lsRefsResp:    "",
			expectedRefs:  nil,
			expectedError: "send ls-refs command",
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

			var (
				client Client
				err    error
			)

			if tt.setupClient != nil {
				client, err = NewHTTPClient(url, tt.setupClient)
			} else {
				client, err = NewHTTPClient(url)
			}
			require.NoError(t, err)
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
	hashify := func(h string) hash.Hash {
		parsedHex, err := hash.FromHex(h)
		require.NoError(t, err)
		return parsedHex
	}

	tests := []struct {
		name          string
		lsRefsResp    string
		refToGet      string
		expectedRef   Ref
		expectedError error
		setupClient   options.Option
	}{
		{
			name: "successful get of existing ref",
			lsRefsResp: func() string {
				pkt, _ := protocol.FormatPacks(
					protocol.PackLine("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d refs/heads/master\n"),
				)
				return string(pkt)
			}(),
			refToGet: "refs/heads/master",
			expectedRef: Ref{
				Name: "refs/heads/master",
				Hash: hashify("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d"),
			},
			expectedError: nil,
		},
		{
			name: "ref not found",
			lsRefsResp: func() string {
				pkt, _ := protocol.FormatPacks(
					protocol.PackLine("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d refs/heads/master\n"),
				)
				return string(pkt)
			}(),
			refToGet:      "refs/heads/non-existent",
			expectedRef:   Ref{},
			expectedError: nil, // We'll check for structured error type
		},
		{
			name:          "empty ref line",
			refToGet:      "",
			expectedRef:   Ref{},
			expectedError: ErrEmptyRefName, // We'll check for error or empty result
		},
		{
			name:          "ls-refs request fails",
			lsRefsResp:    "",
			refToGet:      "refs/heads/master",
			expectedRef:   Ref{},
			expectedError: nil, // We'll check for wrapped error differently
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

			var (
				client Client
				err    error
			)

			if tt.setupClient != nil {
				client, err = NewHTTPClient(url, tt.setupClient)
			} else {
				client, err = NewHTTPClient(url)
			}
			require.NoError(t, err)

			ref, err := client.GetRef(context.Background(), tt.refToGet)
			if tt.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedError)
				require.Equal(t, Ref{}, ref)
			} else if tt.setupClient != nil {
				// For network timeout cases, just check that we got an error
				require.Error(t, err)
				require.Contains(t, err.Error(), "send ls-refs command")
				require.Equal(t, Ref{}, ref)
			} else if tt.refToGet == "refs/heads/non-existent" {
				// Check for structured RefNotFoundError
				require.Error(t, err)
				var notFoundErr *RefNotFoundError
				require.ErrorAs(t, err, &notFoundErr)
				require.Equal(t, "refs/heads/non-existent", notFoundErr.RefName)
				require.Equal(t, Ref{}, ref)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedRef, ref)
			}
		})
	}
}

func TestCreateRef(t *testing.T) {
	hashify := func(h string) hash.Hash {
		parsedHex, err := hash.FromHex(h)
		require.NoError(t, err)
		return parsedHex
	}

	tests := []struct {
		name          string
		refToCreate   Ref
		refExists     bool
		expectedError string
		setupClient   options.Option
	}{
		{
			name: "successful ref creation",
			refToCreate: Ref{
				Name: "refs/heads/main",
				Hash: hashify("1234567890123456789012345678901234567890"),
			},
			refExists:     false,
			expectedError: "",
		},
		{
			name: "create ref that already exists",
			refToCreate: Ref{
				Name: "refs/heads/main",
				Hash: hashify("1234567890123456789012345678901234567890"),
			},
			refExists:     true,
			expectedError: "reference already exists: refs/heads/main",
		},
		{
			name: "empty ref name",
			refToCreate: Ref{
				Name: "",
				Hash: hashify("1234567890123456789012345678901234567890"),
			},
			refExists:     false,
			expectedError: ErrEmptyRefName.Error(),
		},
		{
			name: "ls-refs request fails",
			refToCreate: Ref{
				Name: "refs/heads/main",
				Hash: hashify("1234567890123456789012345678901234567890"),
			},
			refExists:     false,
			expectedError: "send ls-refs command",
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
			shouldCheckBody := tt.expectedError == ""
			if tt.setupClient == nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/git-upload-pack" {
						// Simulate refs list for GetRef in CreateRef tests
						var refsResp string
						if tt.refExists {
							// Ref exists
							pkt, _ := protocol.FormatPacks(
								protocol.PackLine(fmt.Sprintf("%s %s\n", tt.refToCreate.Hash, tt.refToCreate.Name)),
							)
							refsResp = string(pkt)
						} else {
							// Ref does not exist
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

			var (
				client Client
				err    error
			)

			if tt.setupClient != nil {
				client, err = NewHTTPClient(url, tt.setupClient)
			} else {
				client, err = NewHTTPClient(url)
			}
			require.NoError(t, err)

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
	hashify := func(h string) hash.Hash {
		parsedHex, err := hash.FromHex(h)
		require.NoError(t, err)
		return parsedHex
	}

	tests := []struct {
		name          string
		refToUpdate   Ref
		refExists     bool
		expectedError string
		setupClient   options.Option
	}{
		{
			name: "successful ref update",
			refToUpdate: Ref{
				Name: "refs/heads/main",
				Hash: hashify("1234567890123456789012345678901234567890"),
			},
			refExists:     true,
			expectedError: "",
		},
		{
			name: "update non-existent ref",
			refToUpdate: Ref{
				Name: "refs/heads/non-existent",
				Hash: hashify("abcdef1234567890123456789012345678901234"),
			},
			refExists:     false,
			expectedError: "reference not found: refs/heads/non-existent",
		},
		{
			name: "empty ref name",
			refToUpdate: Ref{
				Name: "",
				Hash: hashify("1234567890123456789012345678901234567890"),
			},
			refExists:     false,
			expectedError: ErrEmptyRefName.Error(),
		},
		{
			name: "ls-refs request fails",
			refToUpdate: Ref{
				Name: "refs/heads/main",
				Hash: hashify("1234567890123456789012345678901234567890"),
			},
			refExists:     false,
			expectedError: "send ls-refs command",
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
			shouldCheckBody := tt.expectedError == ""
			if tt.setupClient == nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/git-upload-pack" {
						// Simulate refs list for GetRef in UpdateRef tests
						var refsResp string
						if tt.refExists {
							// Ref exists
							pkt, _ := protocol.FormatPacks(
								protocol.PackLine(fmt.Sprintf("%s %s\n", tt.refToUpdate.Hash, tt.refToUpdate.Name)),
							)
							refsResp = string(pkt)
						} else {
							// Ref does not exist
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

			var (
				client Client
				err    error
			)

			if tt.setupClient != nil {
				client, err = NewHTTPClient(url, tt.setupClient)
			} else {
				client, err = NewHTTPClient(url)
			}
			require.NoError(t, err)

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
		refToDelete   string
		refExists     bool
		expectedError string
		setupClient   options.Option
	}{
		{
			name:          "successful ref deletion",
			refToDelete:   "refs/heads/main",
			refExists:     true,
			expectedError: "",
		},
		{
			name:          "delete non-existent ref",
			refToDelete:   "refs/heads/non-existent",
			refExists:     false,
			expectedError: "reference not found: refs/heads/non-existent",
		},
		{
			name:          "empty ref name",
			refToDelete:   "",
			refExists:     false,
			expectedError: ErrEmptyRefName.Error(),
		},
		{
			name:          "ls-refs request fails",
			refToDelete:   "refs/heads/main",
			refExists:     false,
			expectedError: "send ls-refs command",
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
			shouldCheckBody := tt.expectedError == "" || strings.Contains(tt.expectedError, "send ref update: got status code 500")
			if tt.setupClient == nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/git-upload-pack" {
						// Simulate refs list for GetRef in DeleteRef tests
						var refsResp string
						if tt.refExists {
							// Ref exists
							pkt, _ := protocol.FormatPacks(
								protocol.PackLine(fmt.Sprintf("%s %s\n", "1234567890123456789012345678901234567890", tt.refToDelete)),
							)
							refsResp = string(pkt)
						} else {
							// Ref does not exist
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

			var (
				client Client
				err    error
			)

			if tt.setupClient != nil {
				client, err = NewHTTPClient(url, tt.setupClient)
			} else {
				client, err = NewHTTPClient(url)
			}
			require.NoError(t, err)

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
