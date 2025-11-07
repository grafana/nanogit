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

// TestShouldIncludePath_LongPaths tests filtering with deeply nested paths
func TestShouldIncludePath_LongPaths(t *testing.T) {
	client := &httpClient{}

	tests := []struct {
		name         string
		path         string
		includePaths []string
		excludePaths []string
		want         bool
	}{
		{
			name:         "deeply nested go file - include **/*.go",
			path:         "pkg/services/api/v1/handlers/auth/middleware/jwt/validator/token.go",
			includePaths: []string{"**/*.go"},
			excludePaths: nil,
			want:         true,
		},
		{
			name:         "deeply nested test file - exclude test",
			path:         "pkg/services/api/v1/handlers/auth/middleware/jwt/validator/token_test.go",
			includePaths: []string{"**/*.go"},
			excludePaths: []string{"**/*_test.go"},
			want:         false,
		},
		{
			name:         "deeply nested in excluded directory",
			path:         "src/node_modules/react/dist/cjs/react.development.js",
			includePaths: []string{"src/**"},
			excludePaths: []string{"**/node_modules/**"},
			want:         false,
		},
		{
			name:         "deeply nested mock file",
			path:         "internal/pkg/services/storage/backends/s3/mocks/mock_client.go",
			includePaths: []string{"**/*.go"},
			excludePaths: []string{"**/mocks/**", "**/*_test.go"},
			want:         false,
		},
		{
			name:         "very long path with multiple segments",
			path:         "services/backend/infrastructure/kubernetes/controllers/deployment/reconciler/handlers/update/strategy.go",
			includePaths: []string{"services/backend/**"},
			excludePaths: []string{"**/test/**", "**/testdata/**"},
			want:         true,
		},
		{
			name:         "long path matching directory at depth",
			path:         "projects/frontend/app/components/shared/buttons/icon/test/fixtures/data.json",
			includePaths: []string{"projects/**"},
			excludePaths: []string{"**/test/**"},
			want:         false,
		},
		{
			name:         "long path with multiple wildcards in pattern",
			path:         "src/main/java/com/example/app/services/impl/UserServiceImpl.java",
			includePaths: []string{"**/*.java"},
			excludePaths: []string{"**/test/**", "**/target/**"},
			want:         true,
		},
		{
			name:         "proto files in deeply nested vendor",
			path:         "vendor/github.com/grpc/grpc-go/internal/proto/grpc_lookup_v1/rls.proto",
			includePaths: []string{"**/*.proto"},
			excludePaths: []string{"vendor/**"},
			want:         false,
		},
		{
			name:         "config file in deeply nested path",
			path:         "deployments/production/us-east-1/services/api/config/app.yaml",
			includePaths: []string{"deployments/production/**/*.yaml"},
			excludePaths: []string{"**/test/**", "**/temp/**"},
			want:         true,
		},
		{
			name:         "exclude multiple nested dirs",
			path:         "build/output/temp/cache/artifacts/debug/symbols.db",
			includePaths: []string{"build/**"},
			excludePaths: []string{"**/temp/**", "**/cache/**", "**/debug/**"},
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

// TestShouldIncludePath_ComplexCombinations tests complex include/exclude pattern combinations
func TestShouldIncludePath_ComplexCombinations(t *testing.T) {
	client := &httpClient{}

	tests := []struct {
		name         string
		path         string
		includePaths []string
		excludePaths []string
		want         bool
	}{
		{
			name:         "multiple includes with multiple excludes - match first include",
			path:         "src/components/Button.tsx",
			includePaths: []string{"src/**", "lib/**", "packages/**"},
			excludePaths: []string{"**/*.test.tsx", "**/*.spec.tsx", "**/node_modules/**"},
			want:         true,
		},
		{
			name:         "multiple includes with multiple excludes - match exclude",
			path:         "src/components/Button.test.tsx",
			includePaths: []string{"src/**", "lib/**", "packages/**"},
			excludePaths: []string{"**/*.test.tsx", "**/*.spec.tsx", "**/node_modules/**"},
			want:         false,
		},
		{
			name:         "include specific extensions, exclude specific dirs",
			path:         "pkg/api/models/user.go",
			includePaths: []string{"**/*.go", "**/*.proto", "**/*.sql"},
			excludePaths: []string{"**/vendor/**", "**/mocks/**", "**/*_test.go"},
			want:         true,
		},
		{
			name:         "include specific extensions, exclude specific dirs - vendor match",
			path:         "vendor/google.golang.org/grpc/server.go",
			includePaths: []string{"**/*.go", "**/*.proto", "**/*.sql"},
			excludePaths: []string{"**/vendor/**", "**/mocks/**", "**/*_test.go"},
			want:         false,
		},
		{
			name:         "overlapping patterns - include dir, exclude subdir",
			path:         "docs/api/reference.md",
			includePaths: []string{"docs/**"},
			excludePaths: []string{"**/drafts/**", "**/internal/**"},
			want:         true,
		},
		{
			name:         "overlapping patterns - exclude subdir wins",
			path:         "docs/internal/notes.md",
			includePaths: []string{"docs/**"},
			excludePaths: []string{"**/drafts/**", "**/internal/**"},
			want:         false,
		},
		{
			name:         "multiple file extensions",
			path:         "config/production/database.yaml",
			includePaths: []string{"**/*.yaml", "**/*.yml", "**/*.json", "**/*.toml"},
			excludePaths: []string{"**/test/**", "**/examples/**"},
			want:         true,
		},
		{
			name:         "complex go project - source file",
			path:         "internal/handlers/http/v1/users/create.go",
			includePaths: []string{"internal/**/*.go", "pkg/**/*.go", "cmd/**/*.go"},
			excludePaths: []string{"**/*_test.go", "**/testdata/**", "**/vendor/**", "**/.git/**"},
			want:         true,
		},
		{
			name:         "complex go project - test file excluded",
			path:         "internal/handlers/http/v1/users/create_test.go",
			includePaths: []string{"internal/**/*.go", "pkg/**/*.go", "cmd/**/*.go"},
			excludePaths: []string{"**/*_test.go", "**/testdata/**", "**/vendor/**", "**/.git/**"},
			want:         false,
		},
		{
			name:         "include multiple source dirs, exclude build artifacts",
			path:         "services/auth/src/main/java/Service.java",
			includePaths: []string{"services/**/src/**", "libraries/**/src/**"},
			excludePaths: []string{"**/target/**", "**/build/**", "**/.gradle/**"},
			want:         true,
		},
		{
			name:         "monorepo - include packages, exclude tooling",
			path:         "packages/core/src/index.ts",
			includePaths: []string{"packages/**/*.ts", "packages/**/*.tsx"},
			excludePaths: []string{"**/*.test.ts", "**/*.spec.ts", "**/dist/**", "**/node_modules/**"},
			want:         true,
		},
		{
			name:         "monorepo - exclude dist",
			path:         "packages/core/dist/index.js",
			includePaths: []string{"packages/**/*.ts", "packages/**/*.tsx", "packages/**/*.js"},
			excludePaths: []string{"**/*.test.ts", "**/*.spec.ts", "**/dist/**", "**/node_modules/**"},
			want:         false,
		},
		{
			name:         "include docs, exclude specific formats",
			path:         "documentation/guides/getting-started.md",
			includePaths: []string{"documentation/**", "docs/**"},
			excludePaths: []string{"**/*.draft.md", "**/*.wip.md", "**/archive/**"},
			want:         true,
		},
		{
			name:         "include docs, exclude draft",
			path:         "documentation/guides/new-feature.draft.md",
			includePaths: []string{"documentation/**", "docs/**"},
			excludePaths: []string{"**/*.draft.md", "**/*.wip.md", "**/archive/**"},
			want:         false,
		},
		{
			name:         "infrastructure as code - include configs, exclude secrets",
			path:         "terraform/modules/vpc/main.tf",
			includePaths: []string{"**/*.tf", "**/*.tfvars"},
			excludePaths: []string{"**/*.tfstate", "**/*.backup", "**/*secret*", "**/.terraform/**"},
			want:         true,
		},
		{
			name:         "infrastructure as code - exclude secrets",
			path:         "terraform/modules/vpc/secrets.tfvars",
			includePaths: []string{"**/*.tf", "**/*.tfvars"},
			excludePaths: []string{"**/*.tfstate", "**/*.backup", "**/*secret*", "**/.terraform/**"},
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

// TestShouldIncludePath_EdgeCases tests edge cases and special scenarios
func TestShouldIncludePath_EdgeCases(t *testing.T) {
	client := &httpClient{}

	tests := []struct {
		name         string
		path         string
		includePaths []string
		excludePaths []string
		want         bool
	}{
		{
			name:         "path with dots in directory name",
			path:         "src/v1.0.0/api/handler.go",
			includePaths: []string{"src/**/*.go"},
			excludePaths: nil,
			want:         true,
		},
		{
			name:         "path with dashes and underscores",
			path:         "my-app/src/user_service/api-handler.go",
			includePaths: []string{"my-app/**"},
			excludePaths: []string{"**/test/**"},
			want:         true,
		},
		{
			name:         "hidden directory",
			path:         ".github/workflows/ci.yml",
			includePaths: []string{".github/**"},
			excludePaths: nil,
			want:         true,
		},
		{
			name:         "exclude hidden directories",
			path:         "src/.cache/data.json",
			includePaths: []string{"src/**"},
			excludePaths: []string{"**/.cache/**", "**/.tmp/**"},
			want:         false,
		},
		{
			name:         "multiple dots in filename",
			path:         "config/app.development.local.json",
			includePaths: []string{"**/*.json"},
			excludePaths: []string{"**/*.local.*"},
			want:         false,
		},
		{
			name:         "uppercase file extension",
			path:         "docs/README.MD",
			includePaths: []string{"**/*.MD", "**/*.md"},
			excludePaths: nil,
			want:         true,
		},
		{
			name:         "file with no extension",
			path:         "scripts/build",
			includePaths: []string{"scripts/**"},
			excludePaths: []string{"**/*.log", "**/*.tmp"},
			want:         true,
		},
		{
			name:         "root level hidden file",
			path:         ".gitignore",
			includePaths: []string{".*"},
			excludePaths: nil,
			want:         true,
		},
		{
			name:         "pattern matching directory name in middle",
			path:         "app/cache/data/users.json",
			includePaths: []string{"app/**"},
			excludePaths: []string{"**/cache/**"},
			want:         false,
		},
		{
			name:         "similar directory names",
			path:         "app/test-utils/helper.go",
			includePaths: []string{"app/**"},
			excludePaths: []string{"**/test/**"},
			want:         true,
		},
		{
			name:         "exact directory name match needed",
			path:         "app/testing/helper.go",
			includePaths: []string{"app/**"},
			excludePaths: []string{"**/test/**"},
			want:         true,
		},
		{
			name:         "pattern with directory that contains pattern chars",
			path:         "app/[bracket]/file.go",
			includePaths: []string{"app/**"},
			excludePaths: []string{"**/tmp/**"},
			want:         true,
		},
		{
			name:         "multiple consecutive slashes handled",
			path:         "src/app/main.go",
			includePaths: []string{"src/**"},
			excludePaths: nil,
			want:         true,
		},
		{
			name:         "single character directory",
			path:         "a/b/c/d.txt",
			includePaths: []string{"a/**"},
			excludePaths: nil,
			want:         true,
		},
		{
			name:         "pattern matching across multiple directory levels",
			path:         "level1/level2/level3/level4/level5/file.go",
			includePaths: []string{"level1/**/level5/**"},
			excludePaths: nil,
			want:         true,
		},
		{
			name:         "wildcard at beginning and end",
			path:         "src/services/user/handler.go",
			includePaths: []string{"**/services/**"},
			excludePaths: nil,
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

// TestShouldIncludePath_RealWorldScenarios tests realistic repository filtering scenarios
func TestShouldIncludePath_RealWorldScenarios(t *testing.T) {
	client := &httpClient{}

	tests := []struct {
		name         string
		description  string
		path         string
		includePaths []string
		excludePaths []string
		want         bool
	}{
		{
			name:         "kubernetes operator - include go code, exclude generated",
			description:  "K8s operator project with code generation",
			path:         "controllers/pod_controller.go",
			includePaths: []string{"**/*.go", "go.mod", "go.sum", "Makefile"},
			excludePaths: []string{"**/zz_generated.*", "**/vendor/**", "**/*_test.go"},
			want:         true,
		},
		{
			name:         "kubernetes operator - exclude generated",
			description:  "K8s operator project with code generation",
			path:         "api/v1/zz_generated.deepcopy.go",
			includePaths: []string{"**/*.go", "go.mod", "go.sum", "Makefile"},
			excludePaths: []string{"**/zz_generated.*", "**/vendor/**", "**/*_test.go"},
			want:         false,
		},
		{
			name:         "full stack app - include frontend source",
			description:  "Full stack app, clone only React frontend",
			path:         "frontend/src/components/Dashboard/UserTable.tsx",
			includePaths: []string{"frontend/src/**/*.tsx", "frontend/src/**/*.ts", "frontend/src/**/*.css"},
			excludePaths: []string{"**/node_modules/**", "**/dist/**", "**/*.test.tsx"},
			want:         true,
		},
		{
			name:         "full stack app - exclude tests",
			description:  "Full stack app, clone only React frontend",
			path:         "frontend/src/components/Dashboard/UserTable.test.tsx",
			includePaths: []string{"frontend/src/**/*.tsx", "frontend/src/**/*.ts", "frontend/src/**/*.css"},
			excludePaths: []string{"**/node_modules/**", "**/dist/**", "**/*.test.tsx"},
			want:         false,
		},
		{
			name:         "microservices monorepo - include specific service",
			description:  "Monorepo with multiple services",
			path:         "services/payment-api/internal/handlers/webhook.go",
			includePaths: []string{"services/payment-api/**", "pkg/shared/**"},
			excludePaths: []string{"**/*_test.go", "**/vendor/**", "**/testdata/**"},
			want:         true,
		},
		{
			name:         "microservices monorepo - exclude other services",
			description:  "Monorepo with multiple services",
			path:         "services/user-api/internal/handlers/auth.go",
			includePaths: []string{"services/payment-api/**", "pkg/shared/**"},
			excludePaths: []string{"**/*_test.go", "**/vendor/**", "**/testdata/**"},
			want:         false,
		},
		{
			name:         "microservices monorepo - include shared pkg",
			description:  "Monorepo with multiple services",
			path:         "pkg/shared/logger/logger.go",
			includePaths: []string{"services/payment-api/**", "pkg/shared/**"},
			excludePaths: []string{"**/*_test.go", "**/vendor/**", "**/testdata/**"},
			want:         true,
		},
		{
			name:         "docs site - include only content",
			description:  "Documentation site, clone only markdown content",
			path:         "content/docs/guides/authentication.md",
			includePaths: []string{"content/**/*.md", "content/**/*.mdx"},
			excludePaths: []string{"**/node_modules/**", "**/public/**", "**/.next/**"},
			want:         true,
		},
		{
			name:         "docs site - exclude build output",
			description:  "Documentation site, clone only markdown content",
			path:         "public/docs/guides/authentication.html",
			includePaths: []string{"content/**/*.md", "content/**/*.mdx"},
			excludePaths: []string{"**/node_modules/**", "**/public/**", "**/.next/**"},
			want:         false,
		},
		{
			name:         "CI/CD repo - include workflows and scripts",
			description:  "Repository with CI/CD configurations",
			path:         ".github/workflows/deploy-production.yml",
			includePaths: []string{".github/**", "scripts/**", "Makefile"},
			excludePaths: []string{"**/*.log", "**/cache/**"},
			want:         true,
		},
		{
			name:         "python ML project - include notebooks and source",
			description:  "Machine learning project with notebooks",
			path:         "notebooks/experiments/model-training-v2.ipynb",
			includePaths: []string{"**/*.py", "**/*.ipynb", "requirements.txt"},
			excludePaths: []string{"**/__pycache__/**", "**/*.pyc", "**/venv/**", "**/.pytest_cache/**"},
			want:         true,
		},
		{
			name:         "python ML project - exclude cache",
			description:  "Machine learning project with notebooks",
			path:         "src/__pycache__/model.cpython-39.pyc",
			includePaths: []string{"**/*.py", "**/*.ipynb", "requirements.txt"},
			excludePaths: []string{"**/__pycache__/**", "**/*.pyc", "**/venv/**", "**/.pytest_cache/**"},
			want:         false,
		},
		{
			name:         "mobile app - include iOS source only",
			description:  "Mobile app with iOS and Android",
			path:         "ios/MyApp/ViewControllers/HomeViewController.swift",
			includePaths: []string{"ios/**/*.swift", "ios/**/*.storyboard", "ios/**/*.xcodeproj/**"},
			excludePaths: []string{"**/Pods/**", "**/DerivedData/**", "**/build/**"},
			want:         true,
		},
		{
			name:         "mobile app - exclude pods",
			description:  "Mobile app with iOS and Android",
			path:         "ios/Pods/Alamofire/Source/Request.swift",
			includePaths: []string{"ios/**/*.swift", "ios/**/*.storyboard", "ios/**/*.xcodeproj/**"},
			excludePaths: []string{"**/Pods/**", "**/DerivedData/**", "**/build/**"},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.shouldIncludePath(tt.path, tt.includePaths, tt.excludePaths)
			require.Equal(t, tt.want, got, "[%s] shouldIncludePath(%q, %v, %v) = %v, want %v",
				tt.description, tt.path, tt.includePaths, tt.excludePaths, got, tt.want)
		})
	}
}
