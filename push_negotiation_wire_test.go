package nanogit

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// formatInfoRefsBody renders a v1-style info/refs?service=git-receive-pack
// body advertising the given capabilities on the first ref line.
func formatInfoRefsBody(t *testing.T, caps string) []byte {
	t.Helper()
	body, err := protocol.FormatPacks(
		protocol.PackLine("# service=git-receive-pack\n"),
		protocol.FlushPacket,
		protocol.PackLine("aabbccddeeff00112233445566778899aabbccdd refs/heads/main\x00"+caps+"\n"),
		protocol.FlushPacket,
	)
	require.NoError(t, err)
	return body
}

// TestNegotiation_DropsServerOmittedCapabilities is the core wire-level
// guarantee: when the server advertises a strict subset, the receive-pack
// POST that follows must advertise only the intersection (plus the
// always-retained client caps). This is the property capability negotiation
// buys us — and it's the property Gitea integration tests cannot prove from
// outside the container.
func TestNegotiation_DropsServerOmittedCapabilities(t *testing.T) {
	refHash, err := hash.FromHex("1234567890123456789012345678901234567890")
	require.NoError(t, err)

	var (
		infoRefsHits       atomic.Int32
		gotReceivePackBody []byte
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/info/refs") && r.URL.Query().Get("service") == "git-receive-pack":
			infoRefsHits.Add(1)
			// Strict server: omits side-band-64k. This is the GitLab-shaped
			// case the option is designed to handle defensively.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(formatInfoRefsBody(t, "report-status-v2 quiet object-format=sha1 agent=git/2.43"))
		case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
			// CreateRef calls GetRef first; "0000" signals no refs match.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("0000"))
		case strings.HasSuffix(r.URL.Path, "/git-receive-pack"):
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			gotReceivePackBody = body
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL, options.WithCapabilityNegotiation())
	require.NoError(t, err)

	require.NoError(t, client.CreateRef(context.Background(), Ref{
		Name: "refs/heads/feature",
		Hash: refHash,
	}))

	// Exactly one info/refs fetch for capability negotiation.
	assert.Equal(t, int32(1), infoRefsHits.Load(),
		"capability negotiation should fetch info/refs exactly once")

	// The receive-pack POST must NOT advertise side-band-64k because the
	// server didn't advertise it. This is the visible behavior change.
	require.NotEmpty(t, gotReceivePackBody)
	assert.NotContains(t, string(gotReceivePackBody), string(protocol.CapSideBand64k),
		"side-band-64k should be filtered out when the server doesn't advertise it")

	// The always-retained capabilities (report-status-v2 and our agent=)
	// must still be present, regardless.
	assert.Contains(t, string(gotReceivePackBody), string(protocol.CapReportStatusV2))
	assert.Contains(t, string(gotReceivePackBody), string(protocol.CapAgent("nanogit")),
		"agent= should always carry the client's identifier, not the server's")
}

// TestNegotiation_CachedAcrossRefOps verifies sync.Once caching: a sequence
// of CreateRef ops on the same client must perform exactly one info/refs
// fetch, not one per call.
func TestNegotiation_CachedAcrossRefOps(t *testing.T) {
	refHash, err := hash.FromHex("1234567890123456789012345678901234567890")
	require.NoError(t, err)

	var infoRefsHits atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/info/refs") && r.URL.Query().Get("service") == "git-receive-pack":
			infoRefsHits.Add(1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(formatInfoRefsBody(t, "report-status-v2 side-band-64k quiet object-format=sha1 agent=git/2.43"))
		case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("0000"))
		case strings.HasSuffix(r.URL.Path, "/git-receive-pack"):
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL, options.WithCapabilityNegotiation())
	require.NoError(t, err)

	// Three create-ref ops, one client. The negotiation fetch must run exactly once.
	for _, name := range []string{"refs/heads/a", "refs/heads/b", "refs/heads/c"} {
		require.NoError(t, client.CreateRef(context.Background(), Ref{Name: name, Hash: refHash}))
	}

	assert.Equal(t, int32(1), infoRefsHits.Load(),
		"capability negotiation should be cached across ref ops via sync.Once")
}

