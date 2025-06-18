package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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

// GetStandardSpecs returns predefined repository specifications for Grafana-focused repositories
func GetStandardSpecs() []RepoSpec {
	return []RepoSpec{
		{
			Name:        "small",
			FileCount:   100,                      // Small team: ~100 dashboards/resources
			CommitCount: 50,                       // More frequent updates typical of dashboard repos
			MaxDepth:    4,                        // Organized folder structure
			FileSizes:   []int{2000, 8000, 25000}, // Typical JSON dashboard sizes (2KB-25KB)
			BinaryFiles: 5,                        // Few binary assets (logos, images)
			Branches:    3,                        // main, staging, dev
		},
		{
			Name:        "medium",
			FileCount:   750,                             // Medium org: ~750 dashboards/resources
			CommitCount: 200,                             // Regular dashboard updates
			MaxDepth:    5,                               // More organized structure
			FileSizes:   []int{1500, 5000, 15000, 50000}, // Range of dashboard complexities
			BinaryFiles: 20,                              // More assets (team logos, custom images)
			Branches:    5,                               // main, staging, dev, feature branches
		},
		{
			Name:        "large",
			FileCount:   3000,                                    // Large enterprise: ~3000 dashboards/resources
			CommitCount: 800,                                     // High activity with many teams
			MaxDepth:    6,                                       // Deep organization by teams/products
			FileSizes:   []int{1000, 3000, 10000, 30000, 100000}, // Very complex dashboards
			BinaryFiles: 100,                                     // Many team assets, custom panels
			Branches:    8,                                       // Multiple environments and feature branches
		},
		{
			Name:        "xlarge",
			FileCount:   15000,                                        // Global enterprise: ~15,000 dashboards/resources across all services
			CommitCount: 3000,                                         // Massive scale with hundreds of teams contributing
			MaxDepth:    8,                                            // Very deep organization (global regions/services/teams/components)
			FileSizes:   []int{800, 2500, 8000, 25000, 75000, 200000}, // Wide range from simple to extremely complex dashboards
			BinaryFiles: 500,                                          // Extensive asset library (logos, custom visualizations, plugins)
			Branches:    15,                                           // Multiple environments, regions, and extensive feature branch workflows
		},
	}
}

func main() {
	ctx := context.Background()

	// Create temporary directory for repository generation
	tempDir, err := os.MkdirTemp("", "nanogit-testdata-*")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Printf("Generating test repositories in %s\n", tempDir)

	// Create testdata directory if it doesn't exist
	testdataDir := "./testdata"
	if err := os.MkdirAll(testdataDir, 0755); err != nil {
		log.Fatalf("Failed to create testdata directory: %v", err)
	}

	specs := GetStandardSpecs()

	for _, spec := range specs {
		fmt.Printf("Generating %s repository...\n", spec.Name)

		// Create repository directory
		repoDir := filepath.Join(tempDir, spec.Name+"-repo")
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			log.Fatalf("Failed to create repo directory: %v", err)
		}

		// Generate repository content using git CLI
		if err := generateRepository(ctx, repoDir, spec); err != nil {
			log.Fatalf("Failed to generate %s repository: %v", spec.Name, err)
		}

		// Create compressed archive
		archivePath := filepath.Join(testdataDir, spec.Name+"-repo.tar.gz")
		if err := createArchive(repoDir, archivePath); err != nil {
			log.Fatalf("Failed to create archive for %s: %v", spec.Name, err)
		}

		fmt.Printf("Created %s\n", archivePath)
	}

	fmt.Println("Test repository generation complete!")
}

func generateRepository(ctx context.Context, repoDir string, spec RepoSpec) error {
	// Initialize bare git repository
	if err := runGitCommand(ctx, repoDir, "init", "--bare"); err != nil {
		return fmt.Errorf("failed to initialize bare repository: %w", err)
	}

	// Create a working directory to build the repository content
	workDir := repoDir + "-work"
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}
	// defer os.RemoveAll(workDir)

	// Clone the bare repo to work on it
	if err := runGitCommand(ctx, "", "clone", repoDir, workDir); err != nil {
		return fmt.Errorf("failed to clone bare repository: %w", err)
	}

	// Configure git user
	if err := runGitCommand(ctx, workDir, "config", "user.name", "Performance Test"); err != nil {
		return fmt.Errorf("failed to configure git user: %w", err)
	}
	if err := runGitCommand(ctx, workDir, "config", "user.email", "test@example.com"); err != nil {
		return fmt.Errorf("failed to configure git email: %w", err)
	}

	// Generate files and commit them
	return generateRepositoryContent(ctx, workDir, spec)
}

func runGitCommand(ctx context.Context, dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s failed: %s (output: %s)", strings.Join(args, " "), err.Error(), string(output))
	}
	return nil
}

