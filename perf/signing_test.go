package performance

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/internal/testsigning"
	"github.com/grafana/nanogit/perf/clients"
)

// signVariant is one (client, format) combination to benchmark.
type signVariant struct {
	client   string
	scenario string
	create   func(ctx context.Context, repoURL, path, content, message string) error
}

func TestSignedCommitPerformance(t *testing.T) {
	if os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		t.Skip("Performance tests disabled. Set RUN_PERFORMANCE_TESTS=true to run.")
	}

	ctx := context.Background()
	suite := getGlobalSuite()
	require.NotNil(t, suite, "Global suite not initialized - TestMain should have set this up")

	repo := suite.GetRepository("small")
	if repo == nil {
		t.Skip("small repository not available")
	}

	baseline := []signVariant{
		{"nanogit", "unsigned", clients.NewNanogitClient().CreateFile},
		{"go-git", "unsigned", clients.NewGoGitClient().CreateFile},
		{"git-cli", "unsigned", newGitCLI(t).CreateFile},
	}
	signed := buildSignedVariants(t)

	record := func(op string, variants []signVariant) {
		for _, v := range variants {
			for i := 0; i < 3; i++ {
				i := i
				suite.collector.RecordOperation(v.client, op, v.scenario, "small", 0, func() error {
					path := fmt.Sprintf("sign-perf/%s-%s-%d.txt", v.client, v.scenario, i)
					return v.create(ctx, repo.AuthURL(), path, "content", "perf signing")
				})
			}
		}
	}

	record("CreateFile", baseline)
	record("CreateFileSigned", signed)

	require.NoError(t, suite.collector.SaveReport("./reports"))
}

func buildSignedVariants(t *testing.T) []signVariant {
	t.Helper()
	gpg := testsigning.LoadGPG(t)
	sshKey := testsigning.LoadSSH(t)
	smime := testsigning.LoadSMIME(t)

	var variants []signVariant

	// nanogit signs all three formats natively.
	ngGPG := clients.NewNanogitClient()
	ngGPG.SetSignerOptions(nanogit.WithGPGSigner(gpg.ArmoredKey))
	ngSSH := clients.NewNanogitClient()
	ngSSH.SetSignerOptions(nanogit.WithSSHSigner(sshKey.PrivateKey))
	ngSMIME := clients.NewNanogitClient()
	ngSMIME.SetSignerOptions(nanogit.WithSMIMESigner(smime.KeyPEM, smime.CertPEM))
	variants = append(variants,
		signVariant{"nanogit", "gpg", ngGPG.CreateFile},
		signVariant{"nanogit", "ssh", ngSSH.CreateFile},
		signVariant{"nanogit", "smime", ngSMIME.CreateFile},
	)

	// go-git signs OpenPGP only.
	ggGPG := clients.NewGoGitClient()
	ggGPG.SetSignKey(gpg.Entity)
	variants = append(variants, signVariant{"go-git", "gpg", ggGPG.CreateFile})

	// git CLI signs all three via gpg / ssh-keygen / gpgsm.
	if _, err := exec.LookPath("gpg"); err == nil {
		home, fpr := gpgHome(t, gpg.KeyPath)
		cli := newGitCLI(t)
		cli.SetSigning([]string{
			"-c", "gpg.format=openpgp",
			"-c", "user.signingkey=" + fpr,
			"-c", "commit.gpgsign=true",
			"-c", "gpg.program=gpg",
		}, []string{"GNUPGHOME=" + home})
		variants = append(variants, signVariant{"git-cli", "gpg", cli.CreateFile})
	}
	if _, err := exec.LookPath("ssh-keygen"); err == nil {
		cli := newGitCLI(t)
		cli.SetSigning([]string{
			"-c", "gpg.format=ssh",
			"-c", "user.signingkey=" + sshKey.PrivateKeyPath,
			"-c", "commit.gpgsign=true",
			"-c", "gpg.ssh.program=ssh-keygen",
		}, nil)
		variants = append(variants, signVariant{"git-cli", "ssh", cli.CreateFile})
	}
	if _, hasGpgsm := exec.LookPath("gpgsm"); hasGpgsm == nil {
		if _, hasSSL := exec.LookPath("openssl"); hasSSL == nil {
			home, fpr := gpgsmHome(t, smime.CertPath, smime.KeyPath)
			cli := newGitCLI(t)
			cli.SetSigning([]string{
				"-c", "gpg.format=x509",
				"-c", "gpg.x509.program=gpgsm",
				"-c", "user.signingkey=" + fpr,
				"-c", "commit.gpgsign=true",
			}, []string{"GNUPGHOME=" + home})
			variants = append(variants, signVariant{"git-cli", "smime", cli.CreateFile})
		}
	}

	return variants
}

func newGitCLI(t *testing.T) *clients.GitCLIClient {
	t.Helper()
	cli, err := clients.NewGitCLIClient()
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Cleanup() })
	return cli
}

func gpgHome(t *testing.T, keyPath string) (home, fpr string) {
	t.Helper()
	home = shortTempDir(t, "perf-gpg-")
	t.Setenv("GNUPGHOME", home)
	runOK(t, "gpg", "--batch", "--import", keyPath)
	out := runOut(t, "gpg", "--batch", "--with-colons", "--list-secret-keys")
	return home, colonField(t, out, "fpr")
}

func gpgsmHome(t *testing.T, certPath, keyPath string) (home, fpr string) {
	t.Helper()
	home = shortTempDir(t, "perf-gpgsm-")
	t.Setenv("GNUPGHOME", home)
	require.NoError(t, os.WriteFile(filepath.Join(home, "gpg-agent.conf"), []byte("allow-mark-trusted\n"), 0o600))

	// gpgsm needs the private key to sign; import a PKCS#12 bundle of key+cert.
	p12 := filepath.Join(home, "signer.p12")
	runOK(t, "openssl", "pkcs12", "-export", "-inkey", keyPath, "-in", certPath, "-out", p12, "-passout", "pass:")
	runOK(t, "gpgsm", "--batch", "--pinentry-mode", "loopback", "--passphrase", "", "--import", p12)

	out := runOut(t, "gpgsm", "--batch", "--with-colons", "--list-keys")
	fpr = colonField(t, out, "fpr")
	require.NoError(t, os.WriteFile(filepath.Join(home, "trustlist.txt"), []byte(fpr+" S\n"), 0o600))
	return home, fpr
}

func colonField(t *testing.T, colons, prefix string) string {
	t.Helper()
	for _, line := range strings.Split(colons, "\n") {
		if strings.HasPrefix(line, prefix+":") {
			cols := strings.Split(line, ":")
			require.Greater(t, len(cols), 9, "unexpected %s line: %q", prefix, line)
			return cols[9]
		}
	}
	t.Fatalf("no %s line found:\n%s", prefix, colons)
	return ""
}

func shortTempDir(t *testing.T, prefix string) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", prefix)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	require.NoError(t, os.Chmod(dir, 0o700))
	return dir
}

func runOK(t *testing.T, name string, args ...string) {
	t.Helper()
	out, err := exec.Command(name, args...).CombinedOutput()
	require.NoError(t, err, "%s %s: %s", name, strings.Join(args, " "), out)
}

func runOut(t *testing.T, name string, args ...string) string {
	t.Helper()
	out, err := exec.Command(name, args...).Output()
	require.NoError(t, err)
	return string(out)
}
