package protocol

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/grafana/nanogit/protocol/hash"
)

// FIXME: This logic is pretty hard to follow and test. So it's missing coverage for now
// Review it once we have some more integration testing so that we don't break things unintentionally.

const (
	ErrNoPackfileSignature        = strError("the given payload has no packfile signature")
	ErrUnsupportedPackfileVersion = strError("the version of the packfile payload is unsupported")
	ErrUnsupportedObjectType      = strError("the type of the object is unsupported")
	ErrInflatedDataIncorrectSize  = strError("the data is the wrong size post-inflation")
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
	// The hash of the object. Might be unset.
	Hash hash.Hash

	// If Type == ObjectTypeRefDelta, this is set.
	Delta *Delta
	// If Type == ObjectTypeOfsDelta, this is set.
	RelativeOffset int
	// If Type == ObjectTypeTree, this is set.
	Tree []PackfileTreeEntry
	// If Type == ObjectTypeCommit, this is set.
	Commit *PackfileCommit
}

func (e *PackfileObject) parseTree() error {
	reader := bufio.NewReader(bytes.NewReader(e.Data))

	for {
		fileModeStr, err := reader.ReadString(' ')
		if err != nil {
			if fileModeStr == "" && errors.Is(err, io.EOF) {
				// The last entry was already entered.
				break
			}
			return eofIsUnexpected(err)
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

func (e *PackfileObject) parseCommit() error {
	reader := bufio.NewReader(bytes.NewReader(e.Data))

	e.Commit = &PackfileCommit{}
	writingMsg := false
	msg := strings.Builder{}
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if len(line) == 0 && errors.Is(err, io.EOF) {
			break
		}

		if writingMsg {
			msg.Write(line)
			continue
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			writingMsg = true
			continue
		}

		command, data, _ := bytes.Cut(line, []byte(" "))
		switch string(command) {
		case "committer":
			identity, err := ParseIdentity(string(data))
			if err != nil {
				return fmt.Errorf("parsing committer: %w", err)
			}
			e.Commit.Committer = identity
		case "tree":
			e.Commit.Tree, err = hash.FromHex(string(data))
		case "author":
			identity, err := ParseIdentity(string(data))
			if err != nil {
				return fmt.Errorf("parsing author: %w", err)
			}
			e.Commit.Author = identity
		case "parent":
			e.Commit.Parent, err = hash.FromHex(string(data))
		default:
			if e.Commit.Fields == nil {
				e.Commit.Fields = make(map[string][]byte, 8)
			}
			e.Commit.Fields[string(command)] = data
		}
		if err != nil {
			return err
		}
	}
	e.Commit.Message = msg.String()
	return nil
}

func (e *PackfileObject) parseDelta(parent string) error {
	var err error
	e.Delta, err = parseDelta(parent, e.Data)
	return eofIsUnexpected(err)
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

// PackfileCommit represents a single commit within a packfile.
//
// The wire-format looks as follows:
//   - A set of attribute fields delimited by '\n's. They are a name, a space (0x20), and a value.
//   - An empty line (i.e. just \n).
//   - The commit message. A PGP signature is optionally included here, which will then have a '\n \n\n' at the end of it.
//
// Resource: https://github.com/go-git/go-git/blob/63343bf5f918ea5384ae63bfd22bb36689fa0151/plumbing/object/commit.go#L185-L275
type PackfileCommit struct {
	Tree      hash.Hash
	Author    *Identity
	Committer *Identity
	Parent    hash.Hash
	Message   string
	// Fields contains any fields beyond the fields that are statically defined.
	// If a field is statically defined, it SHOULD not show up here.
	Fields map[string][]byte

	// There is also a gpgsig field.
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
	algo             crypto.Hash

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
	if p == nil {
		return PackfileEntry{}, io.EOF
	}

	if p.err != nil {
		return PackfileEntry{}, fmt.Errorf("ReadObject called after error returned: %w", p.err)
	}

	var entry PackfileEntry
	entry, p.err = p.readObject()
	if !p.trailerRead {
		p.err = eofIsUnexpected(p.err)
	}
	return entry, p.err
}

func (p *PackfileReader) readObject() (PackfileEntry, error) {
	entry := PackfileEntry{}
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

	var buf [1]byte
	if _, err := p.reader.Read(buf[:]); err != nil {
		return entry, err
	}

	entry.Object = &PackfileObject{}

	// The first byte is a 3-bit type (stored in 4 bits).
	// The remaining 4 bits are the start of a varint containing the size.
	entry.Object.Type = ObjectType((buf[0] >> 4) & 0b111)
	size := int(buf[0] & 0b1111)
	shift := 4
	for buf[0]&0x80 == 0x80 {
		if _, err := p.reader.Read(buf[:]); err != nil {
			return entry, err
		}

		size += int(buf[0]&0x7f) << shift
		shift += 7
	}

	var err error
	switch entry.Object.Type {
	case ObjectTypeBlob, ObjectTypeCommit, ObjectTypeTag, ObjectTypeTree:
		entry.Object.Data, err = p.readAndInflate(size)
		if err != nil {
			return entry, eofIsUnexpected(err)
		}

		entry.Object.Hash, err = Object(p.algo, entry.Object.Type, entry.Object.Data)
		if err != nil {
			return entry, eofIsUnexpected(err)
		}

		if entry.Object.Type == ObjectTypeTree {
			if err := entry.Object.parseTree(); err != nil {
				return entry, err
			}
		}
		if entry.Object.Type == ObjectTypeCommit {
			if err := entry.Object.parseCommit(); err != nil {
				return entry, err
			}
		}

	case ObjectTypeRefDelta:
		ref := make([]byte, p.algo.Size())
		if _, err := p.reader.Read(ref); err != nil {
			return entry, err
		}
		entry.Object.Data, err = p.readAndInflate(size)
		if err != nil {
			return entry, err
		}
		if err := entry.Object.parseDelta(hex.EncodeToString(ref[:])); err != nil {
			return entry, err
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

func ParsePackfile(payload []byte) (*PackfileReader, error) {
	// TODO: Accept an io.Reader to the function.
	if len(payload) < 4 || !slices.Equal(payload[:4], []byte("PACK")) {
		return nil, ErrNoPackfileSignature
	}
	payload = payload[4:] // Without "PACK"

	version := binary.BigEndian.Uint32(payload[:4])
	if version != 2 && version != 3 {
		return nil, fmt.Errorf("version %d: %w", version, ErrUnsupportedPackfileVersion)
	}
	payload = payload[4:] // Without version

	countObjects := binary.BigEndian.Uint32(payload[:4])
	payload = payload[4:] // Without countObjects

	// The payload now contains just objects. These are multiple and can be quite large.
	// Let's pass it off to a caller to read the rest of what's in here.
	// Eventually, we can even accept an io.Reader directly here, such that we don't need to
	//   keep the whole original payload in memory, either.
	return &PackfileReader{
		reader:           bytes.NewReader(payload),
		remainingObjects: countObjects,
		algo:             crypto.SHA1, // TODO: Support SHA256
	}, nil
}

// PackfileWriter helps create Git objects and pack them into a packfile.
// It maintains state about the objects being written and handles the packfile format.
type PackfileWriter struct {
	// Objects that will be written to the packfile
	objects []PackfileObject
	// The hash algorithm to use (SHA1 or SHA256)
	algo crypto.Hash
	// Buffer to store the final packfile
	buf bytes.Buffer
}

// NewPackfileWriter creates a new PackfileWriter with the specified hash algorithm.
func NewPackfileWriter(algo crypto.Hash) *PackfileWriter {
	return &PackfileWriter{
		objects: make([]PackfileObject, 0),
		algo:    algo,
	}
}

// AddBlob adds a blob object to the packfile.
// The blob contains the raw file contents.
func (w *PackfileWriter) AddBlob(data []byte) (hash.Hash, error) {
	obj := PackfileObject{
		Type: ObjectTypeBlob,
		Data: data,
	}

	h, err := Object(w.algo, obj.Type, obj.Data)
	if err != nil {
		return hash.Hash{}, fmt.Errorf("computing blob hash: %w", err)
	}
	obj.Hash = h

	w.objects = append(w.objects, obj)
	return h, nil
}

// AddTree adds a tree object to the packfile.
// The tree represents a directory structure with file modes and hashes.
func (w *PackfileWriter) AddTree(entries []PackfileTreeEntry) (hash.Hash, error) {
	// Sort entries by name for consistent hashing
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].FileName < entries[j].FileName
	})

	// Build tree data
	var data bytes.Buffer
	for _, entry := range entries {
		// Write mode as octal string
		fmt.Fprintf(&data, "%o ", entry.FileMode)
		// Write filename
		data.WriteString(entry.FileName)
		data.WriteByte(0) // NUL byte
		// Write hash
		hashBytes, err := hex.DecodeString(entry.Hash)
		if err != nil {
			return hash.Hash{}, fmt.Errorf("decoding hash for %s: %w", entry.FileName, err)
		}
		data.Write(hashBytes)
	}

	obj := PackfileObject{
		Type: ObjectTypeTree,
		Data: data.Bytes(),
		Tree: entries,
	}

	h, err := Object(w.algo, obj.Type, obj.Data)
	if err != nil {
		return hash.Hash{}, fmt.Errorf("computing tree hash: %w", err)
	}
	obj.Hash = h

	w.objects = append(w.objects, obj)
	return h, nil
}

// AddCommit adds a commit object to the packfile.
// The commit references a tree and optionally a parent commit.
func (w *PackfileWriter) AddCommit(tree, parent hash.Hash, author, committer *Identity, message string) (hash.Hash, error) {
	var data bytes.Buffer

	// Write tree
	fmt.Fprintf(&data, "tree %s\n", tree.String())

	// Write parent if provided
	if !parent.Is(hash.Zero) {
		fmt.Fprintf(&data, "parent %s\n", parent.String())
	}

	// Write author
	fmt.Fprintf(&data, "author %s\n", author.String())

	// Write committer
	fmt.Fprintf(&data, "committer %s\n", committer.String())

	// Write message
	data.WriteString("\n")
	data.WriteString(message)

	obj := PackfileObject{
		Type: ObjectTypeCommit,
		Data: data.Bytes(),
		Commit: &PackfileCommit{
			Tree:      tree,
			Parent:    parent,
			Author:    author,
			Committer: committer,
			Message:   message,
		},
	}

	h, err := Object(w.algo, obj.Type, obj.Data)
	if err != nil {
		return hash.Hash{}, fmt.Errorf("computing commit hash: %w", err)
	}
	obj.Hash = h

	w.objects = append(w.objects, obj)
	return h, nil
}

// WritePackfile writes all objects to a packfile and returns the packfile data.
// The packfile format is:
// - Reference update command: <old-value> <new-value> <ref-name>\000<capabilities>\n
// - Flush packet (0000)
// - 4-byte signature: "PACK"
// - 4-byte version number (2)
// - 4-byte number of objects
// - Object entries
// - 20-byte SHA1 of the packfile
func (w *PackfileWriter) WritePackfile() ([]byte, error) {
	// Write signature
	if _, err := w.buf.WriteString("PACK"); err != nil {
		return nil, fmt.Errorf("writing packfile signature: %w", err)
	}

	// Write version (2)
	if err := binary.Write(&w.buf, binary.BigEndian, uint32(2)); err != nil {
		return nil, fmt.Errorf("writing packfile version: %w", err)
	}

	// Write number of objects
	numObjects := len(w.objects)
	if numObjects > math.MaxUint32 {
		return nil, fmt.Errorf("too many objects: %d exceeds maximum of %d", numObjects, math.MaxUint32)
	}

	if err := binary.Write(&w.buf, binary.BigEndian, uint32(numObjects)); err != nil {
		return nil, fmt.Errorf("writing object count: %w", err)
	}

	// Write each object
	for _, obj := range w.objects {
		if err := w.writeObject(obj); err != nil {
			return nil, fmt.Errorf("writing object: %w", err)
		}
	}

	// Compute and write packfile hash
	packHash := w.algo.New()
	if _, err := packHash.Write(w.buf.Bytes()); err != nil {
		return nil, fmt.Errorf("writing to pack hash: %w", err)
	}
	if _, err := w.buf.Write(packHash.Sum(nil)); err != nil {
		return nil, fmt.Errorf("writing pack hash: %w", err)
	}

	// Get the packfile data
	packfileData := w.buf.Bytes()

	// Find the commit hash from the objects
	var parentHash, commitHash hash.Hash
	for _, obj := range w.objects {
		if obj.Type == ObjectTypeCommit {
			commitHash = obj.Hash
			parentHash = obj.Commit.Parent
			break
		}
	}

	if commitHash.Is(hash.Zero) {
		return nil, errors.New("no commit object found in packfile")
	}

	// Create the complete message with reference update command
	// Format: <old-value> <new-value> <ref-name>\000<capabilities>\n0000<packfile data>
	refUpdate := fmt.Sprintf("%s %s %s\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n",
		parentHash.String(), // old value (zero hash for new refs)
		commitHash.String(), // new value (the commit hash)
		"refs/heads/main")   // ref name

	// Calculate the length of the ref update line (including the 4 bytes of length)
	refUpdateLen := len(refUpdate) + 4
	refUpdateLine := fmt.Sprintf("%04x%s", refUpdateLen, refUpdate)

	// Combine everything
	completeData := make([]byte, 0, len(refUpdateLine)+4+len(packfileData))
	completeData = append(completeData, []byte(refUpdateLine)...)
	completeData = append(completeData, []byte("0000")...) // flush packet
	completeData = append(completeData, packfileData...)

	return completeData, nil
}

// writeObject writes a single object to the packfile.
// The object format is:
// - Type and size (variable length)
// - Compressed object data
func (w *PackfileWriter) writeObject(obj PackfileObject) error {
	// Write type and size
	size := len(obj.Data)
	firstByte := byte(obj.Type)<<4 | byte(size&0x0f)
	size >>= 4

	for size > 0 {
		firstByte |= 0x80
		w.buf.WriteByte(firstByte)
		firstByte = byte(size & 0x7f)
		size >>= 7
	}
	w.buf.WriteByte(firstByte)

	// Compress and write data
	zw := zlib.NewWriter(&w.buf)
	if _, err := zw.Write(obj.Data); err != nil {
		return fmt.Errorf("compressing object data: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("closing zlib writer: %w", err)
	}

	return nil
}
