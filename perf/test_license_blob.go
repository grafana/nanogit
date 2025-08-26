package performance

import (
	"context"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
)

// TestLicenseBlobFetch tests fetching the LICENSE file blob that's missing from nanogit
func TestLicenseBlobFetch(t *testing.T) {
	ctx := context.Background()

	// Create HTTP client for public repository
	client, err := nanogit.NewHTTPClient("https://github.com/grafana/grafana.git")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// LICENSE file blob hash from git ls-tree command
	licenseBlobHashStr := "be3f7b28e564e7dd05eaf59d64adba1a4065ac0e"
	licenseBlobHash, err := hash.FromHex(licenseBlobHashStr)
	if err != nil {
		t.Fatalf("Failed to parse LICENSE blob hash: %v", err)
	}

	t.Logf("🔍 Testing individual fetch of LICENSE blob: %s", licenseBlobHashStr)

	// Try to fetch the LICENSE blob individually
	blob, err := client.GetBlob(ctx, licenseBlobHash)
	if err != nil {
		t.Logf("❌ LICENSE blob fetch failed: %v", err)
		t.Logf("This indicates the LICENSE blob is not available via Smart HTTP protocol")
	} else {
		t.Logf("✅ LICENSE blob fetch succeeded!")
		t.Logf("   • Blob hash: %s", blob.Hash.String())
		t.Logf("   • Content size: %d bytes", len(blob.Content))
		t.Logf("   • Content starts with: %s", string(blob.Content[:minInt(200, len(blob.Content))]))
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}