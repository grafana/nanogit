package output

import (
	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
)

// Formatter defines the interface for different output formats
type Formatter interface {
	// FormatRefs outputs a list of Git references (branches/tags)
	FormatRefs(refs []nanogit.Ref) error

	// FormatTreeEntries outputs tree entries (files and directories)
	FormatTreeEntries(entries []nanogit.FlatTreeEntry) error

	// FormatBlobContent outputs file content
	FormatBlobContent(path string, hash hash.Hash, content []byte) error

	// FormatCloneResult outputs clone operation results
	FormatCloneResult(result *nanogit.CloneResult) error
}

// Get returns the appropriate formatter based on format type
func Get(format string) Formatter {
	switch format {
	case "json":
		return NewJSONFormatter()
	default:
		return NewHumanFormatter()
	}
}
