package nanogit_test

import (
	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/internal/testhelpers"
	clientpkg "github.com/grafana/nanogit/protocol/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Repository", func() {
	Context("RepoExists functionality", func() {
		var (
			client nanogit.Client
			remote *testhelpers.RemoteRepo
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, remote, _, _ = QuickSetup()
		})

		It("should confirm existence of existing repository", func() {
			exists, err := client.RepoExists(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should handle non-existent repository", func() {
			By("Creating client for non-existent repository")
			nonExistentClient, err := nanogit.NewHTTPClient(remote.URL()+"/nonexistent", clientpkg.WithBasicAuth(remote.User.Username, remote.User.Password))
			Expect(err).NotTo(HaveOccurred())

			exists, err := nonExistentClient.RepoExists(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should handle unauthorized access", func() {
			By("Creating client with wrong credentials")
			unauthorizedClient, err := nanogit.NewHTTPClient(remote.URL(), clientpkg.WithBasicAuth("wronguser", "wrongpass"))
			Expect(err).NotTo(HaveOccurred())

			exists, err := unauthorizedClient.RepoExists(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("401 Unauthorized"))
			Expect(exists).To(BeFalse())
		})
	})
})
