package integration_test

import (
	"context"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Repository", func() {
	Context("RepoExists functionality", func() {
		var (
			client nanogit.Client
			remote *helpers.RemoteRepo
		)

		BeforeEach(func() {
			By("Setting up test repository")
			client, remote, _, _ = QuickSetup()
		})

		It("should confirm existence of existing repository", func() {
			exists, err := client.RepoExists(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should handle non-existent repository", func() {
			By("Creating client for non-existent repository")
			nonExistentClient, err := nanogit.NewHTTPClient(remote.URL()+"/nonexistent", nanogit.WithBasicAuth(remote.User.Username, remote.User.Password))
			Expect(err).NotTo(HaveOccurred())

			exists, err := nonExistentClient.RepoExists(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should handle unauthorized access", func() {
			By("Creating client with wrong credentials")
			unauthorizedClient, err := nanogit.NewHTTPClient(remote.URL(), nanogit.WithBasicAuth("wronguser", "wrongpass"))
			Expect(err).NotTo(HaveOccurred())

			exists, err := unauthorizedClient.RepoExists(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("401 Unauthorized"))
			Expect(exists).To(BeFalse())
		})
	})
})
