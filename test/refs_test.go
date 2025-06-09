package integration_test

import (
	"errors"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
)

// TestListRefs tests listing all references
func (s *IntegrationTestSuite) TestListRefs() {
	// Get fresh repository with initial setup
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create initial commit and set up refs
	local.CreateFile(s.T(), "test.txt", "test content")
	local.Git(s.T(), "add", "test.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")
	firstCommitStr := local.Git(s.T(), "rev-parse", "HEAD")
	firstCommit, err := hash.FromHex(firstCommitStr)
	s.NoError(err)

	// Set up branches and tags
	local.Git(s.T(), "branch", "-M", "main")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")
	local.Git(s.T(), "branch", "test-branch")
	local.Git(s.T(), "push", "origin", "test-branch", "--force")
	local.Git(s.T(), "tag", "v1.0.0")
	local.Git(s.T(), "push", "origin", "v1.0.0", "--force")

	ctx, cancel := s.CreateContext(s.StandardTimeout())
	defer cancel()

	client := remote.Client(s.T())
	refs, err := client.ListRefs(ctx)
	s.NoError(err, "ListRefs failed")
	s.Len(refs, 4, "should have 4 references")

	wantRefs := []nanogit.Ref{
		{Name: "HEAD", Hash: firstCommit},
		{Name: "refs/heads/main", Hash: firstCommit},
		{Name: "refs/heads/test-branch", Hash: firstCommit},
		{Name: "refs/tags/v1.0.0", Hash: firstCommit},
	}

	s.ElementsMatch(wantRefs, refs)
}

// TestGetRef tests getting individual references
func (s *IntegrationTestSuite) TestGetRef() {
	// Get fresh repository with initial setup
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create initial commit and set up refs
	local.CreateFile(s.T(), "test.txt", "test content")
	local.Git(s.T(), "add", "test.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")
	firstCommitStr := local.Git(s.T(), "rev-parse", "HEAD")
	firstCommit, err := hash.FromHex(firstCommitStr)
	s.NoError(err)

	// Set up branches and tags
	local.Git(s.T(), "branch", "-M", "main")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")
	local.Git(s.T(), "branch", "test-branch")
	local.Git(s.T(), "push", "origin", "test-branch", "--force")
	local.Git(s.T(), "tag", "v1.0.0")
	local.Git(s.T(), "push", "origin", "v1.0.0", "--force")

	client := remote.Client(s.T())
	wantRefs := []nanogit.Ref{
		{Name: "HEAD", Hash: firstCommit},
		{Name: "refs/heads/main", Hash: firstCommit},
		{Name: "refs/heads/test-branch", Hash: firstCommit},
		{Name: "refs/tags/v1.0.0", Hash: firstCommit},
	}

	s.Run("existing refs", func() {
		s.T().Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Getting refs one by one")
		for _, wantRef := range wantRefs {
			ref, err := client.GetRef(ctx, wantRef.Name)
			s.NoError(err, "GetRef failed for %s", wantRef.Name)
			s.Equal(wantRef.Name, ref.Name)
			s.Equal(firstCommit, ref.Hash)
		}
	})

	s.Run("non-existent ref", func() {
		s.T().Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Getting ref with non-existent ref")
		_, err := client.GetRef(ctx, "refs/heads/non-existent")

		var notFoundErr *nanogit.RefNotFoundError
		s.True(errors.As(err, &notFoundErr))
		s.Equal("refs/heads/non-existent", notFoundErr.RefName)
	})
}

