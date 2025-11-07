package nanogit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestShouldIncludePath_ExcludePatterns tests exclude path filtering with various patterns
func TestShouldIncludePath_ExcludePatterns(t *testing.T) {
	client := &httpClient{}

	tests := []struct {
		name         string
		path         string
		excludePaths []string
		want         bool
	}{
		// Directory exclusion with /**
		{
			name:         "exclude directory with /** - direct child",
			path:         "node_modules/package.json",
			excludePaths: []string{"node_modules/**"},
			want:         false,
		},
		{
			name:         "exclude directory with /** - nested file",
			path:         "node_modules/react/index.js",
			excludePaths: []string{"node_modules/**"},
			want:         false,
		},
		{
			name:         "exclude directory with /** - directory itself",
			path:         "node_modules",
			excludePaths: []string{"node_modules/**"},
			want:         false,
		},
		{
			name:         "exclude directory with /** - not matching",
			path:         "src/node_modules.txt",
			excludePaths: []string{"node_modules/**"},
			want:         true,
		},

		// File extension exclusion at root level
		{
			name:         "exclude *.log - root level match",
			path:         "debug.log",
			excludePaths: []string{"*.log"},
			want:         false,
		},
		{
			name:         "exclude *.log - nested not matched",
			path:         "logs/debug.log",
			excludePaths: []string{"*.log"},
			want:         true,
		},

		// File extension exclusion at any depth with **
		{
			name:         "exclude **/*.log - root level match",
			path:         "debug.log",
			excludePaths: []string{"**/*.log"},
			want:         false,
		},
		{
			name:         "exclude **/*.log - nested match",
			path:         "logs/debug.log",
			excludePaths: []string{"**/*.log"},
			want:         false,
		},
		{
			name:         "exclude **/*.log - deeply nested match",
			path:         "src/server/logs/debug.log",
			excludePaths: []string{"**/*.log"},
			want:         false,
		},

		// Specific path exclusion
		{
			name:         "exclude specific path - exact match",
			path:         "src/test/fixtures",
			excludePaths: []string{"src/test/fixtures"},
			want:         false,
		},
		{
			name:         "exclude specific path - not matching",
			path:         "src/test/main.go",
			excludePaths: []string{"src/test/fixtures"},
			want:         true,
		},

		// Directory at any depth with **/dir/**
		{
			name:         "exclude **/test/** - top level",
			path:         "test/helper.go",
			excludePaths: []string{"**/test/**"},
			want:         false,
		},
		{
			name:         "exclude **/test/** - nested",
			path:         "src/test/helper.go",
			excludePaths: []string{"**/test/**"},
			want:         false,
		},
		{
			name:         "exclude **/test/** - deeply nested",
			path:         "app/pkg/test/helper.go",
			excludePaths: []string{"**/test/**"},
			want:         false,
		},

		// Direct children with dir/*
		{
			name:         "exclude src/* - direct child file",
			path:         "src/main.go",
			excludePaths: []string{"src/*"},
			want:         false,
		},
		{
			name:         "exclude src/* - direct child dir",
			path:         "src/utils",
			excludePaths: []string{"src/*"},
			want:         false,
		},
		{
			name:         "exclude src/* - nested not matched",
			path:         "src/utils/helper.go",
			excludePaths: []string{"src/*"},
			want:         true,
		},
		{
			name:         "exclude src/* - dir itself not matched",
			path:         "src",
			excludePaths: []string{"src/*"},
			want:         true,
		},

		// Multiple exclude patterns
		{
			name:         "multiple excludes - first pattern matches",
			path:         "node_modules/react/index.js",
			excludePaths: []string{"node_modules/**", "*.tmp", "test/**"},
			want:         false,
		},
		{
			name:         "multiple excludes - second pattern matches",
			path:         "cache.tmp",
			excludePaths: []string{"node_modules/**", "*.tmp", "test/**"},
			want:         false,
		},
		{
			name:         "multiple excludes - no match",
			path:         "src/main.go",
			excludePaths: []string{"node_modules/**", "*.tmp", "test/**"},
			want:         true,
		},

		// Filename at any depth with **/filename
		{
			name:         "exclude **/README.md - root level",
			path:         "README.md",
			excludePaths: []string{"**/README.md"},
			want:         false,
		},
		{
			name:         "exclude **/README.md - nested",
			path:         "docs/README.md",
			excludePaths: []string{"**/README.md"},
			want:         false,
		},
		{
			name:         "exclude **/README.md - deeply nested",
			path:         "docs/api/v1/README.md",
			excludePaths: []string{"**/README.md"},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.shouldIncludePath(tt.path, nil, tt.excludePaths)
			require.Equal(t, tt.want, got, "shouldIncludePath(%q, nil, %v) = %v, want %v",
				tt.path, tt.excludePaths, got, tt.want)
		})
	}
}

