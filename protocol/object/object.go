// Package object defines the types of objects that can be stored in a Git repository.
//
// Git stores all content as objects in its object database. Each object has a type
// that determines how Git interprets its contents. The object types are:
//
//   - Commit: A snapshot of the repository at a point in time, containing metadata
//     about the commit (author, committer, message) and references to tree and parent
//     objects.
//   - Tree: A directory listing, containing references to blobs and other trees.
//   - Blob: A file's contents.
//   - Tag: A reference to a specific object, usually a commit, with additional metadata.
//
// Additionally, Git uses two special object types for pack files:
//   - OfsDelta: A delta object that references its base by offset within the pack.
//   - RefDelta: A delta object that references its base by its object ID.
//
// For more details about Git's object types and their formats, see:
// https://git-scm.com/book/en/v2/Git-Internals-Git-Objects
// https://git-scm.com/docs/pack-format#_object_types
package object

import "fmt"

// Type represents a Git object type. The values are chosen to match Git's internal
// representation in pack files, where the type is stored as a 3-bit value.
type Type uint8

// The object types. The values are chosen to match Git's internal representation
// in pack files, where the type is stored as a 3-bit value. Type 5 is reserved
// for future use, and 0 is invalid.
const (
	TypeInvalid  Type = 0 // 0b000 - Invalid type
	TypeCommit   Type = 1 // 0b001 - Commit object
	TypeTree     Type = 2 // 0b010 - Tree object
	TypeBlob     Type = 3 // 0b011 - Blob object
	TypeTag      Type = 4 // 0b100 - Tag object
	TypeReserved Type = 5 // 0b101 - Reserved for future use
	TypeOfsDelta Type = 6 // 0b110 - Offset delta in pack file
	TypeRefDelta Type = 7 // 0b111 - Reference delta in pack file
)

// String returns the string representation of the object type.
// This is used for debugging and error messages.
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

// Bytes returns the byte representation of the object type as used in Git's
// object format. This is the string that appears in the object header,
// e.g., "commit", "tree", "blob", etc.
//
// For more details about Git's object format, see:
// https://git-scm.com/book/en/v2/Git-Internals-Git-Objects#_object_storage
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
	case TypeInvalid, TypeReserved:
		fallthrough
	default:
		return []byte("unknown")
	}
}
