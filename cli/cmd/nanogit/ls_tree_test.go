package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterEntriesByPath(t *testing.T) {
	entries := []nanogit.FlatTreeEntry{
		{Path: "README.md", Name: "README.md"},
		{Path: "docs/getting-started.md", Name: "getting-started.md"},
		{Path: "docs/api.md", Name: "api.md"},
		{Path: "src/main.go", Name: "main.go"},
		{Path: "src/utils/helper.go", Name: "helper.go"},
	}

	tests := []struct {
		name     string
		prefix   string
		expected int
		check    func(t *testing.T, filtered []nanogit.FlatTreeEntry)
	}{
		{
			name:     "empty prefix returns all",
			prefix:   "",
			expected: 5,
			check: func(t *testing.T, filtered []nanogit.FlatTreeEntry) {
				assert.Len(t, filtered, 5)
			},
		},
		{
			name:     "filter by docs directory",
			prefix:   "docs",
			expected: 2,
			check: func(t *testing.T, filtered []nanogit.FlatTreeEntry) {
				assert.Len(t, filtered, 2)
				for _, entry := range filtered {
					assert.Contains(t, entry.Path, "docs/")
				}
			},
		},
		{
			name:     "filter by src directory",
			prefix:   "src",
			expected: 2,
			check: func(t *testing.T, filtered []nanogit.FlatTreeEntry) {
				assert.Len(t, filtered, 2)
				for _, entry := range filtered {
					assert.Contains(t, entry.Path, "src/")
				}
			},
		},
		{
			name:     "filter by nested directory",
			prefix:   "src/utils",
			expected: 1,
			check: func(t *testing.T, filtered []nanogit.FlatTreeEntry) {
				assert.Len(t, filtered, 1)
				assert.Equal(t, "src/utils/helper.go", filtered[0].Path)
			},
		},
		{
			name:     "prefix with trailing slash",
			prefix:   "docs/",
			expected: 2,
			check: func(t *testing.T, filtered []nanogit.FlatTreeEntry) {
				assert.Len(t, filtered, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterEntriesByPath(entries, tt.prefix)
			tt.check(t, filtered)
		})
	}
}

func TestObjectTypeToString(t *testing.T) {
	tests := []struct {
		objType  protocol.ObjectType
		expected string
	}{
		{protocol.ObjectTypeBlob, "blob"},
		{protocol.ObjectTypeTree, "tree"},
		{protocol.ObjectTypeCommit, "commit"},
		{protocol.ObjectTypeTag, "tag"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := objectTypeToString(tt.objType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOutputTreeJSON(t *testing.T) {
	entries := []nanogit.TreeEntry{
		{
			Name: "README.md",
			Mode: 0o100644,
			Type: protocol.ObjectTypeBlob,
			Hash: hash.MustFromHex("1234567890123456789012345678901234567890"),
		},
		{
			Name: "docs",
			Mode: 0o040000,
			Type: protocol.ObjectTypeTree,
			Hash: hash.MustFromHex("2345678901234567890123456789012345678901"),
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputTreeJSON(entries)
	require.NoError(t, err)

	// Restore stdout
	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Verify JSON is valid
	var output []treeEntryJSON
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)
	assert.Len(t, output, 2)
	assert.Equal(t, "README.md", output[0].Name)
	assert.Equal(t, "100644", output[0].Mode)
	assert.Equal(t, "blob", output[0].Type)
	assert.Equal(t, "1234567890123456789012345678901234567890", output[0].Hash)
	assert.Equal(t, "docs", output[1].Name)
	assert.Equal(t, "040000", output[1].Mode)
	assert.Equal(t, "tree", output[1].Type)
}

func TestOutputFlatTreeJSON(t *testing.T) {
	entries := []nanogit.FlatTreeEntry{
		{
			Name: "README.md",
			Path: "README.md",
			Mode: 0o100644,
			Type: protocol.ObjectTypeBlob,
			Hash: hash.MustFromHex("1234567890123456789012345678901234567890"),
		},
		{
			Name: "api.md",
			Path: "docs/api.md",
			Mode: 0o100644,
			Type: protocol.ObjectTypeBlob,
			Hash: hash.MustFromHex("2345678901234567890123456789012345678901"),
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputFlatTreeJSON(entries)
	require.NoError(t, err)

	// Restore stdout
	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Verify JSON is valid
	var output []flatTreeEntryJSON
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)
	assert.Len(t, output, 2)
	assert.Equal(t, "README.md", output[0].Name)
	assert.Equal(t, "README.md", output[0].Path)
	assert.Equal(t, "100644", output[0].Mode)
	assert.Equal(t, "blob", output[0].Type)
	assert.Equal(t, "api.md", output[1].Name)
	assert.Equal(t, "docs/api.md", output[1].Path)
}

func TestOutputTreeHuman(t *testing.T) {
	entries := []nanogit.TreeEntry{
		{
			Name: "README.md",
			Mode: 0o100644,
			Type: protocol.ObjectTypeBlob,
			Hash: hash.MustFromHex("1234567890123456789012345678901234567890"),
		},
	}

	// Test simple output
	t.Run("simple output", func(t *testing.T) {
		lsTreeLong = false

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := outputTreeHuman(entries)
		require.NoError(t, err)

		require.NoError(t, w.Close())
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "README.md")
		assert.NotContains(t, output, "100644") // Should not show mode in simple output
	})

	// Test long output
	t.Run("long output", func(t *testing.T) {
		lsTreeLong = true

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := outputTreeHuman(entries)
		require.NoError(t, err)

		require.NoError(t, w.Close())
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "100644")
		assert.Contains(t, output, "blob")
		assert.Contains(t, output, "1234567890123456789012345678901234567890")
		assert.Contains(t, output, "README.md")

		// Reset flag
		lsTreeLong = false
	})
}

func TestLsTreeCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "no arguments returns error",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "one argument returns error",
			args:        []string{"url"},
			expectError: true,
		},
		{
			name:        "too many arguments returns error",
			args:        []string{"url", "ref", "extra"},
			expectError: true,
		},
		{
			name:        "valid arguments accepted",
			args:        []string{"https://github.com/grafana/nanogit.git", "main"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			lsTreeRecursive = false
			lsTreeLong = false
			globalJSON = false
			lsTreePath = ""
			globalUsername = ""
			globalToken = ""

			// Test argument validation
			lsTreeCmd.SetArgs(tt.args)
			err := lsTreeCmd.Args(lsTreeCmd, tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