// TestShouldIncludePath_IncludePatterns tests include path filtering with various patterns
func TestShouldIncludePath_IncludePatterns(t *testing.T) {
	client := &httpClient{}

	tests := []struct {
		name         string
		path         string
		includePaths []string
		want         bool
	}{
		// Directory inclusion with /**
		{
			name:         "include src/** - direct child",
			path:         "src/main.go",
			includePaths: []string{"src/**"},
			want:         true,
		},
		{
			name:         "include src/** - nested file",
			path:         "src/utils/helper.go",
			includePaths: []string{"src/**"},
			want:         true,
		},
		{
			name:         "include src/** - directory itself",
			path:         "src",
			includePaths: []string{"src/**"},
			want:         true,
		},
		{
			name:         "include src/** - not matching",
			path:         "docs/README.md",
			includePaths: []string{"src/**"},
			want:         false,
		},

		// File extension inclusion at root level
		{
			name:         "include *.go - root level match",
			path:         "main.go",
			includePaths: []string{"*.go"},
			want:         true,
		},
		{
			name:         "include *.go - nested not matched",
			path:         "src/main.go",
			includePaths: []string{"*.go"},
			want:         false,
		},

		// File extension inclusion at any depth with **
		{
			name:         "include **/*.go - root level match",
			path:         "main.go",
			includePaths: []string{"**/*.go"},
			want:         true,
		},
		{
			name:         "include **/*.go - nested match",
			path:         "src/main.go",
			includePaths: []string{"**/*.go"},
			want:         true,
		},
		{
			name:         "include **/*.go - deeply nested match",
			path:         "src/utils/pkg/helper.go",
			includePaths: []string{"**/*.go"},
			want:         true,
		},
		{
			name:         "include **/*.go - not matching",
			path:         "README.md",
			includePaths: []string{"**/*.go"},
			want:         false,
		},

		// Specific path inclusion
		{
			name:         "include specific path - exact match",
			path:         "go.mod",
			includePaths: []string{"go.mod"},
			want:         true,
		},
		{
			name:         "include specific path - not matching",
			path:         "go.sum",
			includePaths: []string{"go.mod"},
			want:         false,
		},

		// Multiple directories with /**
		{
			name:         "include multiple dirs - first matches",
			path:         "src/main.go",
			includePaths: []string{"src/**", "docs/**"},
			want:         true,
		},
		{
			name:         "include multiple dirs - second matches",
			path:         "docs/api/README.md",
			includePaths: []string{"src/**", "docs/**"},
			want:         true,
		},
		{
			name:         "include multiple dirs - neither matches",
			path:         "test/helper.go",
			includePaths: []string{"src/**", "docs/**"},
			want:         false,
		},

		// Direct children with dir/*
		{
			name:         "include src/* - direct child file",
			path:         "src/main.go",
			includePaths: []string{"src/*"},
			want:         true,
		},
		{
			name:         "include src/* - nested not matched",
			path:         "src/utils/helper.go",
			includePaths: []string{"src/*"},
			want:         false,
		},

		// Root level files with specific names
		{
			name:         "include root files - go.mod match",
			path:         "go.mod",
			includePaths: []string{"go.mod", "go.sum", "README.md"},
			want:         true,
		},
		{
			name:         "include root files - go.sum match",
			path:         "go.sum",
			includePaths: []string{"go.mod", "go.sum", "README.md"},
			want:         true,
		},
		{
			name:         "include root files - no match",
			path:         "main.go",
			includePaths: []string{"go.mod", "go.sum", "README.md"},
			want:         false,
		},

		// Filename at any depth with **/filename
		{
			name:         "include **/Makefile - root level",
			path:         "Makefile",
			includePaths: []string{"**/Makefile"},
			want:         true,
		},
		{
			name:         "include **/Makefile - nested",
			path:         "scripts/Makefile",
			includePaths: []string{"**/Makefile"},
			want:         true,
		},
		{
			name:         "include **/Makefile - not matching",
			path:         "Makefile.txt",
			includePaths: []string{"**/Makefile"},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.shouldIncludePath(tt.path, tt.includePaths, nil)
			require.Equal(t, tt.want, got, "shouldIncludePath(%q, %v, nil) = %v, want %v",
				tt.path, tt.includePaths, got, tt.want)
		})
	}
}

