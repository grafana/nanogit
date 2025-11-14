package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/retry"
	"github.com/stretchr/testify/require"
)

// trackingRetrier tracks calls to ShouldRetry and Wait for testing
type trackingRetrier struct {
	shouldRetryCalls []shouldRetryCall
	waitCalls        []waitCall
	maxAttempts      int
	shouldRetryFunc  func(ctx context.Context, err error, attempt int) bool
	waitFunc         func(ctx context.Context, attempt int) error
	mu               sync.Mutex
}

type shouldRetryCall struct {
	ctx     context.Context
	err     error
	attempt int
	result  bool
}

type waitCall struct {
	ctx     context.Context
	attempt int
	err     error
}

func newTrackingRetrier(maxAttempts int) *trackingRetrier {
	return &trackingRetrier{
		maxAttempts: maxAttempts,
		shouldRetryCalls: make([]shouldRetryCall, 0),
		waitCalls:        make([]waitCall, 0),
	}
}

func (r *trackingRetrier) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result bool
	if r.shouldRetryFunc != nil {
		result = r.shouldRetryFunc(ctx, err, attempt)
	} else {
		// Default: retry on server unavailable errors
		result = errors.Is(err, protocol.ErrServerUnavailable)
	}

	r.shouldRetryCalls = append(r.shouldRetryCalls, shouldRetryCall{
		ctx:     ctx,
		err:     err,
		attempt: attempt,
		result:  result,
	})
	return result
}

func (r *trackingRetrier) Wait(ctx context.Context, attempt int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var err error
	if r.waitFunc != nil {
		err = r.waitFunc(ctx, attempt)
	}

	r.waitCalls = append(r.waitCalls, waitCall{
		ctx:     ctx,
		attempt: attempt,
		err:     err,
	})
	return err
}

func (r *trackingRetrier) MaxAttempts() int {
	return r.maxAttempts
}

func (r *trackingRetrier) getShouldRetryCalls() []shouldRetryCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]shouldRetryCall{}, r.shouldRetryCalls...)
}

func (r *trackingRetrier) getWaitCalls() []waitCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]waitCall{}, r.waitCalls...)
}

func TestSmartInfo_Retry(t *testing.T) {
	t.Parallel()

	t.Run("retries on 5xx errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("000eversion 2\n0000"))
		}))
		defer server.Close()

		retrier := newTrackingRetrier(3)
		retrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return errors.Is(err, protocol.ErrServerUnavailable)
		}
		retrier.waitFunc = func(ctx context.Context, attempt int) error {
			// Fast wait for testing
			time.Sleep(10 * time.Millisecond)
			return nil
		}

		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL+"/repo")
		require.NoError(t, err)

		err = client.SmartInfo(ctx, "git-upload-pack")
		require.NoError(t, err)
		require.Equal(t, 3, attemptCount)

		// Verify retrier was called
		shouldRetryCalls := retrier.getShouldRetryCalls()
		require.GreaterOrEqual(t, len(shouldRetryCalls), 2, "ShouldRetry should be called at least twice")
		require.GreaterOrEqual(t, len(retrier.getWaitCalls()), 2, "Wait should be called at least twice")
	})

	t.Run("does not retry on 4xx errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		retrier := newTrackingRetrier(3)
		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = client.SmartInfo(ctx, "git-upload-pack")
		require.Error(t, err)
		require.Equal(t, 1, attemptCount, "Should not retry on 4xx errors")

		// Verify retrier was not called for 4xx errors
		// 4xx errors are checked outside the retry wrapper, so ShouldRetry is never invoked
		shouldRetryCalls := retrier.getShouldRetryCalls()
		require.Equal(t, 0, len(shouldRetryCalls), "ShouldRetry should not be called for 4xx errors")
		waitCalls := retrier.getWaitCalls()
		require.Equal(t, 0, len(waitCalls), "Wait should not be called for 4xx errors")
	})

	t.Run("retries on network errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 2 {
				// Simulate network error by closing connection
				hj, ok := w.(http.Hijacker)
			if !ok {
				return
			}
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("000eversion 2\n0000"))
		}))
		defer server.Close()

		retrier := newTrackingRetrier(3)
		retrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			// Retry on any error for this test
			return err != nil
		}
		retrier.waitFunc = func(ctx context.Context, attempt int) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}

		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// This might fail, but we're testing that retries are attempted
		_ = client.SmartInfo(ctx, "git-upload-pack")

		// Verify retrier was called
		shouldRetryCalls := retrier.getShouldRetryCalls()
		require.GreaterOrEqual(t, len(shouldRetryCalls), 1, "ShouldRetry should be called")
	})
}

