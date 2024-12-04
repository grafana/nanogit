package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type githubClient struct {
	base   *url.URL
	client *http.Client
}

func newGitHubClient(u *url.URL) (*githubClient, error) {
	// There's a pointer in url.URL, but it's inmutable, so it's safe to do
	// a shallow-copy here.
	u2 := *u

	u2.Path = strings.TrimRight(u2.Path, "/")
	u2.Path = strings.TrimSuffix(u2.Path, ".git")

	c := githubClient{
		base:   &u2,
		client: &http.Client{},
	}

	return &c, nil
}

func (c *githubClient) addHeaders(req *http.Request) {
	req.Header.Add("Git-Protocol", "version=2")
	req.Header.Add("User-Agent", "nanogit/0")

	if username, password := os.Getenv("GHUSER"), os.Getenv("GHPASS"); username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	if token := os.Getenv("GH_TOKEN"); token != "" {
		req.Header.Add("Authorization", "token "+token)
	}
}

func (c *githubClient) SendCommands(ctx context.Context, data []byte) ([]byte, error) {
	body := bytes.NewReader(data)

	u := c.base.JoinPath("git-upload-pack").String()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return nil, err
	}

	c.addHeaders(req)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	return io.ReadAll(res.Body)
}

func (c *githubClient) SmartInfoRequest(ctx context.Context) ([]byte, error) {
	u := c.base.JoinPath("info/refs")

	query := make(url.Values)
	query.Set("service", "git-upload-pack")
	u.RawQuery = query.Encode()

	slog.Info("SmartInfoRequest", "url", u.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	c.addHeaders(req)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	return io.ReadAll(res.Body)
}
