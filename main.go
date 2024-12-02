package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/grafana/hackathon-2024-12-nanogit/protocol"
)

func main() {
	if err := run(); err != nil {
		slog.Error("app run returned error", "err", err)
		os.Exit(1)
	}
}

func cmd(ctx context.Context, org, repo string, data []byte) ([]byte, error) {
	body := io.NopCloser(bytes.NewReader(data))
	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/"+org+"/"+repo+"/git-upload-pack", body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Git-Protocol", "version=2")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}
	b, err := io.ReadAll(res.Body)
	return b, err
}

func run() error {
	owner, repo := "grafana", "grafana"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	refsData, err := cmd(ctx, owner, repo, protocol.FormatPacket([]byte("command=ls-refs\n"), []byte("object-format=sha1\n")))
	if err != nil {
		return err
	}
	lines, remainder, err := protocol.ParsePacket(refsData)
	if err != nil {
		return err
	}
	for _, line := range lines {
		slog.Info("line in data", "line", string(line))
	}
	slog.Info("and here's the remainder", "remainder", remainder)

	return nil
}
