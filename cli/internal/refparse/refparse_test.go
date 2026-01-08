package refparse

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/mocks"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRefOrHash_WithCommitHash(t *testing.T) {
	ctx := context.Background()
	validHash := "0123456789abcdef0123456789abcdef01234567"
	parsedHash, err := hash.FromHex(validHash)
	require.NoError(t, err)

	tests := []struct {
		name        string
		input       string
		setupMock   func(*mocks.FakeClient)
		expectHash  hash.Hash
		expectError bool
	}{
		{
			name:  "valid commit hash",
			input: validHash,
			setupMock: func(client *mocks.FakeClient) {
				client.GetCommitReturns(&nanogit.Commit{
					Hash: parsedHash,
				}, nil)
			},
			expectHash:  parsedHash,
			expectError: false,
		},
		{
			name:  "invalid commit hash - not found",
			input: validHash,
			setupMock: func(client *mocks.FakeClient) {
				client.GetCommitReturns(nil, errors.New("commit not found"))
			},
			expectError: true,
		},
		{
			name:  "invalid format - too short (treated as ref)",
			input: "abc123",
			setupMock: func(client *mocks.FakeClient) {
				// Will try as ref, all attempts should fail
				client.GetRefReturns(nanogit.Ref{}, errors.New("not found"))
			},
			expectError: true,
		},
		{
			name:  "invalid format - non-hex characters (treated as ref)",
			input: "ghijklmnopqrstuvwxyz01234567890123456789",
			setupMock: func(client *mocks.FakeClient) {
				// Will try as ref, all attempts should fail
				client.GetRefReturns(nanogit.Ref{}, errors.New("not found"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mocks.FakeClient{}
			tt.setupMock(client)

			result, err := ResolveRefOrHash(ctx, client, tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectHash, result)
			}
		})
	}
}

func TestResolveRefOrHash_WithFullRef(t *testing.T) {
	ctx := context.Background()
	expectedHash, err := hash.FromHex("0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)

	tests := []struct {
		name        string
		input       string
		setupMock   func(*mocks.FakeClient)
		expectHash  hash.Hash
		expectError bool
	}{
		{
			name:  "full branch reference",
			input: "refs/heads/main",
			setupMock: func(client *mocks.FakeClient) {
				client.GetRefReturns(nanogit.Ref{
					Name: "refs/heads/main",
					Hash: expectedHash,
				}, nil)
			},
			expectHash:  expectedHash,
			expectError: false,
		},
		{
			name:  "full tag reference",
			input: "refs/tags/v1.0.0",
			setupMock: func(client *mocks.FakeClient) {
				client.GetRefReturns(nanogit.Ref{
					Name: "refs/tags/v1.0.0",
					Hash: expectedHash,
				}, nil)
			},
			expectHash:  expectedHash,
			expectError: false,
		},
		{
			name:  "full reference not found",
			input: "refs/heads/nonexistent",
			setupMock: func(client *mocks.FakeClient) {
				client.GetRefReturns(nanogit.Ref{}, errors.New("reference not found"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mocks.FakeClient{}
			tt.setupMock(client)

			result, err := ResolveRefOrHash(ctx, client, tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectHash, result)
			}
		})
	}
}

func TestResolveRefOrHash_WithShortRef(t *testing.T) {
	ctx := context.Background()
	expectedHash, err := hash.FromHex("0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)

	tests := []struct {
		name        string
		input       string
		setupMock   func(*mocks.FakeClient)
		expectHash  hash.Hash
		expectError bool
		description string
	}{
		{
			name:  "short branch name - found in refs/heads",
			input: "main",
			setupMock: func(client *mocks.FakeClient) {
				// First try: refs/heads/main - success
				client.GetRefReturnsOnCall(0, nanogit.Ref{
					Name: "refs/heads/main",
					Hash: expectedHash,
				}, nil)
			},
			expectHash:  expectedHash,
			expectError: false,
			description: "Should find branch in refs/heads/",
		},
		{
			name:  "short tag name - found in refs/tags",
			input: "v1.0.0",
			setupMock: func(client *mocks.FakeClient) {
				// First try: refs/heads/v1.0.0 - fail
				client.GetRefReturnsOnCall(0, nanogit.Ref{}, errors.New("not found"))
				// Second try: refs/tags/v1.0.0 - success
				client.GetRefReturnsOnCall(1, nanogit.Ref{
					Name: "refs/tags/v1.0.0",
					Hash: expectedHash,
				}, nil)
			},
			expectHash:  expectedHash,
			expectError: false,
			description: "Should find tag in refs/tags/",
		},
		{
			name:  "short name - found as exact ref",
			input: "HEAD",
			setupMock: func(client *mocks.FakeClient) {
				// First try: refs/heads/HEAD - fail
				client.GetRefReturnsOnCall(0, nanogit.Ref{}, errors.New("not found"))
				// Second try: refs/tags/HEAD - fail
				client.GetRefReturnsOnCall(1, nanogit.Ref{}, errors.New("not found"))
				// Third try: HEAD exact - success
				client.GetRefReturnsOnCall(2, nanogit.Ref{
					Name: "HEAD",
					Hash: expectedHash,
				}, nil)
			},
			expectHash:  expectedHash,
			expectError: false,
			description: "Should find exact ref name as fallback",
		},
		{
			name:  "short name - not found anywhere",
			input: "nonexistent",
			setupMock: func(client *mocks.FakeClient) {
				// All attempts fail
				client.GetRefReturns(nanogit.Ref{}, errors.New("not found"))
			},
			expectError: true,
			description: "Should fail when ref not found in any location",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mocks.FakeClient{}
			tt.setupMock(client)

			result, err := ResolveRefOrHash(ctx, client, tt.input)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)
				assert.Equal(t, tt.expectHash, result, tt.description)
			}
		})
	}
}

func TestResolveRefOrHash_EdgeCases(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "whitespace only",
			input:       "   ",
			expectError: true,
		},
		{
			name:        "special characters",
			input:       "refs/heads/feature/my-branch",
			expectError: true, // Will fail because mock returns error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mocks.FakeClient{}
			client.GetRefReturns(nanogit.Ref{}, errors.New("not found"))

			_, err := ResolveRefOrHash(ctx, client, tt.input)

			if tt.expectError {
				assert.Error(t, err)
			}
		})
	}
}
