package protocol

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// ParseReceivePackInfoRefs reads the body of GET info/refs?service=git-receive-pack
// and returns the capabilities the server advertised after the NUL byte on
// the first ref line. The Git Smart HTTP v1-style discovery format is:
//
//	001e# service=git-receive-pack\n
//	0000
//	<sha> <refname>\x00<cap1> <cap2> ...\n
//	<sha> <refname>\n
//	...
//	0000
//
// Empty repositories advertise a synthetic "capabilities^{}" ref:
//
//	0000000000000000000000000000000000000000 capabilities^{}\x00<caps>
//
// The parser is intentionally pure: it accepts an io.Reader, never touches
// net/http, and never blocks the caller on transport state. The caller is
// responsible for reading response bodies and for reporting transport errors.
//
// Returns an empty slice (not nil) when the first ref line has no NUL — i.e.,
// the server advertised zero capabilities. Returns an error when the response
// is malformed (not a pkt-line stream, missing service header, no ref lines).
func ParseReceivePackInfoRefs(r io.Reader) ([]Capability, error) {
	parser := NewParser(r)

	// First pkt-line: "# service=git-receive-pack\n". Parser.Next surfaces
	// flush packets as io.EOF — the intermediate flush between header and ref
	// list will produce EOF too, so we keep reading past it until we see the
	// first ref line or run out of input.
	first, err := readNonFlush(parser)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("receive-pack info/refs: empty response")
		}
		return nil, fmt.Errorf("receive-pack info/refs: read service header: %w", err)
	}
	service := strings.TrimRight(string(first), "\n")
	if service != "# service=git-receive-pack" {
		return nil, fmt.Errorf("receive-pack info/refs: expected service header, got %q", service)
	}

	refLine, err := readNonFlush(parser)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("receive-pack info/refs: no ref lines after service header")
		}
		return nil, fmt.Errorf("receive-pack info/refs: read first ref line: %w", err)
	}

	// First ref line: "<sha> <refname>\x00<caps>\n" — split on NUL.
	line := strings.TrimRight(string(refLine), "\n")
	_, capPart, ok := strings.Cut(line, "\x00")
	if !ok || capPart == "" {
		// Either no NUL (server advertised no capabilities on the first ref —
		// unusual but technically legal) or NUL with empty suffix. Return an
		// empty (non-nil) slice so callers can distinguish "advertised
		// nothing" from "parse error".
		return []Capability{}, nil
	}

	tokens := strings.Split(capPart, " ")
	caps := make([]Capability, 0, len(tokens))
	for _, t := range tokens {
		if t == "" {
			continue
		}
		caps = append(caps, Capability(t))
	}
	return caps, nil
}

// readNonFlush returns the next non-flush pkt-line from p. Parser.Next maps
// every flush packet (length == 0000) to io.EOF, so we retry on EOF: if the
// EOF was a flush separator the retry reads the following pkt-line; if the
// stream is genuinely exhausted the retry returns io.EOF again.
func readNonFlush(p *Parser) ([]byte, error) {
	line, err := p.Next()
	if err == nil {
		return line, nil
	}
	if !errors.Is(err, io.EOF) {
		return nil, err
	}
	return p.Next()
}
