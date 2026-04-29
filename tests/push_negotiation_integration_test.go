package integration_test

import (
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol/hash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Capability negotiation happy path against the Gitea container. Gitea
// advertises the full default set, so the negotiated intersection is
// observably equivalent to the static defaults — what we exercise here is
// that the lazy fetch + sync.Once cache + intersection wiring all run
// cleanly through a real receive-pack discovery and a real push.
var _ = Describe("Push with capability negotiation", func() {
	author := nanogit.Author{
		Name:  "Test Author",
		Email: "test@example.com",
		Time:  time.Now(),
	}
	committer := nanogit.Committer{
		Name:  "Test Committer",
		Email: "test@example.com",
		Time:  time.Now(),
	}

	It("creates and pushes a commit when negotiation is enabled", func() {
		client, _, local, _ := QuickSetup(options.WithCapabilityNegotiation())

		headOutput, err := local.Git("rev-parse", "HEAD")
		Expect(err).NotTo(HaveOccurred())
		head, err := hash.FromHex(headOutput)
		Expect(err).NotTo(HaveOccurred())

		writer, err := client.NewStagedWriter(ctx, nanogit.Ref{
			Name: "refs/heads/main",
			Hash: head,
		})
		Expect(err).NotTo(HaveOccurred())

		_, err = writer.CreateBlob(ctx, "negotiation.txt", []byte("negotiated"))
		Expect(err).NotTo(HaveOccurred())

		_, err = writer.Commit(ctx, "add file via negotiation", author, committer)
		Expect(err).NotTo(HaveOccurred())

		Expect(writer.Push(ctx)).To(Succeed())
	})

	It("reuses the negotiated set across multiple ref operations", func() {
		// sync.Once on httpClient guarantees we don't fetch info/refs more
		// than once even when the caller drives many ref ops in a row. We
		// can't observe the round-trip count from outside Gitea, but the
		// operations must still all succeed end-to-end.
		client, _, _, _ := QuickSetup(options.WithCapabilityNegotiation())

		for range 3 {
			refs, err := client.ListRefs(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(refs).NotTo(BeEmpty())
		}
	})
})
