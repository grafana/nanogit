// A Git-oriented hashing implementation. Supports all hashing algorithms that Git does.
package hash

import (
	"encoding/hex"
	"hash"
	"slices"
)

type Hash []byte

var Zero Hash

func FromHex(hs string) (Hash, error) {
	if len(hs) == 0 {
		return Zero, nil
	}

	b, err := hex.DecodeString(hs)
	if err != nil {
		return Zero, err
	}
	return Hash(b), err
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

func (h Hash) String() string {
	return hex.EncodeToString(h)
}

func (h Hash) Is(other Hash) bool {
	return slices.Equal(h, other)
}

type Hasher struct {
	hash.Hash
}
