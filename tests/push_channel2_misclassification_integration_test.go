package integration_test

import (
	"errors"
	"strings"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// FIXME(grafana/grafana#124392): This spec documents the current, buggy
// behavior of nanogit's receive-pack response classifier. It is the
// regression baseline for the upcoming fix and MUST be inverted once that
// fix lands.
//
// Issue: https://github.com/grafana/grafana/issues/124392
//
// Bug summary
// -----------
// The receive-pack response parser (protocol/pack.go: detectError /
// isErrorOrFatalMessageOptimized) treats any pkt-line whose payload (or
// side-band-wrapped payload) starts with "fatal:" or "error:" as a fatal
// protocol error, regardless of which side-band channel carried it.
//
// Per gitprotocol-pack, side-band channel 2 (0x02) is the *informational*
// stream that stock `git push` renders as `remote: ...` lines. Only
// channel 1 (report-status / report-status-v2) and channel 3 (true fatal
// protocol errors) should drive success/failure classification of a push.
//
// The original report is GitLab CE 17.1.1 emitting
//
//	fatal: cannot exec 'exit 0 #': No such file or directory
//
// on channel 2 because the Meson-built Git records "/usr/bin/sh" as the
// hook-stub interpreter and the host (Ubuntu/Debian without UsrMerge)
// only ships "/bin/sh". The push itself succeeds — channel 1 carries
// "unpack ok" + "ok refs/heads/<branch>", the commit lands on the remote,
// and stock `git push` exits 0. Only nanogit promotes the channel-2
// warning into an error.
//
// Repro recipe used below
// -----------------------
// We can't easily ship GitLab CE in testcontainers, but the bug is
// triggered by the *channel-2 substring*, not by the specific message or
// server. A pre-receive hook that writes a "fatal: ..." line to stderr
// and exits 0 lets git-receive-pack complete normally (channel 1 reports
// "unpack ok" + "ok refs/heads/<branch>") while channel 2 still carries
// the offending line — exactly the GitLab #124392 byte pattern.
//
// Expected post-fix behavior (what this test must assert after the fix)
// ---------------------------------------------------------------------
//   - writer.Push(ctx) returns nil.
//   - The branch advances to the staged commit on the remote.
//   - The channel-2 "fatal:" line is surfaced as informational context
//     (e.g. via RemoteRejectionError-style RemoteMessages on success, or
//     simply ignored), never as a returned error.
var _ = Describe("Receive-pack channel-2 misclassification (issue #124392)", func() {
	author := nanogit.Author{Name: "Test Author", Email: "test@example.com", Time: time.Now()}
	committer := nanogit.Committer{Name: "Test Committer", Email: "test@example.com", Time: time.Now()}

	// The exact line from grafana/grafana#124392. The bug is triggered by
	// the "fatal:" substring on channel 2, so any text containing it
	// would reproduce — using the literal message keeps the test
	// self-documenting against the issue.
	const channel2FatalLine = "fatal: cannot exec 'exit 0 #': No such file or directory"

	It("currently misclassifies a channel-2 'fatal:' line as a push error, even though channel 1 reports success", func() {
		client, repo, local, _ := QuickSetup()

		By("Seeding the repo with a main branch")
		_, err := local.Git("branch", "-M", "main")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("push", "-u", "origin", "main", "--force")
		Expect(err).NotTo(HaveOccurred())

		mainRef, err := client.GetRef(ctx, "refs/heads/main")
		Expect(err).NotTo(HaveOccurred())

		// Pre-receive hook: print a "fatal:" line to stderr (which the
		// server multiplexes onto side-band channel 2) but exit 0 so
		// receive-pack still reports "unpack ok" + "ok refs/heads/main"
		// on channel 1. This mimics the GitLab byte pattern from #124392.
		hookScript := "#!/bin/sh\n" +
			"echo \"" + channel2FatalLine + "\" 1>&2\n" +
			"exit 0\n"

		By("Installing a pre-receive hook that emits 'fatal:' on stderr but exits 0")
		Expect(setRepoServerHook(ctx, gitServer, repo, "pre-receive", hookScript)).To(Succeed())

		By("Staging a commit and pushing")
		writer, err := client.NewStagedWriter(ctx, mainRef)
		Expect(err).NotTo(HaveOccurred())

		_, err = writer.CreateBlob(ctx, "issue-124392.txt", []byte("commit lands despite the channel-2 noise"))
		Expect(err).NotTo(HaveOccurred())
		commit, err := writer.Commit(ctx, "issue-124392 repro", author, committer)
		Expect(err).NotTo(HaveOccurred())

		pushErr := writer.Push(ctx)

		// FIXME(grafana/grafana#124392): These assertions encode the
		// *current, incorrect* behavior so this regression test passes
		// against unfixed main. When the parser fix lands, replace this
		// whole block with:
		//
		//   Expect(pushErr).To(Succeed())
		//   updated, err := client.GetRef(ctx, "refs/heads/main")
		//   Expect(err).NotTo(HaveOccurred())
		//   Expect(updated.Hash).To(Equal(commit.Hash))
		//
		// and (optionally) assert the channel-2 line is exposed as a
		// non-error informational message rather than dropped silently.
		By("Asserting the current (buggy) behavior: push fails with the channel-2 fatal text surfaced as a server error")
		Expect(pushErr).To(HaveOccurred(),
			"pre-fix: nanogit promotes any channel-2 'fatal:' line into a returned error")

		var serverErr *protocol.GitServerError
		Expect(errors.As(pushErr, &serverErr)).To(BeTrue(),
			"pre-fix: misclassified channel-2 line surfaces as GitServerError, got %T: %v", pushErr, pushErr)
		Expect(serverErr.ErrorType).To(Equal("fatal"))
		Expect(serverErr.Message).To(ContainSubstring("cannot exec 'exit 0 #'"))

		By("Confirming the push actually succeeded at the protocol level — the commit landed on the remote")
		// This is the canary that proves the bug: channel 1 reported
		// success (unpack ok + ok refs/heads/main) and the ref advanced,
		// even though nanogit returned an error to the caller.
		_, err = local.Git("fetch", "origin", "main")
		Expect(err).NotTo(HaveOccurred())
		remoteHead, err := local.Git("rev-parse", "origin/main")
		Expect(err).NotTo(HaveOccurred())
		Expect(strings.TrimSpace(remoteHead)).To(Equal(commit.Hash.String()),
			"the commit must have landed on origin/main — proving the returned error is a misclassification, not a real failure")

		remoteTree, err := local.Git("ls-tree", "-r", "--name-only", "origin/main")
		Expect(err).NotTo(HaveOccurred())
		Expect(strings.Split(strings.TrimSpace(remoteTree), "\n")).To(ContainElement("issue-124392.txt"),
			"the staged file must be present in the remote tree")
	})
})