// TestCreateRef tests creating new references
func (s *IntegrationTestSuite) TestCreateRef() {
	// Get fresh repository with initial setup
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create initial commit
	local.CreateFile(s.T(), "test.txt", "test content")
	local.Git(s.T(), "add", "test.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")
	firstCommitStr := local.Git(s.T(), "rev-parse", "HEAD")
	firstCommit, err := hash.FromHex(firstCommitStr)
	s.NoError(err)

	local.Git(s.T(), "branch", "-M", "main")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")

	client := remote.Client(s.T())

	s.Run("create branch ref", func() {
		s.T().Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Creating ref with new-branch")
		err := client.CreateRef(ctx, nanogit.Ref{Name: "refs/heads/new-branch", Hash: firstCommit})
		s.NoError(err)

		s.Logger.Info("Getting ref with new-branch")
		ref, err := client.GetRef(ctx, "refs/heads/new-branch")
		s.NoError(err)
		s.Equal(firstCommit, ref.Hash)
	})

	s.Run("create tag ref", func() {
		s.T().Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		s.Logger.Info("Creating tag with v2.0.0")
		err := client.CreateRef(ctx, nanogit.Ref{Name: "refs/tags/v2.0.0", Hash: firstCommit})
		s.NoError(err)

		s.Logger.Info("Getting ref with new tag")
		ref, err := client.GetRef(ctx, "refs/tags/v2.0.0")
		s.NoError(err)
		s.Equal(firstCommit, ref.Hash)
	})
}

// TestUpdateRef tests updating existing references
func (s *IntegrationTestSuite) TestUpdateRef() {
	// Get fresh repository with initial setup
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create initial commit
	local.CreateFile(s.T(), "test.txt", "test content")
	local.Git(s.T(), "add", "test.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")
	firstCommitStr := local.Git(s.T(), "rev-parse", "HEAD")
	firstCommit, err := hash.FromHex(firstCommitStr)
	s.NoError(err)

	local.Git(s.T(), "branch", "-M", "main")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")

	ctx, cancel := s.CreateContext(s.StandardTimeout())
	defer cancel()

	client := remote.Client(s.T())

	// First create a ref to update
	s.Logger.Info("Creating ref for update test")
	err = client.CreateRef(ctx, nanogit.Ref{Name: "refs/heads/update-test", Hash: firstCommit})
	s.NoError(err)

	// Create a new commit to update to
	s.Logger.Info("Creating a new commit")
	local.Git(s.T(), "commit", "--allow-empty", "-m", "new commit")
	newHashStr := local.Git(s.T(), "rev-parse", "HEAD")
	newHash, err := hash.FromHex(newHashStr)
	s.NoError(err)
	local.Git(s.T(), "push", "origin", "main", "--force")

	// Update the ref
	s.Logger.Info("Updating ref to point to new commit")
	err = client.UpdateRef(ctx, nanogit.Ref{Name: "refs/heads/update-test", Hash: newHash})
	s.NoError(err)

	// Verify the update
	s.Logger.Info("Getting ref and verifying it points to new commit")
	ref, err := client.GetRef(ctx, "refs/heads/update-test")
	s.NoError(err)
	s.Equal(newHash, ref.Hash)
}