func generateRepositoryContent(ctx context.Context, workDir string, spec RepoSpec) error {
	// Generate initial file structure
	files := generateFileStructure(spec)

	// Create all files
	for _, file := range files {
		fullPath := filepath.Join(workDir, file.Path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", file.Path, err)
		}

		if err := os.WriteFile(fullPath, []byte(file.Content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", file.Path, err)
		}
	}

	// Add all files
	if err := runGitCommand(ctx, workDir, "add", "."); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	// Initial commit
	if err := runGitCommand(ctx, workDir, "commit", "-m", "Initial commit with test data"); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	// Generate additional commits
	for i := 1; i < spec.CommitCount; i++ {
		changes := generateCommitChanges(spec, i)
		if len(changes) > 0 {
			// Apply changes
			for _, change := range changes {
				switch strings.ToLower(change.Action) {
				case "create", "update":
					fullPath := filepath.Join(workDir, change.Path)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						return fmt.Errorf("failed to create directory for %s: %w", change.Path, err)
					}
					if err := os.WriteFile(fullPath, []byte(change.Content), 0644); err != nil {
						return fmt.Errorf("failed to write file %s: %w", change.Path, err)
					}
					if err := runGitCommand(ctx, workDir, "add", change.Path); err != nil {
						return fmt.Errorf("failed to add file %s: %w", change.Path, err)
					}
				}
			}

			// Commit changes
			message := fmt.Sprintf("Commit %d: %s", i+1, generateCommitMessage(changes))
			if err := runGitCommand(ctx, workDir, "commit", "-m", message); err != nil {
				return fmt.Errorf("failed to create commit %d: %w", i+1, err)
			}
		}
	}

	// Push to bare repository
	if err := runGitCommand(ctx, workDir, "push", "origin", "main"); err != nil {
		return fmt.Errorf("failed to push to bare repository: %w", err)
	}

	return nil
}

func createArchive(sourceDir, archivePath string) error {
	// Create archive file
	file, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer file.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Walk through source directory and add files to archive
	return filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from source directory
		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Write file content (if it's a regular file)
		if info.Mode().IsRegular() {
			sourceFile, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer sourceFile.Close()

			_, err = io.Copy(tarWriter, sourceFile)
			return err
		}

		return nil
	})
}

// FileChange represents a file operation
type FileChange struct {
	Path    string
	Content string
	Action  string
}

// generateFileStructure creates the initial file structure
func generateFileStructure(spec RepoSpec) []FileChange {
	rand.Seed(time.Now().UnixNano())
	files := make([]FileChange, 0, spec.FileCount)

	// Generate directory structure
	dirs := generateDirectoryStructure(spec.MaxDepth)

	// Distribute files across directories
	for i := 0; i < spec.FileCount; i++ {
		dir := dirs[rand.Intn(len(dirs))]
		filename := generateFilename(i, spec)
		path := fmt.Sprintf("%s/%s", dir, filename)
		if dir == "" {
			path = filename
		}

		content := generateFileContent(i, spec)

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
			Content: generateReadme(spec),
			Action:  "create",
		},
		{
			Path:    ".gitignore",
			Content: generateGitignore(),
			Action:  "create",
		},
		{
			Path:    "LICENSE",
			Content: generateLicense(),
			Action:  "create",
		},
	}...)

	return files
}

// generateDirectoryStructure creates a realistic Grafana repository directory structure
func generateDirectoryStructure(maxDepth int) []string {
	dirs := []string{""}

	// Typical Grafana repository structure
	baseDirs := []string{"dashboards", "alerts", "datasources", "folders", "assets", "docs"}

	for _, baseDir := range baseDirs {
		dirs = append(dirs, baseDir)

		// Create subdirectories based on typical Grafana organization
		var subDirs []string
		switch baseDir {
		case "dashboards":
			subDirs = []string{"infrastructure", "applications", "business", "security", "platform", "monitoring"}
		case "alerts":
			subDirs = []string{"critical", "warning", "info", "sla"}
		case "datasources":
			subDirs = []string{"prometheus", "loki", "tempo", "elasticsearch", "mysql"}
		case "folders":
			subDirs = []string{"teams", "products", "environments", "shared"}
		case "assets":
			subDirs = []string{"logos", "images", "icons", "plugins"}
		case "docs":
			subDirs = []string{"runbooks", "playbooks", "guides", "templates"}
		}

		// Create nested structure
		for depth := 1; depth < maxDepth; depth++ {
			for _, subDir := range subDirs {
				if rand.Float32() < 0.7 { // 70% chance for Grafana-specific dirs
					path := baseDir
					for i := 1; i <= depth; i++ {
						if i == 1 {
							path = fmt.Sprintf("%s/%s", path, subDir)
						} else {
							// Add team/product specific subdirs
							teamDirs := []string{"frontend", "backend", "devops", "data", "ml", "security"}
							path = fmt.Sprintf("%s/%s", path, teamDirs[rand.Intn(len(teamDirs))])
						}
					}
					dirs = append(dirs, path)
				}
			}
		}
	}

	return dirs
}

