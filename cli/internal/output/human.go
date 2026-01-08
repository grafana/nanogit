package output

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
)

// HumanFormatter outputs in human-readable format with colors
type HumanFormatter struct {
	success *color.Color
	info    *color.Color
	dim     *color.Color
}

// NewHumanFormatter creates a new human-readable formatter
func NewHumanFormatter() *HumanFormatter {
	return &HumanFormatter{
		success: color.New(color.FgGreen),
		info:    color.New(color.FgCyan),
		dim:     color.New(color.Faint),
	}
}

// FormatRefs outputs references in human-readable format
func (f *HumanFormatter) FormatRefs(refs []nanogit.Ref) error {
	for _, ref := range refs {
		fmt.Printf("%s\t%s\n", f.dim.Sprint(ref.Hash.String()[:8]+"..."), ref.Name)
	}
	return nil
}

// FormatTreeEntries outputs tree entries in human-readable format
func (f *HumanFormatter) FormatTreeEntries(entries []nanogit.FlatTreeEntry) error {
	for _, entry := range entries {
		// Format: [mode] [type] [hash]  [path]
		modeStr := fmt.Sprintf("%06o", entry.Mode)
		hashStr := entry.Hash.String()[:8] + "..."
		fmt.Printf("%s %s %s  %s\n",
			f.dim.Sprint(modeStr),
			f.info.Sprint(entry.Type.String()[:4]),
			f.dim.Sprint(hashStr),
			entry.Path)
	}
	return nil
}

// FormatBlobContent outputs file content (raw to stdout)
func (f *HumanFormatter) FormatBlobContent(path string, hash hash.Hash, content []byte) error {
	// Just output raw content to stdout
	_, err := os.Stdout.Write(content)
	return err
}

// FormatCloneResult outputs clone results in human-readable format
func (f *HumanFormatter) FormatCloneResult(result *nanogit.CloneResult) error {
	f.success.Printf("✓ Cloned %d files to %s\n", result.FilteredFiles, result.Path)
	if result.FilteredFiles != result.TotalFiles {
		fmt.Printf("  (filtered %d of %d total files)\n", result.FilteredFiles, result.TotalFiles)
	}
	fmt.Printf("  Commit: %s\n", result.Commit.Hash.String()[:8]+"...")
	return nil
}
