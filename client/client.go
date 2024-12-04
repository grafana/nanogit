package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

type Client interface {
	// TODO(mem): this is probably not the right interface.
	SendCommands(ctx context.Context, data []byte) ([]byte, error)
	SmartInfoRequest(ctx context.Context) ([]byte, error)
}

// New returns a new Client for the given repository.
//
// TODO(mem): this is a temporary implementation. It probably needs to have
// some kind of options parameter so that we can pass authentication
// information to the client. It's possible that basic auth is not going to be
// enough for all possible situations.
func New(repo string) (Client, error) {
	u, err := url.Parse(repo)
	if err != nil {
		return nil, fmt.Errorf("parsing url: %w", err)
	}

	switch u.Host {
	// This won't work with GitHub Enterprise because in that case it's
	// possible to have a custom domain name.
	case "github.com":
		return newGitHubClient(u)

	default:
		return nil, errors.New("unsupported host")
	}
}
