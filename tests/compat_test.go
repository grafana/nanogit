package integration_test

import "github.com/grafana/nanogit/testutil"

// Type aliases for backward compatibility with existing tests
// These allow existing test code to continue working without modification
// while using the new testutil package under the hood

type (
	// User is an alias for testutil.User
	User = testutil.User

	// LocalGitRepo is an alias for testutil.LocalRepo
	LocalGitRepo = testutil.LocalRepo
)

// RemoteRepo wraps testutil.Repo to add backward-compatible URL() method
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
