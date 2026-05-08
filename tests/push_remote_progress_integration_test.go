package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/protocol"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// setRepoServerHook installs a server-side git hook on a Gitea repo
// via the admin API. Valid hook IDs are "pre-receive", "update", and
// "post-receive". The hook script runs inside the Gitea container on
// every push; stderr/stdout is relayed to the client over side-band
// channel 2 as `remote: …` lines, which is exactly the path the new
// RemoteRejectionError wrapper captures.
//
// Note: requires GITEA__security__DISABLE_GIT_HOOKS=false on the
// container (set in gittest.Server) and a site-admin user (test users
// are created with --admin).
func setRepoServerHook(ctx context.Context, server *gittest.Server, repo *gittest.RemoteRepository, hookID, content string) error {
	body, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/hooks/git/%s", server.URL(), repo.Owner, repo.Name, hookID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(repo.User.Username, repo.User.Password)

	cli := &http.Client{Timeout: 15 * time.Second}
	res, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode >= 300 {
		respBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("set %s hook: status %d: %s", hookID, res.StatusCode, respBody)
	}
	return nil
}

// These specs exercise the receive-pack error path against a real Git
// server (Gitea in testcontainers) when a server-side pre-receive hook
// rejects the push. They verify the end-to-end behavior added by the
// channel-2 capture: the hook's stderr text is attached to the push
// error as RemoteRejectionError.RemoteMessages, and the underlying
// typed error (GitReferenceUpdateError) survives via Unwrap.
var _ = Describe("Receive-pack remote progress on rejected push", func() {
	author := nanogit.Author{Name: "Test Author", Email: "test@example.com", Time: time.Now()}
	committer := nanogit.Committer{Name: "Test Committer", Email: "test@example.com", Time: time.Now()}

	It("attaches pre-receive hook stderr to the error and preserves the typed cause", func() {
		client, repo, local, _ := QuickSetup()

		By("Seeding the repo with a main branch")
		_, err := local.Git("branch", "-M", "main")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("push", "-u", "origin", "main", "--force")
		Expect(err).NotTo(HaveOccurred())

		mainRef, err := client.GetRef(ctx, "refs/heads/main")
		Expect(err).NotTo(HaveOccurred())

		const hookMsg = "test policy: pushes to this repo are denied for this test"
		hookScript := "#!/bin/sh\n" +
			"echo '" + hookMsg + "' 1>&2\n" +
			"exit 1\n"

		By("Installing a pre-receive hook that prints to stderr and rejects")
		Expect(setRepoServerHook(ctx, gitServer, repo, "pre-receive", hookScript)).To(Succeed())

		By("Staging a commit and pushing — expect rejection")
		writer, err := client.NewStagedWriter(ctx, mainRef)
		Expect(err).NotTo(HaveOccurred())

		_, err = writer.CreateBlob(ctx, "rejected.txt", []byte("should not land"))
		Expect(err).NotTo(HaveOccurred())
		_, err = writer.Commit(ctx, "rejected commit", author, committer)
		Expect(err).NotTo(HaveOccurred())

		pushErr := writer.Push(ctx)
		Expect(pushErr).To(HaveOccurred(), "hook should reject the push")

		By("Confirming the underlying typed error survives via Unwrap")
		var refErr *protocol.GitReferenceUpdateError
		Expect(errors.As(pushErr, &refErr)).To(BeTrue(),
			"expected GitReferenceUpdateError in chain, got %T: %v", pushErr, pushErr)
		Expect(refErr.RefName).To(Equal("refs/heads/main"))

		By("Confirming RemoteRejectionError carries the hook stderr line")
		var wrapped *protocol.RemoteRejectionError
		Expect(errors.As(pushErr, &wrapped)).To(BeTrue(),
			"expected RemoteRejectionError in chain, got %T: %v", pushErr, pushErr)
		Expect(wrapped.RemoteMessages).To(ContainElement(hookMsg),
			"expected hook stderr line in RemoteMessages, got %v", wrapped.RemoteMessages)

		By("Confirming the surfaced error string includes the remote: line")
		Expect(pushErr.Error()).To(ContainSubstring("remote: " + hookMsg))

		By("Confirming the branch did not advance — silent-success regression guard")
		currentMain, err := client.GetRef(ctx, "refs/heads/main")
		Expect(err).NotTo(HaveOccurred())
		Expect(currentMain.Hash).To(Equal(mainRef.Hash),
			"main must not have advanced; if it did, the hook rejection was silently swallowed")
	})

	It("captures multi-line hook output as separate remote: lines", func() {
		client, repo, local, _ := QuickSetup()

		_, err := local.Git("branch", "-M", "main")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("push", "-u", "origin", "main", "--force")
		Expect(err).NotTo(HaveOccurred())

		mainRef, err := client.GetRef(ctx, "refs/heads/main")
		Expect(err).NotTo(HaveOccurred())

		const (
			line1 = "GL-HOOK-ERR: Commit message must reference an issue."
			line2 = "GL-HOOK-ERR: See https://example.com/policy for details."
		)
		hookScript := "#!/bin/sh\n" +
			"echo '" + line1 + "' 1>&2\n" +
			"echo '" + line2 + "' 1>&2\n" +
			"exit 1\n"

		Expect(setRepoServerHook(ctx, gitServer, repo, "pre-receive", hookScript)).To(Succeed())

		writer, err := client.NewStagedWriter(ctx, mainRef)
		Expect(err).NotTo(HaveOccurred())

		_, err = writer.CreateBlob(ctx, "rejected.txt", []byte("should not land"))
		Expect(err).NotTo(HaveOccurred())
		_, err = writer.Commit(ctx, "rejected commit", author, committer)
		Expect(err).NotTo(HaveOccurred())

		pushErr := writer.Push(ctx)
		Expect(pushErr).To(HaveOccurred())

		var wrapped *protocol.RemoteRejectionError
		Expect(errors.As(pushErr, &wrapped)).To(BeTrue())
		Expect(wrapped.RemoteMessages).To(ContainElements(line1, line2))
	})
})

