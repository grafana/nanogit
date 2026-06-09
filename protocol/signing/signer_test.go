package signing_test

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/protocol/signing"
)

func signAndStore(t *testing.T, signer signing.Signer) (repo, sha string) {
	t.Helper()
	c := newTestCommit("verify roundtrip\n")
	sig, err := signer.Sign(c.BuildUnsigned())
	require.NoError(t, err)
	c.Signature = sig
	signed := c.Build()

	repo = filepath.Join(t.TempDir(), "repo.git")
	run(t, "", "git", "init", "--bare", repo)

	cmd := exec.Command("git", "-C", repo, "hash-object", "-w", "-t", "commit", "--stdin")
	cmd.Stdin = bytes.NewReader(signed)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "hash-object: %s", out)
	return repo, strings.TrimSpace(string(out))
}

func run(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s %s: %s", name, strings.Join(args, " "), out)
	return string(out)
}

func newTestCommit(msg string) *protocol.PackfileCommit {
	ident := &protocol.Identity{Name: "A", Email: "a@b", Timestamp: 1234567890, Timezone: "+0000"}
	return &protocol.PackfileCommit{
		Tree:      hash.Zero,
		Parent:    hash.Zero,
		Author:    ident,
		Committer: ident,
		Message:   msg,
	}
}
