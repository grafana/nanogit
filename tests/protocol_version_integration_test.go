package integration_test

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// formatPacks is a helper to create properly formatted pkt-line responses for integration tests
func formatPacks(packs ...protocol.Pack) string {
	data, err := protocol.FormatPacks(packs...)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

var _ = Describe("Protocol Version Detection", func() {
	Context("Protocol v1 Detection", func() {
		It("should detect v1-only servers", func() {
			By("Creating a mock Git server that only supports protocol v1")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate Git protocol v1 info/refs response
				// v1 response format: <hash> <refname>\0<capabilities>
				if r.URL.Path == "/repo.git/info/refs" {
					// Real Git smart HTTP format for v1
					v1Response := formatPacks(
						protocol.PackLine("# service=git-upload-pack\n"),
						protocol.FlushPacket,
						protocol.PackLine("1234567890abcdef1234567890abcdef12345678 refs/heads/main\000caps\n"),
						protocol.PackLine("1234567890abcdef1234567890abcdef12345678 refs/heads/dev\n"))
					w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(v1Response))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Creating a nanogit client for the v1-only server")
			client, err := nanogit.NewHTTPClient(server.URL+"/repo.git",
				options.WithBasicAuth("test", "test"))
			Expect(err).NotTo(HaveOccurred())

			By("Checking protocol compatibility - should return false for v1")
			compatible, err := client.IsServerCompatible(ctx)
			Expect(err).NotTo(HaveOccurred(), "Should successfully check compatibility")
			Expect(compatible).To(BeFalse(), "Should be incompatible with v1")
			logger.Info("Protocol v1 server correctly detected as incompatible")
		})

		It("should detect v1 with single ref advertisement", func() {
			By("Creating a mock Git server that only supports protocol v1")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					// v1 advertisement format
					v1Response := formatPacks(
						protocol.PackLine("# service=git-upload-pack\n"),
						protocol.FlushPacket,
						protocol.PackLine("abcdef1234567890abcdef1234567890abcdef12 refs/heads/master\000caps\n"))
					w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(v1Response))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Creating nanogit client")
			client, err := nanogit.NewHTTPClient(server.URL+"/repo.git",
				options.WithBasicAuth("test", "test"))
			Expect(err).NotTo(HaveOccurred())

			By("Checking protocol compatibility - should return false for v1")
			compatible, err := client.IsServerCompatible(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(compatible).To(BeFalse())
			logger.Info("Protocol v1 server correctly detected as incompatible")
		})

		It("should detect v1 when multiple refs are advertised without v2 indicators", func() {
			By("Creating a server with multiple v1 ref advertisements")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					// Multiple refs in v1 format
					v1Response := formatPacks(
						protocol.PackLine("# service=git-upload-pack\n"),
						protocol.FlushPacket,
						protocol.PackLine("1111111111111111111111111111111111111111 HEAD\000caps\n"),
						protocol.PackLine("2222222222222222222222222222222222222222 refs/heads/main\n"),
						protocol.PackLine("3333333333333333333333333333333333333333 refs/heads/dev\n"),
						protocol.PackLine("4444444444444444444444444444444444444444 refs/tags/v1.0.0\n"))
					w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(v1Response))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Creating client")
			client, err := nanogit.NewHTTPClient(server.URL + "/repo.git")
			Expect(err).NotTo(HaveOccurred())

			By("Checking protocol compatibility - should return false for v1")
			compatible, err := client.IsServerCompatible(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(compatible).To(BeFalse())
		})
	})

	Context("Protocol v2 Detection", func() {
		It("should successfully connect to v2 servers with version announcement", func() {
			By("Creating a mock Git server with v2 version announcement")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					// v2 response with version announcement
					v2Response := formatPacks(
						protocol.PackLine("# service=git-upload-pack\n"),
						protocol.FlushPacket,
						protocol.PackLine("version 2\n"))
					w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(v2Response))
					return
				}
				if r.URL.Path == "/repo.git/git-upload-pack" {
					// Minimal v2 ls-refs response
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(string(protocol.FlushPacket)))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Creating nanogit client")
			client, err := nanogit.NewHTTPClient(server.URL+"/repo.git",
				options.WithBasicAuth("test", "test"))
			Expect(err).NotTo(HaveOccurred())

			By("Checking protocol compatibility - should return true for v2")
			compatible, err := client.IsServerCompatible(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(compatible).To(BeTrue())
			logger.Info("Server is compatible (v2)")
		})

		It("should successfully detect v2 servers with capability lines", func() {
			By("Creating a mock Git server with v2 capability lines")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					// v2 response with capability lines (start with =)
					v2Response := formatPacks(
						protocol.PackLine("# service=git-upload-pack\n"),
						protocol.FlushPacket,
						protocol.PackLine("=ls-refs=unborn\n"),
						protocol.PackLine("=fetch=shallow\n"))
					w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(v2Response))
					return
				}
				if r.URL.Path == "/repo.git/git-upload-pack" {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(string(protocol.FlushPacket)))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Creating nanogit client")
			client, err := nanogit.NewHTTPClient(server.URL + "/repo.git")
			Expect(err).NotTo(HaveOccurred())

			By("Checking protocol compatibility - should return true for v2")
			compatible, err := client.IsServerCompatible(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(compatible).To(BeTrue())
			logger.Info("Server is compatible (v2)")
		})

		It("should work with real Gitea server (v2 compatible)", func() {
			By("Using the shared integration test Gitea server")
			client, _, _, _ := QuickSetup()

			By("Listing refs from Gitea server")
			refs, err := client.ListRefs(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(refs)).To(BeNumerically(">", 0))
			logger.Info("Successfully connected to Gitea server", "refs_count", len(refs))

			By("Verifying Gitea is compatible")
			compatible, err := client.IsServerCompatible(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(compatible).To(BeTrue())
			logger.Info("Gitea server is compatible (v2)")
		})

		It("should detect v1-only real Gitea server as incompatible", func() {
			By("Creating a real Gitea server with protocol v1 only")
			// Use Gitea 1.16 which uses Git 2.31 - before full v2 support was standard
			// Gitea 1.22 always advertises v2 capabilities regardless of ENABLE_AUTO_GIT_WIRE_PROTOCOL
			v1Server, err := gittest.NewServer(ctx,
				gittest.WithLogger(gittest.NewStructuredLogger(logger)),
				gittest.WithGiteaVersion("1.16"),
				gittest.WithProtocolV1Only(),
				gittest.WithTimeout(60*time.Second))
			Expect(err).NotTo(HaveOccurred())
			defer v1Server.Cleanup()

			By("Creating a test user")
			user, err := v1Server.CreateUser(ctx)
			Expect(err).NotTo(HaveOccurred())

			By("Generating access token")
			token, err := v1Server.CreateToken(ctx, user.Username)
			Expect(err).NotTo(HaveOccurred())

			By("Creating a test repository")
			repo, err := v1Server.CreateRepo(ctx, "test-v1-repo", user)
			Expect(err).NotTo(HaveOccurred())

			By("Initializing repository with a commit")
			localRepo, err := gittest.NewLocalRepo(ctx, gittest.WithRepoLogger(gittest.NewStructuredLogger(logger)))
			Expect(err).NotTo(HaveOccurred())
			defer localRepo.Cleanup()

			connInfo, err := localRepo.InitWithRemote(user, repo)
			Expect(err).NotTo(HaveOccurred())

			By("Creating nanogit client")
			client, err := nanogit.NewHTTPClient(connInfo.URL,
				options.WithTokenAuth(token))
			Expect(err).NotTo(HaveOccurred())

			By("Checking protocol compatibility - should return false for v1")
			compatible, err := client.IsServerCompatible(ctx)
			Expect(err).NotTo(HaveOccurred(), "Should successfully check compatibility")
			Expect(compatible).To(BeFalse(), "Should be incompatible with v1-only server")
			logger.Info("Real Gitea v1-only server correctly detected as incompatible")

			By("Verifying ListRefs still works (uses v1 protocol)")
			refs, err := client.ListRefs(ctx)
			Expect(err).NotTo(HaveOccurred(), "ListRefs should work even on v1 servers")
			Expect(len(refs)).To(BeNumerically(">", 0))
			logger.Info("ListRefs still works on v1 server", "refs_count", len(refs))
		})
	})

	Context("Protocol Version Edge Cases", func() {
		It("should handle empty responses gracefully", func() {
			By("Creating a server with empty response")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(string(protocol.FlushPacket))) // Just flush packet
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Creating client")
			client, err := nanogit.NewHTTPClient(server.URL + "/repo.git")
			Expect(err).NotTo(HaveOccurred())

			By("Checking protocol compatibility - should return error for unknown")
			compatible, err := client.IsServerCompatible(ctx)
			Expect(err).To(HaveOccurred(), "Should error when protocol version is unknown")
			Expect(err.Error()).To(ContainSubstring("could not determine protocol version"))
			Expect(compatible).To(BeFalse())
			logger.Info("Empty response correctly returned error")
		})

		It("should prioritize v2 when both v1 and v2 indicators are present", func() {
			By("Creating a server that sends mixed v1/v2 response")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					// Mixed response: v2 version announcement + v1 refs
					mixedResponse := formatPacks(
						protocol.PackLine("# service=git-upload-pack\n"),
						protocol.FlushPacket,
						protocol.PackLine("version 2\n"), // v2 indicator
						protocol.PackLine("1234567890abcdef1234567890abcdef12345678 refs/heads/main\n")) // v1-style ref
					w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(mixedResponse))
					return
				}
				if r.URL.Path == "/repo.git/git-upload-pack" {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(string(protocol.FlushPacket)))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Creating client")
			client, err := nanogit.NewHTTPClient(server.URL + "/repo.git")
			Expect(err).NotTo(HaveOccurred())

			By("Should be compatible (v2), not v1")
			compatible, err := client.IsServerCompatible(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(compatible).To(BeTrue())
			logger.Info("Mixed v1/v2 response correctly detected as compatible")
		})

		It("should handle malformed responses without crashing", func() {
			By("Creating a server with malformed packet data")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					// Invalid packet format
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("invalid packet data that is not pkt-line format"))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Creating client")
			client, err := nanogit.NewHTTPClient(server.URL + "/repo.git")
			Expect(err).NotTo(HaveOccurred())

			By("Should handle gracefully without panic - returns error")
			compatible, err := client.IsServerCompatible(ctx)
			Expect(err).To(HaveOccurred(), "Should error when response is malformed")
			Expect(err.Error()).To(ContainSubstring("could not determine protocol version"))
			Expect(compatible).To(BeFalse())
			logger.Info("Malformed response correctly returned error")
		})
	})
})
