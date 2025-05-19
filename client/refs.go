package client

import (
	"bytes"
	"context"

	//nolint:gosec // git uses sha1 for the pack file
	"crypto/sha1"
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
	_, err := c.SmartInfo(ctx, "git-upload-pack")
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

	refsData, err := c.UploadPack(ctx, pkt)
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

// GetRef sends a request to get a single reference in the repository.
// It returns the reference name, hash, and any error encountered.
// FIXME: In protocol v1, you cannot filter the refs you want to get.
func (c *clientImpl) GetRef(ctx context.Context, refName string) (Ref, error) {
	refs, err := c.ListRefs(ctx)
	if err != nil {
		return Ref{}, fmt.Errorf("list refs: %w", err)
	}

	for _, r := range refs {
		if r.Name == refName {
			return r, nil
		}
	}

	return Ref{}, ErrRefNotFound
}

// CreateRef creates a new reference in the repository.
// It returns any error encountered.
func (c *clientImpl) CreateRef(ctx context.Context, ref Ref) error {
	_, err := c.GetRef(ctx, ref.Name)
	switch {
	case err != nil && !errors.Is(err, ErrRefNotFound):
		return fmt.Errorf("get ref: %w", err)
	case err == nil:
		return fmt.Errorf("ref %s already exists", ref.Name)
	}

	// First get the initial capability advertisement
	_, err = c.SmartInfo(ctx, "git-receive-pack")
	if err != nil {
		return fmt.Errorf("get receive-pack capability: %w", err)
	}

	// Create the ref using receive-pack
	// Format: <old-value> <new-value> <ref-name>\000<capabilities>\n
	// Old value is the zero hash for new refs
	refLine := fmt.Sprintf("%s %s %s\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n", strings.Repeat("0", 40), ref.Hash, strings.TrimSpace(ref.Name))

	// Calculate the correct length (including the 4 bytes of the length field)
	lineLen := len(refLine) + 4
	pkt := []byte(fmt.Sprintf("%04x%s0000", lineLen, refLine))

	// Add empty pack file
	// Pack file format: PACK + version(4) + object count(4) + SHA1(20)
	emptyPack := []byte("PACK\x00\x00\x00\x02\x00\x00\x00\x00")

	// Calculate SHA1 of the pack file
	//nolint:gosec // git uses sha1 for the pack file
	h := sha1.New()
	h.Write(emptyPack)
	packSha1 := h.Sum(nil)
	emptyPack = append(emptyPack, packSha1...)

	// Send pack file as raw data (not as a pkt-line)
	pkt = append(pkt, emptyPack...)

	// Add final flush packet
	pkt = append(pkt, []byte("0000")...)
	_, err = c.SmartInfo(ctx, "git-receive-pack")
	if err != nil {
		return fmt.Errorf("get receive-pack capability: %w", err)
	}

	// Send the ref update
	_, err = c.ReceivePack(ctx, pkt)
	if err != nil {
		return fmt.Errorf("send ref update: %w", err)
	}

	return nil
}

// UpdateRef updates an existing reference in the repository.
// It returns any error encountered.
func (c *clientImpl) UpdateRef(ctx context.Context, ref Ref) error {
	// First check if the ref exists
	oldRef, err := c.GetRef(ctx, ref.Name)
	if err != nil {
		if errors.Is(err, ErrRefNotFound) {
			return fmt.Errorf("ref %s does not exist", ref.Name)
		}
		return fmt.Errorf("get ref: %w", err)
	}

	// First get the initial capability advertisement
	_, err = c.SmartInfo(ctx, "git-receive-pack")
	if err != nil {
		return fmt.Errorf("get receive-pack capability: %w", err)
	}

	// Update the ref using receive-pack
	// Format: <old-value> <new-value> <ref-name>\000<capabilities>\n
	refLine := fmt.Sprintf("%s %s %s\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n", oldRef.Hash, ref.Hash, strings.TrimSpace(ref.Name))

	// Calculate the correct length (including the 4 bytes of the length field)
	lineLen := len(refLine) + 4
	pkt := []byte(fmt.Sprintf("%04x%s0000", lineLen, refLine))

	// Add empty pack file
	// Pack file format: PACK + version(4) + object count(4) + SHA1(20)
	emptyPack := []byte("PACK\x00\x00\x00\x02\x00\x00\x00\x00")

	// Calculate SHA1 of the pack file
	//nolint:gosec // git uses sha1 for the pack file
	h := sha1.New()
	h.Write(emptyPack)
	packSha1 := h.Sum(nil)
	emptyPack = append(emptyPack, packSha1...)

	// Send pack file as raw data (not as a pkt-line)
	pkt = append(pkt, emptyPack...)

	// Add final flush packet
	pkt = append(pkt, []byte("0000")...)

	// Send the ref update
	_, err = c.ReceivePack(ctx, pkt)
	if err != nil {
		return fmt.Errorf("update ref: %w", err)
	}

	return nil
}

// DeleteRef deletes a reference from the repository.
// It returns any error encountered.
func (c *clientImpl) DeleteRef(ctx context.Context, refName string) error {
	// First check if the ref exists
	_, err := c.GetRef(ctx, refName)
	if err != nil {
		if errors.Is(err, ErrRefNotFound) {
			return fmt.Errorf("ref %s does not exist", refName)
		}
		return fmt.Errorf("get ref: %w", err)
	}

	// First get the initial capability advertisement
	_, err = c.SmartInfo(ctx, "git-receive-pack")
	if err != nil {
		return fmt.Errorf("get receive-pack capability: %w", err)
	}

	// Delete the ref using receive-pack
	// Format: <old-value> <new-value> <ref-name>\000<capabilities>\n
	// For deletion, new-value is the zero hash (40 zeros)
	refLine := fmt.Sprintf("%s %s %s\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n", strings.Repeat("0", 40), strings.Repeat("0", 40), strings.TrimSpace(refName))

	// Calculate the correct length (including the 4 bytes of the length field)
	lineLen := len(refLine) + 4
	pkt := []byte(fmt.Sprintf("%04x%s0000", lineLen, refLine))

	// Add empty pack file
	// Pack file format: PACK + version(4) + object count(4) + SHA1(20)
	emptyPack := []byte("PACK\x00\x00\x00\x02\x00\x00\x00\x00")

	// Calculate SHA1 of the pack file
	//nolint:gosec // git uses sha1 for the pack file
	h := sha1.New()
	h.Write(emptyPack)
	packSha1 := h.Sum(nil)
	emptyPack = append(emptyPack, packSha1...)

	// Send pack file as raw data (not as a pkt-line)
	pkt = append(pkt, emptyPack...)

	// Add final flush packet
	pkt = append(pkt, []byte("0000")...)

	// Send the ref update
	_, err = c.ReceivePack(ctx, pkt)
	if err != nil {
		return fmt.Errorf("delete ref: %w", err)
	}

	return nil
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
