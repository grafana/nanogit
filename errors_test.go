package nanogit

import (
	"errors"
	"testing"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/require"
)

func TestGlobalErrors(t *testing.T) {
	t.Parallel()

	// Test that all global error variables are not nil and have proper messages
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrObjectNotFound", ErrObjectNotFound, "object not found"},
		{"ErrObjectAlreadyExists", ErrObjectAlreadyExists, "object already exists"},
		{"ErrUnexpectedObjectType", ErrUnexpectedObjectType, "unexpected object type"},
		{"ErrNothingToPush", ErrNothingToPush, "nothing to push"},
		{"ErrNothingToCommit", ErrNothingToCommit, "nothing to commit"},
		{"ErrUnexpectedObjectCount", ErrUnexpectedObjectCount, "unexpected object count"},
		{"ErrEmptyCommitMessage", ErrEmptyCommitMessage, "empty commit message"},
		{"ErrEmptyPath", ErrEmptyPath, "empty path"},
		{"ErrEmptyRefName", ErrEmptyRefName, "empty ref name"},
		{"ErrInvalidAuthor", ErrInvalidAuthor, "invalid author information"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tt.err)
			require.Equal(t, tt.msg, tt.err.Error())
		})
	}
}

func TestObjectNotFoundError(t *testing.T) {
	t.Parallel()

	testHash, err := hash.FromHex("1234567890123456789012345678901234567890")
	require.NoError(t, err)

	t.Run("constructor", func(t *testing.T) {
		t.Parallel()
		err := NewObjectNotFoundError(testHash)
		require.NotNil(t, err)
		require.Equal(t, testHash, err.ObjectID)
	})

	t.Run("error message", func(t *testing.T) {
		t.Parallel()
		err := NewObjectNotFoundError(testHash)
		expected := "object 1234567890123456789012345678901234567890 not found"
		require.Equal(t, expected, err.Error())
	})

	t.Run("unwrap", func(t *testing.T) {
		t.Parallel()
		err := NewObjectNotFoundError(testHash)
		require.Equal(t, ErrObjectNotFound, err.Unwrap())
	})

	t.Run("errors.Is compatibility", func(t *testing.T) {
		t.Parallel()
		err := NewObjectNotFoundError(testHash)
		require.ErrorIs(t, err, ErrObjectNotFound)
		require.NotErrorIs(t, err, ErrObjectAlreadyExists)
	})

	t.Run("errors.As compatibility", func(t *testing.T) {
		t.Parallel()
		originalErr := NewObjectNotFoundError(testHash)
		wrappedErr := errors.New("wrapped: " + originalErr.Error())

		var targetErr *ObjectNotFoundError
		// This should NOT work since we're not using %w verb
		require.False(t, errors.As(wrappedErr, &targetErr))

		// But this should work with direct error
		require.ErrorAs(t, originalErr, &targetErr)
		require.Equal(t, testHash, targetErr.ObjectID)
	})
}

func TestObjectAlreadyExistsError(t *testing.T) {
	t.Parallel()

	testHash, err := hash.FromHex("abcdef1234567890123456789012345678901234")
	require.NoError(t, err)

	t.Run("constructor", func(t *testing.T) {
		t.Parallel()
		err := NewObjectAlreadyExistsError(testHash)
		require.NotNil(t, err)
		require.Equal(t, testHash, err.ObjectID)
	})

	t.Run("error message", func(t *testing.T) {
		t.Parallel()
		err := NewObjectAlreadyExistsError(testHash)
		expected := "object abcdef1234567890123456789012345678901234 already exists"
		require.Equal(t, expected, err.Error())
	})

	t.Run("unwrap", func(t *testing.T) {
		t.Parallel()
		err := NewObjectAlreadyExistsError(testHash)
		require.Equal(t, ErrObjectAlreadyExists, err.Unwrap())
	})

	t.Run("errors.Is compatibility", func(t *testing.T) {
		t.Parallel()
		err := NewObjectAlreadyExistsError(testHash)
		require.ErrorIs(t, err, ErrObjectAlreadyExists)
		require.NotErrorIs(t, err, ErrObjectNotFound)
	})
}

