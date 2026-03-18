package nanogit

import (
	"testing"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetectRenames_MultipleFilesWithSameHash tests the critical edge case where
// multiple files share the same content hash. This was a bug in the original implementation
// where using map[string]CommitFile would cause later entries to overwrite earlier ones,
// and then all changes with that hash would be dropped as "renamed".
func TestDetectRenames_MultipleFilesWithSameHash(t *testing.T) {
	// Create a common hash for files with identical content
	identicalHash, err := hash.FromHex("1234567890abcdef1234567890abcdef12345678")
	require.NoError(t, err)

	tests := []struct {
		name     string
		changes  []CommitFile
		expected []CommitFile
	}{
		{
			name: "two deletes and one add with same hash - one rename, one delete",
			changes: []CommitFile{
				{Path: "file1.txt", OldHash: identicalHash, Status: protocol.FileStatusDeleted, Mode: 0644},
				{Path: "file2.txt", OldHash: identicalHash, Status: protocol.FileStatusDeleted, Mode: 0644},
				{Path: "renamed.txt", Hash: identicalHash, Status: protocol.FileStatusAdded, Mode: 0644},
			},
			expected: []CommitFile{
				// One file should be paired as a rename (file1 or file2 -> renamed)
				{
					Path:    "renamed.txt",
					OldPath: "file1.txt", // First deleted file gets paired
					Hash:    identicalHash,
					OldHash: identicalHash,
					Status:  protocol.FileStatusRenamed,
					Mode:    0644,
					OldMode: 0644,
				},
				// One delete should remain unpaired
				{Path: "file2.txt", OldHash: identicalHash, Status: protocol.FileStatusDeleted, Mode: 0644},
			},
		},
		{
			name: "one delete and two adds with same hash - one rename, one add",
			changes: []CommitFile{
				{Path: "original.txt", OldHash: identicalHash, Status: protocol.FileStatusDeleted, Mode: 0644},
				{Path: "copy1.txt", Hash: identicalHash, Status: protocol.FileStatusAdded, Mode: 0644},
				{Path: "copy2.txt", Hash: identicalHash, Status: protocol.FileStatusAdded, Mode: 0644},
			},
			expected: []CommitFile{
				// One add should be paired as a rename
				{
					Path:    "copy1.txt", // First added file gets paired
					OldPath: "original.txt",
					Hash:    identicalHash,
					OldHash: identicalHash,
					Status:  protocol.FileStatusRenamed,
					Mode:    0644,
					OldMode: 0644,
				},
				// One add should remain unpaired
				{Path: "copy2.txt", Hash: identicalHash, Status: protocol.FileStatusAdded, Mode: 0644},
			},
		},
		{
			name: "three deletes and three adds with same hash - three renames",
			changes: []CommitFile{
				{Path: "a.txt", OldHash: identicalHash, Status: protocol.FileStatusDeleted, Mode: 0644},
				{Path: "b.txt", OldHash: identicalHash, Status: protocol.FileStatusDeleted, Mode: 0644},
				{Path: "c.txt", OldHash: identicalHash, Status: protocol.FileStatusDeleted, Mode: 0644},
				{Path: "x.txt", Hash: identicalHash, Status: protocol.FileStatusAdded, Mode: 0644},
				{Path: "y.txt", Hash: identicalHash, Status: protocol.FileStatusAdded, Mode: 0644},
				{Path: "z.txt", Hash: identicalHash, Status: protocol.FileStatusAdded, Mode: 0644},
			},
			expected: []CommitFile{
				// All three should be paired as renames (one-to-one)
				{
					Path:    "x.txt",
					OldPath: "a.txt",
					Hash:    identicalHash,
					OldHash: identicalHash,
					Status:  protocol.FileStatusRenamed,
					Mode:    0644,
					OldMode: 0644,
				},
				{
					Path:    "y.txt",
					OldPath: "b.txt",
					Hash:    identicalHash,
					OldHash: identicalHash,
					Status:  protocol.FileStatusRenamed,
					Mode:    0644,
					OldMode: 0644,
				},
				{
					Path:    "z.txt",
					OldPath: "c.txt",
					Hash:    identicalHash,
					OldHash: identicalHash,
					Status:  protocol.FileStatusRenamed,
					Mode:    0644,
					OldMode: 0644,
				},
			},
		},
		{
			name: "mixed: identical content files with other changes",
			changes: []CommitFile{
				// Identical content files
				{Path: "dup1.txt", OldHash: identicalHash, Status: protocol.FileStatusDeleted, Mode: 0644},
				{Path: "dup2.txt", Hash: identicalHash, Status: protocol.FileStatusAdded, Mode: 0644},
				// Different content file (modified)
				{
					Path:    "different.txt",
					Hash:    hash.MustFromHex("abcdef1234567890abcdef1234567890abcdef12"),
					OldHash: hash.MustFromHex("fedcba0987654321fedcba0987654321fedcba09"),
					Status:  protocol.FileStatusModified,
					Mode:    0644,
					OldMode: 0644,
				},
			},
			expected: []CommitFile{
				// Identical files should be paired as rename
				{
					Path:    "dup2.txt",
					OldPath: "dup1.txt",
					Hash:    identicalHash,
					OldHash: identicalHash,
					Status:  protocol.FileStatusRenamed,
					Mode:    0644,
					OldMode: 0644,
				},
				// Modified file should remain unchanged
				{
					Path:    "different.txt",
					Hash:    hash.MustFromHex("abcdef1234567890abcdef1234567890abcdef12"),
					OldHash: hash.MustFromHex("fedcba0987654321fedcba0987654321fedcba09"),
					Status:  protocol.FileStatusModified,
					Mode:    0644,
					OldMode: 0644,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectRenames(tt.changes)

			// Verify we have the expected number of results
			assert.Len(t, result, len(tt.expected), "Result count mismatch")

			// Verify each expected change is present
			for _, expected := range tt.expected {
				found := false
				for _, actual := range result {
					if actual.Status == expected.Status &&
						actual.Path == expected.Path &&
						actual.OldPath == expected.OldPath {
						found = true
						assert.Equal(t, expected.Hash, actual.Hash, "Hash mismatch for %s", actual.Path)
						assert.Equal(t, expected.OldHash, actual.OldHash, "OldHash mismatch for %s", actual.Path)
						assert.Equal(t, expected.Mode, actual.Mode, "Mode mismatch for %s", actual.Path)
						assert.Equal(t, expected.OldMode, actual.OldMode, "OldMode mismatch for %s", actual.Path)
						break
					}
				}
				assert.True(t, found, "Expected change not found: %+v", expected)
			}
		})
	}
}

// TestDetectRenames_EmptyAndNilCases tests edge cases with empty inputs
func TestDetectRenames_EmptyAndNilCases(t *testing.T) {
	tests := []struct {
		name     string
		changes  []CommitFile
		expected []CommitFile
	}{
		{
			name:     "nil input",
			changes:  nil,
			expected: []CommitFile{},
		},
		{
			name:     "empty input",
			changes:  []CommitFile{},
			expected: []CommitFile{},
		},
		{
			name: "only modified files - no renames",
			changes: []CommitFile{
				{
					Path:    "file.txt",
					Hash:    hash.MustFromHex("1111111111111111111111111111111111111111"),
					OldHash: hash.MustFromHex("2222222222222222222222222222222222222222"),
					Status:  protocol.FileStatusModified,
					Mode:    0644,
					OldMode: 0644,
				},
			},
			expected: []CommitFile{
				{
					Path:    "file.txt",
					Hash:    hash.MustFromHex("1111111111111111111111111111111111111111"),
					OldHash: hash.MustFromHex("2222222222222222222222222222222222222222"),
					Status:  protocol.FileStatusModified,
					Mode:    0644,
					OldMode: 0644,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectRenames(tt.changes)
			assert.Equal(t, tt.expected, result)
		})
	}
}
