package protocol

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDelta(t *testing.T) {
	parent := "test" // doesn't matter
	payload := []byte{
		// ExpectedSourceLength, varint
		4, // 4 bytes. Doesn't set byte 0x80, so this is just the 7 bits of data.
		// deltaSize
		8, // TODO: Set correct size
		// Actual delta
		0x80 | // cmd. Copy from source.
			// we have no offset: copy from position 0.
			1<<4, // we have a size.
		4, // size1: copy 4 bytes from source.
		// deltaSize should now be 4 bytes smaller.
		0x00 | // cmd. Add data instruction
			3, // size: we have 3 bytes of data
		0x12, 0x34, 0x45, 0x80, // 4 bytes of data
	}
	_, err := parseDelta(parent, payload)
	assert.NoError(t, err)
}
