package output

import (
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHumanFormatter_FormatRefs(t *testing.T) {
	formatter := NewHumanFormatter()

	refs := []nanogit.Ref{
		{
			Name: "refs/heads/main",
			Hash: hash.MustFromHex("0123456789abcdef0123456789abcdef01234567"),
		},
		{
			Name: "refs/heads/develop",
			Hash: hash.MustFromHex("1111111111111111111111111111111111111111"),
		},
	}

	// Should not error
	err := formatter.FormatRefs(refs)
	assert.NoError(t, err)
}

func TestHumanFormatter_FormatTreeEntries(t *testing.T) {
	formatter := NewHumanFormatter()

	entries := []nanogit.FlatTreeEntry{
		{
			Name: "README.md",
			Path: "README.md",
			Mode: 0o100644,
			Hash: hash.MustFromHex("0123456789abcdef0123456789abcdef01234567"),
			Type: protocol.ObjectTypeBlob,
		},
		{
			Name: "src",
			Path: "src",
			Mode: 0o40000,
			Hash: hash.MustFromHex("1111111111111111111111111111111111111111"),
			Type: protocol.ObjectTypeTree,
		},
	}

	// Should not error
	err := formatter.FormatTreeEntries(entries)
	assert.NoError(t, err)
}

func TestHumanFormatter_FormatBlobContent(t *testing.T) {
	formatter := NewHumanFormatter()

	content := []byte("Hello, World!")
	blobHash := hash.MustFromHex("0123456789abcdef0123456789abcdef01234567")

	// Should not error
	err := formatter.FormatBlobContent("test.txt", blobHash, content)
	assert.NoError(t, err)
}

func TestHumanFormatter_FormatCloneResult(t *testing.T) {
	formatter := NewHumanFormatter()

	result := &nanogit.CloneResult{
		Commit: &nanogit.Commit{
			Hash: hash.MustFromHex("0123456789abcdef0123456789abcdef01234567"),
		},
		TotalFiles:    100,
		FilteredFiles: 50,
		Path:          "/tmp/repo",
	}

	// Should not error
	err := formatter.FormatCloneResult(result)
	assert.NoError(t, err)
}

func TestHumanFormatter_EmptyRefs(t *testing.T) {
	formatter := NewHumanFormatter()

	// Should handle empty slice without error
	err := formatter.FormatRefs([]nanogit.Ref{})
	assert.NoError(t, err)
}

func TestHumanFormatter_EmptyTreeEntries(t *testing.T) {
	formatter := NewHumanFormatter()

	// Should handle empty slice without error
	err := formatter.FormatTreeEntries([]nanogit.FlatTreeEntry{})
	assert.NoError(t, err)
}

func TestHumanFormatter_EmptyBlobContent(t *testing.T) {
	formatter := NewHumanFormatter()

	blobHash := hash.MustFromHex("0123456789abcdef0123456789abcdef01234567")

	// Should handle empty content without error
	err := formatter.FormatBlobContent("empty.txt", blobHash, []byte{})
	assert.NoError(t, err)
}

func TestHumanFormatter_CloneResultNoFiltering(t *testing.T) {
	formatter := NewHumanFormatter()

	result := &nanogit.CloneResult{
		Commit: &nanogit.Commit{
			Hash: hash.MustFromHex("0123456789abcdef0123456789abcdef01234567"),
		},
		TotalFiles:    100,
		FilteredFiles: 100, // No filtering
		Path:          "/tmp/repo",
	}

	// Should not show filtering message when all files cloned
	err := formatter.FormatCloneResult(result)
	require.NoError(t, err)
}

func TestHumanFormatter_CloneResultWithFiltering(t *testing.T) {
	formatter := NewHumanFormatter()

	result := &nanogit.CloneResult{
		Commit: &nanogit.Commit{
			Hash: hash.MustFromHex("0123456789abcdef0123456789abcdef01234567"),
		},
		TotalFiles:    100,
		FilteredFiles: 25, // Filtered to 25%
		Path:          "/tmp/repo",
	}

	// Should show filtering message
	err := formatter.FormatCloneResult(result)
	require.NoError(t, err)
}
