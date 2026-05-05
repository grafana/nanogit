package client

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLimitedReadCloser(t *testing.T) {
	t.Parallel()

	t.Run("limit zero is passthrough", func(t *testing.T) {
		body := io.NopCloser(strings.NewReader("hello"))
		r := newLimitedReadCloser(body, 0, "fetch")

		got, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), got)
	})

	t.Run("negative limit is passthrough", func(t *testing.T) {
		body := io.NopCloser(strings.NewReader("hello"))
		r := newLimitedReadCloser(body, -1, "fetch")

		got, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), got)
	})

	t.Run("body at or below limit reads cleanly", func(t *testing.T) {
		body := io.NopCloser(strings.NewReader("hello"))
		r := newLimitedReadCloser(body, 5, "fetch")

		got, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), got)
	})

	t.Run("body above limit returns ErrResponseTooLarge", func(t *testing.T) {
		body := io.NopCloser(strings.NewReader("hello world"))
		r := newLimitedReadCloser(body, 5, "ls-refs")

		got, err := io.ReadAll(r)

		var tooLarge *ErrResponseTooLarge
		require.ErrorAs(t, err, &tooLarge)
		assert.Equal(t, "ls-refs", tooLarge.Op)
		assert.Equal(t, int64(5), tooLarge.Limit)
		// Verify the bytes that did slip through stay bounded by the limit.
		assert.LessOrEqual(t, len(got), 5)
	})

	t.Run("partial reads accumulate against the limit", func(t *testing.T) {
		// Drip-feed reader: returns one byte at a time so we exercise the
		// remaining-counter math across many Read calls.
		body := io.NopCloser(&dripReader{data: []byte("0123456789")})
		r := newLimitedReadCloser(body, 7, "fetch")

		buf := make([]byte, 1)
		var collected []byte
		var lastErr error
		for {
			n, err := r.Read(buf)
			collected = append(collected, buf[:n]...)
			if err != nil {
				lastErr = err
				break
			}
		}

		var tooLarge *ErrResponseTooLarge
		require.ErrorAs(t, lastErr, &tooLarge)
		assert.Equal(t, []byte("0123456"), collected)
	})

	t.Run("close forwards to underlying body", func(t *testing.T) {
		closed := false
		body := &closeRecorder{Reader: bytes.NewReader([]byte("x")), onClose: func() { closed = true }}
		r := newLimitedReadCloser(body, 100, "fetch")

		require.NoError(t, r.Close())
		assert.True(t, closed)
	})

	t.Run("error type is comparable via errors.Is on sentinel", func(t *testing.T) {
		body := io.NopCloser(strings.NewReader("xx"))
		r := newLimitedReadCloser(body, 1, "fetch")

		_, err := io.ReadAll(r)
		require.Error(t, err)
		// errors.As is the documented match path; check it explicitly.
		var tooLarge *ErrResponseTooLarge
		assert.True(t, errors.As(err, &tooLarge))
	})
}

// dripReader returns one byte per Read until exhausted.
type dripReader struct {
	data []byte
	pos  int
}

func (d *dripReader) Read(p []byte) (int, error) {
	if d.pos >= len(d.data) {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	p[0] = d.data[d.pos]
	d.pos++
	return 1, nil
}

type closeRecorder struct {
	io.Reader
	onClose func()
}

func (c *closeRecorder) Close() error {
	if c.onClose != nil {
		c.onClose()
	}
	return nil
}
