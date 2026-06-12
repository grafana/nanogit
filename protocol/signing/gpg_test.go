package signing_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol/signing"
	"github.com/grafana/nanogit/protocol/signing/testsigning"
)

func TestGPGSignerVerify(t *testing.T) {
	t.Setenv("TMPDIR", "/tmp")
	home := t.TempDir()
	t.Setenv("GNUPGHOME", home)

	gpg := testsigning.LoadGPG(t)
	signer, err := signing.NewGPGSigner(gpg.ArmoredKey)
	require.NoError(t, err)
	repo, sha := signAndStore(t, signer)

	pub := filepath.Join(home, "gpg.pub.asc")
	require.NoError(t, os.WriteFile(pub, gpg.ArmoredPublic, 0o644))
	run(t, "", "gpg", "--batch", "--import", pub)
	out := run(t, repo, "git", "verify-commit", "--raw", sha)
	require.Contains(t, out, "GOODSIG")
}

func TestNewGPGSigner(t *testing.T) {
	t.Run("invalid armor", func(t *testing.T) {
		_, err := signing.NewGPGSigner([]byte("not a key"))
		require.Error(t, err)
	})

	t.Run("public key only", func(t *testing.T) {
		gpg := testsigning.LoadGPG(t)
		_, err := signing.NewGPGSigner(gpg.ArmoredPublic)
		require.ErrorContains(t, err, "private component")
	})
}

func TestGPGSignerSign(t *testing.T) {
	gpg := testsigning.LoadGPG(t)
	signer, err := signing.NewGPGSigner(gpg.ArmoredKey)
	require.NoError(t, err)
	sig, err := signer.Sign([]byte("hello"))
	require.NoError(t, err)
	require.Contains(t, sig, "-----BEGIN PGP SIGNATURE-----")
}