// TestShouldIncludePath_IncludeExcludePrecedence tests that exclude patterns take precedence over include patterns
func TestShouldIncludePath_IncludeExcludePrecedence(t *testing.T) {
	client := &httpClient{}

	tests := []struct {
		name         string
		path         string
		includePaths []string
		excludePaths []string
		want         bool
	}{
		{
			name:         "exclude takes precedence - same directory",
			path:         "src/test/helper.go",
			includePaths: []string{"src/**"},
			excludePaths: []string{"**/test/**"},
			want:         false,
		},
		{
			name:         "exclude takes precedence - file extension",
			path:         "src/debug.log",
			includePaths: []string{"src/**"},
			excludePaths: []string{"**/*.log"},
			want:         false,
		},
		{
			name:         "exclude takes precedence - specific file",
			path:         "src/secrets.txt",
			includePaths: []string{"src/**"},
			excludePaths: []string{"**/secrets.txt"},
			want:         false,
		},
		{
			name:         "include wins when no exclude match",
			path:         "src/main.go",
			includePaths: []string{"src/**"},
			excludePaths: []string{"**/test/**", "**/*.log"},
			want:         true,
		},
		{
			name:         "both patterns match - exclude wins",
			path:         "node_modules/package.json",
			includePaths: []string{"**/*.json"},
			excludePaths: []string{"node_modules/**"},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.shouldIncludePath(tt.path, tt.includePaths, tt.excludePaths)
			require.Equal(t, tt.want, got, "shouldIncludePath(%q, %v, %v) = %v, want %v",
				tt.path, tt.includePaths, tt.excludePaths, got, tt.want)
		})
	}
}

// TestShouldIncludePath_EmptyPatterns tests behavior when no patterns are provided
func TestShouldIncludePath_EmptyPatterns(t *testing.T) {
	client := &httpClient{}

	tests := []struct {
		name         string
		path         string
		includePaths []string
		excludePaths []string
		want         bool
	}{
		{
			name:         "no patterns - include everything",
			path:         "src/main.go",
			includePaths: nil,
			excludePaths: nil,
			want:         true,
		},
		{
			name:         "empty include - include everything not excluded",
			path:         "src/main.go",
			includePaths: []string{},
			excludePaths: []string{"test/**"},
			want:         true,
		},
		{
			name:         "empty include - exclude matches",
			path:         "test/helper.go",
			includePaths: []string{},
			excludePaths: []string{"test/**"},
			want:         false,
		},
		{
			name:         "empty exclude - only include matches",
			path:         "src/main.go",
			includePaths: []string{"src/**"},
			excludePaths: []string{},
			want:         true,
		},
		{
			name:         "empty exclude - include doesn't match",
			path:         "docs/README.md",
			includePaths: []string{"src/**"},
			excludePaths: []string{},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.shouldIncludePath(tt.path, tt.includePaths, tt.excludePaths)
			require.Equal(t, tt.want, got, "shouldIncludePath(%q, %v, %v) = %v, want %v",
				tt.path, tt.includePaths, tt.excludePaths, got, tt.want)
		})
	}
}

// TestShouldIncludePath_ComplexPatterns tests more complex real-world scenarios
func TestShouldIncludePath_ComplexPatterns(t *testing.T) {
	client := &httpClient{}

	tests := []struct {
		name         string
		path         string
		includePaths []string
		excludePaths []string
		want         bool
	}{
		{
			name:         "go project - include source exclude generated",
			path:         "pkg/api/client.go",
			includePaths: []string{"**/*.go", "go.mod", "go.sum"},
			excludePaths: []string{"**/*_test.go", "**/mocks/**"},
			want:         true,
		},
		{
			name:         "go project - exclude test files",
			path:         "pkg/api/client_test.go",
			includePaths: []string{"**/*.go", "go.mod", "go.sum"},
			excludePaths: []string{"**/*_test.go", "**/mocks/**"},
			want:         false,
		},
		{
			name:         "go project - exclude mocks",
			path:         "pkg/mocks/mock_client.go",
			includePaths: []string{"**/*.go", "go.mod", "go.sum"},
			excludePaths: []string{"**/*_test.go", "**/mocks/**"},
			want:         false,
		},
		{
			name:         "go project - include go.mod",
			path:         "go.mod",
			includePaths: []string{"**/*.go", "go.mod", "go.sum"},
			excludePaths: []string{"**/*_test.go", "**/mocks/**"},
			want:         true,
		},
		{
			name:         "docs only - include markdown",
			path:         "docs/api/reference.md",
			includePaths: []string{"docs/**"},
			excludePaths: []string{"**/*.tmp", "**/drafts/**"},
			want:         true,
		},
		{
			name:         "docs only - exclude drafts",
			path:         "docs/drafts/notes.md",
			includePaths: []string{"docs/**"},
			excludePaths: []string{"**/*.tmp", "**/drafts/**"},
			want:         false,
		},
		{
			name:         "scripts and config - multiple includes",
			path:         "scripts/build.sh",
			includePaths: []string{"scripts/**", "*.mk", "Makefile"},
			excludePaths: []string{"**/*.log", "**/*.tmp"},
			want:         true,
		},
		{
			name:         "scripts and config - root makefile",
			path:         "Makefile",
			includePaths: []string{"scripts/**", "*.mk", "Makefile"},
			excludePaths: []string{"**/*.log", "**/*.tmp"},
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.shouldIncludePath(tt.path, tt.includePaths, tt.excludePaths)
			require.Equal(t, tt.want, got, "shouldIncludePath(%q, %v, %v) = %v, want %v",
				tt.path, tt.includePaths, tt.excludePaths, got, tt.want)
		})
	}
}
