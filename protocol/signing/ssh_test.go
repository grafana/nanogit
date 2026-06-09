package signing_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol/signing"
	"github.com/grafana/nanogit/protocol/signing/testsigning"
)

func TestSSHSignerVerify(t *testing.T) {
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
	require.Contains(t, out, "Good \"git\" signature")
}

func TestNewSSHSigner(t *testing.T) {
	_, err := signing.NewSSHSigner([]byte("not a key"))
	require.Error(t, err)
}

func TestSSHSignerSign(t *testing.T) {
	k := testsigning.LoadSSH(t)
	signer, err := signing.NewSSHSigner(k.PrivateKey)
	require.NoError(t, err)
	sig, err := signer.Sign([]byte("hello"))
	require.NoError(t, err)
	require.Contains(t, sig, "-----BEGIN SSH SIGNATURE-----")
}