func TestUnexpectedObjectCountError(t *testing.T) {
	t.Parallel()

	testHash1, err := hash.FromHex("1111111111111111111111111111111111111111")
	require.NoError(t, err)
	testHash2, err := hash.FromHex("2222222222222222222222222222222222222222")
	require.NoError(t, err)

	objects := []*protocol.PackfileObject{
		{Hash: testHash1, Type: protocol.ObjectTypeBlob},
		{Hash: testHash2, Type: protocol.ObjectTypeTree},
	}

	t.Run("constructor", func(t *testing.T) {
		t.Parallel()
		err := NewUnexpectedObjectCountError(1, objects)
		require.NotNil(t, err)
		require.Equal(t, 1, err.ExpectedCount)
		require.Equal(t, 2, err.ActualCount)
		require.Equal(t, objects, err.Objects)
	})

	t.Run("error message", func(t *testing.T) {
		t.Parallel()
		err := NewUnexpectedObjectCountError(1, objects)
		expected := "unexpected object count: expected 1 but got 2 objects: OBJ_BLOB/1111111111111111111111111111111111111111, OBJ_TREE/2222222222222222222222222222222222222222"
		require.Equal(t, expected, err.Error())
	})

	t.Run("error message with empty objects", func(t *testing.T) {
		t.Parallel()
		err := NewUnexpectedObjectCountError(5, []*protocol.PackfileObject{})
		expected := "unexpected object count: expected 5 but got 0 objects: "
		require.Equal(t, expected, err.Error())
	})

	t.Run("unwrap", func(t *testing.T) {
		t.Parallel()
		err := NewUnexpectedObjectCountError(1, objects)
		require.Equal(t, ErrUnexpectedObjectCount, err.Unwrap())
	})

	t.Run("errors.Is compatibility", func(t *testing.T) {
		t.Parallel()
		err := NewUnexpectedObjectCountError(1, objects)
		require.ErrorIs(t, err, ErrUnexpectedObjectCount)
		require.NotErrorIs(t, err, ErrObjectNotFound)
	})
}

func TestUnexpectedObjectTypeError(t *testing.T) {
	t.Parallel()

	testHash, err := hash.FromHex("fedcba0987654321098765432109876543210987")
	require.NoError(t, err)

	t.Run("constructor", func(t *testing.T) {
		t.Parallel()
		err := NewUnexpectedObjectTypeError(testHash, protocol.ObjectTypeTree, protocol.ObjectTypeBlob)
		require.NotNil(t, err)
		require.Equal(t, testHash, err.ObjectID)
		require.Equal(t, protocol.ObjectTypeTree, err.ExpectedType)
		require.Equal(t, protocol.ObjectTypeBlob, err.ActualType)
	})

	t.Run("error message", func(t *testing.T) {
		t.Parallel()
		err := NewUnexpectedObjectTypeError(testHash, protocol.ObjectTypeTree, protocol.ObjectTypeBlob)
		expected := "object fedcba0987654321098765432109876543210987 has unexpected type OBJ_BLOB (expected OBJ_TREE)"
		require.Equal(t, expected, err.Error())
	})

	t.Run("unwrap", func(t *testing.T) {
		t.Parallel()
		err := NewUnexpectedObjectTypeError(testHash, protocol.ObjectTypeTree, protocol.ObjectTypeBlob)
		require.Equal(t, ErrUnexpectedObjectType, err.Unwrap())
	})

	t.Run("errors.Is compatibility", func(t *testing.T) {
		t.Parallel()
		err := NewUnexpectedObjectTypeError(testHash, protocol.ObjectTypeTree, protocol.ObjectTypeBlob)
		require.ErrorIs(t, err, ErrUnexpectedObjectType)
		require.NotErrorIs(t, err, ErrObjectNotFound)
	})
}

