package integration_test

import (
	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/testutil"
	. "github.com/onsi/gomega"
)

// Type aliases and wrappers for backward compatibility with existing tests.
// These allow existing test code to continue working without modification
// while using the new testutil package under the hood with Ginkgo-friendly error handling.

// User is an alias for testutil.User
type User = testutil.User

// LocalGitRepo wraps testutil.LocalRepo and adds Ginkgo-friendly methods
// that automatically fail tests on errors.
type LocalGitRepo struct {
	*testutil.LocalRepo
}

// CreateFile creates a file and fails the test if there's an error.
func (r *LocalGitRepo) CreateFile(path, content string) error {
	err := r.LocalRepo.CreateFile(path, content)
	Expect(err).NotTo(HaveOccurred())
	return err
}

// UpdateFile updates a file and fails the test if there's an error.
func (r *LocalGitRepo) UpdateFile(path, content string) error {
	err := r.LocalRepo.UpdateFile(path, content)
	Expect(err).NotTo(HaveOccurred())
	return err
}

// DeleteFile deletes a file and fails the test if there's an error.
func (r *LocalGitRepo) DeleteFile(path string) error {
	err := r.LocalRepo.DeleteFile(path)
	Expect(err).NotTo(HaveOccurred())
	return err
}

// CreateDirPath creates a directory path and fails the test if there's an error.
func (r *LocalGitRepo) CreateDirPath(dirpath string) error {
	err := r.LocalRepo.CreateDirPath(dirpath)
	Expect(err).NotTo(HaveOccurred())
	return err
}

// Git runs a git command and fails the test if there's an error.
// This maintains backward compatibility with the old Git() method that only returned string.
func (r *LocalGitRepo) Git(args ...string) string {
	output, err := r.LocalRepo.Git(args...)
	Expect(err).NotTo(HaveOccurred(), "git command failed: %v", err)
	return output
}

// QuickInit initializes the repository with a remote and fails the test on error.
func (r *LocalGitRepo) QuickInit(user *testutil.User, remoteURL string) (nanogit.Client, string, error) {
	client, url, err := r.LocalRepo.QuickInit(user, remoteURL)
	Expect(err).NotTo(HaveOccurred())
	return client, url, err
}

// RemoteRepo wraps testutil.Repo to add backward-compatible methods
type RemoteRepo struct {
	*testutil.Repo
}

// URL returns the public URL (for backward compatibility with old tests)
func (r *RemoteRepo) URL() string {
	return r.Repo.URL
}

// AuthURL returns the authenticated URL (for backward compatibility)
func (r *RemoteRepo) AuthURL() string {
	return r.Repo.AuthURL
}