// generateFilename creates realistic Grafana-focused filenames
func generateFilename(index int, spec RepoSpec) string {
	// Grafana repository files are predominantly JSON with some YAML and docs
	extensions := []string{".json", ".json", ".json", ".json", ".yaml", ".yml", ".md", ".txt"} // Heavy bias toward JSON

	// Binary file extensions for Grafana assets
	binaryExtensions := []string{".png", ".jpg", ".svg", ".ico", ".gif"}

	var ext string
	if index < spec.BinaryFiles {
		ext = binaryExtensions[rand.Intn(len(binaryExtensions))]
	} else {
		ext = extensions[rand.Intn(len(extensions))]
	}

	// Grafana-specific naming patterns
	var prefix string
	switch ext {
	case ".json":
		dashboardPrefixes := []string{"dashboard", "panel", "alert", "datasource", "folder", "notification", "team", "user", "org"}
		prefix = dashboardPrefixes[rand.Intn(len(dashboardPrefixes))]
	case ".yaml", ".yml":
		configPrefixes := []string{"config", "alert-rules", "recording-rules", "provisioning", "datasource-config"}
		prefix = configPrefixes[rand.Intn(len(configPrefixes))]
	case ".md":
		docPrefixes := []string{"README", "runbook", "playbook", "guide", "troubleshooting", "setup"}
		prefix = docPrefixes[rand.Intn(len(docPrefixes))]
	default:
		prefix = "resource"
	}

	return fmt.Sprintf("%s-%04d%s", prefix, index, ext)
}

// generateFileContent creates realistic file content
func generateFileContent(index int, spec RepoSpec) string {
	// Choose a random file size from the spec
	size := spec.FileSizes[rand.Intn(len(spec.FileSizes))]

	// For binary files (first spec.BinaryFiles files), generate binary-looking content
	if index < spec.BinaryFiles {
		return generateBinaryContent(size)
	}

	// Generate text content
	return generateTextContent(size, index)
}

// generateBinaryContent creates binary-like content (base64 encoded)
func generateBinaryContent(size int) string {
	// Generate random bytes and encode as base64 to simulate binary content
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	content := make([]byte, size)
	for i := range content {
		content[i] = chars[rand.Intn(len(chars))]
	}
	return string(content)
}

// generateTextContent creates realistic text content
func generateTextContent(size int, index int) string {
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
func generateCommitChanges(spec RepoSpec, commitIndex int) []FileChange {
	rand.Seed(time.Now().UnixNano() + int64(commitIndex))

	// Much smaller number of changes per commit (1-5 files max)
	maxChanges := min(5, max(1, spec.FileCount/200))
	changeCount := rand.Intn(maxChanges) + 1

	changes := make([]FileChange, 0, changeCount)

	for i := 0; i < changeCount; i++ {
		action := chooseAction()

		switch action {
		case "create":
			// Only occasionally create new files (20% chance)
			if rand.Float32() < 0.2 {
				changes = append(changes, FileChange{
					Path:    generateNewFilePath(spec, commitIndex, i),
					Content: generateTextContent(spec.FileSizes[rand.Intn(len(spec.FileSizes))], commitIndex*1000+i),
					Action:  "create",
				})
			}
		case "update":
			// Update existing files by generating realistic paths from initial structure
			changes = append(changes, FileChange{
				Path:    generateExistingFilePath(spec, commitIndex, i),
				Content: generateTextContent(spec.FileSizes[rand.Intn(len(spec.FileSizes))], commitIndex*1000+i),
				Action:  "update",
			})
		}
	}

	return changes
}

// Helper methods
func chooseAction() string {
	r := rand.Float32()
	if r < 0.5 {
		return "update"
	} else {
		return "create"
	}
}

func generateNewFilePath(spec RepoSpec, commitIndex, fileIndex int) string {
	dirs := []string{"dashboards/new", "alerts/new", "docs/new"}
	dir := dirs[rand.Intn(len(dirs))]
	return fmt.Sprintf("%s/commit_%d_file_%d.json", dir, commitIndex, fileIndex)
}

// generateExistingFilePath creates paths that should match existing files
func generateExistingFilePath(spec RepoSpec, commitIndex, fileIndex int) string {
	// Generate paths that match the initial file structure
	baseDirs := []string{"dashboards", "alerts", "datasources", "folders", "assets", "docs"}
	baseDir := baseDirs[rand.Intn(len(baseDirs))]
	
	// Generate a file index that should exist in the initial structure
	existingIndex := rand.Intn(spec.FileCount)
	filename := generateFilename(existingIndex, spec)
	
	// Sometimes add subdirectory
	if rand.Float32() < 0.6 {
		var subDirs []string
		switch baseDir {
		case "dashboards":
			subDirs = []string{"infrastructure", "applications", "business"}
		case "alerts":
			subDirs = []string{"critical", "warning", "info"}
		case "datasources":
			subDirs = []string{"prometheus", "loki", "tempo"}
		default:
			subDirs = []string{"shared", "common"}
		}
		subDir := subDirs[rand.Intn(len(subDirs))]
		return fmt.Sprintf("%s/%s/%s", baseDir, subDir, filename)
	}
	
	return fmt.Sprintf("%s/%s", baseDir, filename)
}

func generateCommitMessage(changes []FileChange) string {
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

func generateReadme(spec RepoSpec) string {
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

func generateGitignore() string {
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

func generateLicense() string {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
