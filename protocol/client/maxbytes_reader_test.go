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

	t.Run("ErrResponseTooLarge.Error formats limit and op", func(t *testing.T) {
		// Stringification matters because operators read this in logs;
		// the test pins the exact format so changes are intentional.
		err := &ErrResponseTooLarge{Limit: 1024, Op: "fetch"}
		assert.Equal(t,
			"nanogit: fetch response exceeded 1024 byte limit",
			err.Error())
	})

	t.Run("limit equal to body size reads cleanly with io.EOF", func(t *testing.T) {
		// Boundary: remaining hits zero exactly when the body ends.
		// The next Read must surface io.EOF, not ErrResponseTooLarge.
		body := io.NopCloser(strings.NewReader("hello"))
		r := newLimitedReadCloser(body, 5, "fetch")

		got, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), got)

		// Drain past the boundary explicitly.
		buf := make([]byte, 1)
		n, err := r.Read(buf)
		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("zero-byte non-EOF Read at the boundary forwards underlying error", func(t *testing.T) {
		// Edge case in the probe path: underlying reader returns
		// (0, someErr) where someErr is not EOF and not nil. The
		// limited reader must forward someErr as-is, not coerce it
		// to io.EOF and not invent ErrResponseTooLarge.
		injected := errors.New("injected transport failure")
		body := &errAfterCloser{
			Reader: strings.NewReader("ab"),
			err:    injected,
		}
		r := newLimitedReadCloser(body, 2, "fetch")

		// Drain the two valid bytes first.
		buf := make([]byte, 2)
		n, err := r.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 2, n)

		// Now we are at the boundary; the next Read enters the
		// probe path. The fake body returns (0, injected), which
		// must propagate untouched.
		n, err = r.Read(buf)
		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, injected)

		var tooLarge *ErrResponseTooLarge
		assert.False(t, errors.As(err, &tooLarge),
			"a non-EOF zero-byte read must NOT be reported as oversized")
	})

	t.Run("zero-byte nil-error Read at the boundary surfaces io.EOF", func(t *testing.T) {
		// Edge case: underlying reader returns (0, nil) at the
		// probe. limitedReadCloser must coerce that to io.EOF so
		// the caller sees a deterministic stream end.
		body := &fixedReader{data: []byte("ab")}
		r := newLimitedReadCloser(body, 2, "fetch")

		// Drain the body.
		buf := make([]byte, 2)
		n, err := r.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 2, n)

		// Probe: fixedReader returns (0, nil) when exhausted.
		n, err = r.Read(buf)
		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, io.EOF)
	})
}

// errAfterCloser is an io.ReadCloser that wraps a strings.Reader and
// returns a configured error once the inner reader is exhausted (instead
// of io.EOF). Lets us exercise the probe path without introducing extra
// dependencies.
type errAfterCloser struct {
	*strings.Reader
	err error
}

func (e *errAfterCloser) Read(p []byte) (int, error) {
	n, err := e.Reader.Read(p)
	if errors.Is(err, io.EOF) {
		return n, e.err
	}
	return n, err
}

func (e *errAfterCloser) Close() error { return nil }

// fixedReader is an io.ReadCloser that returns (0, nil) once exhausted —
// a quirk allowed by io.Reader's contract that callers must tolerate.
// We use it to exercise the (n=0, err=nil) branch of the probe path.
type fixedReader struct {
	data []byte
	pos  int
}

func (f *fixedReader) Read(p []byte) (int, error) {
	if f.pos >= len(f.data) {
		return 0, nil
	}
	n := copy(p, f.data[f.pos:])
	f.pos += n
	return n, nil
}

func (f *fixedReader) Close() error { return nil }

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
