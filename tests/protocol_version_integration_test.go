package integration_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
	protocolclient "github.com/grafana/nanogit/protocol/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Protocol Version Detection", func() {
	Context("Protocol v1 Detection", func() {
		It("should detect and reject v1-only servers with clear error message", func() {
			By("Creating a mock Git server that only supports protocol v1")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate Git protocol v1 info/refs response
				// v1 response format: <hash> <refname>\0<capabilities>
				if r.URL.Path == "/repo.git/info/refs" {
					// Real Git smart HTTP format for v1
					v1Response := "001e# service=git-upload-pack\n" + // service announcement
						"0000" + // flush after service announcement
						"003f1234567890abcdef1234567890abcdef12345678 refs/heads/main\000caps\n" + // ref with capabilities
						"00351234567890abcdef1234567890abcdef12345678 refs/heads/dev\n" + // additional ref
						"0000" // final flush
					w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(v1Response))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Attempting to create a nanogit client for the v1-only server")
			client, err := nanogit.NewHTTPClient(server.URL+"/repo.git",
				options.WithBasicAuth("test", "test"))
			Expect(err).NotTo(HaveOccurred())

			By("Checking protocol compatibility - should fail with protocol v1 error")
			compatible, err := client.IsProtocolCompatible(ctx)
			Expect(err).To(HaveOccurred(), "Should fail when connecting to v1-only server")
			Expect(compatible).To(BeFalse())

			By("Verifying error is ErrProtocolV1NotSupported")
			Expect(errors.Is(err, protocolclient.ErrProtocolV1NotSupported)).To(BeTrue(),
				"Error should be ErrProtocolV1NotSupported")
			logger.Info("Protocol v1 detected and rejected as expected", "error", err.Error())

			By("Verifying error message is informative")
			Expect(err.Error()).To(ContainSubstring("git protocol v1 is not supported"))
			Expect(err.Error()).To(ContainSubstring("protocol v2"))
		})

		It("should detect v1 when attempting to get ref from v1-only server", func() {
			By("Creating a mock Git server that only supports protocol v1")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					// v1 advertisement format
					v1Response := "001e# service=git-upload-pack\n" +
						"0000" +
						"003fabcdef1234567890abcdef1234567890abcdef12 refs/heads/master\000caps\n" +
						"0000"
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

			By("Checking protocol compatibility - should fail with protocol v1 error")
			_, err = client.IsProtocolCompatible(ctx)
			Expect(err).To(HaveOccurred())

			By("Verifying protocol v1 error")
			Expect(errors.Is(err, protocolclient.ErrProtocolV1NotSupported)).To(BeTrue())
			logger.Info("GetRef failed with protocol v1 error as expected")
		})

		It("should detect v1 when multiple refs are advertised without v2 indicators", func() {
			By("Creating a server with multiple v1 ref advertisements")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					// Multiple refs in v1 format
					v1Response := "001e# service=git-upload-pack\n" +
						"0000" +
						"003f1111111111111111111111111111111111111111 HEAD\000caps\n" +
						"00352222222222222222222222222222222222222222 refs/heads/main\n" +
						"00353333333333333333333333333333333333333333 refs/heads/dev\n" +
						"003a4444444444444444444444444444444444444444 refs/tags/v1.0.0\n" +
						"0000"
					w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(v1Response))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Creating client and attempting operation")
			client, err := nanogit.NewHTTPClient(server.URL+"/repo.git")
			Expect(err).NotTo(HaveOccurred())

			By("Checking protocol compatibility should fail with v1 error")
			_, err = client.IsProtocolCompatible(ctx)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, protocolclient.ErrProtocolV1NotSupported)).To(BeTrue())
		})
	})

	Context("Protocol v2 Detection", func() {
		It("should successfully connect to v2 servers with version announcement", func() {
			By("Creating a mock Git server with v2 version announcement")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					// v2 response with version announcement
					v2Response := "001e# service=git-upload-pack\n" +
						"0000" +
						"000eversion 2\n" +
						"0000"
					w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(v2Response))
					return
				}
				if r.URL.Path == "/repo.git/git-upload-pack" {
					// Minimal v2 ls-refs response
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("0000")) // Empty refs for simplicity
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Creating nanogit client")
			client, err := nanogit.NewHTTPClient(server.URL+"/repo.git",
				options.WithBasicAuth("test", "test"))
			Expect(err).NotTo(HaveOccurred())

			By("Checking protocol compatibility should succeed")
			compatible, err := client.IsProtocolCompatible(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(compatible).To(BeTrue())
			logger.Info("Successfully connected to v2 server")
		})

		It("should successfully connect to v2 servers with capability lines", func() {
			By("Creating a mock Git server with v2 capability lines")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					// v2 response with capability lines (start with =)
					v2Response := "001e# service=git-upload-pack\n" +
						"0000" +
						"0014=ls-refs=unborn\n" +
						"0012=fetch=shallow\n" +
						"0000"
					w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(v2Response))
					return
				}
				if r.URL.Path == "/repo.git/git-upload-pack" {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("0000"))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Creating nanogit client")
			client, err := nanogit.NewHTTPClient(server.URL+"/repo.git")
			Expect(err).NotTo(HaveOccurred())

			By("Checking protocol compatibility should succeed")
			compatible, err := client.IsProtocolCompatible(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(compatible).To(BeTrue())
			logger.Info("Successfully connected to v2 server with capability lines")
		})

		It("should work with real Gitea server (v2 compatible)", func() {
			By("Using the shared integration test Gitea server")
			client, _, _, _ := QuickSetup()

			By("Listing refs from Gitea server")
			refs, err := client.ListRefs(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(refs)).To(BeNumerically(">", 0))
			logger.Info("Successfully connected to Gitea server with v2 protocol", "refs_count", len(refs))

			By("Verifying Gitea supports v2 (implicit - no error)")
			// If we got this far without ErrProtocolV1NotSupported, Gitea is using v2
		})
	})

	Context("Protocol Version Edge Cases", func() {
		It("should handle empty responses gracefully", func() {
			By("Creating a server with empty response")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("0000")) // Just flush packet
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Creating client - should not fail on protocol detection")
			client, err := nanogit.NewHTTPClient(server.URL+"/repo.git")
			Expect(err).NotTo(HaveOccurred())

			By("Operations may fail later, but not due to protocol v1 detection")
			_, err = client.IsProtocolCompatible(ctx)
			// It's OK if this fails, but it should NOT be a protocol v1 error
			if err != nil {
				Expect(errors.Is(err, protocolclient.ErrProtocolV1NotSupported)).To(BeFalse(),
					"Empty response should not be detected as v1")
			}
		})

		It("should prioritize v2 when both v1 and v2 indicators are present", func() {
			By("Creating a server that sends mixed v1/v2 response")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					// Mixed response: v2 version announcement + v1 refs
					mixedResponse := "001e# service=git-upload-pack\n" +
						"0000" +
						"000eversion 2\n" + // v2 indicator
						"003f1234567890abcdef1234567890abcdef12345678 refs/heads/main\n" + // v1-style ref
						"0000"
					w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(mixedResponse))
					return
				}
				if r.URL.Path == "/repo.git/git-upload-pack" {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("0000"))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Creating client")
			client, err := nanogit.NewHTTPClient(server.URL+"/repo.git")
			Expect(err).NotTo(HaveOccurred())

			By("Should be detected as v2, not v1")
			compatible, err := client.IsProtocolCompatible(ctx)
			Expect(err).NotTo(HaveOccurred(), "Should succeed when v2 indicator is present")
			Expect(compatible).To(BeTrue())
			logger.Info("Mixed v1/v2 response correctly prioritized v2")
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
			client, err := nanogit.NewHTTPClient(server.URL+"/repo.git")
			Expect(err).NotTo(HaveOccurred())

			By("Should handle gracefully without panic")
			_, err = client.IsProtocolCompatible(ctx)
			// Operation will likely fail, but should not crash
			if err != nil {
				logger.Info("Malformed response handled gracefully", "error", err.Error())
			}
		})
	})

	Context("Error Message Quality", func() {
		It("should provide helpful error messages for v1-only servers", func() {
			By("Creating v1-only server")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repo.git/info/refs" {
					v1Response := "001e# service=git-upload-pack\n" +
						"0000" +
						"003f1234567890abcdef1234567890abcdef12345678 refs/heads/main\000caps\n" +
						"0000"
					w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(v1Response))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			By("Attempting to connect")
			client, err := nanogit.NewHTTPClient(server.URL+"/repo.git")
			Expect(err).NotTo(HaveOccurred())

			_, err = client.IsProtocolCompatible(context.Background())
			Expect(err).To(HaveOccurred())

			By("Checking error message quality")
			errMsg := err.Error()

			// Should mention protocol v1
			Expect(errMsg).To(ContainSubstring("protocol v1"))

			// Should mention that v2 is required
			Expect(errMsg).To(ContainSubstring("protocol v2"))

			// Should provide actionable guidance
			Expect(errMsg).To(Or(
				ContainSubstring("upgrade"),
				ContainSubstring("GitHub"),
				ContainSubstring("GitLab"),
				ContainSubstring("Bitbucket"),
			))

			logger.Info("Error message provides helpful context", "message", errMsg)
		})
	})
})
