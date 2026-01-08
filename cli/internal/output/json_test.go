package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONFormatter_FormatRefs(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewJSONFormatter()
	formatter.encoder = json.NewEncoder(&buf)

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

	err := formatter.FormatRefs(refs)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "refs")
	refsArray := result["refs"].([]interface{})
	assert.Len(t, refsArray, 2)

	firstRef := refsArray[0].(map[string]interface{})
	assert.Equal(t, "refs/heads/main", firstRef["name"])
	assert.Equal(t, "0123456789abcdef0123456789abcdef01234567", firstRef["hash"])
}

func TestJSONFormatter_FormatTreeEntries(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewJSONFormatter()
	formatter.encoder = json.NewEncoder(&buf)

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

	err := formatter.FormatTreeEntries(entries)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "entries")
	entriesArray := result["entries"].([]interface{})
	assert.Len(t, entriesArray, 2)

	firstEntry := entriesArray[0].(map[string]interface{})
	assert.Equal(t, "README.md", firstEntry["path"])
	assert.Equal(t, "100644", firstEntry["mode"])
}

func TestJSONFormatter_FormatBlobContent(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewJSONFormatter()
	formatter.encoder = json.NewEncoder(&buf)

	content := []byte("Hello, World!")
	blobHash := hash.MustFromHex("0123456789abcdef0123456789abcdef01234567")

	err := formatter.FormatBlobContent("test.txt", blobHash, content)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "test.txt", result["path"])
	assert.Equal(t, "0123456789abcdef0123456789abcdef01234567", result["hash"])
	assert.Equal(t, "Hello, World!", result["content"])
}

func TestJSONFormatter_FormatCloneResult(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewJSONFormatter()
	formatter.encoder = json.NewEncoder(&buf)

	result := &nanogit.CloneResult{
		Commit: &nanogit.Commit{
			Hash: hash.MustFromHex("0123456789abcdef0123456789abcdef01234567"),
		},
		TotalFiles:    100,
		FilteredFiles: 50,
		Path:          "/tmp/repo",
	}

	err := formatter.FormatCloneResult(result)
	require.NoError(t, err)

	var output map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.Equal(t, "0123456789abcdef0123456789abcdef01234567", output["commit"])
	assert.Equal(t, float64(100), output["total_files"])
	assert.Equal(t, float64(50), output["filtered_files"])
	assert.Equal(t, "/tmp/repo", output["path"])
}

func TestJSONFormatter_EmptyRefs(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewJSONFormatter()
	formatter.encoder = json.NewEncoder(&buf)

	err := formatter.FormatRefs([]nanogit.Ref{})
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "refs")
	refsArray := result["refs"].([]interface{})
	assert.Len(t, refsArray, 0)
}

func TestJSONFormatter_EmptyTreeEntries(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewJSONFormatter()
	formatter.encoder = json.NewEncoder(&buf)

	err := formatter.FormatTreeEntries([]nanogit.FlatTreeEntry{})
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "entries")
	entriesArray := result["entries"].([]interface{})
	assert.Len(t, entriesArray, 0)
}
