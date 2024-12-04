package protocol

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
)

var (
	ErrNoPackfileSignature        = errors.New("the given payload has no packfile signature")
	ErrUnsupportedPackfileVersion = errors.New("the version of the packfile payload is unsupported")
	ErrUnsupportedObjectType      = errors.New("the type of the object is unsupported")
	ErrInflatedDataIncorrectSize  = errors.New("the data is the wrong size post-inflation")
)

// MaxUnpackedObjectSize is the maximum size of an unpacked object.
const MaxUnpackedObjectSize = 10 * 1024 * 1024

type PackfileEntry struct {
	Object  *PackfileObject
	Trailer *PackfileTrailer
}

type PackfileObject struct {
	// The type of the object. 3-bit field.
	Type ObjectType
	// The data, uncompressed.
	// If Type is one of ObjectTypeRefDelta and ObjectTypeOfsDelta, this is a delta.
	Data []byte

	// If Type == ObjectTypeRefDelta, this is set.
	ObjectName string
	// If Type == ObjectTypeOfsDelta, this is set.
	RelativeOffset int
	// If Type == ObjectTypeTree, this is set.
	Tree []PackfileTreeEntry
}

func (e *PackfileObject) parseTree() error {
	reader := bufio.NewReader(bytes.NewReader(e.Data))

	for {
		fileModeStr, err := reader.ReadString(' ')
		if err != nil {
			if errors.Is(err, io.EOF) {
				// The last entry was already entered.
				break
			}
			return err
		}
		fileModeStr = fileModeStr[:len(fileModeStr)-1] // ReadString includes delim
		fileMode, err := strconv.ParseUint(fileModeStr, 8, 32)
		if err != nil {
			return err
		}

		name, err := reader.ReadString(0)
		if err != nil {
			return eofIsUnexpected(err)
		}
		name = name[:len(name)-1] // ReadString includes delim

		var hash [20]byte
		if _, err := io.ReadFull(reader, hash[:]); err != nil {
			return eofIsUnexpected(err)
		}

		e.Tree = append(e.Tree, PackfileTreeEntry{
			FileName: name,
			FileMode: uint32(fileMode),
			Hash:     hex.EncodeToString(hash[:]),
		})
	}

	return nil
}

// PackfileTreeEntry represents a part of a packfile tree.
//
// The wire-format looks as follows:
//   - File mode as ASCII text. Dirs are 0o40000.
//   - A space (0x20).
//   - A file name. NUL bytes are not legal.
//   - A NUL byte.
//   - A hash. 20 or 32 bytes for SHA-1 and SHA-256 respectively.
//   - Repeat until EOF.
//
// Resource: https://github.com/go-git/go-git/blob/63343bf5f918ea5384ae63bfd22bb36689fa0151/plumbing/object/tree.go#L216-L273
type PackfileTreeEntry struct {
	FileMode uint32
	FileName string
	Hash     string
}

type PackfileTrailer struct {
	// TODO: Checksum here. Are there multiple??
}

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
	reader           io.Reader
	remainingObjects uint32

	// State that shouldn't be set when constructed.
	trailerRead bool
	err         error
}

// ReadObject reads an object from the packfile.
// The final object is always a PackfileTrailer. It comes with a nil error, not an io.EOF.
// When the final object is read, a nil and io.EOF is returned.
// If another error is ever returned, the object is "tainted", and will not read more objects.
//
// This function is not concurrency-safe. Use a mutex or a single goroutine+channel when dealing with this.
// Objects returned are no longer owned by this function once returned; you can pass them around goroutines freely.
func (p *PackfileReader) ReadObject() (PackfileEntry, error) {
	// TODO: probably smart to use a mutex here.

	entry := PackfileEntry{}
	if p.err != nil {
		return entry, fmt.Errorf("ReadObject called after error returned: %w", p.err)
	}

	if p.remainingObjects == 0 {
		// It's time for the trailer.
		if p.trailerRead {
			// We've already read it, so there's no more to do here.
			return entry, io.EOF
		}

		// TODO: Read & parse trailer. No idea how that'll work.
		entry.Trailer = &PackfileTrailer{}

		p.trailerRead = true
		return entry, nil
	}
	p.remainingObjects--

	// TODO(mariell): kinda ugly hack... let's just call another method and set this when an error is returned from it.
	var err error
	defer func() {
		p.err = err
	}()

	var buf [1]byte
	if _, err = p.reader.Read(buf[:]); err != nil {
		return entry, eofIsUnexpected(err)
	}

	entry.Object = &PackfileObject{}

	// The first byte is a 3-bit type (stored in 4 bits).
	// The remaining 4 bits are the start of a varint containing the size.
	entry.Object.Type = ObjectType((buf[0] >> 4) & 0b111)
	size := int(buf[0] & 0b1111)
	shift := 4
	for buf[0]&0x80 == 0x80 {
		if _, err = p.reader.Read(buf[:]); err != nil {
			return entry, eofIsUnexpected(err)
		}

		size += int(buf[0]&0x7f) << shift
		shift += 7
	}

	switch entry.Object.Type {
	case ObjectTypeBlob, ObjectTypeCommit, ObjectTypeTag, ObjectTypeTree:
		entry.Object.Data, err = p.readAndInflate(size)
		if err != nil {
			return entry, eofIsUnexpected(err)
		}
		if entry.Object.Type == ObjectTypeTree {
			if err = entry.Object.parseTree(); err != nil {
				return entry, eofIsUnexpected(err)
			}
		}

	case ObjectTypeRefDelta:
		var ref [20]byte
		if _, err = p.reader.Read(ref[:]); err != nil {
			return entry, eofIsUnexpected(err)
		}
		entry.Object.ObjectName = hex.EncodeToString(ref[:])
		entry.Object.Data, err = p.readAndInflate(size)
		if err != nil {
			return entry, eofIsUnexpected(err)
		}

	case ObjectTypeOfsDelta:
		// TODO(mariell): we need to handle a ref delta, at least.
		//   Maybe OFS too? I don't think we need them as that's a
		//   capability to negotiate.
		fallthrough

	case ObjectTypeInvalid, ObjectTypeReserved:
		// TODO(mem): do we need to do something about these? No
		// special handling for them yet.
		fallthrough

	default:
		err = fmt.Errorf("%w (%s; original byte: %08b)",
			ErrUnsupportedObjectType, entry.Object.Type, buf[0])
		return entry, err
	}

	return entry, nil
}

func (p *PackfileReader) readAndInflate(sz int) ([]byte, error) {
	zr, err := zlib.NewReader(p.reader)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	// TODO(mem): this should be limited to the size the packet says it
	// carries, and we should limit that size above (i.e. if the packet
	// says it's carrying a huge amount of data we should bail out).
	lr := io.LimitReader(zr, MaxUnpackedObjectSize)

	var data bytes.Buffer
	if _, err := io.Copy(&data, lr); err != nil {
		return nil, eofIsUnexpected(err)
	}

	if data.Len() != sz {
		return nil, ErrInflatedDataIncorrectSize
	}

	return data.Bytes(), nil
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
	payload = payload[4:] // Without version

	// The payload now contains just objects. These are multiple and can be quite large.
	// Let's pass it off to a caller to read the rest of what's in here.
	// Eventually, we can even accept an io.Reader directly here, such that we don't need to
	//   keep the whole original payload in memory, either.
	return &PackfileReader{reader: bytes.NewReader(payload), remainingObjects: countObjects}, nil
}