func TestPathNotFoundError(t *testing.T) {
	t.Parallel()

	testPath := "some/path/to/file"

	t.Run("constructor", func(t *testing.T) {
		t.Parallel()
		err := NewPathNotFoundError(testPath)
		require.NotNil(t, err)
		require.Equal(t, testPath, err.Path)
	})

	t.Run("error message", func(t *testing.T) {
		t.Parallel()
		err := NewPathNotFoundError(testPath)
		expected := "path not found: some/path/to/file"
		require.Equal(t, expected, err.Error())
	})

	t.Run("unwrap", func(t *testing.T) {
		t.Parallel()
		err := NewPathNotFoundError(testPath)
		require.Equal(t, ErrObjectNotFound, err.Unwrap())
	})

	t.Run("errors.Is compatibility", func(t *testing.T) {
		t.Parallel()
		err := NewPathNotFoundError(testPath)
		require.ErrorIs(t, err, ErrObjectNotFound)
		require.NotErrorIs(t, err, ErrObjectAlreadyExists)
	})
}

func TestRefNotFoundError(t *testing.T) {
	t.Parallel()

	refName := "refs/heads/feature-branch"

	t.Run("constructor", func(t *testing.T) {
		t.Parallel()
		err := NewRefNotFoundError(refName)
		require.NotNil(t, err)
		require.Equal(t, refName, err.RefName)
	})

	t.Run("error message", func(t *testing.T) {
		t.Parallel()
		err := NewRefNotFoundError(refName)
		expected := "reference not found: refs/heads/feature-branch"
		require.Equal(t, expected, err.Error())
	})

	t.Run("unwrap", func(t *testing.T) {
		t.Parallel()
		err := NewRefNotFoundError(refName)
		require.Equal(t, ErrObjectNotFound, err.Unwrap())
	})

	t.Run("errors.Is compatibility", func(t *testing.T) {
		t.Parallel()
		err := NewRefNotFoundError(refName)
		require.ErrorIs(t, err, ErrObjectNotFound)
		require.NotErrorIs(t, err, ErrObjectAlreadyExists)
	})
}

func TestRefAlreadyExistsError(t *testing.T) {
	t.Parallel()

	refName := "refs/heads/existing-branch"

	t.Run("constructor", func(t *testing.T) {
		t.Parallel()
		err := NewRefAlreadyExistsError(refName)
		require.NotNil(t, err)
		require.Equal(t, refName, err.RefName)
	})

	t.Run("error message", func(t *testing.T) {
		t.Parallel()
		err := NewRefAlreadyExistsError(refName)
		expected := "reference already exists: refs/heads/existing-branch"
		require.Equal(t, expected, err.Error())
	})

	t.Run("unwrap", func(t *testing.T) {
		t.Parallel()
		err := NewRefAlreadyExistsError(refName)
		require.Equal(t, ErrObjectAlreadyExists, err.Unwrap())
	})

	t.Run("errors.Is compatibility", func(t *testing.T) {
		t.Parallel()
		err := NewRefAlreadyExistsError(refName)
		require.ErrorIs(t, err, ErrObjectAlreadyExists)
		require.NotErrorIs(t, err, ErrObjectNotFound)
	})
}

func TestAuthorError(t *testing.T) {
	t.Parallel()

	field := "email"
	reason := "invalid format"

	t.Run("constructor", func(t *testing.T) {
		t.Parallel()
		err := NewAuthorError(field, reason)
		require.NotNil(t, err)
		require.Equal(t, field, err.Field)
		require.Equal(t, reason, err.Reason)
	})

	t.Run("error message", func(t *testing.T) {
		t.Parallel()
		err := NewAuthorError(field, reason)
		expected := "invalid author email: invalid format"
		require.Equal(t, expected, err.Error())
	})

	t.Run("unwrap", func(t *testing.T) {
		t.Parallel()
		err := NewAuthorError(field, reason)
		require.Equal(t, ErrInvalidAuthor, err.Unwrap())
	})

	t.Run("errors.Is compatibility", func(t *testing.T) {
		t.Parallel()
		err := NewAuthorError(field, reason)
		require.ErrorIs(t, err, ErrInvalidAuthor)
		require.NotErrorIs(t, err, ErrObjectNotFound)
	})
}

