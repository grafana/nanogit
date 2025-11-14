package integration_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server Unavailable Error Handling", func() {
	Context("HTTP 5xx Status Codes", func() {
		It("should return ServerUnavailableError for 500 Internal Server Error", func() {
			By("Creating a mock server that returns 500")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling SmartInfo should return ServerUnavailableError")
			_, err = client.IsAuthorized(ctx)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrServerUnavailable)).To(BeTrue(), "error should be ErrServerUnavailable")

			var serverErr *nanogit.ServerUnavailableError
			Expect(errors.As(err, &serverErr)).To(BeTrue(), "error should be ServerUnavailableError type")
			Expect(serverErr.StatusCode).To(Equal(http.StatusInternalServerError))
			Expect(serverErr.Underlying).NotTo(BeNil())
		})

		It("should return ServerUnavailableError for 502 Bad Gateway", func() {
			By("Creating a mock server that returns 502")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
				w.Write([]byte("Bad Gateway"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling SmartInfo should return ServerUnavailableError")
			_, err = client.IsAuthorized(ctx)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrServerUnavailable)).To(BeTrue())

			var serverErr *nanogit.ServerUnavailableError
			Expect(errors.As(err, &serverErr)).To(BeTrue())
			Expect(serverErr.StatusCode).To(Equal(http.StatusBadGateway))
		})

		It("should return ServerUnavailableError for 503 Service Unavailable", func() {
			By("Creating a mock server that returns 503")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("Service Unavailable"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling SmartInfo should return ServerUnavailableError")
			_, err = client.IsAuthorized(ctx)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrServerUnavailable)).To(BeTrue())

			var serverErr *nanogit.ServerUnavailableError
			Expect(errors.As(err, &serverErr)).To(BeTrue())
			Expect(serverErr.StatusCode).To(Equal(http.StatusServiceUnavailable))
		})

		It("should return ServerUnavailableError for 504 Gateway Timeout", func() {
			By("Creating a mock server that returns 504")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusGatewayTimeout)
				w.Write([]byte("Gateway Timeout"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling SmartInfo should return ServerUnavailableError")
			_, err = client.IsAuthorized(ctx)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrServerUnavailable)).To(BeTrue())

			var serverErr *nanogit.ServerUnavailableError
			Expect(errors.As(err, &serverErr)).To(BeTrue())
			Expect(serverErr.StatusCode).To(Equal(http.StatusGatewayTimeout))
		})

		It("should not return ServerUnavailableError for 4xx errors", func() {
			By("Creating a mock server that returns 404")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("Not Found"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling SmartInfo should return error but not ServerUnavailableError")
			_, err = client.IsAuthorized(ctx)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrServerUnavailable)).To(BeFalse(), "4xx errors should not be ServerUnavailableError")
		})
	})

	Context("Error Propagation Through Client Methods", func() {
		It("should propagate ServerUnavailableError from UploadPack", func() {
			By("Creating a mock server that returns 500 for upload-pack")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "git-upload-pack") {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal Server Error"))
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("000eversion 2\n0000"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling ListRefs which uses UploadPack internally")
			_, err = client.ListRefs(ctx)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, nanogit.ErrServerUnavailable)).To(BeTrue(), "error should propagate through ListRefs")

			var serverErr *nanogit.ServerUnavailableError
			Expect(errors.As(err, &serverErr)).To(BeTrue())
			Expect(serverErr.StatusCode).To(Equal(http.StatusInternalServerError))
		})

	})

	Context("Error Wrapping and Unwrapping", func() {
		It("should preserve underlying error information", func() {
			By("Creating a mock server that returns 500")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling SmartInfo should return ServerUnavailableError")
			_, err = client.IsAuthorized(ctx)
			Expect(err).To(HaveOccurred())

			var serverErr *nanogit.ServerUnavailableError
			Expect(errors.As(err, &serverErr)).To(BeTrue())

			By("Verifying underlying error is preserved")
			Expect(serverErr.Underlying).NotTo(BeNil())
			Expect(serverErr.Underlying.Error()).To(ContainSubstring("status code 500"))
			Expect(serverErr.Underlying.Error()).To(ContainSubstring("Internal Server Error"))
		})

		It("should work with errors.Is for error checking", func() {
			By("Creating a mock server that returns 503")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("Service Unavailable"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Calling SmartInfo should return ServerUnavailableError")
			_, err = client.IsAuthorized(ctx)
			Expect(err).To(HaveOccurred())

			By("Verifying errors.Is works correctly")
			Expect(errors.Is(err, nanogit.ErrServerUnavailable)).To(BeTrue())
			Expect(errors.Is(err, protocol.ErrServerUnavailable)).To(BeTrue(), "should also work with protocol package error")
		})
	})

	Context("Retry Logic Support", func() {
		It("should allow users to implement retry logic based on ServerUnavailableError", func() {
			By("Creating a mock server that returns 500 then succeeds")
			attemptCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attemptCount++
				if attemptCount == 1 {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal Server Error"))
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("000eversion 2\n0000"))
			}))
			defer server.Close()

			By("Creating client pointing to mock server")
			client, err := nanogit.NewHTTPClient(server.URL + "/test.git")
			Expect(err).NotTo(HaveOccurred())

			By("Implementing retry logic")
			maxRetries := 3
			var lastErr error
			for i := 0; i < maxRetries; i++ {
				_, lastErr = client.IsAuthorized(ctx)
				if lastErr == nil {
					break
				}
				if errors.Is(lastErr, nanogit.ErrServerUnavailable) {
					logger.Info("Server unavailable, retrying", "attempt", i+1, "max_retries", maxRetries)
					continue
				}
				break
			}

			By("Verifying retry succeeded")
			Expect(lastErr).NotTo(HaveOccurred(), "should succeed after retry")
			Expect(attemptCount).To(Equal(2), "should have retried once")
		})
	})
})

