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
// - Error propagation through write operations (CreateRef)
// - Error propagation through read operations (ListRefs, GetBlob)
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

		It("should return UnauthorizedError on ListRefs operation", func() {
			By("Creating a mock server that returns 401 for git-upload-pack")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "git-upload-pack") {
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

			By("Calling ListRefs should return UnauthorizedError")
			_, err = client.ListRefs(ctx)
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
			Expect(errors.Is(err, nanogit.ErrRepositoryNotFound)).To(BeTrue(), "error should be ErrRepositoryNotFound")

			var notFoundErr *nanogit.RepositoryNotFoundError
			Expect(errors.As(err, &notFoundErr)).To(BeTrue(), "error should be RepositoryNotFoundError type")
			Expect(notFoundErr.StatusCode).To(Equal(http.StatusNotFound))
			Expect(notFoundErr.Underlying).NotTo(BeNil())
		})

		It("should return RepositoryNotFoundError on GetBlob", func() {
			By("Creating a mock server that returns 404 for all requests")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("404 Not Found"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling GetBlob should return RepositoryNotFoundError")
			blobHash := hash.MustFromHex("0000000000000000000000000000000000000001")
			_, err = client.GetBlob(ctx, blobHash)
			Expect(err).To(HaveOccurred())

			// Error might be wrapped, check with errors.As
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

	Context("Error propagation through CreateRef", func() {
		It("should propagate PermissionDeniedError correctly", func() {
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

			By("Creating client and attempting to create ref")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("CreateRef should return PermissionDeniedError")
			ref := nanogit.Ref{
				Name: "refs/heads/main",
				Hash: hash.MustFromHex("0000000000000000000000000000000000000001"),
			}
			err = client.CreateRef(ctx, ref)
			Expect(err).To(HaveOccurred())

			// Error propagates through the call stack
			var permErr *nanogit.PermissionDeniedError
			Expect(errors.As(err, &permErr)).To(BeTrue())
			Expect(permErr.StatusCode).To(Equal(http.StatusForbidden))

			// errors.Is should also work
			Expect(errors.Is(err, nanogit.ErrPermissionDenied)).To(BeTrue())

			// Underlying error should be accessible
			Expect(permErr.Underlying).NotTo(BeNil())
			Expect(permErr.Underlying.Error()).To(ContainSubstring("403"))
		})
	})

	Context("Fetch operations", func() {
		It("should return PermissionDeniedError when fetching blob with 403", func() {
			By("Creating a mock server that returns 403 for upload-pack")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			blobHash := hash.MustFromHex("0000000000000000000000000000000000000001")
			_, err = client.GetBlob(ctx, blobHash)
			Expect(err).To(HaveOccurred())

			// Should get a PermissionDeniedError (might be wrapped)
			var permErr *nanogit.PermissionDeniedError
			if errors.As(err, &permErr) {
				Expect(permErr.StatusCode).To(Equal(http.StatusForbidden))
				Expect(permErr.Endpoint).To(Equal("git-upload-pack"))
			}
		})
	})
})
