package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/protocol"
)

var (
	ErrRefNotFound = errors.New("ref not found")
)

type Ref struct {
	Name string
	Hash string
}

// ListRefs sends a request to list all references in the repository.
// It returns a map of reference names to their commit hashes.
func (c *clientImpl) ListRefs(ctx context.Context) ([]Ref, error) {
	// First get the initial capability advertisement
	_, err := c.SmartInfoRequest(ctx)
	if err != nil {
		return nil, fmt.Errorf("get repository info: %w", err)
	}

	// Now send the ls-refs command
	pkt, err := protocol.FormatPacks(
		protocol.PackLine("command=ls-refs\n"),
		protocol.PackLine("object-format=sha1\n"),
		protocol.FlushPacket,
	)
	if err != nil {
		return nil, fmt.Errorf("format ls-refs command: %w", err)
	}

	refsData, err := c.SendCommands(ctx, pkt)
	if err != nil {
		return nil, fmt.Errorf("send ls-refs command: %w", err)
	}

	refs := make([]Ref, 0)
	lines, _, err := protocol.ParsePack(refsData)
	if err != nil {
		return nil, fmt.Errorf("parse refs response: %w", err)
	}

	for _, line := range lines {
		ref, hash, err := parseRefLine(line)
		if err != nil {
			return nil, fmt.Errorf("parse ref line: %w", err)
		}
		if ref != "" {
			refs = append(refs, Ref{Name: ref, Hash: hash})
		}
	}

	return refs, nil
}

// parseRefLine parses a single reference line from the git response.
// Returns the reference name, hash, and any error encountered.
func parseRefLine(line []byte) (ref, hash string, err error) {
	// Skip empty lines and pkt-line flush markers
	if len(line) == 0 || bytes.Equal(line, []byte("0000")) {
		return "", "", nil
	}

	// Skip capability lines (they start with =)
	if len(line) > 0 && line[0] == '=' {
		return "", "", nil
	}

	// Split into hash and rest
	parts := bytes.SplitN(line, []byte(" "), 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid ref format: %s", line)
	}

	// Ensure we have a full 40-character SHA-1 hash
	hash = string(parts[0])
	if len(hash) != 40 {
		return "", "", fmt.Errorf("invalid hash length: got %d, want 40", len(hash))
	}

	refName := strings.TrimSpace(string(parts[1]))

	// Handle HEAD reference with capabilities
	if strings.HasPrefix(refName, "HEAD") {
		symref := extractSymref(refName)
		if symref != "" {
			return symref, hash, nil
		}

		return refName, hash, nil
	}

	// Remove capability suffix if present
	if idx := bytes.IndexByte(parts[1], '\x00'); idx > 0 {
		refName = string(parts[1][:idx])
	}

	return strings.TrimSpace(refName), strings.TrimSpace(hash), nil
}

// extractSymref extracts the symref value from a line.
// It returns the symref value if present, and an error if it is not present.
// Example:
// 00437fd1a60b01f91b314f59955a4e4d4e80d8edf11d HEAD symref=HEAD:refs/heads/master
// The symref value is "refs/heads/master".
func extractSymref(line string) string {
	// Check for symref in the reference line
	parts := strings.Split(line, " ")
	if len(parts) == 1 {
		return ""
	}

	if idx := strings.Index(line, "symref="); idx > 0 {
		symref := line[idx+7:]
		if colonIdx := strings.Index(symref, ":"); colonIdx > 0 {
			return strings.TrimSpace(symref[colonIdx+1:])
		}

		return strings.TrimSpace(symref)
	}

	return ""
}