var _ = Describe("Receive-pack remote progress on successful push", func() {
	author := nanogit.Author{Name: "Test Author", Email: "test@example.com", Time: time.Now()}
	committer := nanogit.Committer{Name: "Test Committer", Email: "test@example.com", Time: time.Now()}

	It("does not fail when the remote emits 'fatal:' on side-band channel 2", func() {
		client, repo, local, _ := QuickSetup()

		By("Seeding the repo with a main branch")
		_, err := local.Git("branch", "-M", "main")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("push", "-u", "origin", "main", "--force")
		Expect(err).NotTo(HaveOccurred())

		mainRef, err := client.GetRef(ctx, "refs/heads/main")
		Expect(err).NotTo(HaveOccurred())

		const hookMsg = "fatal: cannot exec 'exit 0 #': No such file or directory"
		hookScript := "#!/bin/sh\n" +
			"echo '" + hookMsg + "' 1>&2\n" +
			"exit 0\n"

		By("Installing a hook that prints 'fatal:' to stderr but exits 0 (success)")
		Expect(setRepoServerHook(ctx, gitServer, repo, "pre-receive", hookScript)).To(Succeed())

		By("Staging a commit and pushing — expect success since exit 0")
		writer, err := client.NewStagedWriter(ctx, mainRef)
		Expect(err).NotTo(HaveOccurred())

		_, err = writer.CreateBlob(ctx, "success.txt", []byte("should land"))
		Expect(err).NotTo(HaveOccurred())
		commit, err := writer.Commit(ctx, "successful commit", author, committer)
		Expect(err).NotTo(HaveOccurred())

		pushErr := writer.Push(ctx)
		Expect(pushErr).NotTo(HaveOccurred(), "push should succeed despite 'fatal:' on channel 2")

		By("Confirming the branch did advance")
		currentMain, err := client.GetRef(ctx, "refs/heads/main")
		Expect(err).NotTo(HaveOccurred())
		Expect(currentMain.Hash).To(Equal(commit.Hash),
			"main must have advanced, reflecting successful push")
	})
})
