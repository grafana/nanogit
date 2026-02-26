package integration_test

import (
	"github.com/grafana/nanogit/testutil"
	. "github.com/onsi/gomega"
)

// gitNoError is a helper that wraps LocalRepo.Git and fails the test on error
// This maintains backward compatibility with the old Git() method that didn't return errors
func gitNoError(local *testutil.LocalRepo, args ...string) string {
	output, err := local.Git(args...)
	Expect(err).NotTo(HaveOccurred(), "git command failed: %v", err)
	return output
}

// mustGit is an alias for gitNoError for tests that prefer this naming
func mustGit(local *testutil.LocalRepo, args ...string) string {
	return gitNoError(local, args...)
}
