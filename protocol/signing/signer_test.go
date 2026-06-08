package signing_test

import (
	"bytes"
	"crypto/sha1"
	"fmt"
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

func TestVerifySignature(t *testing.T) {
	t.Setenv("TMPDIR", "/tmp")
	home := t.TempDir()
	t.Setenv("GNUPGHOME", home)

	t.Run("gpg", func(t *testing.T) {
		gpg := testsigning.LoadGPG(t)
		signer, err := signing.NewGPGSigner(gpg.ArmoredKey)
		require.NoError(t, err)
		repo, sha := signAndStore(t, signer)

		pub := filepath.Join(home, "gpg.pub.asc")
		require.NoError(t, os.WriteFile(pub, gpg.ArmoredPublic, 0o644))
		run(t, "", "gpg", "--batch", "--import", pub)
		out := run(t, repo, "git", "verify-commit", "--raw", sha)
		t.Log(string(out))
		require.Contains(t, out, "GOODSIG")
	})

	t.Run("ssh", func(t *testing.T) {
		k := testsigning.LoadSSH(t)
		signer, err := signing.NewSSHSigner(k.PrivateKey)
		require.NoError(t, err)
		repo, sha := signAndStore(t, signer)

		allowed := filepath.Join(t.TempDir(), "allowed_signers")
		require.NoError(t, os.WriteFile(allowed,
			[]byte("signer@test.invalid namespaces=\"git\" "+string(k.PublicLine)), 0o644))
		out := run(t, repo, "git",
			"-c", "gpg.format=ssh",
			"-c", "gpg.ssh.allowedSignersFile="+allowed,
			"verify-commit", "--raw", sha)
		t.Log(string(out))
		require.Contains(t, out, "Good \"git\" signature")
	})

	t.Run("smime", func(t *testing.T) {
		s := testsigning.LoadSMIME(t)
		signer, err := signing.NewSMIMESigner(s.KeyPEM, s.CertPEM)
		require.NoError(t, err)
		repo, sha := signAndStore(t, signer)

		require.NoError(t, os.WriteFile(filepath.Join(home, "gpgsm.conf"), []byte("disable-crl-checks\n"), 0o644))
		fpr := fmt.Sprintf("%X", sha1.Sum(s.Certificate.Raw))
		require.NoError(t, os.WriteFile(filepath.Join(home, "trustlist.txt"), []byte(fpr+" S relax\n"), 0o644))
		run(t, "", "gpgsm", "--batch", "--import", s.CertPath)
		out := run(t, repo, "git",
			"-c", "gpg.format=x509",
			"-c", "gpg.x509.program="+gpgsmShim(t),
			"verify-commit", "--raw", sha)
		t.Log(string(out))
		require.Contains(t, out, "GOODSIG")
	})
}

// gpgsmShim wraps gpgsm to rewrite git's stdin marker "-" to "/dev/stdin",
// which gpgsm requires for the detached payload.
func gpgsmShim(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gpgsm-shim")
	script := "#!/bin/sh\nfor a in \"$@\"; do [ \"$a\" = - ] && a=/dev/stdin; set -- \"$@\" \"$a\"; shift; done\nexec gpgsm \"$@\"\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

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
