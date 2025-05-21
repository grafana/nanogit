package nanogit

import (
	"context"
	"fmt"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

func (c *clientImpl) GetBlob(ctx context.Context, blobID hash.Hash) ([]byte, error) {
	obj, err := c.getObject(ctx, blobID)
	if err != nil {
		return nil, fmt.Errorf("getting object: %w", err)
	}

	if obj.Type == protocol.ObjectTypeBlob && obj.Hash.Is(blobID) {
		return obj.Data, nil
	}

	return nil, fmt.Errorf("blob not found: %s", blobID)
}
