// Package nanogit is a lightweight Git client for services that read from and
// write to remote repositories over HTTPS — with no local clone, no .git
// directory, and no git binary. It speaks the Git Smart HTTP protocol v2
// directly, so it works with GitHub, GitLab, Bitbucket, Gitea, and any other
// server that supports protocol v2.
//
// nanogit is deliberately narrow: it implements the essential server-side
// operations (refs, blobs, trees, commits, diffs, staged writes, and
// path-filtered shallow clones) and nothing else. For local worktrees,
// merges, SSH, or full Git functionality, use the git CLI or a
// general-purpose implementation such as go-git.
//
// # Reading
//
// Create a [Client] with [NewHTTPClient], resolve a ref, and read objects:
//
//	client, err := nanogit.NewHTTPClient(
//	    "https://github.com/owner/repo.git",
//	    options.WithBasicAuth("git", token),
//	)
//	if err != nil {
//	    return err
//	}
//	ref, err := client.GetRef(ctx, "refs/heads/main")
//	if err != nil {
//	    return err
//	}
//	commit, err := client.GetCommit(ctx, ref.Hash)
//	if err != nil {
//	    return err
//	}
//	blob, err := client.GetBlobByPath(ctx, commit.Tree, "README.md")
//	if err != nil {
//	    return err
//	}
//
// # Writing
//
// Writes are transactional: a [StagedWriter] stages any number of changes,
// commits them, and pushes the result as a single atomic ref update:
//
//	writer, err := client.NewStagedWriter(ctx, ref)
//	if err != nil {
//	    return err
//	}
//	if _, err := writer.CreateBlob(ctx, "hello.txt", []byte("hi\n")); err != nil {
//	    return err
//	}
//	if _, err := writer.Commit(ctx, "Add hello.txt", author, committer); err != nil {
//	    return err
//	}
//	if err := writer.Push(ctx); err != nil {
//	    return err
//	}
//
// # Configuration
//
// Construction-time behavior (authentication, response size limits,
// receive-pack capabilities) is configured with functional options from
// [github.com/grafana/nanogit/options]. Cross-cutting concerns are injected
// per call through the context: logging via
// [github.com/grafana/nanogit/log.ToContext], retries via
// [github.com/grafana/nanogit/retry.ToContext], and object caching via
// [github.com/grafana/nanogit/storage.ToContext].
//
// Generated mocks for [Client] and [StagedWriter] live in
// [github.com/grafana/nanogit/mocks], and
// [github.com/grafana/nanogit/gittest] runs integration tests against a real
// containerized Git server.
//
// Full documentation, guides, and benchmarks: https://grafana.github.io/nanogit
package nanogit
