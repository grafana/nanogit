package mocks_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/mocks"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Example test showing how to use the Client mock
func TestServiceWithMockedClient(t *testing.T) {
	// Create a mock client
	mockClient := &mocks.FakeClient{}

	// Set up expectations
	testHash, _ := hash.FromHex("abcdef1234567890abcdef1234567890abcdef12")
	expectedRef := nanogit.Ref{
		Name: "refs/heads/main",
		Hash: testHash,
	}

	mockClient.GetRefReturns(expectedRef, nil)
	mockClient.IsAuthorizedReturns(true, nil)

	// Test your service that uses the client
	service := &MyService{client: mockClient}

	ctx := context.Background()
	result, err := service.GetMainBranch(ctx)

	// Verify the results
	require.NoError(t, err)
	assert.Equal(t, expectedRef, result)

	// Verify the mock was called as expected
	assert.Equal(t, 1, mockClient.GetRefCallCount())
	assert.Equal(t, 1, mockClient.IsAuthorizedCallCount())

	// Verify the arguments passed to the mock
	_, refName := mockClient.GetRefArgsForCall(0)
	assert.Equal(t, "refs/heads/main", refName)
}

// Example test showing how to use the StagedWriter mock
func TestServiceWithMockedStagedWriter(t *testing.T) {
	// Create mock client and writer
	mockClient := &mocks.FakeClient{}
	mockWriter := &mocks.FakeStagedWriter{}

	// Set up the client to return our mock writer
	mockClient.NewStagedWriterReturns(mockWriter, nil)

	// Set up writer expectations
	expectedBlobHash, _ := hash.FromHex("1234567890abcdef1234567890abcdef12345678")
	commitHash, _ := hash.FromHex("fedcba0987654321fedcba0987654321fedcba09")
	expectedCommit := &nanogit.Commit{
		Hash: commitHash,
		Author: nanogit.Author{
			Name:  "Test Author",
			Email: "test@example.com",
			Time:  time.Now(),
		},
		Message: "Test commit",
	}

	mockWriter.CreateBlobReturns(expectedBlobHash, nil)
	mockWriter.CommitReturns(expectedCommit, nil)
	mockWriter.PushReturns(nil)

	// Test your service
	service := &FileService{client: mockClient}

	ctx := context.Background()
	ref := nanogit.Ref{Name: "refs/heads/main"}
	content := []byte("test content")

	commit, err := service.CreateFileAndCommit(ctx, ref, "test.txt", content, "Add test file")

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, expectedCommit, commit)

	// Verify mock calls
	assert.Equal(t, 1, mockClient.NewStagedWriterCallCount())
	assert.Equal(t, 1, mockWriter.CreateBlobCallCount())
	assert.Equal(t, 1, mockWriter.CommitCallCount())
	assert.Equal(t, 1, mockWriter.PushCallCount())

	// Verify arguments
	_, path, actualContent := mockWriter.CreateBlobArgsForCall(0)
	assert.Equal(t, "test.txt", path)
	assert.Equal(t, content, actualContent)
}

// Example test showing error handling with mocks
func TestServiceErrorHandling(t *testing.T) {
	mockClient := &mocks.FakeClient{}

	// Simulate an authorization error
	mockClient.IsAuthorizedReturns(false, assert.AnError)

	service := &MyService{client: mockClient}

	ctx := context.Background()
	_, err := service.GetMainBranch(ctx)

	// Verify error handling
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authorization failed")
}

// Example service that uses nanogit.Client
type MyService struct {
	client nanogit.Client
}

func (s *MyService) GetMainBranch(ctx context.Context) (nanogit.Ref, error) {
	// Check authorization first
	authorized, err := s.client.IsAuthorized(ctx)
	if err != nil {
		return nanogit.Ref{}, fmt.Errorf("authorization failed: %w", err)
	}
	if !authorized {
		return nanogit.Ref{}, errors.New("authorization failed: not authorized")
	}

	// Get the main branch
	return s.client.GetRef(ctx, "refs/heads/main")
}

// Example service that uses StagedWriter
type FileService struct {
	client nanogit.Client
}

func (s *FileService) CreateFileAndCommit(ctx context.Context, ref nanogit.Ref, path string, content []byte, message string) (*nanogit.Commit, error) {
	// Create a writer
	writer, err := s.client.NewStagedWriter(ctx, ref)
	if err != nil {
		return nil, err
	}

	// Create the file
	_, err = writer.CreateBlob(ctx, path, content)
	if err != nil {
		return nil, err
	}

	// Commit the changes
	author := nanogit.Author{
		Name:  "Automated Service",
		Email: "service@example.com",
		Time:  time.Now(),
	}
	committer := nanogit.Committer{
		Name:  "Automated Service",
		Email: "service@example.com",
		Time:  time.Now(),
	}

	commit, err := writer.Commit(ctx, message, author, committer)
	if err != nil {
		return nil, err
	}

	// Push the changes
	err = writer.Push(ctx)
	if err != nil {
		return nil, err
	}

	return commit, nil
}
