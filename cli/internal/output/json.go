package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
)

// JSONFormatter outputs in JSON format
type JSONFormatter struct {
	encoder *json.Encoder
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() *JSONFormatter {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return &JSONFormatter{
		encoder: enc,
	}
}

// refOutput represents a Git reference for JSON output
type refOutput struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
}

// FormatRefs outputs references in JSON format
func (f *JSONFormatter) FormatRefs(refs []nanogit.Ref) error {
	output := make([]refOutput, len(refs))
	for i, ref := range refs {
		output[i] = refOutput{
			Name: ref.Name,
			Hash: ref.Hash.String(),
		}
	}
	return f.encoder.Encode(map[string]interface{}{
		"refs": output,
	})
}

// treeEntryOutput represents a tree entry for JSON output
type treeEntryOutput struct {
	Type string `json:"type"`
	Mode string `json:"mode"`
	Hash string `json:"hash"`
	Path string `json:"path"`
	Name string `json:"name"`
}

// FormatTreeEntries outputs tree entries in JSON format
func (f *JSONFormatter) FormatTreeEntries(entries []nanogit.FlatTreeEntry) error {
	output := make([]treeEntryOutput, len(entries))
	for i, entry := range entries {
		output[i] = treeEntryOutput{
			Type: entry.Type.String(),
			Mode: fmt.Sprintf("%06o", entry.Mode),
			Hash: entry.Hash.String(),
			Path: entry.Path,
			Name: entry.Name,
		}
	}
	return f.encoder.Encode(map[string]interface{}{
		"entries": output,
	})
}

// blobOutput represents file content for JSON output
type blobOutput struct {
	Path    string `json:"path"`
	Hash    string `json:"hash"`
	Size    int    `json:"size"`
	Content string `json:"content"`
}

// FormatBlobContent outputs file content in JSON format
func (f *JSONFormatter) FormatBlobContent(path string, hash hash.Hash, content []byte) error {
	output := blobOutput{
		Path:    path,
		Hash:    hash.String(),
		Size:    len(content),
		Content: string(content),
	}
	return f.encoder.Encode(output)
}

// cloneResultOutput represents clone result for JSON output
type cloneResultOutput struct {
	Path          string `json:"path"`
	Commit        string `json:"commit"`
	TotalFiles    int    `json:"total_files"`
	FilteredFiles int    `json:"filtered_files"`
}

// FormatCloneResult outputs clone results in JSON format
func (f *JSONFormatter) FormatCloneResult(result *nanogit.CloneResult) error {
	output := cloneResultOutput{
		Path:          result.Path,
		Commit:        result.Commit.Hash.String(),
		TotalFiles:    result.TotalFiles,
		FilteredFiles: result.FilteredFiles,
	}
	return f.encoder.Encode(output)
}
