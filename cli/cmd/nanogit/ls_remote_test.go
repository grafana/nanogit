package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterRefs(t *testing.T) {
	refs := []nanogit.Ref{
		{Name: "refs/heads/main", Hash: mustParseHash(t, "1234567890123456789012345678901234567890")},
		{Name: "refs/heads/develop", Hash: mustParseHash(t, "2345678901234567890123456789012345678901")},
		{Name: "refs/tags/v1.0.0", Hash: mustParseHash(t, "3456789012345678901234567890123456789012")},
		{Name: "refs/tags/v2.0.0", Hash: mustParseHash(t, "4567890123456789012345678901234567890123")},
		{Name: "refs/pull/123/head", Hash: mustParseHash(t, "5678901234567890123456789012345678901234")},
	}

	tests := []struct {
		name     string
		heads    bool
		tags     bool
		expected int
		check    func(t *testing.T, filtered []nanogit.Ref)
	}{
		{
			name:     "no filters returns all refs",
			heads:    false,
			tags:     false,
			expected: 5,
			check: func(t *testing.T, filtered []nanogit.Ref) {
				assert.Len(t, filtered, 5)
			},
		},
		{
			name:     "heads filter returns only branches",
			heads:    true,
			tags:     false,
			expected: 2,
			check: func(t *testing.T, filtered []nanogit.Ref) {
				assert.Len(t, filtered, 2)
				for _, ref := range filtered {
					assert.True(t, strings.HasPrefix(ref.Name, "refs/heads/"))
				}
			},
		},
		{
			name:     "tags filter returns only tags",
			heads:    false,
			tags:     true,
			expected: 2,
			check: func(t *testing.T, filtered []nanogit.Ref) {
				assert.Len(t, filtered, 2)
				for _, ref := range filtered {
					assert.True(t, strings.HasPrefix(ref.Name, "refs/tags/"))
				}
			},
		},
		{
			name:     "both filters returns branches and tags",
			heads:    true,
			tags:     true,
			expected: 4,
			check: func(t *testing.T, filtered []nanogit.Ref) {
				assert.Len(t, filtered, 4)
				for _, ref := range filtered {
					assert.True(t,
						strings.HasPrefix(ref.Name, "refs/heads/") ||
							strings.HasPrefix(ref.Name, "refs/tags/"),
					)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global flags for filter function
			lsRemoteHeads = tt.heads
			lsRemoteTags = tt.tags

			filtered := filterRefs(refs)
			tt.check(t, filtered)
		})
	}
}

func TestOutputJSON(t *testing.T) {
	refs := []nanogit.Ref{
		{Name: "refs/heads/main", Hash: mustParseHash(t, "1234567890123456789012345678901234567890")},
		{Name: "refs/tags/v1.0.0", Hash: mustParseHash(t, "2345678901234567890123456789012345678901")},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSON(refs)
	require.NoError(t, err)

	// Restore stdout
	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Verify JSON is valid and contains expected data
	var output []refJSON
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)
	assert.Len(t, output, 2)
	assert.Equal(t, "refs/heads/main", output[0].Name)
	assert.Equal(t, "1234567890123456789012345678901234567890", output[0].Hash)
	assert.Equal(t, "refs/tags/v1.0.0", output[1].Name)
	assert.Equal(t, "2345678901234567890123456789012345678901", output[1].Hash)
}

func TestOutputHuman(t *testing.T) {
	refs := []nanogit.Ref{
		{Name: "refs/heads/main", Hash: mustParseHash(t, "1234567890123456789012345678901234567890")},
		{Name: "refs/tags/v1.0.0", Hash: mustParseHash(t, "2345678901234567890123456789012345678901")},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputHuman(refs)
	require.NoError(t, err)

	// Restore stdout
	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "1234567890123456789012345678901234567890\trefs/heads/main")
	assert.Contains(t, output, "2345678901234567890123456789012345678901\trefs/tags/v1.0.0")
}

func TestLsRemoteCommand(t *testing.T) {
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
			name:        "too many arguments returns error",
			args:        []string{"url1", "url2"},
			expectError: true,
		},
		{
			name:        "valid url argument accepted",
			args:        []string{"https://github.com/grafana/nanogit"},
			expectError: false, // Will fail when trying to connect, but args validation passes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			lsRemoteHeads = false
			lsRemoteTags = false
			lsRemoteJSON = false
			lsRemoteToken = ""

			// Test argument validation
			lsRemoteCmd.SetArgs(tt.args)
			err := lsRemoteCmd.Args(lsRemoteCmd, tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// mustParseHash parses a hash string or fails the test
func mustParseHash(t *testing.T, s string) hash.Hash {
	t.Helper()
	return hash.MustFromHex(s)
}
