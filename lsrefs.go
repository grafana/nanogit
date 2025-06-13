package nanogit

import (
	"context"
	"fmt"

	"github.com/grafana/nanogit/protocol"
)

type lsRefsOptions struct {
	Prefix string
}

func (c *httpClient) lsRefs(ctx context.Context, opts lsRefsOptions) ([]protocol.RefLine, error) {
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

	refsData, err := c.UploadPack(ctx, pkt)
	if err != nil {
		return nil, fmt.Errorf("send ls-refs command: %w", err)
	}

	refs := make([]protocol.RefLine, 0)
	lines, _, err := protocol.ParsePack(refsData)
	if err != nil {
		return nil, fmt.Errorf("parse refs response: %w", err)
	}

	for _, line := range lines {
		refLine, err := protocol.ParseRefLine(line)
		if err != nil {
			return nil, fmt.Errorf("parse ref line: %w", err)
		}

		if refLine.RefName != "" {
			refs = append(refs, refLine)
		}
	}

	return refs, nil

}