// TestNegotiation_FailurePropagates verifies that a negotiation fetch error
// aborts the push instead of silently falling back to the static set.
// Silent fallback would hide server misconfiguration and contradict the
// explicit opt-in.
func TestNegotiation_FailurePropagates(t *testing.T) {
	refHash, err := hash.FromHex("1234567890123456789012345678901234567890")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/info/refs") && r.URL.Query().Get("service") == "git-receive-pack":
			// 404 here exercises the CheckHTTPClientError path.
			w.WriteHeader(http.StatusNotFound)
		case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
			// CreateRef calls GetRef before the negotiation lookup; let it
			// succeed with "no refs match" so the negotiation step is what
			// actually trips the test.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("0000"))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL, options.WithCapabilityNegotiation())
	require.NoError(t, err)

	err = client.CreateRef(context.Background(), Ref{
		Name: "refs/heads/feature",
		Hash: refHash,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "negotiate receive-pack capabilities",
		"error chain should mention the negotiation step rather than masking it")
}

// TestNegotiation_TransientFailureDoesNotPoisonClient guards the retry
// contract: if the first negotiation fetch fails (transient network error,
// 5xx, etc.), a subsequent call must retry rather than return the cached
// error forever. sync.Once would have poisoned the client; the mutex-based
// implementation only caches successes.
func TestNegotiation_TransientFailureDoesNotPoisonClient(t *testing.T) {
	refHash, err := hash.FromHex("1234567890123456789012345678901234567890")
	require.NoError(t, err)

	var infoRefsHits atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/info/refs") && r.URL.Query().Get("service") == "git-receive-pack":
			n := infoRefsHits.Add(1)
			if n == 1 {
				// First call fails — a transient network blip in disguise.
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Second call succeeds.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(formatInfoRefsBody(t, "report-status-v2 side-band-64k quiet object-format=sha1 agent=git/2.43"))
		case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("0000"))
		case strings.HasSuffix(r.URL.Path, "/git-receive-pack"):
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL, options.WithCapabilityNegotiation())
	require.NoError(t, err)

	// First call: server returns 500 → CreateRef must error out without
	// caching the failure.
	err = client.CreateRef(context.Background(), Ref{Name: "refs/heads/a", Hash: refHash})
	require.Error(t, err, "first call should fail because the server is 500-ing")

	// Second call: server now responds successfully → CreateRef must retry
	// the negotiation fetch (not return the stale error) and succeed.
	require.NoError(t, client.CreateRef(context.Background(), Ref{Name: "refs/heads/b", Hash: refHash}),
		"a transient first failure must not poison the client")

	// Two info/refs hits: one for the failed first attempt, one for the
	// successful retry. Subsequent ops would reuse the cached success.
	assert.Equal(t, int32(2), infoRefsHits.Load(),
		"failed negotiation must not be cached; the next call should retry")
}

// TestNoNegotiation_DefaultBehaviorUnchanged guards the opt-in contract:
// without WithCapabilityNegotiation the client must not fetch info/refs at
// all and must advertise the full static default set.
func TestNoNegotiation_DefaultBehaviorUnchanged(t *testing.T) {
	refHash, err := hash.FromHex("1234567890123456789012345678901234567890")
	require.NoError(t, err)

	var (
		infoRefsHits       atomic.Int32
		gotReceivePackBody []byte
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/info/refs"):
			infoRefsHits.Add(1)
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("0000"))
		case strings.HasSuffix(r.URL.Path, "/git-receive-pack"):
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			gotReceivePackBody = body
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	// No negotiation option — should behave exactly like before.
	client, err := NewHTTPClient(server.URL)
	require.NoError(t, err)

	require.NoError(t, client.CreateRef(context.Background(), Ref{
		Name: "refs/heads/feature",
		Hash: refHash,
	}))

	assert.Equal(t, int32(0), infoRefsHits.Load(),
		"info/refs should not be fetched when negotiation is off")
	assert.Contains(t, string(gotReceivePackBody), string(protocol.CapSideBand64k),
		"the static default set must still be advertised when negotiation is off")
}
