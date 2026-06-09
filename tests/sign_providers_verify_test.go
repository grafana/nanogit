package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
)

const (
	sigPGP  = "PGP"
	sigSSH  = "SSH"
	sigX509 = "X509"
)

type verification struct {
	Type     string
	Verified bool
	Reason   string
}

type getVerificationFunc func(t *testing.T, sha string) *verification

type signTest struct {
	Repo, User, Token, Email string
	Provider                 string
	GPGKey, SSHKey           []byte
	SMIMEKey, SMIMECert      []byte
	Cleanup                  bool

	client          nanogit.Client
	owner, repo     string
	getVerification getVerificationFunc
	branch          string
}

func TestSignProvidersVerify(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testproviders suite in short mode")
	}
	s := &signTest{
		Repo:      os.Getenv("TEST_REPO"),
		User:      os.Getenv("TEST_USER"),
		Token:     os.Getenv("TEST_TOKEN"),
		Email:     os.Getenv("TEST_SIGN_EMAIL"),
		Provider:  os.Getenv("TEST_PROVIDER"),
		GPGKey:    []byte(os.Getenv("TEST_GPG_KEY")),
		SSHKey:    []byte(os.Getenv("TEST_SSH_KEY")),
		SMIMEKey:  []byte(os.Getenv("TEST_SMIME_KEY")),
		SMIMECert: []byte(os.Getenv("TEST_SMIME_CERT")),
		Cleanup:   os.Getenv("TEST_CLEANUP") != "false",
	}
	if s.Repo == "" || s.Token == "" || s.User == "" {
		t.Skip("Skipping testproviders suite: TEST_REPO/TEST_TOKEN/TEST_USER not set")
	}
	if s.Email == "" {
		t.Skip("Skipping signature verification: TEST_SIGN_EMAIL not set")
	}

	switch s.Provider {
	case "github":
		s.getVerification = s.githubVerification
	case "gitlab":
		s.getVerification = s.gitlabVerification
	case "bitbucket":
		t.Log("skipping verification for bitbucket")
	default:
		t.Fatalf("unsupported TEST_PROVIDER %q (want github/gitlab/bitbucket)", s.Provider)
	}

	u, err := url.Parse(s.Repo)
	require.NoError(t, err)
	s.owner = path.Base(path.Dir(u.Path))
	s.repo = strings.TrimSuffix(path.Base(u.Path), ".git")

	client, err := nanogit.NewHTTPClient(s.Repo, options.WithBasicAuth(s.User, s.Token))
	require.NoError(t, err)
	s.client = client

	ctx := t.Context()
	s.branch = fmt.Sprintf("sign-verify-%d", time.Now().UnixNano())
	mainRef, err := client.GetRef(ctx, "refs/heads/main")
	require.NoError(t, err)
	require.NoError(t, client.CreateRef(ctx, nanogit.Ref{Name: "refs/heads/" + s.branch, Hash: mainRef.Hash}))
	if s.Cleanup {
		t.Cleanup(func() {
			if err := client.DeleteRef(context.WithoutCancel(ctx), "refs/heads/"+s.branch); err != nil {
				t.Logf("cleanup: delete branch %s: %v", s.branch, err)
			}
		})
	}

	t.Run("gpg", func(t *testing.T) {
		if len(s.GPGKey) == 0 {
			t.Skip("TEST_GPG_KEY not set")
		}
		sha := s.push(t, nanogit.WithGPGSigner(s.GPGKey))
		if s.getVerification == nil {
			return
		}
		v := s.getVerification(t, sha)
		require.Equal(t, sigPGP, v.Type)
		require.True(t, v.Verified, "gpg signature not verified: %s", v.Reason)
	})
	t.Run("ssh", func(t *testing.T) {
		if len(s.SSHKey) == 0 {
			t.Skip("TEST_SSH_KEY not set")
		}
		sha := s.push(t, nanogit.WithSSHSigner(s.SSHKey))
		if s.getVerification == nil {
			return
		}
		v := s.getVerification(t, sha)
		require.Equal(t, sigSSH, v.Type)
		require.True(t, v.Verified, "ssh signature not verified: %s", v.Reason)
	})
	t.Run("smime", func(t *testing.T) {
		if len(s.SMIMEKey) == 0 || len(s.SMIMECert) == 0 {
			t.Skip("TEST_SMIME_KEY/TEST_SMIME_CERT not set")
		}
		sha := s.push(t, nanogit.WithSMIMESigner(s.SMIMEKey, s.SMIMECert))
		if s.getVerification == nil {
			return
		}
		v := s.getVerification(t, sha)
		// Only the signature type is asserted, not Verified: S/MIME verification requires a cert
		// chaining to a CA in the provider's trust store (GitHub uses the Mozilla/Debian roots).
		// Our test cert is self-signed, so the commit signs fine but providers mark it untrusted.
		require.Equal(t, sigX509, v.Type)
	})
}

func (s *signTest) push(t *testing.T, opt nanogit.WriterOption) string {
	t.Helper()
	ctx := t.Context()

	branchRef, err := s.client.GetRef(ctx, "refs/heads/"+s.branch)
	require.NoError(t, err)
	writer, err := s.client.NewStagedWriter(ctx, branchRef, opt)
	require.NoError(t, err)

	_, err = writer.CreateBlob(ctx, fmt.Sprintf("sign-test-%s.txt", t.Name()), []byte(t.Name()))
	require.NoError(t, err)
	when := time.Now()
	ident := nanogit.Author{Name: "Nanogit Signer", Email: s.Email, Time: when}
	commit, err := writer.Commit(ctx, fmt.Sprintf("signed commit (%s)\n", t.Name()), ident,
		nanogit.Committer{Name: ident.Name, Email: ident.Email, Time: when})
	require.NoError(t, err)
	require.NoError(t, writer.Push(ctx))
	return commit.Hash.String()
}

func (s *signTest) githubVerification(t *testing.T, sha string) *verification {
	t.Helper()
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", s.owner, s.repo, sha)
	req, err := http.NewRequestWithContext(t.Context(), "GET", endpoint, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "github commit lookup: %s", body)

	var got struct {
		Commit struct {
			Verification struct {
				Verified  bool   `json:"verified"`
				Reason    string `json:"reason"`
				Signature string `json:"signature"`
			} `json:"verification"`
		} `json:"commit"`
	}
	require.NoError(t, json.Unmarshal(body, &got))

	v := got.Commit.Verification
	var typ string
	switch {
	case strings.Contains(v.Signature, "-----BEGIN PGP SIGNATURE-----"):
		typ = sigPGP
	case strings.Contains(v.Signature, "-----BEGIN SSH SIGNATURE-----"):
		typ = sigSSH
	case strings.Contains(v.Signature, "-----BEGIN SIGNED MESSAGE-----"):
		typ = sigX509
	}
	return &verification{Type: typ, Verified: v.Verified, Reason: v.Reason}
}

func (s *signTest) gitlabVerification(t *testing.T, sha string) *verification {
	t.Helper()
	project := url.PathEscape(s.owner + "/" + s.repo)
	endpoint := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/repository/commits/%s/signature", project, sha)
	req, err := http.NewRequestWithContext(t.Context(), "GET", endpoint, nil)
	require.NoError(t, err)
	req.Header.Set("PRIVATE-TOKEN", s.Token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "gitlab signature lookup: %s", body)

	var got struct {
		VerificationStatus string `json:"verification_status"`
		SignatureType      string `json:"signature_type"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	return &verification{
		Type:     got.SignatureType,
		Verified: got.VerificationStatus == "verified",
		Reason:   got.VerificationStatus,
	}
}
