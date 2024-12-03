package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
)

var (
	ErrNoPackfileSignature        = errors.New("the given payload has no packfile signature")
	ErrUnsupportedPackfileVersion = errors.New("the version of the packfile payload is unsupported")
	ErrUnsupportedObjectType      = errors.New("the type of the object is unsupported")
)

// A PackfileReader is a reader for a set of compressed files (objects).
// Its wire-format is defined here: https://git-scm.com/docs/pack-format
// Its negotiation is defined here: https://git-scm.com/docs/pack-protocol#_packfile_negotiation
//
// The wire-format goes as such:
//   - 4-byte signature: `[]byte("PACK")`
//   - 4-byte version number (2 or 3; big-endian)
//   - 4-byte number of objects contained in the pack (big-endian)
//   - The pre-defined number of objects follow.
//   - A trailer of all packfile checksums.
//
// The object entries go as such:
//   - For an undeltified representation,
//     there is a n-byte type and length (3-bit type, (n-1)*7+4-bit length).
//     Finally, the compressed object data.
//   - For a deltified representation, the same byte and length is given.
//     Then, we have an object name if OBJ_REF_DELTA or a negative relative offset from the delta object's position in the pack if this is an OBJ_OFS_DELTA object.
//     Finally, the compressed delta data.
type PackfileReader struct {
	reader io.Reader
}

func (p *PackfileReader) ReadObject() (*PackedObject, error) {
	// TODO: probably smart to use a mutex here.

	var buf [1]byte
	if _, err := p.reader.Read(buf[:]); err != nil {
		return nil, err
	}

	// The first byte is a 3-bit type (stored in 4 bits).
	// The remaining 4 bits are the start of a varint containing the size.
	oty := ObjectType((buf[0] >> 4) & 0b111)
	switch oty {
	case ObjectTypeBlob, ObjectTypeCommit, ObjectTypeTag, ObjectTypeTree:
		// All good!
	default:
		return nil, fmt.Errorf("%w (%s)", ErrUnsupportedObjectType, oty)
	}

	size := int(buf[0] & 0b1111)
	shift := 4
	for buf[0]&0x80 == 0x80 {
		if _, err := p.reader.Read(buf[:]); err != nil {
			return nil, err
		}

		size += int(buf[0]&0x7f) << shift
		shift += 7
	}

	data := make([]byte, size)
	if _, err := p.reader.Read(data); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}

	return &PackedObject{Data: data, Type: oty}, nil
}

type PackedObject struct {
	// The type of the object. 3-bit field.
	Type ObjectType
	// If Type == ObjectTypeRefDelta, this is set.
	ObjectName string
	// If Type == ObjectTypeOfsDelta, this is set.
	RelativeOffset int
	// The compressed data.
	// If Type is one of ObjectTypeRefDelta and ObjectTypeOfsDelta, this is a delta.
	Data []byte
}

type ObjectType uint8

// The object types. Type 5 is reserved. 0 is invalid.
const (
	ObjectTypeInvalid  ObjectType = 0 // 0b000
	ObjectTypeCommit   ObjectType = 1 // 0b001
	ObjectTypeTree     ObjectType = 2 // 0b010
	ObjectTypeBlob     ObjectType = 3 // 0b011
	ObjectTypeTag      ObjectType = 4 // 0b100
	ObjectTypeReserved ObjectType = 5 // 0b101
	ObjectTypeOfsDelta ObjectType = 6 // 0b110
	ObjectTypeRefDelta ObjectType = 7 // 0b111
)

func (t ObjectType) String() string {
	switch t {
	case ObjectTypeInvalid:
		return "OBJ_INVALID"
	case ObjectTypeCommit:
		return "OBJ_COMMIT"
	case ObjectTypeTree:
		return "OBJ_TREE"
	case ObjectTypeBlob:
		return "OBJ_BLOB"
	case ObjectTypeTag:
		return "OBJ_TAG"
	case ObjectTypeReserved:
		return "OBJ_RESERVED"
	case ObjectTypeOfsDelta:
		return "OBJ_OFS_DELTA"
	case ObjectTypeRefDelta:
		return "OBJ_REF_DELTA"
	default:
		return fmt.Sprintf("ObjectType(%d)", uint8(t))
	}
}

func (t ObjectType) IsValid() bool {
	return t != ObjectTypeInvalid && t != ObjectTypeReserved && (t & ^ObjectType(0b111)) == 0
}

func ParsePackfile(payload []byte) (*PackfileReader, error) {
	// TODO: Accept an io.Reader to the function.
	if len(payload) < 4 || !slices.Equal(payload[:4], []byte("PACK")) {
		return nil, ErrNoPackfileSignature
	}
	payload = payload[4:] // Without "PACK"

	version := binary.BigEndian.Uint32(payload[:4])
	if version != 2 && version != 3 {
		return nil, ErrUnsupportedPackfileVersion
	}
	payload = payload[4:] // Without version

	countObjects := binary.BigEndian.Uint32(payload[:4])
	_ = countObjects      // just clear the warning, for now...
	payload = payload[4:] // Without version

	// The payload now contains just objects. These are multiple and can be quite large.
	// Let's pass it off to a caller to read the rest of what's in here.
	// Eventually, we can even accept an io.Reader directly here, such that we don't need to
	//   keep the whole original payload in memory, either.
	return &PackfileReader{reader: bytes.NewReader(payload)}, nil
}
