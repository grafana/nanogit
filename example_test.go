package nanogit_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
)

// ExampleNewHTTPClient creates a client for a repository and verifies the
// server speaks Git protocol v2.
func ExampleNewHTTPClient() {
	ctx := context.Background()

	client, err := nanogit.NewHTTPClient(
		"https://github.com/owner/repo.git",
		options.WithBasicAuth("git", "your-token"),
	)
	if err != nil {
		log.Fatal(err)
	}

	compatible, err := client.IsServerCompatible(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("server supports protocol v2:", compatible)
}

// ExampleClient_GetBlobByPath reads a single file from a repository without
// cloning it: resolve the ref, load its commit, and walk the tree to a path.
func ExampleClient_GetBlobByPath() {
	ctx := context.Background()

	client, err := nanogit.NewHTTPClient("https://github.com/owner/repo.git")
	if err != nil {
		log.Fatal(err)
	}

	ref, err := client.GetRef(ctx, "refs/heads/main")
	if err != nil {
		log.Fatal(err)
	}

	commit, err := client.GetCommit(ctx, ref.Hash)
	if err != nil {
		log.Fatal(err)
	}

	blob, err := client.GetBlobByPath(ctx, commit.Tree, "docs/README.md")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d bytes at %s\n", len(blob.Content), blob.Hash)
}

// ExampleClient_NewStagedWriter stages a new file, commits it, and pushes the
// result as a single atomic ref update — entirely over HTTPS.
func ExampleClient_NewStagedWriter() {
	ctx := context.Background()

	client, err := nanogit.NewHTTPClient(
		"https://github.com/owner/repo.git",
		options.WithBasicAuth("git", "your-token"),
	)
	if err != nil {
		log.Fatal(err)
	}

	ref, err := client.GetRef(ctx, "refs/heads/main")
	if err != nil {
		log.Fatal(err)
	}

	writer, err := client.NewStagedWriter(ctx, ref)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := writer.CreateBlob(ctx, "hello/world.txt", []byte("pushed without a checkout\n")); err != nil {
		log.Fatal(err)
	}

	author := nanogit.Author{Name: "Example", Email: "example@example.com", Time: time.Now()}
	committer := nanogit.Committer{Name: "Example", Email: "example@example.com", Time: time.Now()}
	commit, err := writer.Commit(ctx, "Add hello/world.txt", author, committer)
	if err != nil {
		log.Fatal(err)
	}

	if err := writer.Push(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("pushed", commit.Hash)
}

// ExampleClient_Clone writes a path-filtered snapshot of a repository to a
// local directory, fetching only the objects those paths need. No .git
// directory is created.
func ExampleClient_Clone() {
	ctx := context.Background()

	client, err := nanogit.NewHTTPClient("https://github.com/owner/repo.git")
	if err != nil {
		log.Fatal(err)
	}

	ref, err := client.GetRef(ctx, "refs/heads/main")
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.Clone(ctx, nanogit.CloneOptions{
		Path:         "/tmp/repo-docs",
		Hash:         ref.Hash,
		IncludePaths: []string{"docs/**"},
		BatchSize:    50,
		Concurrency:  8,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %d of %d files to %s\n", result.FilteredFiles, result.TotalFiles, result.Path)
}

// ExampleClient_CompareCommits lists the files that changed between two
// commits, with rename detection enabled.
func ExampleClient_CompareCommits() {
	ctx := context.Background()

	client, err := nanogit.NewHTTPClient("https://github.com/owner/repo.git")
	if err != nil {
		log.Fatal(err)
	}

	base, err := client.GetRef(ctx, "refs/tags/v1.0.0")
	if err != nil {
		log.Fatal(err)
	}
	head, err := client.GetRef(ctx, "refs/heads/main")
	if err != nil {
		log.Fatal(err)
	}

	changes, err := client.CompareCommits(ctx, base.Hash, head.Hash, nanogit.WithRenameDetection())
	if err != nil {
		log.Fatal(err)
	}
	for _, change := range changes {
		fmt.Printf("%s %s\n", change.Status, change.Path)
	}
}
