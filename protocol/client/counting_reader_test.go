package client

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountingReadCloser(t *testing.T) {
	t.Parallel()

	t.Run("counts bytes read in a single Read call", func(t *testing.T) {
		body := io.NopCloser(strings.NewReader("hello"))
		r := newCountingReadCloser(body)

		got, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), got)
		assert.Equal(t, int64(5), r.n)
	})

	t.Run("accumulates across multiple partial reads", func(t *testing.T) {
		body := io.NopCloser(&dripReader{data: []byte("0123456789")})
		r := newCountingReadCloser(body)

		got, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, []byte("0123456789"), got)
		assert.Equal(t, int64(10), r.n)
	})

	t.Run("empty body counts zero bytes", func(t *testing.T) {
		body := io.NopCloser(strings.NewReader(""))
		r := newCountingReadCloser(body)

		got, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Equal(t, int64(0), r.n)
	})

	t.Run("count reflects only bytes actually returned on a partial error read", func(t *testing.T) {
		injected := errors.New("injected transport failure")
		body := &errAfterCloser{
			Reader: strings.NewReader("ab"),
			err:    injected,
		}
		r := newCountingReadCloser(body)

		buf := make([]byte, 4)
		n, err := r.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 2, n)

		n, err = r.Read(buf)
		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, injected)

		assert.Equal(t, int64(2), r.n, "count must not advance for a read that returned 0 bytes")
	})

	t.Run("close forwards to underlying body", func(t *testing.T) {
		closed := false
		body := &closeRecorder{Reader: strings.NewReader("x"), onClose: func() { closed = true }}
		r := newCountingReadCloser(body)

		require.NoError(t, r.Close())
		assert.True(t, closed)
	})

	t.Run("wraps another ReadCloser transparently (e.g. limitedReadCloser)", func(t *testing.T) {
		// The real call site wraps countingReadCloser around
		// newLimitedReadCloser's result, so the count must reflect
		// bytes delivered by the wrapped reader, not the raw body.
		body := io.NopCloser(strings.NewReader("hello world"))
		limited := newLimitedReadCloser(body, 5, "fetch")
		r := newCountingReadCloser(limited)

		got, err := io.ReadAll(r)
		var tooLarge *ErrResponseTooLarge
		require.ErrorAs(t, err, &tooLarge)
		assert.LessOrEqual(t, len(got), 5)
		assert.Equal(t, int64(len(got)), r.n)
	})
}
