//go:build integration
// +build integration

package client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ListRefs(t *testing.T) {
	// Use a public repository for testing
	repo := "https://github.com/octocat/Hello-World"
	client, err := New(repo, WithGitHub())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	refs, err := client.ListRefs(ctx)
	require.NoError(t, err, "ListRefs failed: %v", err)

	// Basic validation of the response
	assert.NotEmpty(t, refs, "should have at least one reference")

	// Check for common refs that should exist
	var masterRef *Ref
	for _, ref := range refs {
		if ref.Name == "refs/heads/master" {
			masterRef = &ref
			break
		}
	}
	require.NotNil(t, masterRef, "should have master branch")
	require.Len(t, masterRef.Hash, 40, "hash should be 40 characters")
}