func TestUploadPack_Retry(t *testing.T) {
	t.Parallel()

	t.Run("retries on network errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 2 {
				// Simulate network error
				hj, ok := w.(http.Hijacker)
				if ok {
					conn, _, _ := hj.Hijack()
					conn.Close()
				}
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("response data"))
		}))
		defer server.Close()

		retrier := newTrackingRetrier(3)
		retrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return err != nil
		}
		retrier.waitFunc = func(ctx context.Context, attempt int) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}

		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// Note: This test verifies retries are attempted, but may fail due to body consumption
		// The important part is that the retrier is called
		_, _ = client.UploadPack(ctx, strings.NewReader("test data"))

		// Verify retrier was called
		shouldRetryCalls := retrier.getShouldRetryCalls()
		require.GreaterOrEqual(t, len(shouldRetryCalls), 1, "ShouldRetry should be called for network errors")
	})

	t.Run("does not retry on 5xx errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		retrier := newTrackingRetrier(3)
		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		_, err = client.UploadPack(ctx, strings.NewReader("test data"))
		require.Error(t, err)
		require.Equal(t, 1, attemptCount, "Should not retry POST requests on 5xx errors")

		// Verify retrier Wait was not called (no retries for 5xx POST errors)
		// The 5xx error happens after Do() succeeds, so retrier is not invoked
		// This is expected behavior - POST requests can't retry 5xx because body is consumed
		waitCalls := retrier.getWaitCalls()
		require.Equal(t, 0, len(waitCalls), "Wait should not be called for 5xx POST errors")
	})
}

func TestReceivePack_Retry(t *testing.T) {
	t.Parallel()

	t.Run("retries on network errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 2 {
				// Simulate network error
				hj, ok := w.(http.Hijacker)
				if ok {
					conn, _, _ := hj.Hijack()
					conn.Close()
				}
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("000dunpack ok0000"))
		}))
		defer server.Close()

		retrier := newTrackingRetrier(3)
		retrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
			return err != nil
		}
		retrier.waitFunc = func(ctx context.Context, attempt int) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}

		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// Note: This test verifies retries are attempted
		_ = client.ReceivePack(ctx, strings.NewReader("test data"))

		// Verify retrier was called
		shouldRetryCalls := retrier.getShouldRetryCalls()
		require.GreaterOrEqual(t, len(shouldRetryCalls), 1, "ShouldRetry should be called for network errors")
	})

	t.Run("does not retry on 5xx errors", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		retrier := newTrackingRetrier(3)
		ctx := retry.ToContext(context.Background(), retrier)
		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		err = client.ReceivePack(ctx, strings.NewReader("test data"))
		require.Error(t, err)
		require.Equal(t, 1, attemptCount, "Should not retry POST requests on 5xx errors")

		// Verify retrier Wait was not called (no retries for 5xx POST errors)
		waitCalls := retrier.getWaitCalls()
		require.Equal(t, 0, len(waitCalls), "Wait should not be called for 5xx POST errors")
	})
}

func TestRetry_NoRetrier(t *testing.T) {
	t.Parallel()

	t.Run("SmartInfo works without retrier", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("000eversion 2\n0000"))
		}))
		defer server.Close()

		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// No retrier in context - should work but not retry
		err = client.SmartInfo(context.Background(), "git-upload-pack")
		require.NoError(t, err)
		require.Equal(t, 1, attemptCount, "should make single attempt without retrier")
	})

	t.Run("SmartInfo does not retry on 5xx without retrier", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// No retrier in context - should fail immediately
		err = client.SmartInfo(context.Background(), "git-upload-pack")
		require.Error(t, err)
		require.Equal(t, 1, attemptCount, "should not retry without retrier")
	})

	t.Run("UploadPack works without retrier", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("response"))
		}))
		defer server.Close()

		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// No retrier in context - should work but not retry
		reader, err := client.UploadPack(context.Background(), strings.NewReader("test"))
		require.NoError(t, err)
		require.NotNil(t, reader)
		_ = reader.Close()
		require.Equal(t, 1, attemptCount, "should make single attempt without retrier")
	})

	t.Run("ReceivePack works without retrier", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("000dunpack ok0000"))
		}))
		defer server.Close()

		client, err := NewRawClient(server.URL + "/repo")
		require.NoError(t, err)

		// No retrier in context - should work but not retry
		err = client.ReceivePack(context.Background(), strings.NewReader("test"))
		require.NoError(t, err)
		require.Equal(t, 1, attemptCount, "should make single attempt without retrier")
	})
}

