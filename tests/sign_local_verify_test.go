package integration_test

import (
	"bytes"
	"crypto"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/protocol/signature"
)

const signerEmail = "signer@test.invalid"

func TestSignLocalVerify_GPG(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping local git verify in short mode")
	}
	requireBins(t, "git", "gpg")

	gnupghome := mkShortTempDir(t, "ng-gpg-")
	t.Setenv("GNUPGHOME", gnupghome)

	gpg := gittest.LoadGPG(t)
	runOK(t, "", "gpg", "--batch", "--import", gpg.KeyPath)
	signer, err := signature.NewGPGSigner(gpg.ArmoredKey)
	require.NoError(t, err)
	commit := signEmptyCommit(t, signer)
	commitBytes := commit.Build()

	tmp := t.TempDir()
	repo := initBareRepo(t, tmp)
	commitSHA := writeGitObject(t, repo, "commit", commitBytes)

	gitSHA, err := computeHash(commitBytes)
	require.NoError(t, err)
	require.Equal(t, gitSHA, commitSHA)

	out, err := runIn(repo, "git", "verify-commit", "--raw", commitSHA)
	require.NoError(t, err, "verify-commit output:\n%s", out)
	require.Contains(t, out, "GOODSIG", "expected GOODSIG status in verify-commit --raw output")
}

func TestSignLocalVerify_SSH(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping local git verify in short mode")
	}
	requireBins(t, "git", "ssh-keygen")

	tmp := t.TempDir()
	k := gittest.LoadSSH(t)

	allowed := filepath.Join(tmp, "allowed_signers")
	require.NoError(t, os.WriteFile(allowed,
		[]byte(signerEmail+" namespaces=\"git\" "+string(k.PublicLine)), 0o644))

	signer, err := signature.NewSSHSigner(k.PrivateKey)
	require.NoError(t, err)
	commit := signEmptyCommit(t, signer)
	repo := initBareRepo(t, tmp)
	commitSHA := writeGitObject(t, repo, "commit", commit.Build())

	out, err := runIn(repo, "git",
		"-c", "gpg.format=ssh",
		"-c", "gpg.ssh.allowedSignersFile="+allowed,
		"verify-commit", "--raw", commitSHA)
	require.NoError(t, err, "verify-commit output:\n%s", out)
	require.Contains(t, out, "Good \"git\" signature")
}

func TestSignLocalVerify_SMIME(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping local git verify in short mode")
	}
	requireBins(t, "git", "gpgsm")

	gnupghome := mkShortTempDir(t, "ng-gpgsm-")
	t.Setenv("GNUPGHOME", gnupghome)

	require.NoError(t, os.WriteFile(filepath.Join(gnupghome, "gpg-agent.conf"),
		[]byte("allow-mark-trusted\n"), 0o600))

	s := gittest.LoadSMIME(t)
	runOK(t, "", "gpgsm", "--batch", "--import", s.CertPath)

	fpRaw := runOut(t, "", "gpgsm", "--batch", "--with-colons", "--list-keys")
	fp := extractGPGSMFingerprint(t, fpRaw)
	require.NoError(t, os.WriteFile(filepath.Join(gnupghome, "trustlist.txt"),
		[]byte(fp+" S\n"), 0o600))

	signer, err := signature.NewSMIMESigner(s.KeyPEM, s.CertPEM)
	require.NoError(t, err)
	commit := signEmptyCommit(t, signer)

	repo := initBareRepo(t, t.TempDir())
	_ = writeGitObject(t, repo, "commit", commit.Build())

	tmp := t.TempDir()
	sigPath := filepath.Join(tmp, "sig.pem")
	dataPath := filepath.Join(tmp, "unsigned.bin")
	require.NoError(t, os.WriteFile(sigPath, []byte(commit.Signature), 0o600))
	unsigned := *commit
	unsigned.Signature = ""
	require.NoError(t, os.WriteFile(dataPath, unsigned.Build(), 0o600))

	out, err := runIn("", "gpgsm", "--status-fd=1", "--verify", sigPath, dataPath)
	require.NoError(t, err, "gpgsm --verify output:\n%s", out)
	require.Contains(t, out, "GOODSIG")
}

func signEmptyCommit(t *testing.T, signer signature.Signer) *protocol.PackfileCommit {
	t.Helper()
	emptyTree, err := hash.FromHex("4b825dc642cb6eb9a060e54bf8d69288fbee4904")
	require.NoError(t, err)
	ident := &protocol.Identity{
		Name: "Nanogit Signer", Email: signerEmail,
		Timestamp: 1234567890, Timezone: "+0000",
	}
	c := &protocol.PackfileCommit{
		Tree: emptyTree, Parent: hash.Zero,
		Author: ident, Committer: ident,
		Message: "verify roundtrip\n",
	}
	sig, err := signer.Sign(c.Build())
	require.NoError(t, err)
	c.Signature = sig
	return c
}

func initBareRepo(t *testing.T, parent string) string {
	t.Helper()
	repo := filepath.Join(parent, "repo.git")
	runOK(t, "", "git", "init", "--bare", repo)
	return repo
}

func writeGitObject(t *testing.T, repo, objType string, data []byte) string {
	t.Helper()
	cmd := exec.Command("git", "-C", repo, "hash-object", "-w", "-t", objType, "--stdin")
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "hash-object: %s", out)
	return strings.TrimSpace(string(out))
}

func computeHash(commitBytes []byte) (string, error) {
	h, err := protocol.Object(crypto.SHA1, protocol.ObjectTypeCommit, commitBytes)
	if err != nil {
		return "", err
	}
	return h.String(), nil
}

func extractGPGSMFingerprint(t *testing.T, colons string) string {
	t.Helper()
	for line := range strings.SplitSeq(colons, "\n") {
		if strings.HasPrefix(line, "fpr:") {
			cols := strings.Split(line, ":")
			require.Greater(t, len(cols), 9, "unexpected fpr line: %q", line)
			return cols[9]
		}
	}
	t.Fatalf("no fpr line in gpgsm --list-keys output:\n%s", colons)
	return ""
}

func requireBins(t *testing.T, names ...string) {
	t.Helper()
	for _, n := range names {
		if _, err := exec.LookPath(n); err != nil {
			t.Skipf("required binary %q not found in PATH: %v", n, err)
		}
	}
}

func mkShortTempDir(t *testing.T, prefix string) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", prefix)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	require.NoError(t, os.Chmod(dir, 0o700))
	return dir
}

func runOK(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s %s: %s", name, strings.Join(args, " "), out)
}

func runOut(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	require.NoError(t, err)
	return string(out)
}

func runIn(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
