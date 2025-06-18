package performance

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// TestDataGenerator helps create test repositories with various characteristics
type TestDataGenerator struct {
	baseURL string
	client  GitClient
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator(baseURL string, client GitClient) *TestDataGenerator {
	return &TestDataGenerator{
		baseURL: baseURL,
		client:  client,
	}
}

// RepoSpec defines the characteristics of a test repository
type RepoSpec struct {
	Name        string
	FileCount   int
	CommitCount int
	MaxDepth    int
	FileSizes   []int // Various file sizes in bytes
	BinaryFiles int   // Number of binary files
	Branches    int   // Number of branches
}

// GetStandardSpecs returns predefined repository specifications
func GetStandardSpecs() []RepoSpec {
	return []RepoSpec{
		{
			Name:        "small",
			FileCount:   50,
			CommitCount: 10,
			MaxDepth:    3,
			FileSizes:   []int{100, 1000, 5000},
			BinaryFiles: 2,
			Branches:    2,
		},
		{
			Name:        "medium",
			FileCount:   500,
			CommitCount: 100,
			MaxDepth:    5,
			FileSizes:   []int{500, 2000, 10000, 50000},
			BinaryFiles: 10,
			Branches:    5,
		},
		// {
		// 	Name:        "large",
		// 	FileCount:   2000,
		// 	CommitCount: 500,
		// 	MaxDepth:    8,
		// 	FileSizes:   []int{1000, 5000, 25000, 100000, 500000},
		// 	BinaryFiles: 50,
		// 	Branches:    10,
		// },
	}
}

// GenerateRepository creates a test repository according to the specification
func (g *TestDataGenerator) GenerateRepository(ctx context.Context, spec RepoSpec) error {
	// Use the baseURL directly - it should already be the complete authenticated URL
	repoURL := g.baseURL

	// Generate initial file structure
	files := g.generateFileStructure(spec)

	// Create repository with initial commit
	err := g.client.BulkCreateFiles(ctx, repoURL, files, "Initial commit with test data")
	if err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	// Generate additional commits
	for i := 1; i < spec.CommitCount; i++ {
		changes := g.generateCommitChanges(spec, i)
		if len(changes) > 0 {
			message := fmt.Sprintf("Commit %d: %s", i+1, g.generateCommitMessage(changes))
			err := g.client.BulkCreateFiles(ctx, repoURL, changes, message)
			if err != nil {
				return fmt.Errorf("failed to create commit %d: %w", i+1, err)
			}
		}
	}

	return nil
}

// generateFileStructure creates the initial file structure
func (g *TestDataGenerator) generateFileStructure(spec RepoSpec) []FileChange {
	rand.Seed(time.Now().UnixNano())
	files := make([]FileChange, 0, spec.FileCount)

	// Generate directory structure
	dirs := g.generateDirectoryStructure(spec.MaxDepth)

	// Distribute files across directories
	for i := 0; i < spec.FileCount; i++ {
		dir := dirs[rand.Intn(len(dirs))]
		filename := g.generateFilename(i, spec)
		path := fmt.Sprintf("%s/%s", dir, filename)
		if dir == "" {
			path = filename
		}

		content := g.generateFileContent(i, spec)

		files = append(files, FileChange{
			Path:    path,
			Content: content,
			Action:  "create",
		})
	}

	// Add some standard files
	files = append(files, []FileChange{
		{
			Path:    "README.md",
			Content: g.generateReadme(spec),
			Action:  "create",
		},
		{
			Path:    ".gitignore",
			Content: g.generateGitignore(),
			Action:  "create",
		},
		{
			Path:    "LICENSE",
			Content: g.generateLicense(),
			Action:  "create",
		},
	}...)

	return files
}

// generateDirectoryStructure creates a realistic directory structure
func (g *TestDataGenerator) generateDirectoryStructure(maxDepth int) []string {
	dirs := []string{""}

	baseDirs := []string{"src", "docs", "tests", "config", "scripts", "assets"}

	for _, baseDir := range baseDirs {
		dirs = append(dirs, baseDir)

		// Create subdirectories
		for depth := 1; depth < maxDepth; depth++ {
			subDirs := []string{"utils", "components", "models", "handlers", "common"}
			for _, subDir := range subDirs {
				if rand.Float32() < 0.6 { // 60% chance of creating subdirectory
					path := baseDir
					for i := 1; i <= depth; i++ {
						path = fmt.Sprintf("%s/%s", path, subDir)
					}
					dirs = append(dirs, path)
				}
			}
		}
	}

	return dirs
}

// generateFilename creates realistic filenames
func (g *TestDataGenerator) generateFilename(index int, spec RepoSpec) string {
	extensions := []string{".go", ".js", ".py", ".java", ".cpp", ".h", ".md", ".txt", ".json", ".yaml", ".xml"}

	// Binary file extensions
	binaryExtensions := []string{".png", ".jpg", ".pdf", ".zip", ".exe", ".so", ".dll"}

	var ext string
	if index < spec.BinaryFiles {
		ext = binaryExtensions[rand.Intn(len(binaryExtensions))]
	} else {
		ext = extensions[rand.Intn(len(extensions))]
	}

	prefixes := []string{"test", "main", "util", "helper", "config", "model", "handler", "service", "component"}
	prefix := prefixes[rand.Intn(len(prefixes))]

	return fmt.Sprintf("%s_%04d%s", prefix, index, ext)
}

// generateFileContent creates realistic file content
func (g *TestDataGenerator) generateFileContent(index int, spec RepoSpec) string {
	// Choose a random file size from the spec
	size := spec.FileSizes[rand.Intn(len(spec.FileSizes))]

	// For binary files (first spec.BinaryFiles files), generate binary-looking content
	if index < spec.BinaryFiles {
		return g.generateBinaryContent(size)
	}

	// Generate text content
	return g.generateTextContent(size, index)
}

// generateBinaryContent creates binary-like content (base64 encoded)
func (g *TestDataGenerator) generateBinaryContent(size int) string {
	// Generate random bytes and encode as base64 to simulate binary content
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	content := make([]byte, size)
	for i := range content {
		content[i] = chars[rand.Intn(len(chars))]
	}
	return string(content)
}

// generateTextContent creates realistic text content
func (g *TestDataGenerator) generateTextContent(size int, index int) string {
	words := []string{
		"function", "variable", "constant", "class", "method", "interface", "struct",
		"package", "import", "export", "return", "if", "else", "for", "while",
		"switch", "case", "default", "try", "catch", "finally", "throw", "new",
		"this", "super", "static", "public", "private", "protected", "async",
		"await", "promise", "callback", "event", "listener", "handler", "service",
		"component", "module", "library", "framework", "application", "system",
		"database", "query", "result", "response", "request", "client", "server",
		"configuration", "parameter", "argument", "value", "property", "attribute",
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("// File %d - Generated test content\n", index))
	content.WriteString(fmt.Sprintf("// Size target: %d bytes\n\n", size))

	currentSize := content.Len()
	for currentSize < size {
		line := fmt.Sprintf("const %s_%d = \"%s %s %s\";\n",
			words[rand.Intn(len(words))],
			rand.Intn(1000),
			words[rand.Intn(len(words))],
			words[rand.Intn(len(words))],
			words[rand.Intn(len(words))],
		)
		content.WriteString(line)
		currentSize = content.Len()
	}

	return content.String()
}

// generateCommitChanges creates realistic changes for subsequent commits
func (g *TestDataGenerator) generateCommitChanges(spec RepoSpec, commitIndex int) []FileChange {
	rand.Seed(time.Now().UnixNano() + int64(commitIndex))

	// Number of changes per commit (1-10% of total files)
	maxChanges := max(1, spec.FileCount/10)
	changeCount := rand.Intn(maxChanges) + 1

	changes := make([]FileChange, 0, changeCount)

	for i := 0; i < changeCount; i++ {
		action := g.chooseAction()

		switch action {
		case "create":
			changes = append(changes, FileChange{
				Path:    g.generateNewFilePath(spec, commitIndex, i),
				Content: g.generateTextContent(spec.FileSizes[rand.Intn(len(spec.FileSizes))], commitIndex*1000+i),
				Action:  "create",
			})
		case "update":
			// Update an existing file (simulate by creating a new version)
			changes = append(changes, FileChange{
				Path:    g.generateNewFilePath(spec, commitIndex, i), // Use new file path for updates too
				Content: g.generateTextContent(spec.FileSizes[rand.Intn(len(spec.FileSizes))], commitIndex*1000+i),
				Action:  "update",
			})
		case "delete":
			// Skip delete operations for now to avoid complexity
			// changes = append(changes, FileChange{
			//	Path:   g.generateExistingFilePath(spec, commitIndex),
			//	Action: "delete",
			// })
		}
	}

	return changes
}

// Helper methods
func (g *TestDataGenerator) chooseAction() string {
	r := rand.Float32()
	if r < 0.5 {
		return "update"
	} else if r < 0.8 {
		return "create"
	} else {
		return "delete"
	}
}

func (g *TestDataGenerator) generateNewFilePath(spec RepoSpec, commitIndex, fileIndex int) string {
	dirs := []string{"src", "docs", "tests", "new_features"}
	dir := dirs[rand.Intn(len(dirs))]
	return fmt.Sprintf("%s/commit_%d_file_%d.go", dir, commitIndex, fileIndex)
}

func (g *TestDataGenerator) generateExistingFilePath(spec RepoSpec, commitIndex int) string {
	// Generate a path that would likely exist from previous commits
	return fmt.Sprintf("src/util_%04d.go", rand.Intn(spec.FileCount/2))
}

func (g *TestDataGenerator) generateCommitMessage(changes []FileChange) string {
	messages := []string{
		"Add new functionality",
		"Fix bug in processing",
		"Update documentation",
		"Refactor code structure",
		"Improve performance",
		"Add error handling",
		"Update dependencies",
		"Fix memory leak",
		"Add unit tests",
		"Update configuration",
	}

	return messages[rand.Intn(len(messages))]
}

func (g *TestDataGenerator) generateReadme(spec RepoSpec) string {
	return fmt.Sprintf(`# %s Test Repository

This is a generated test repository for performance benchmarking.

## Specifications

- Files: %d
- Commits: %d
- Max Depth: %d
- Binary Files: %d
- Branches: %d

## Description

This repository contains generated test data for evaluating Git client performance
across different repository sizes and structures.

Generated on: %s
`, spec.Name, spec.FileCount, spec.CommitCount, spec.MaxDepth,
		spec.BinaryFiles, spec.Branches, time.Now().Format(time.RFC3339))
}

func (g *TestDataGenerator) generateGitignore() string {
	return `# Generated gitignore for test repository
*.log
*.tmp
.DS_Store
node_modules/
build/
dist/
.env
*.swp
*.swo
*~
`
}

func (g *TestDataGenerator) generateLicense() string {
	return `MIT License

Copyright (c) 2024 Performance Test Repository

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

