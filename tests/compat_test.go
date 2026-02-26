package integration_test

import (
	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	. "github.com/onsi/gomega"
)

// Type aliases and wrappers for backward compatibility with existing tests.
// These allow existing test code to continue working without modification
// while using the new testutil package under the hood with Ginkgo-friendly error handling.

// User is an alias for gittest.User
type User = gittest.User

// LocalGitRepo wraps gittest.LocalRepo and adds Ginkgo-friendly methods
// that automatically fail tests on errors.
type LocalGitRepo struct {
	*gittest.LocalRepo
}

// CreateFile creates a file and fails the test if there's an error.
func (r *LocalGitRepo) CreateFile(path, content string) {
	err := r.LocalRepo.CreateFile(path, content)
	Expect(err).NotTo(HaveOccurred())
}

// UpdateFile updates a file and fails the test if there's an error.
func (r *LocalGitRepo) UpdateFile(path, content string) {
	err := r.LocalRepo.UpdateFile(path, content)
	Expect(err).NotTo(HaveOccurred())
}

// DeleteFile deletes a file and fails the test if there's an error.
func (r *LocalGitRepo) DeleteFile(path string) {
	err := r.LocalRepo.DeleteFile(path)
	Expect(err).NotTo(HaveOccurred())
}

// CreateDirPath creates a directory path and fails the test if there's an error.
func (r *LocalGitRepo) CreateDirPath(dirpath string) {
	err := r.LocalRepo.CreateDirPath(dirpath)
	Expect(err).NotTo(HaveOccurred())
}

// Git runs a git command and fails the test if there's an error.
// This maintains backward compatibility with the old Git() method that only returned string.
func (r *LocalGitRepo) Git(args ...string) string {
	output, err := r.LocalRepo.Git(args...)
	Expect(err).NotTo(HaveOccurred(), "git command failed: %v", err)
	return output
}

// QuickInit initializes the repository with a remote and fails the test on error.
func (r *LocalGitRepo) QuickInit(user *gittest.User, remoteURL string) nanogit.Client {
	client, err := r.LocalRepo.QuickInit(user, remoteURL)
	Expect(err).NotTo(HaveOccurred())
	return client
}

// RemoteRepo wraps gittest.Repo to add backward-compatible methods
type RemoteRepo struct {
	*gittest.Repo
}

// URL returns the public URL (for backward compatibility with old tests)
func (r *RemoteRepo) URL() string {
	return r.Repo.URL
}

// AuthURL returns the authenticated URL (for backward compatibility)
func (r *RemoteRepo) AuthURL() string {
	return r.Repo.AuthURL
}
