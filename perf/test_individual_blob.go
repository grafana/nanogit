package performance

import (
	"context"
	"testing"

	"github.com/grafana/nanogit"
	nanolog "github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol/hash"
)

// TestIndividualBlobFetch tests fetching a specific missing blob directly  
func TestIndividualBlobFetch(t *testing.T) {
	ctx := context.Background()
	logger := &simpleLogger{}
	ctx = nanolog.ToContext(ctx, logger)

	// Create HTTP client for public repository
	client, err := nanogit.NewHTTPClient("https://github.com/grafana/grafana.git")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test one of the consistently missing blob hashes from debug output
	missingBlobHashStr := "68d3c8b7e7618d7aa3063cf2e9076c9310a9892c" // dataplaneservicespec.go
	missingBlobHash, err := hash.FromHex(missingBlobHashStr)
	if err != nil {
		t.Fatalf("Failed to parse blob hash: %v", err)
	}

	t.Logf("🔍 Testing individual fetch of missing blob: %s", missingBlobHashStr)

	// Try to fetch this specific blob individually
	blob, err := client.GetBlob(ctx, missingBlobHash)
	if err != nil {
		t.Logf("❌ Individual blob fetch failed: %v", err)
		t.Logf("This confirms the blob is not available via Git Smart HTTP protocol")
	} else {
		t.Logf("✅ Individual blob fetch succeeded!")
		t.Logf("   • Blob hash: %s", blob.Hash.String())
		t.Logf("   • Content size: %d bytes", len(blob.Content))
		t.Logf("   • Content preview: %s", string(blob.Content[:min(100, len(blob.Content))]))
	}

	// This test confirms whether missing blobs are available individually via Git Smart HTTP protocol
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}