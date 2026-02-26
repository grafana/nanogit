package integration_test

import "github.com/grafana/nanogit/testutil"

// Type aliases for backward compatibility with existing tests
// These allow existing test code to continue working without modification
// while using the new testutil package under the hood

type (
	// User is an alias for testutil.User
	User = testutil.User

	// RemoteRepo is an alias for testutil.Repo
	RemoteRepo = testutil.Repo

	// LocalGitRepo is an alias for testutil.LocalRepo
	LocalGitRepo = testutil.LocalRepo
)
