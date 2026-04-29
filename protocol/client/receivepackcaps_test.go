package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/nanogit/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchReceivePackCapabilities(t *testing.T) {
	t.Parallel()

	t.Run("parses caps from typical Gitea response", func(t *testing.T) {
		body, err := protocol.FormatPacks(
			protocol.PackLine("# service=git-receive-pack\n"),
			protocol.FlushPacket,
			protocol.PackLine("aabbccddeeff00112233445566778899aabbccdd refs/heads/main\x00report-status-v2 side-band-64k quiet object-format=sha1 agent=git/2.43\n"),
			protocol.FlushPacket,
		)
		require.NoError(t, err)

		var gotPath, gotService, gotMethod string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotService = r.URL.Query().Get("service")
			gotMethod = r.Method
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		caps, err := client.FetchReceivePackCapabilities(context.Background())
		require.NoError(t, err)

		assert.Equal(t, "/repo.git/info/refs", gotPath)
		assert.Equal(t, "git-receive-pack", gotService)
		assert.Equal(t, http.MethodGet, gotMethod)
		assert.Contains(t, caps, protocol.CapSideBand64k)
		assert.Contains(t, caps, protocol.CapReportStatusV2)
		assert.Contains(t, caps, protocol.CapAgent("git/2.43"))
	})

	t.Run("maps 404 to ErrRepositoryNotFound", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		_, err = client.FetchReceivePackCapabilities(context.Background())
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrRepositoryNotFound),
			"expected ErrRepositoryNotFound, got %v", err)
	})

	t.Run("propagates parse errors verbatim", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not a pkt-line stream"))
		}))
		defer server.Close()

		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		_, err = client.FetchReceivePackCapabilities(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse receive-pack info/refs")
	})
}
