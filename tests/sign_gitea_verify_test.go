package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol/signature/testsigning"
)

func TestSignGiteaVerify_GPG(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping gitea verify in short mode")
	}

	ctx := t.Context()
	server, err := gittest.NewServer(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Cleanup() })

	user, err := server.CreateUser(ctx)
	require.NoError(t, err)
	user.Token, err = server.CreateToken(ctx, user.Username)
	require.NoError(t, err)

	gpg := testsigning.LoadGPG(t)
	setUserPrimaryEmail(t, server.URL(), user, signerEmail)
	uploadGPGKey(t, server.URL(), user.Token, gpg.ArmoredPublic)

	repo, err := server.CreateRepo(ctx, "signing-verify", user)
	require.NoError(t, err)
	local, err := gittest.NewLocalRepo(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = local.Cleanup() })
	conn, err := local.InitWithRemote(user, repo)
	require.NoError(t, err)

	client, err := nanogit.NewHTTPClient(conn.URL,
		options.WithBasicAuth(conn.Username, conn.Password))
	require.NoError(t, err)
	ref, err := client.GetRef(ctx, "refs/heads/main")
	require.NoError(t, err)

	writer, err := client.NewStagedWriter(ctx, ref, nanogit.WithGPGSigner(gpg.ArmoredKey))
	require.NoError(t, err)
	_, err = writer.CreateBlob(ctx, "sign-test.txt", []byte("hi"))
	require.NoError(t, err)
	when := time.Now()
	ident := nanogit.Author{Name: "Nanogit Signer", Email: signerEmail, Time: when}
	commit, err := writer.Commit(ctx, "signed commit\n", ident,
		nanogit.Committer{Name: ident.Name, Email: ident.Email, Time: when})
	require.NoError(t, err)
	require.NoError(t, writer.Push(ctx))

	verified, reason := giteaCommitVerification(t, server.URL(), user.Token,
		user.Username, repo.Name, commit.Hash.String())
	require.True(t, verified, "Gitea reported the commit as unverified (reason: %q)", reason)
}

func setUserPrimaryEmail(t *testing.T, baseURL string, user *gittest.User, email string) {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"email":      email,
		"source_id":  0,
		"login_name": user.Username,
	})
	require.NoError(t, err)
	req, err := http.NewRequestWithContext(t.Context(), "PATCH",
		baseURL+"/api/v1/admin/users/"+user.Username, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user.Username, user.Password)
	doOK(t, req, "set primary email")
	user.Email = email
}

func uploadGPGKey(t *testing.T, baseURL, token string, armoredPublic []byte) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"armored_public_key": string(armoredPublic)})
	require.NoError(t, err)
	req, err := http.NewRequestWithContext(t.Context(), "POST", baseURL+"/api/v1/user/gpg_keys", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	doOK(t, req, "upload gpg key")
}

func giteaCommitVerification(t *testing.T, baseURL, token, owner, repo, sha string) (bool, string) {
	t.Helper()
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/git/commits/%s?verification=true", baseURL, owner, repo, sha)
	req, err := http.NewRequestWithContext(t.Context(), "GET", url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, http.StatusOK, resp.StatusCode, "gitea commit lookup: %s", body)

	var got struct {
		Commit struct {
			Verification struct {
				Verified bool   `json:"verified"`
				Reason   string `json:"reason"`
			} `json:"verification"`
		} `json:"commit"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	return got.Commit.Verification.Verified, got.Commit.Verification.Reason
}

func doOK(t *testing.T, req *http.Request, what string) {
	t.Helper()
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		out, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s: %s: %s", what, resp.Status, out)
	}
}
