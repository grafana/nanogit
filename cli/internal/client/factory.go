package client

import (
	"context"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/cli/internal/auth"
)

// New creates a nanogit client with the provided authentication and options.
func New(ctx context.Context, url string, authConfig *auth.Config) (nanogit.Client, error) {
	opts := authConfig.ToOptions()
	return nanogit.NewHTTPClient(url, opts...)
}
