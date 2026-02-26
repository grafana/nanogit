package integration_test

import (
	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	. "github.com/onsi/gomega"
)

// Type aliases and wrappers for backward compatibility with existing tests.
// These allow existing test code to continue working without modification
// while using the new gittest package under the hood with Ginkgo-friendly error handling.

// User is an alias for gittest.User
type User = gittest.User

// LocalRepository wraps gittest.LocalRepo and adds Ginkgo-friendly methods
// that automatically fail tests on errors.
type LocalRepository struct {
	*gittest.LocalRepo
}

// CreateFile creates a file and fails the test if there's an error.
func (r *LocalRepository) CreateFile(path, content string) {
	err := r.LocalRepo.CreateFile(path, content)
	Expect(err).NotTo(HaveOccurred())
}

// UpdateFile updates a file and fails the test if there's an error.
func (r *LocalRepository) UpdateFile(path, content string) {
	err := r.LocalRepo.UpdateFile(path, content)
	Expect(err).NotTo(HaveOccurred())
}

// DeleteFile deletes a file and fails the test if there's an error.
func (r *LocalRepository) DeleteFile(path string) {
	err := r.LocalRepo.DeleteFile(path)
	Expect(err).NotTo(HaveOccurred())
}

// CreateDirPath creates a directory path and fails the test if there's an error.
func (r *LocalRepository) CreateDirPath(dirpath string) {
	err := r.LocalRepo.CreateDirPath(dirpath)
	Expect(err).NotTo(HaveOccurred())
}

// Git runs a git command and fails the test if there's an error.
// This maintains backward compatibility with the old Git() method that only returned string.
func (r *LocalRepository) Git(args ...string) string {
	output, err := r.LocalRepo.Git(args...)
	Expect(err).NotTo(HaveOccurred(), "git command failed: %v", err)
	return output
}

// InitWithRemote initializes the repository with a remote and fails the test on error.
func (r *LocalRepository) InitWithRemote(user *gittest.User, remote *gittest.RemoteRepository) nanogit.Client {
	client, err := r.LocalRepo.InitWithRemote(user, remote)
	Expect(err).NotTo(HaveOccurred())
	return client
}

// RemoteRepo wraps gittest.RemoteRepository to add backward-compatible methods
type RemoteRepo struct {
	*gittest.RemoteRepository
}

// URL returns the public URL (for backward compatibility with old tests)
func (r *RemoteRepo) URL() string {
	return r.RemoteRepository.URL
}

// AuthURL returns the authenticated URL (for backward compatibility)
func (r *RemoteRepo) AuthURL() string {
	return r.RemoteRepository.AuthURL
}

// RepoName returns the repository name (for backward compatibility)
func (r *RemoteRepo) RepoName() string {
	return r.RemoteRepository.Name
}

// GitServer wraps gittest.Server to add backward-compatible methods
type GitServer struct {
	*gittest.Server
}

// GenerateUserToken generates a user token (backward compatible method).
// This wraps the new CreateToken API and maintains the old signature that didn't return errors.
func (s *GitServer) GenerateUserToken(username, password string) string {
	token, err := s.Server.CreateToken(ctx, username)
	Expect(err).NotTo(HaveOccurred())
	return token
}
