package signing_test

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol/signing"
	"github.com/grafana/nanogit/protocol/signing/testsigning"
)

func TestSMIMESignerVerify(t *testing.T) {
	t.Setenv("TMPDIR", "/tmp")
	home := t.TempDir()
	t.Setenv("GNUPGHOME", home)

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
	require.Contains(t, out, "GOODSIG")
}

func TestNewSMIMESigner(t *testing.T) {
	s := testsigning.LoadSMIME(t)

	t.Run("invalid certificate", func(t *testing.T) {
		_, err := signing.NewSMIMESigner(s.KeyPEM, []byte("not a cert"))
		require.ErrorContains(t, err, "certificate")
	})

	t.Run("invalid private key", func(t *testing.T) {
		_, err := signing.NewSMIMESigner([]byte("not a key"), s.CertPEM)
		require.ErrorContains(t, err, "private key")
	})
}

func TestSMIMESignerSign(t *testing.T) {
	s := testsigning.LoadSMIME(t)
	signer, err := signing.NewSMIMESigner(s.KeyPEM, s.CertPEM)
	require.NoError(t, err)
	sig, err := signer.Sign([]byte("hello"))
	require.NoError(t, err)
	require.Contains(t, sig, "-----BEGIN SIGNED MESSAGE-----")
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
