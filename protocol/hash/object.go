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

var ErrUnlinkedAlgorithm = errors.New("the algorithm is not linked into the binary")

func Object(algo crypto.Hash, t object.Type, data []byte) (Hash, error) {
	h, err := NewHasher(algo, t, int64(len(data)))
	if err != nil {
		return nil, err
	}
	h.Write(data)
	return h.Sum(nil), nil
}

func NewHasher(algo crypto.Hash, t object.Type, size int64) (Hasher, error) {
	if !algo.Available() { // Avoid a panic
		return Hasher{}, ErrUnlinkedAlgorithm
	}
	h := Hasher{Hash: algo.New()}
	h.Write(t.Bytes())
	h.Write([]byte(" "))
	h.Write([]byte(strconv.FormatInt(size, 10)))
	h.Write([]byte{0})
	return h, nil
}
