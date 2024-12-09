package object

import "fmt"

type Type uint8

// The object types. Type 5 is reserved. 0 is invalid.
const (
	TypeInvalid  Type = 0 // 0b000
	TypeCommit   Type = 1 // 0b001
	TypeTree     Type = 2 // 0b010
	TypeBlob     Type = 3 // 0b011
	TypeTag      Type = 4 // 0b100
	TypeReserved Type = 5 // 0b101
	TypeOfsDelta Type = 6 // 0b110
	TypeRefDelta Type = 7 // 0b111
)

func (t Type) String() string {
	switch t {
	case TypeInvalid:
		return "OBJ_INVALID"
	case TypeCommit:
		return "OBJ_COMMIT"
	case TypeTree:
		return "OBJ_TREE"
	case TypeBlob:
		return "OBJ_BLOB"
	case TypeTag:
		return "OBJ_TAG"
	case TypeReserved:
		return "OBJ_RESERVED"
	case TypeOfsDelta:
		return "OBJ_OFS_DELTA"
	case TypeRefDelta:
		return "OBJ_REF_DELTA"
	default:
		return fmt.Sprintf("object.Type(%d)", uint8(t))
	}
}

func (t Type) Bytes() []byte {
	switch t {
	case TypeCommit:
		return []byte("commit")
	case TypeTree:
		return []byte("tree")
	case TypeBlob:
		return []byte("blob")
	case TypeTag:
		return []byte("tag")
	case TypeOfsDelta:
		return []byte("ofs-delta")
	case TypeRefDelta:
		return []byte("ref-delta")
	default:
		return []byte("unknown")
	}
}
