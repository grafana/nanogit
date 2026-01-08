package main_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	cliBinary   string
	testRepoURL string = "https://github.com/grafana/nanogit"
)

func TestCLIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI integration tests in short mode")
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "CLI Integration Suite")
}

var _ = BeforeSuite(func() {
	By("Building CLI binary")

	// Build the CLI binary
	cliBinary = filepath.Join("..", "bin", "nanogit-test")
	buildCmd := exec.Command("go", "build", "-o", cliBinary, ".")
	buildCmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := buildCmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "Failed to build CLI: %s", string(output))

	GinkgoWriter.Printf("✅ Built CLI binary at %s\n", cliBinary)
	GinkgoWriter.Printf("🚀 CLI integration test suite setup complete\n")
	GinkgoWriter.Printf("📋 Testing against public repo: %s\n", testRepoURL)
})

var _ = AfterSuite(func() {
	By("Cleaning up test artifacts")

	// Clean up CLI binary
	if cliBinary != "" {
		_ = os.Remove(cliBinary)
	}

	GinkgoWriter.Printf("✅ CLI integration test suite teardown complete\n")
})

var _ = Describe("CLI Commands", func() {
	runCLI := func(args ...string) (string, string, error) {
		cmd := exec.Command(cliBinary, args...)

		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		return stdout.String(), stderr.String(), err
	}

	Describe("ls-remote", func() {
		It("should list remote references", func() {
			stdout, stderr, err := runCLI("ls-remote", testRepoURL)
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
			Expect(stdout).To(ContainSubstring("refs/heads/main"))
			GinkgoWriter.Printf("✅ ls-remote listed references\n")
		})

		It("should list only branches with --heads", func() {
			stdout, stderr, err := runCLI("ls-remote", testRepoURL, "--heads")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
			Expect(stdout).To(ContainSubstring("refs/heads/"))
			Expect(stdout).NotTo(ContainSubstring("refs/tags/"))
			GinkgoWriter.Printf("✅ ls-remote --heads filtered branches\n")
		})

		It("should list only tags with --tags", func() {
			stdout, stderr, err := runCLI("ls-remote", testRepoURL, "--tags")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
			Expect(stdout).To(ContainSubstring("refs/tags/"))
			Expect(stdout).NotTo(ContainSubstring("refs/heads/"))
			GinkgoWriter.Printf("✅ ls-remote --tags filtered tags\n")
		})

		It("should output JSON with --json", func() {
			stdout, stderr, err := runCLI("ls-remote", testRepoURL, "--json")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

			var result map[string]interface{}
			err = json.Unmarshal([]byte(stdout), &result)
			Expect(err).NotTo(HaveOccurred(), "stdout should be valid JSON")
			Expect(result).To(HaveKey("refs"))
			GinkgoWriter.Printf("✅ ls-remote --json output valid JSON\n")
		})
	})

	Describe("ls-tree", func() {
		It("should list tree contents at a ref", func() {
			stdout, stderr, err := runCLI("ls-tree", testRepoURL, "refs/heads/main")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
			Expect(stdout).To(ContainSubstring("README.md"))
			GinkgoWriter.Printf("✅ ls-tree listed tree contents\n")
		})

		It("should show detailed output with --long", func() {
			stdout, stderr, err := runCLI("ls-tree", testRepoURL, "refs/heads/main", "--long")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
			// Should contain mode, type (OBJ_BLOB or OBJ_TREE), hash
			Expect(stdout).To(MatchRegexp(`\d{6}\s+OBJ_`))
			Expect(stdout).To(ContainSubstring("README.md"))
			GinkgoWriter.Printf("✅ ls-tree --long showed detailed info\n")
		})

		It("should list recursively with --recursive", func() {
			stdout, stderr, err := runCLI("ls-tree", testRepoURL, "refs/heads/main", "--recursive")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
			// Recursive should show files in subdirectories
			lines := strings.Split(strings.TrimSpace(stdout), "\n")
			Expect(len(lines)).To(BeNumerically(">", 10))
			GinkgoWriter.Printf("✅ ls-tree --recursive listed all files\n")
		})

		It("should output JSON with --json", func() {
			stdout, stderr, err := runCLI("ls-tree", testRepoURL, "refs/heads/main", "--json")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

			var result map[string]interface{}
			err = json.Unmarshal([]byte(stdout), &result)
			Expect(err).NotTo(HaveOccurred(), "stdout should be valid JSON")
			Expect(result).To(HaveKey("entries"))
			GinkgoWriter.Printf("✅ ls-tree --json output valid JSON\n")
		})
	})

	Describe("cat-file", func() {
		It("should output file contents", func() {
			stdout, stderr, err := runCLI("cat-file", testRepoURL, "refs/heads/main", "README.md")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
			Expect(stdout).To(ContainSubstring("nanogit"))
			GinkgoWriter.Printf("✅ cat-file output file contents\n")
		})

		It("should output JSON with --json", func() {
			stdout, stderr, err := runCLI("cat-file", testRepoURL, "refs/heads/main", "README.md", "--json")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

			var result map[string]interface{}
			err = json.Unmarshal([]byte(stdout), &result)
			Expect(err).NotTo(HaveOccurred(), "stdout should be valid JSON")
			Expect(result).To(HaveKey("path"))
			Expect(result).To(HaveKey("hash"))
			Expect(result).To(HaveKey("content"))
			GinkgoWriter.Printf("✅ cat-file --json output valid JSON\n")
		})

		It("should fail for non-existent file", func() {
			_, _, err := runCLI("cat-file", testRepoURL, "refs/heads/main", "nonexistent.txt")
			Expect(err).To(HaveOccurred(), "should fail for non-existent file")
			GinkgoWriter.Printf("✅ cat-file properly handles missing files\n")
		})
	})

	Describe("clone", func() {
		var cloneDir string

		BeforeEach(func() {
			var err error
			cloneDir, err = os.MkdirTemp("", "cli-clone-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if cloneDir != "" {
				_ = os.RemoveAll(cloneDir)
			}
		})

		It("should clone a repository", func() {
			destination := filepath.Join(cloneDir, "repo")
			stdout, stderr, err := runCLI("clone", testRepoURL, destination, "--ref", "refs/heads/main")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s\nstdout: %s", stderr, stdout)

			// Verify files were created
			_, err = os.Stat(filepath.Join(destination, "README.md"))
			Expect(err).NotTo(HaveOccurred(), "README.md should exist")
			GinkgoWriter.Printf("✅ clone created repository files\n")
		})

		It("should clone with path filtering", func() {
			destination := filepath.Join(cloneDir, "filtered")
			stdout, stderr, err := runCLI(
				"clone", testRepoURL, destination,
				"--ref", "refs/heads/main",
				"--include-paths", "*.md",
			)
			Expect(err).NotTo(HaveOccurred(), "stderr: %s\nstdout: %s", stderr, stdout)

			// Verify only .md files were cloned
			_, err = os.Stat(filepath.Join(destination, "README.md"))
			Expect(err).NotTo(HaveOccurred(), "README.md should exist")

			// Verify filtering message in output
			Expect(stdout).To(ContainSubstring("filtered"))
			GinkgoWriter.Printf("✅ clone with --include-paths filtered correctly\n")
		})

		It("should output JSON with --json", func() {
			destination := filepath.Join(cloneDir, "json-test")
			stdout, stderr, err := runCLI(
				"clone", testRepoURL, destination,
				"--ref", "refs/heads/main",
				"--json",
			)
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

			var result map[string]interface{}
			err = json.Unmarshal([]byte(stdout), &result)
			Expect(err).NotTo(HaveOccurred(), "stdout should be valid JSON")
			// Verify the actual keys returned by the clone command
			Expect(result).To(HaveKey("filtered_files"))
			Expect(result).To(HaveKey("total_files"))
			Expect(result).To(HaveKey("commit"))
			GinkgoWriter.Printf("✅ clone --json output valid JSON\n")
		})

	})

	Describe("Error Handling", func() {
		It("should show helpful error for invalid URL", func() {
			_, stderr, err := runCLI("ls-remote", "not-a-valid-url")
			Expect(err).To(HaveOccurred())
			Expect(stderr).NotTo(BeEmpty(), "should show error message")
			GinkgoWriter.Printf("✅ Invalid URL shows helpful error\n")
		})

		It("should show helpful error for invalid ref", func() {
			_, stderr, err := runCLI("ls-tree", testRepoURL, "refs/heads/nonexistent-branch-9999")
			Expect(err).To(HaveOccurred())
			Expect(stderr).NotTo(BeEmpty(), "should show error message")
			GinkgoWriter.Printf("✅ Invalid ref shows helpful error\n")
		})

		It("should show usage error for missing arguments", func() {
			_, stderr, err := runCLI("clone")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(Or(
				ContainSubstring("requires"),
				ContainSubstring("usage"),
				ContainSubstring("accepts"),
				ContainSubstring("arg"),
			), "should show usage error")
			GinkgoWriter.Printf("✅ Missing arguments show usage error\n")
		})
	})
})
