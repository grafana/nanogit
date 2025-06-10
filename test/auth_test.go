package integration_test

import (
	"context"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/test/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Authorization", func() {
	var (
		remote    *helpers.RemoteRepo
		user      *helpers.User
		remoteURL string
	)

	BeforeEach(func() {
		By("Setting up test repository using shared Git server")
		_, remote, _, user = QuickSetup()
		remoteURL = remote.URL()
	})

	It("should successfully authorize with basic auth", func() {
		By("Creating client with correct basic auth credentials")
		authClient, err := nanogit.NewHTTPClient(remoteURL, nanogit.WithBasicAuth(user.Username, user.Password), nanogit.WithLogger(logger))
		Expect(err).NotTo(HaveOccurred())

		By("Checking authorization")
		auth, err := authClient.IsAuthorized(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(auth).To(BeTrue())
	})

	It("should fail authorization with wrong credentials", func() {
		By("Creating client with incorrect credentials")
		unauthorizedClient, err := nanogit.NewHTTPClient(remoteURL, nanogit.WithBasicAuth("wronguser", "wrongpass"))
		Expect(err).NotTo(HaveOccurred())

		By("Checking authorization should fail")
		auth, err := unauthorizedClient.IsAuthorized(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(auth).To(BeFalse())
	})

	It("should successfully authorize with access token", func() {
		By("Generating access token for user")
		token := gitServer.GenerateUserToken(user.Username, user.Password)
		Expect(token).NotTo(BeEmpty())

		By("Creating client with access token")
		tokenClient, err := nanogit.NewHTTPClient(remoteURL, nanogit.WithTokenAuth(token), nanogit.WithLogger(logger))
		Expect(err).NotTo(HaveOccurred())

		By("Checking authorization")
		auth, err := tokenClient.IsAuthorized(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(auth).To(BeTrue())
	})

	It("should fail authorization with invalid token", func() {
		By("Creating client with invalid token")
		invalidToken := "token invalid-token"
		invalidClient, err := nanogit.NewHTTPClient(remoteURL, nanogit.WithTokenAuth(invalidToken), nanogit.WithLogger(logger))
		Expect(err).NotTo(HaveOccurred())

		By("Checking authorization should fail")
		auth, err := invalidClient.IsAuthorized(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(auth).To(BeFalse())
	})
})
