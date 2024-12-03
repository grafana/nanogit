package protocol

// A Packfile is a compressed, communicable file.
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
type Packfile struct {
}

type PackedObject struct {
	// The type of the object. 3-bit field.
	Type ObjectType
	// Is this a deltified representation?
	Deltified bool
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

func (t ObjectType) IsValid() bool {
	return t != ObjectTypeInvalid && t != ObjectTypeReserved && (t & ^ObjectType(0b111)) == 0
}

func ParsePackfile(payload []byte) (*Packfile, error) {
	return nil, nil
}
