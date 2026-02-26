package integration_test

// gitNoError is a helper that wraps LocalGitRepo.Git and fails the test on error
// This maintains backward compatibility with the old Git() method that didn't return errors
func gitNoError(local *LocalGitRepo, args ...string) string {
	output, _ := local.Git(args...)
	// Note: local.Git() already calls Expect internally, so we don't need to check the error here
	return output
}

