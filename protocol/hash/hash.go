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

func (h Hash) String() string {
	return hex.EncodeToString(h)
}

func (h Hash) Is(other Hash) bool {
	return slices.Equal(h, other)
}

type Hasher struct {
	hash.Hash
}