// TestDeleteRef tests deleting references
func (s *IntegrationTestSuite) TestDeleteRef() {
	// Get fresh repository with initial setup
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create initial commit
	local.CreateFile(s.T(), "test.txt", "test content")
	local.Git(s.T(), "add", "test.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")
	firstCommitStr := local.Git(s.T(), "rev-parse", "HEAD")
	firstCommit, err := hash.FromHex(firstCommitStr)
	s.NoError(err)

	local.Git(s.T(), "branch", "-M", "main")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")

	client := remote.Client(s.T())

	s.Run("delete branch ref", func() {
		s.T().Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		// Create a ref to delete
		s.Logger.Info("Creating ref for delete test")
		err := client.CreateRef(ctx, nanogit.Ref{Name: "refs/heads/delete-test", Hash: firstCommit})
		s.NoError(err)

		// Delete the ref
		s.Logger.Info("Deleting ref")
		err = client.DeleteRef(ctx, "refs/heads/delete-test")
		s.NoError(err)

		// Verify it's gone
		s.Logger.Info("Verifying ref is deleted")
		_, err = client.GetRef(ctx, "refs/heads/delete-test")
		var notFoundErr *nanogit.RefNotFoundError
		s.True(errors.As(err, &notFoundErr))
		s.Equal("refs/heads/delete-test", notFoundErr.RefName)
	})

	s.Run("delete tag ref", func() {
		s.T().Parallel()

		ctx, cancel := s.CreateContext(s.StandardTimeout())
		defer cancel()

		// Create a tag to delete
		s.Logger.Info("Creating tag for delete test")
		err := client.CreateRef(ctx, nanogit.Ref{Name: "refs/tags/delete-test", Hash: firstCommit})
		s.NoError(err)

		// Delete the tag
		s.Logger.Info("Deleting tag")
		err = client.DeleteRef(ctx, "refs/tags/delete-test")
		s.NoError(err)

		// Verify it's gone
		s.Logger.Info("Verifying tag is deleted")
		_, err = client.GetRef(ctx, "refs/tags/delete-test")
		var notFoundErr *nanogit.RefNotFoundError
		s.True(errors.As(err, &notFoundErr))
		s.Equal("refs/tags/delete-test", notFoundErr.RefName)
	})
}

// TestRefsIntegrationFlow tests a complete workflow of ref operations
func (s *IntegrationTestSuite) TestRefsIntegrationFlow() {
	// Get fresh repository with initial setup
	remote, _ := s.CreateTestRepo()
	local := remote.Local(s.T())

	// Create initial commit
	local.CreateFile(s.T(), "test.txt", "test content")
	local.Git(s.T(), "add", "test.txt")
	local.Git(s.T(), "commit", "-m", "Initial commit")
	firstCommitStr := local.Git(s.T(), "rev-parse", "HEAD")
	firstCommit, err := hash.FromHex(firstCommitStr)
	s.NoError(err)

	local.Git(s.T(), "branch", "-M", "main")
	local.Git(s.T(), "push", "-u", "origin", "main", "--force")

	ctx, cancel := s.CreateContext(s.StandardTimeout())
	defer cancel()

	client := remote.Client(s.T())

	// This test validates the complete flow without parallel sub-tests
	// since it needs sequential operations

	refName := "refs/heads/integration-flow"

	// 1. Create ref
	s.Logger.Info("Creating ref for integration flow")
	err = client.CreateRef(ctx, nanogit.Ref{Name: refName, Hash: firstCommit})
	s.NoError(err)

	// 2. Get ref
	s.Logger.Info("Getting created ref")
	ref, err := client.GetRef(ctx, refName)
	s.NoError(err)
	s.Equal(firstCommit, ref.Hash)

	// 3. Create new commit and update ref
	s.Logger.Info("Creating new commit for update")
	local.Git(s.T(), "commit", "--allow-empty", "-m", "integration flow commit")
	newHashStr := local.Git(s.T(), "rev-parse", "HEAD")
	newHash, err := hash.FromHex(newHashStr)
	s.NoError(err)
	local.Git(s.T(), "push", "origin", "main", "--force")

	s.Logger.Info("Updating ref to new commit")
	err = client.UpdateRef(ctx, nanogit.Ref{Name: refName, Hash: newHash})
	s.NoError(err)

	// 4. Verify update
	s.Logger.Info("Verifying ref update")
	ref, err = client.GetRef(ctx, refName)
	s.NoError(err)
	s.Equal(newHash, ref.Hash)

	// 5. Delete ref
	s.Logger.Info("Deleting ref")
	err = client.DeleteRef(ctx, refName)
	s.NoError(err)

	// 6. Verify deletion
	s.Logger.Info("Verifying ref deletion")
	_, err = client.GetRef(ctx, refName)
	var notFoundErr *nanogit.RefNotFoundError
	s.True(errors.As(err, &notFoundErr))
	s.Equal(refName, notFoundErr.RefName)
}
