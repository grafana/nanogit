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
			expectedError: "line declared 59 bytes but unexpected EOF occurred",
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
			expectedError: &RefNotFoundError{RefName: "refs/heads/non-existent"},
		},
		{
			name:          "empty ref line",
			refToGet:      "",
			expectedRef:   Ref{},
			expectedError: ErrEmptyRefName, // We'll check for error or empty result
		},
		{
			name: "multiple refs found - exact match",
			lsRefsResp: func() string {
				pkt, _ := protocol.FormatPacks(
					protocol.PackLine("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d refs/heads/test\n"),
					protocol.PackLine("8fd1a60b01f91b314f59955a4e4d4e80d8edf11e refs/heads/test-longer\n"),
				)
				return string(pkt)
			}(),
			refToGet: "refs/heads/test",
			expectedRef: Ref{
				Name: "refs/heads/test",
				Hash: hashify("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d"),
			},
			expectedError: nil,
		},
		{
			name: "multiple refs found - no exact match",
			lsRefsResp: func() string {
				pkt, _ := protocol.FormatPacks(
					protocol.PackLine("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d refs/heads/test-branch\n"),
					protocol.PackLine("8fd1a60b01f91b314f59955a4e4d4e80d8edf11e refs/heads/test-longer\n"),
				)
				return string(pkt)
			}(),
			refToGet:      "refs/heads/test",
			expectedRef:   Ref{},
			expectedError: &RefNotFoundError{RefName: "refs/heads/test"},
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
				if refNotFoundErr, ok := tt.expectedError.(*RefNotFoundError); ok {
					var actualErr *RefNotFoundError
					require.ErrorAs(t, err, &actualErr)
					require.Equal(t, refNotFoundErr.RefName, actualErr.RefName)
				} else {
					require.ErrorIs(t, err, tt.expectedError)
				}
				require.Equal(t, Ref{}, ref)
			} else if tt.setupClient != nil {
				// For network timeout cases, just check that we got an error
				require.Error(t, err)
				require.Contains(t, err.Error(), "send ls-refs command")
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

			server := setupUpdateRefTestServer(t, tt)
			defer func() {
				if server != nil {
					server.Close()
				}
			}()

			client := createTestClient(t, server, tt.setupClient)
			runUpdateRefTest(t, client, tt)
		})
	}
}

// setupUpdateRefTestServer creates a test server for UpdateRef tests
func setupUpdateRefTestServer(t *testing.T, tt struct {
	name          string
	refToUpdate   Ref
	refExists     bool
	expectedError string
	setupClient   options.Option
}) *httptest.Server {
	if tt.setupClient != nil {
		return nil
	}

	shouldCheckBody := tt.expectedError == ""
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/git-upload-pack":
			handleUploadPackRequest(t, w, tt)
		case "/git-receive-pack":
			handleReceivePackRequest(t, w, r, tt, shouldCheckBody)
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
	}))
}

// handleUploadPackRequest handles git-upload-pack requests in tests
func handleUploadPackRequest(t *testing.T, w http.ResponseWriter, tt struct {
	name          string
	refToUpdate   Ref
	refExists     bool
	expectedError string
	setupClient   options.Option
}) {
	var refsResp string
	if tt.refExists {
		pkt, _ := protocol.FormatPacks(
			protocol.PackLine(fmt.Sprintf("%s %s\n", tt.refToUpdate.Hash, tt.refToUpdate.Name)),
		)
		refsResp = string(pkt)
	} else {
		refsResp = "0000"
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(refsResp)); err != nil {
		t.Errorf("failed to write response: %v", err)
	}
}

// handleReceivePackRequest handles git-receive-pack requests in tests
func handleReceivePackRequest(t *testing.T, w http.ResponseWriter, r *http.Request, tt struct {
	name          string
	refToUpdate   Ref
	refExists     bool
	expectedError string
	setupClient   options.Option
}, shouldCheckBody bool) {
	if tt.expectedError == "ref refs/heads/non-existent does not exist" {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("error: ref refs/heads/non-existent does not exist")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
		return
	}

	if shouldCheckBody {
		validateReceivePackBody(t, r, tt)
	}
	w.WriteHeader(http.StatusOK)
}

