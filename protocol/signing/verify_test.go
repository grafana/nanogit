package signing_test

import (
	"bytes"
	"crypto"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/protocol/signing"
	"github.com/grafana/nanogit/protocol/signing/testsigning"
)

const signerEmail = "signer@test.invalid"

func TestVerifyWithGit(t *testing.T) {
	t.Run("gpg", func(t *testing.T) {
		requireBins(t, "git", "gpg")
		gpg := testsigning.LoadGPG(t)
		signer, err := signing.NewGPGSigner(gpg.ArmoredKey)
		require.NoError(t, err)
		repo, sha := signAndStore(t, signer)

		t.Setenv("GNUPGHOME", t.TempDir())
		run(t, "", "gpg", "--batch", "--import", gpg.KeyPath)
		out := run(t, repo, "git", "verify-commit", "--raw", sha)
		require.Contains(t, out, "GOODSIG")
	})

	t.Run("ssh", func(t *testing.T) {
		requireBins(t, "git", "ssh-keygen")
		k := testsigning.LoadSSH(t)
		signer, err := signing.NewSSHSigner(k.PrivateKey)
		require.NoError(t, err)
		repo, sha := signAndStore(t, signer)

		allowed := filepath.Join(t.TempDir(), "allowed_signers")
		require.NoError(t, os.WriteFile(allowed,
			[]byte(signerEmail+" namespaces=\"git\" "+string(k.PublicLine)), 0o644))
		out := run(t, repo, "git",
			"-c", "gpg.format=ssh",
			"-c", "gpg.ssh.allowedSignersFile="+allowed,
			"verify-commit", "--raw", sha)
		require.Contains(t, out, "Good \"git\" signature")
	})
}

func signAndStore(t *testing.T, signer signing.Signer) (repo, sha string) {
	t.Helper()
	emptyTree, err := hash.FromHex("4b825dc642cb6eb9a060e54bf8d69288fbee4904")
	require.NoError(t, err)
	ident := &protocol.Identity{Name: "Nanogit Signer", Email: signerEmail, Timestamp: 1234567890, Timezone: "+0000"}
	c := &protocol.PackfileCommit{Tree: emptyTree, Parent: hash.Zero, Author: ident, Committer: ident, Message: "verify roundtrip\n"}
	unsignedBytes := c.Build(false)
	sig, err := signer.Sign(unsignedBytes)
	require.NoError(t, err)
	c.Signature = sig
	signed := c.Build(true)

	repo = filepath.Join(t.TempDir(), "repo.git")
	run(t, "", "git", "init", "--bare", repo)

	cmd := exec.Command("git", "-C", repo, "hash-object", "-w", "-t", "commit", "--stdin")
	cmd.Stdin = bytes.NewReader(signed)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "hash-object: %s", out)
	sha = strings.TrimSpace(string(out))

	want, err := protocol.Object(crypto.SHA1, protocol.ObjectTypeCommit, signed)
	require.NoError(t, err)
	require.Equal(t, want.String(), sha)
	return repo, sha
}

func requireBins(t *testing.T, names ...string) {
	t.Helper()
	for _, n := range names {
		if _, err := exec.LookPath(n); err != nil {
			t.Skipf("required binary %q not found in PATH: %v", n, err)
		}
	}
}

func run(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s %s: %s", name, strings.Join(args, " "), out)
	return string(out)
}