func TestErrorsChaining(t *testing.T) {
	t.Parallel()

	// Test error chaining and wrapping with fmt.Errorf and %w verb
	t.Run("wrapped errors maintain Is compatibility", func(t *testing.T) {
		t.Parallel()
		testHash, err := hash.FromHex("1234567890123456789012345678901234567890")
		require.NoError(t, err)

		originalErr := NewObjectNotFoundError(testHash)
		wrappedErr := errors.New("operation failed: " + originalErr.Error())

		// Direct error should match
		require.ErrorIs(t, originalErr, ErrObjectNotFound)

		// Wrapped without %w should NOT match with errors.Is
		require.NotErrorIs(t, wrappedErr, ErrObjectNotFound)
	})

	t.Run("multiple error types can be distinguished", func(t *testing.T) {
		t.Parallel()
		testHash, err := hash.FromHex("1234567890123456789012345678901234567890")
		require.NoError(t, err)

		notFoundErr := NewObjectNotFoundError(testHash)
		alreadyExistsErr := NewObjectAlreadyExistsError(testHash)
		typeErr := NewUnexpectedObjectTypeError(testHash, protocol.ObjectTypeTree, protocol.ObjectTypeBlob)

		// Each error should only match its own base error
		require.ErrorIs(t, notFoundErr, ErrObjectNotFound)
		require.NotErrorIs(t, notFoundErr, ErrObjectAlreadyExists)
		require.NotErrorIs(t, notFoundErr, ErrUnexpectedObjectType)

		require.ErrorIs(t, alreadyExistsErr, ErrObjectAlreadyExists)
		require.NotErrorIs(t, alreadyExistsErr, ErrObjectNotFound)
		require.NotErrorIs(t, alreadyExistsErr, ErrUnexpectedObjectType)

		require.ErrorIs(t, typeErr, ErrUnexpectedObjectType)
		require.NotErrorIs(t, typeErr, ErrObjectNotFound)
		require.NotErrorIs(t, typeErr, ErrObjectAlreadyExists)
	})
}

func TestSpecialCases(t *testing.T) {
	t.Parallel()

	t.Run("zero hash handling", func(t *testing.T) {
		t.Parallel()
		zeroHash, err := hash.FromHex("0000000000000000000000000000000000000000")
		require.NoError(t, err)
		err = NewObjectNotFoundError(zeroHash)
		expected := "object 0000000000000000000000000000000000000000 not found"
		require.Equal(t, expected, err.Error())
	})

	t.Run("nil hash handling", func(t *testing.T) {
		t.Parallel()
		err := NewObjectNotFoundError(hash.Zero)
		expected := "object 0000000000000000000000000000000000000000 not found"
		require.Equal(t, expected, err.Error())
	})

	t.Run("empty strings handling", func(t *testing.T) {
		t.Parallel()
		pathErr := NewPathNotFoundError("")
		require.Equal(t, "path not found: ", pathErr.Error())

		refErr := NewRefNotFoundError("")
		require.Equal(t, "reference not found: ", refErr.Error())

		authorErr := NewAuthorError("", "")
		require.Equal(t, "invalid author : ", authorErr.Error())
	})

	t.Run("object types string representation", func(t *testing.T) {
		t.Parallel()
		testHash, err := hash.FromHex("1234567890123456789012345678901234567890")
		require.NoError(t, err)

		// Test different object type combinations
		types := []struct {
			expected protocol.ObjectType
			actual   protocol.ObjectType
		}{
			{protocol.ObjectTypeCommit, protocol.ObjectTypeTree},
			{protocol.ObjectTypeTree, protocol.ObjectTypeBlob},
			{protocol.ObjectTypeBlob, protocol.ObjectTypeCommit},
			{protocol.ObjectTypeTag, protocol.ObjectTypeCommit},
		}

		for _, tt := range types {
			err := NewUnexpectedObjectTypeError(testHash, tt.expected, tt.actual)
			require.Contains(t, err.Error(), tt.expected.String())
			require.Contains(t, err.Error(), tt.actual.String())
		}
	})
}