// validateReceivePackBody validates the request body for receive-pack requests
func validateReceivePackBody(t *testing.T, r *http.Request, tt struct {
	name          string
	refToUpdate   Ref
	refExists     bool
	expectedError string
	setupClient   options.Option
}) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("failed to read request body: %v", err)
		return
	}

	expectedRefLine := fmt.Sprintf("%s %s %s\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n0000",
		tt.refToUpdate.Hash,
		tt.refToUpdate.Hash,
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
	}
}

// createTestClient creates a test client with optional custom configuration
func createTestClient(t *testing.T, server *httptest.Server, setupClient options.Option) Client {
	url := "http://127.0.0.1:0"
	if server != nil {
		url = server.URL
	}

	var (
		client Client
		err    error
	)

	if setupClient != nil {
		client, err = NewHTTPClient(url, setupClient)
	} else {
		client, err = NewHTTPClient(url)
	}
	require.NoError(t, err)
	return client
}

// runUpdateRefTest executes the UpdateRef test and validates results
func runUpdateRefTest(t *testing.T, client Client, tt struct {
	name          string
	refToUpdate   Ref
	refExists     bool
	expectedError string
	setupClient   options.Option
}) {
	err := client.UpdateRef(context.Background(), tt.refToUpdate)
	if tt.expectedError != "" {
		require.Error(t, err)
		require.Contains(t, err.Error(), tt.expectedError)
	} else {
		require.NoError(t, err)
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

			server := setupDeleteRefTestServer(t, tt)
			defer func() {
				if server != nil {
					server.Close()
				}
			}()

			client := createTestClient(t, server, tt.setupClient)
			runDeleteRefTest(t, client, tt)
		})
	}
}

// setupDeleteRefTestServer creates a test server for DeleteRef tests
func setupDeleteRefTestServer(t *testing.T, tt struct {
	name          string
	refToDelete   string
	refExists     bool
	expectedError string
	setupClient   options.Option
}) *httptest.Server {
	if tt.setupClient != nil {
		return nil
	}

	shouldCheckBody := tt.expectedError == "" || strings.Contains(tt.expectedError, "send ref update: got status code 500")
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/git-upload-pack":
			handleDeleteRefUploadPack(t, w, tt)
		case "/git-receive-pack":
			handleDeleteRefReceivePack(t, w, r, tt, shouldCheckBody)
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
	}))
}

// handleDeleteRefUploadPack handles upload-pack requests for delete ref tests
func handleDeleteRefUploadPack(t *testing.T, w http.ResponseWriter, tt struct {
	name          string
	refToDelete   string
	refExists     bool
	expectedError string
	setupClient   options.Option
}) {
	var refsResp string
	if tt.refExists {
		pkt, _ := protocol.FormatPacks(
			protocol.PackLine(fmt.Sprintf("%s %s\n", "1234567890123456789012345678901234567890", tt.refToDelete)),
		)
		refsResp = string(pkt)
	} else {
		refsResp = "0000"
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(refsResp)); err != nil {
		t.Errorf("failed to write response: %v", err)
	}
}

// handleDeleteRefReceivePack handles receive-pack requests for delete ref tests
func handleDeleteRefReceivePack(t *testing.T, w http.ResponseWriter, r *http.Request, tt struct {
	name          string
	refToDelete   string
	refExists     bool
	expectedError string
	setupClient   options.Option
}, shouldCheckBody bool) {
	if shouldCheckBody {
		validateDeleteRefBody(t, r, tt)
	}
	w.WriteHeader(http.StatusOK)
}

// validateDeleteRefBody validates the request body for delete ref requests
func validateDeleteRefBody(t *testing.T, r *http.Request, tt struct {
	name          string
	refToDelete   string
	refExists     bool
	expectedError string
	setupClient   options.Option
}) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("failed to read request body: %v", err)
		return
	}

	expectedRefLine := fmt.Sprintf("%s %s %s\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n0000",
		"1234567890123456789012345678901234567890",
		protocol.ZeroHash,
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
	}
}

// runDeleteRefTest executes the DeleteRef test and validates results
func runDeleteRefTest(t *testing.T, client Client, tt struct {
	name          string
	refToDelete   string
	refExists     bool
	expectedError string
	setupClient   options.Option
}) {
	err := client.DeleteRef(context.Background(), tt.refToDelete)
	if tt.expectedError != "" {
		require.Error(t, err)
		require.Contains(t, err.Error(), tt.expectedError)
	} else {
		require.NoError(t, err)
	}
}
