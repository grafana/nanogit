package nanogit

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/protocol/object"
)

func (c *clientImpl) GetBlob(ctx context.Context, blobID hash.Hash) ([]byte, error) {
	obj, err := c.GetObject(ctx, blobID)
	if err != nil {
		if strings.Contains(err.Error(), "not our ref") {
			return nil, fmt.Errorf("blob not found: %s", blobID)
		}

		return nil, fmt.Errorf("getting object: %w", err)
	}

	if obj.Type == object.TypeBlob && obj.Hash.Is(blobID) {
		return obj.Data, nil
	}

	return nil, fmt.Errorf("blob not found: %s", blobID)
}
