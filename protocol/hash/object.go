// Package hash provides functionality for hashing Git objects.
//
// For more details about Git's object format, see:
// https://git-scm.com/book/en/v2/Git-Internals-Git-Objects
package hash

import (
	"crypto"
	"errors"
	"strconv"

	// Linking the algorithms Git supports into the binary.
	// Their init functions register the hash in the `crypto` package.

	// Git still uses sha1 for the most part: https://git-scm.com/docs/hash-function-transition
	//nolint:gosec
	_ "crypto/sha1"
	_ "crypto/sha256"

	"github.com/grafana/nanogit/protocol/object"
)

// ErrUnlinkedAlgorithm is returned when trying to use a hash algorithm that is not
// linked into the binary (e.g., MD5).
var ErrUnlinkedAlgorithm = errors.New("the algorithm is not linked into the binary")

// Object computes the hash of a Git object. Git objects are stored with a header followed by the content.
// The header format is: "<type> <size>\0" where:
//   - <type> is the object type (commit, tree, blob, or tag)
//   - <size> is the size of the content in bytes
//   - \0 is a null byte
//
// For example, a blob containing "test" would be stored as:
//
//	"blob 4\0test"
//
// The hash is computed over both the header and the content. This ensures that:
//  1. Objects of different types with the same content have different hashes
//  2. The size is verified when the object is read
//  3. The object type is verified when the object is read
//
// For more details about Git's object format and internals, see:
// https://git-scm.com/book/en/v2/Git-Internals-Git-Objects
//
// By default, Git uses SHA-1 for object hashes, but is transitioning to SHA-256:
// https://git-scm.com/docs/hash-function-transition
func Object(algo crypto.Hash, t object.Type, data []byte) (Hash, error) {
	h, err := NewHasher(algo, t, int64(len(data)))
	if err != nil {
		return nil, err
	}

	if _, err = h.Write(data); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// NewHasher creates a new hasher for a Git object. It writes the object header
// to the hash before returning, so the caller only needs to write the object content.
//
// The header consists of:
//  1. The object type as a string (e.g., "commit", "tree", "blob", "tag")
//  2. A space character
//  3. The size of the content as a decimal string
//  4. A null byte
//
// For example, for a blob of size 42, the header would be:
//
//	"blob 42\0"
//
// This matches Git's internal object format, ensuring hash compatibility with Git.
// For more details about Git's object format and internals, see:
// https://git-scm.com/book/en/v2/Git-Internals-Git-Objects
func NewHasher(algo crypto.Hash, t object.Type, size int64) (Hasher, error) {
	if !algo.Available() { // Avoid a panic
		return Hasher{}, ErrUnlinkedAlgorithm
	}
	h := Hasher{Hash: algo.New()}

	chunks := [][]byte{
		t.Bytes(),
		[]byte(" "),
		[]byte(strconv.FormatInt(size, 10)),
		{0},
	}

	for _, chunk := range chunks {
		if _, err := h.Write(chunk); err != nil {
			return Hasher{}, err
		}
	}

	return h, nil
}
