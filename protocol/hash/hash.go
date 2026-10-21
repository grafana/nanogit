// Package hash represents SHA-1 Git object identifiers: parsing, formatting,
// and computing the 20-byte hashes Git uses to name objects.
//
// Only SHA-1 is supported; repositories using the SHA-256 object format are
// not supported.
package hash

import (
	"encoding/hex"
	"hash"
)

// Hash is a 20-byte SHA-1 Git object identifier.
type Hash [20]byte

// Zero is the all-zero Hash. It is the zero value of Hash and denotes a
// nonexistent object, such as the old value of a ref being created or the
// new value of a ref being deleted.
var Zero Hash

// FromHex parses a 40-character hexadecimal string into a Hash. As a special
// case, the empty string yields (Zero, nil) rather than an error; any other
// length is rejected with hex.InvalidByteError.
func FromHex(hs string) (Hash, error) {
	if len(hs) == 0 {
		return Zero, nil
	}

	if len(hs) != 40 {
		return Zero, hex.InvalidByteError(len(hs))
	}

	var h Hash
	_, err := hex.Decode(h[:], []byte(hs))
	if err != nil {
		return Zero, err
	}
	return h, nil
}

// MustFromHex is like FromHex but panics if the hex string is invalid.
// It is intended for use in tests and other situations where the hex string
// is known to be valid.
func MustFromHex(hs string) Hash {
	h, err := FromHex(hs)
	if err != nil {
		panic(err)
	}
	return h
}

// String returns the lowercase 40-character hexadecimal form of h.
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// Is reports whether h equals other.
func (h Hash) Is(other Hash) bool {
	return h == other
}

// Hasher computes a Git object identifier incrementally. Construct it with
// protocol.NewHasher, which writes the object header ("<type> <size>\x00")
// before the caller writes the object content.
type Hasher struct {
	hash.Hash
}
