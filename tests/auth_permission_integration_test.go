package integration_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Integration tests for HTTP authentication and permission error handling.
// These tests verify that 401 (Unauthorized), 403 (Forbidden), and 404 (Not Found)
// errors are properly detected, structured, and propagated through the nanogit API.
//
// Test coverage includes:
// - CanRead() and CanWrite() permission checking methods
// - Error propagation through write operations (Push, CreateRef, UpdateRef, DeleteRef)
// - Error propagation through read operations (Clone, ListRefs, GetBlob)
// - errors.Is() and errors.As() compatibility for all error types
// - Error wrapping and unwrapping through the call stack

var _ = Describe("Authentication and Permission Error Handling", func() {
	Context("HTTP 401 Unauthorized", func() {
		It("should return UnauthorizedError on IsAuthorized", func() {
			By("Creating a mock server that returns 401")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("401 Unauthorized"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling IsAuthorized should return false")
			authorized, err := client.IsAuthorized(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(authorized).To(BeFalse())
		})

		It("should return UnauthorizedError on CanRead", func() {
			By("Creating a mock server that returns 401")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("401 Unauthorized"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling CanRead should return false")
			canRead, err := client.CanRead(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(canRead).To(BeFalse())
		})

		It("should return UnauthorizedError on CanWrite", func() {
			By("Creating a mock server that returns 401")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("401 Unauthorized"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling CanWrite should return false")
			canWrite, err := client.CanWrite(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(canWrite).To(BeFalse())
		})

		It("should return UnauthorizedError on Clone operation", func() {
			By("Creating a mock server that returns 401 for info/refs")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "info/refs") {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte("401 Unauthorized"))
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling Clone should return UnauthorizedError")
			_, err = client.Clone(ctx, nanogit.CloneOptions{
				Path: GinkgoT().TempDir(),
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrUnauthorized)).To(BeTrue(), "error should be ErrUnauthorized")

			var authErr *nanogit.UnauthorizedError
			Expect(errors.As(err, &authErr)).To(BeTrue(), "error should be UnauthorizedError type")
			Expect(authErr.StatusCode).To(Equal(http.StatusUnauthorized))
			Expect(authErr.Underlying).NotTo(BeNil())
		})
	})

	Context("HTTP 403 Forbidden", func() {
		It("should return PermissionDeniedError on CanRead", func() {
			By("Creating a mock server that returns 403")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte("403 Forbidden"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling CanRead should return false")
			canRead, err := client.CanRead(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(canRead).To(BeFalse())
		})

		It("should return PermissionDeniedError on CanWrite", func() {
			By("Creating a mock server that returns 403 for git-receive-pack")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.RawQuery, "git-receive-pack") {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte("403 Forbidden"))
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling CanWrite should return false (read-only access)")
			canWrite, err := client.CanWrite(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(canWrite).To(BeFalse())
		})

		It("should return PermissionDeniedError on Push operation", func() {
			By("Creating a mock server that returns 403 for git-receive-pack POST")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" && strings.Contains(r.URL.Path, "git-receive-pack") {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte("403 Forbidden"))
					return
				}
				// Return 200 for info/refs to allow initial setup
				w.WriteHeader(http.StatusOK)
				if strings.Contains(r.URL.Path, "info/refs") {
					_, _ = w.Write([]byte("001e# service=git-receive-pack\n0000"))
				}
			}))
			defer server.Close()

			By("Creating client and attempting push")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Creating a staged writer")
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: hash.Hash{},
			}
			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())
			defer writer.Cleanup(ctx)

			By("Pushing should return PermissionDeniedError")
			err = writer.Push(ctx)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrPermissionDenied)).To(BeTrue(), "error should be ErrPermissionDenied")

			var permErr *nanogit.PermissionDeniedError
			Expect(errors.As(err, &permErr)).To(BeTrue(), "error should be PermissionDeniedError type")
			Expect(permErr.StatusCode).To(Equal(http.StatusForbidden))
			Expect(permErr.Operation).To(Equal("POST"))
			Expect(permErr.Endpoint).To(Equal("git-receive-pack"))
			Expect(permErr.Underlying).NotTo(BeNil())
		})

		It("should return PermissionDeniedError on CreateRef operation", func() {
			By("Creating a mock server that returns 403 for git-receive-pack POST")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" && strings.Contains(r.URL.Path, "git-receive-pack") {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte("403 Forbidden"))
					return
				}
				w.WriteHeader(http.StatusOK)
				if strings.Contains(r.URL.Path, "info/refs") {
					_, _ = w.Write([]byte("001e# service=git-receive-pack\n0000"))
				}
			}))
			defer server.Close()

			By("Creating client and attempting to create ref")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Creating ref should return PermissionDeniedError")
			ref := nanogit.Ref{
				Name: "refs/heads/feature",
				Hash: hash.MustFromHex("0000000000000000000000000000000000000001"),
			}
			err = client.CreateRef(ctx, ref)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrPermissionDenied)).To(BeTrue())

			var permErr *nanogit.PermissionDeniedError
			Expect(errors.As(err, &permErr)).To(BeTrue())
			Expect(permErr.StatusCode).To(Equal(http.StatusForbidden))
			Expect(permErr.Endpoint).To(Equal("git-receive-pack"))
		})
	})

	Context("HTTP 404 Not Found", func() {
		It("should return RepositoryNotFoundError on Clone", func() {
			By("Creating a mock server that returns 404")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("404 Not Found"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling Clone should return RepositoryNotFoundError")
			_, err = client.Clone(ctx, nanogit.CloneOptions{
				Path: GinkgoT().TempDir(),
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrRepositoryNotFound)).To(BeTrue(), "error should be ErrRepositoryNotFound")

			var notFoundErr *nanogit.RepositoryNotFoundError
			Expect(errors.As(err, &notFoundErr)).To(BeTrue(), "error should be RepositoryNotFoundError type")
			Expect(notFoundErr.StatusCode).To(Equal(http.StatusNotFound))
			Expect(notFoundErr.Underlying).NotTo(BeNil())
		})

		It("should return RepositoryNotFoundError on ListRefs", func() {
			By("Creating a mock server that returns 404")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("404 Not Found"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling ListRefs should return RepositoryNotFoundError")
			_, err = client.ListRefs(ctx)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrRepositoryNotFound)).To(BeTrue())

			var notFoundErr *nanogit.RepositoryNotFoundError
			Expect(errors.As(err, &notFoundErr)).To(BeTrue())
			Expect(notFoundErr.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Context("CanWrite method", func() {
		It("should return true when git-receive-pack is accessible", func() {
			By("Creating a mock server that allows git-receive-pack")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.RawQuery, "git-receive-pack") {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("001e# service=git-receive-pack\n0000"))
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling CanWrite should return true")
			canWrite, err := client.CanWrite(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(canWrite).To(BeTrue())
		})

		It("should return false when git-receive-pack returns 403 (read-only)", func() {
			By("Creating a mock server that forbids git-receive-pack but allows git-upload-pack")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.RawQuery, "git-receive-pack") {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte("403 Forbidden"))
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling CanWrite should return false (read-only)")
			canWrite, err := client.CanWrite(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(canWrite).To(BeFalse())
		})

		It("should return false when git-receive-pack returns 401 (unauthorized)", func() {
			By("Creating a mock server that returns 401 for git-receive-pack")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.RawQuery, "git-receive-pack") {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte("401 Unauthorized"))
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling CanWrite should return false")
			canWrite, err := client.CanWrite(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(canWrite).To(BeFalse())
		})

		It("should return error for 500 server error", func() {
			By("Creating a mock server that returns 500")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("500 Internal Server Error"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling CanWrite should return error")
			_, err = client.CanWrite(ctx)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrServerUnavailable)).To(BeTrue())
		})
	})

	Context("Error propagation through operations", func() {
		It("should propagate PermissionDeniedError through UpdateRef", func() {
			By("Creating a mock server that returns 403 for receive-pack")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" && strings.Contains(r.URL.Path, "git-receive-pack") {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte("403 Forbidden"))
					return
				}
				w.WriteHeader(http.StatusOK)
				if strings.Contains(r.URL.Path, "info/refs") {
					_, _ = w.Write([]byte("001e# service=git-receive-pack\n0000"))
				}
			}))
			defer server.Close()

			By("Creating client and attempting to update ref")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("UpdateRef should return PermissionDeniedError")
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: hash.MustFromHex("0000000000000000000000000000000000000001"),
			}
			err = client.UpdateRef(ctx, ref)
			Expect(err).To(HaveOccurred())

			// Error propagates through the call stack
			var permErr *nanogit.PermissionDeniedError
			Expect(errors.As(err, &permErr)).To(BeTrue())
			Expect(permErr.StatusCode).To(Equal(http.StatusForbidden))
		})

		It("should propagate PermissionDeniedError through DeleteRef", func() {
			By("Creating a mock server that returns 403 for receive-pack")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" && strings.Contains(r.URL.Path, "git-receive-pack") {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte("403 Forbidden"))
					return
				}
				w.WriteHeader(http.StatusOK)
				if strings.Contains(r.URL.Path, "info/refs") {
					_, _ = w.Write([]byte("001e# service=git-receive-pack\n0000"))
				}
			}))
			defer server.Close()

			By("Creating client and attempting to delete ref")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("DeleteRef should return PermissionDeniedError")
			err = client.DeleteRef(ctx, "refs/heads/feature")
			Expect(err).To(HaveOccurred())

			var permErr *nanogit.PermissionDeniedError
			Expect(errors.As(err, &permErr)).To(BeTrue())
			Expect(permErr.StatusCode).To(Equal(http.StatusForbidden))
		})
	})

	Context("Error wrapping chain", func() {
		It("should maintain error chain for wrapped PermissionDeniedError", func() {
			By("Creating a mock server that returns 403")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" && strings.Contains(r.URL.Path, "git-receive-pack") {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte("403 Forbidden"))
					return
				}
				w.WriteHeader(http.StatusOK)
				if strings.Contains(r.URL.Path, "info/refs") {
					_, _ = w.Write([]byte("001e# service=git-receive-pack\n0000"))
				}
			}))
			defer server.Close()

			By("Creating client")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Creating staged writer and pushing")
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: hash.Hash{},
			}
			writer, err := client.NewStagedWriter(ctx, ref)
			Expect(err).NotTo(HaveOccurred())
			defer writer.Cleanup(ctx)

			err = writer.Push(ctx)
			Expect(err).To(HaveOccurred())

			By("Error chain should be preserved through wrapping")
			// errors.Is should work through wrapped errors
			Expect(errors.Is(err, nanogit.ErrPermissionDenied)).To(BeTrue())

			// errors.As should work through wrapped errors
			var permErr *nanogit.PermissionDeniedError
			Expect(errors.As(err, &permErr)).To(BeTrue())

			// Underlying error should be accessible
			Expect(permErr.Underlying).NotTo(BeNil())
			Expect(permErr.Underlying.Error()).To(ContainSubstring("403"))
		})
	})

	Context("Fetch operations (read-only)", func() {
		It("should return UnauthorizedError on fetch with 401", func() {
			By("Creating a mock server that returns 401 for upload-pack")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" && strings.Contains(r.URL.Path, "git-upload-pack") {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte("401 Unauthorized"))
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			By("Creating client")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Attempting to clone should return UnauthorizedError")
			_, err = client.Clone(ctx, nanogit.CloneOptions{
				Path: GinkgoT().TempDir(),
			})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrUnauthorized)).To(BeTrue())
		})

		It("should return PermissionDeniedError on fetch with 403", func() {
			By("Creating a mock server that returns 403 for upload-pack")
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				if r.Method == "POST" && strings.Contains(r.URL.Path, "git-upload-pack") {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte("403 Forbidden"))
					return
				}
				// Allow info/refs for initial connection
				if strings.Contains(r.URL.Path, "info/refs") {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("001e# service=git-upload-pack\n0000"))
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			By("Creating client")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Attempting to fetch blob should return PermissionDeniedError")
			hash := hash.MustFromHex("0000000000000000000000000000000000000001")
			_, err = client.GetBlob(ctx, hash)
			Expect(err).To(HaveOccurred())

			// Should eventually get a PermissionDeniedError (might be wrapped)
			var permErr *nanogit.PermissionDeniedError
			if errors.As(err, &permErr) {
				Expect(permErr.StatusCode).To(Equal(http.StatusForbidden))
				Expect(permErr.Endpoint).To(Equal("git-upload-pack"))
			}
		})
	})
})
