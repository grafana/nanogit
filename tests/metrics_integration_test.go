package integration_test

import (
	"strings"

	"github.com/grafana/nanogit/metrics"
	"github.com/grafana/nanogit/mocks"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/storage"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// These integration tests exercise metrics.Recorder end-to-end against the
// shared Gitea testcontainer, verifying that the events documented in
// docs/architecture/metrics.md actually fire from real HTTP round trips and
// real packfile fetches — not just against the httptest.Server fakes used
// by the protocol/client unit tests.
var _ = Describe("Metrics", func() {
	It("records HTTPRequest with OperationSmartInfo for RepoExists", func() {
		client, _, _, _ := QuickSetup()

		recorder := &mocks.FakeRecorder{}
		metricsCtx := metrics.ToContext(ctx, recorder)

		exists, err := client.RepoExists(metricsCtx)
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())

		Expect(recorder.HTTPRequestCallCount()).To(BeNumerically(">=", 1))
		_, operation, statusCode, _, attempt := recorder.HTTPRequestArgsForCall(0)
		Expect(operation).To(Equal(metrics.OperationSmartInfo))
		Expect(statusCode).To(Equal(200))
		Expect(attempt).To(Equal(1))
	})

	It("records HTTPRequest with OperationUploadPack and ObjectsFetched for GetBlob", func() {
		client, _, local, _ := QuickSetup()

		By("Pushing a blob to fetch")
		Expect(local.CreateFile("hello.txt", "hello world")).To(Succeed())
		_, err := local.Git("add", "hello.txt")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("commit", "-m", "add hello")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("push", "origin", "main", "--force")
		Expect(err).NotTo(HaveOccurred())

		blobHashOutput, err := local.Git("rev-parse", "HEAD:hello.txt")
		Expect(err).NotTo(HaveOccurred())
		blobHash, err := hash.FromHex(strings.TrimSpace(blobHashOutput))
		Expect(err).NotTo(HaveOccurred())

		recorder := &mocks.FakeRecorder{}
		metricsCtx := metrics.ToContext(ctx, recorder)

		blob, err := client.GetBlob(metricsCtx, blobHash)
		Expect(err).NotTo(HaveOccurred())
		Expect(blob.Content).To(Equal([]byte("hello world")))

		By("Verifying an upload-pack HTTPRequest was recorded")
		Expect(recorder.HTTPRequestCallCount()).To(BeNumerically(">=", 1))
		foundUploadPack := false
		for i := range recorder.HTTPRequestCallCount() {
			_, operation, statusCode, _, _ := recorder.HTTPRequestArgsForCall(i)
			if operation == metrics.OperationUploadPack {
				foundUploadPack = true
				Expect(statusCode).To(Equal(200))
			}
		}
		Expect(foundUploadPack).To(BeTrue(), "expected at least one upload-pack HTTPRequest")

		By("Verifying ObjectsFetched was recorded for the network fetch")
		Expect(recorder.ObjectsFetchedCallCount()).To(Equal(1))
		_, count, bytes := recorder.ObjectsFetchedArgsForCall(0)
		Expect(count).To(Equal(1))
		Expect(bytes).To(BeNumerically(">", 0))
	})

	It("records CacheAccess misses then hits across repeated fetches with shared storage", func() {
		client, _, local, _ := QuickSetup()

		By("Pushing a blob to fetch twice")
		Expect(local.CreateFile("cached.txt", "cache me")).To(Succeed())
		_, err := local.Git("add", "cached.txt")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("commit", "-m", "add cached file")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("push", "origin", "main", "--force")
		Expect(err).NotTo(HaveOccurred())

		blobHashOutput, err := local.Git("rev-parse", "HEAD:cached.txt")
		Expect(err).NotTo(HaveOccurred())
		blobHash, err := hash.FromHex(strings.TrimSpace(blobHashOutput))
		Expect(err).NotTo(HaveOccurred())

		recorder := &mocks.FakeRecorder{}
		// A shared storage.PackfileStorage across both GetBlob calls is
		// what makes the second lookup a cache hit — without it, each
		// call gets its own ephemeral cache and CacheAccess is never
		// recorded at all (see docs/architecture/metrics.md).
		metricsCtx := metrics.ToContext(storage.ToContext(ctx, storage.NewInMemoryStorage(ctx)), recorder)

		By("First GetBlob: cache miss, network fetch")
		_, err = client.GetBlob(metricsCtx, blobHash)
		Expect(err).NotTo(HaveOccurred())
		Expect(recorder.CacheAccessCallCount()).To(Equal(1))
		_, hit := recorder.CacheAccessArgsForCall(0)
		Expect(hit).To(BeFalse())
		Expect(recorder.ObjectsFetchedCallCount()).To(Equal(1))

		By("Second GetBlob: cache hit, no network fetch")
		_, err = client.GetBlob(metricsCtx, blobHash)
		Expect(err).NotTo(HaveOccurred())
		Expect(recorder.CacheAccessCallCount()).To(Equal(2))
		_, hit = recorder.CacheAccessArgsForCall(1)
		Expect(hit).To(BeTrue())
		Expect(recorder.ObjectsFetchedCallCount()).To(Equal(1),
			"a cache hit must skip the network fetch entirely, so ObjectsFetched must not fire again")
	})

	It("reports nothing when no recorder is in context (NoopRecorder default)", func() {
		client, _, _, _ := QuickSetup()

		// Using the shared suite ctx directly (no metrics.ToContext) must
		// not panic or error — nanogit falls back to metrics.NoopRecorder.
		exists, err := client.RepoExists(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
	})
})
