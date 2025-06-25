package client

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
)

type LsRefsOptions struct {
	Prefix string
}

func (c *rawClient) LsRefs(ctx context.Context, opts LsRefsOptions) ([]protocol.RefLine, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Ls-refs", "prefix", opts.Prefix)

	// Send the ls-refs command directly - Protocol v2 allows this without needing
	// a separate capability advertisement request
	packs := []protocol.Pack{
		protocol.PackLine("command=ls-refs\n"),
		protocol.PackLine("object-format=sha1\n"),
	}

	if opts.Prefix != "" {
		packs = append(packs, protocol.DelimeterPacket)
		packs = append(packs, protocol.PackLine(fmt.Sprintf("ref-prefix %s\n", opts.Prefix)))
	}

	packs = append(packs, protocol.FlushPacket)
	pkt, err := protocol.FormatPacks(packs...)
	if err != nil {
		return nil, fmt.Errorf("format ls-refs command: %w", err)
	}

	logger.Debug("Send Ls-refs request", "requestSize", len(pkt))
	logger.Debug("Ls-refs raw request", "request", string(pkt))

	refsReader, err := c.UploadPack(ctx, bytes.NewReader(pkt))
	if err != nil {
		return nil, fmt.Errorf("send ls-refs command: %w", err)
	}
	defer refsReader.Close()

	// Read response for logging and parsing
	refsData, err := io.ReadAll(refsReader)
	if err != nil {
		return nil, fmt.Errorf("read refs response: %w", err)
	}

	logger.Debug("Received ls-refs response", "responseSize", len(refsData))
	logger.Debug("Ls-refs raw response", "response", string(refsData))

	refs := make([]protocol.RefLine, 0)
	lines, _, err := protocol.ParsePack(refsData)
	if err != nil {
		return nil, fmt.Errorf("parse refs response: %w", err)
	}

	logger.Debug("Parse ref lines", "lineCount", len(lines))
	for _, line := range lines {
		refLine, err := protocol.ParseRefLine(line)
		if err != nil {
			return nil, fmt.Errorf("parse ref line: %w", err)
		}

		if refLine.RefName != "" {
			refs = append(refs, refLine)
		}
	}

	logger.Debug("Ls-refs completed", "refCount", len(refs))
	return refs, nil
}
