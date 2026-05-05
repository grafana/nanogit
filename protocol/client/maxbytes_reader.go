package client

import (
	"fmt"
	"io"
)

// ErrResponseTooLarge is returned when a server response exceeds the
// configured byte limit for an operation. Embedders can match it with
// errors.As to surface a friendly error or escalate to a security alert.
type ErrResponseTooLarge struct {
	// Limit is the byte cap that was exceeded.
	Limit int64
	// Op identifies the operation class that hit the cap, e.g. "fetch",
	// "ls-refs", "receive-pack", "compatibility".
	Op string
}

func (e *ErrResponseTooLarge) Error() string {
	return fmt.Sprintf("nanogit: %s response exceeded %d byte limit", e.Op, e.Limit)
}

// newLimitedReadCloser wraps body so Read returns *ErrResponseTooLarge once
// the cumulative number of bytes returned would exceed limit. A limit <= 0
// disables the check (passthrough).
//
// Unlike io.LimitReader, hitting the cap returns an explicit typed error
// rather than a silent io.EOF, so parsers cannot mistake an attack for a
// well-formed truncated response.
func newLimitedReadCloser(body io.ReadCloser, limit int64, op string) io.ReadCloser {
	if limit <= 0 {
		return body
	}
	return &limitedReadCloser{body: body, remaining: limit, limit: limit, op: op}
}

type limitedReadCloser struct {
	body      io.ReadCloser
	remaining int64
	limit     int64
	op        string
}

func (l *limitedReadCloser) Read(p []byte) (int, error) {
	if l.remaining <= 0 {
		// Probe one extra byte: if the body is also at EOF we forward
		// io.EOF; if it has more data we report ErrResponseTooLarge.
		var probe [1]byte
		n, err := l.body.Read(probe[:])
		if n > 0 {
			return 0, &ErrResponseTooLarge{Limit: l.limit, Op: l.op}
		}
		if err == nil {
			err = io.EOF
		}
		return 0, err
	}

	if int64(len(p)) > l.remaining {
		p = p[:l.remaining]
	}
	n, err := l.body.Read(p)
	l.remaining -= int64(n)
	return n, err
}

func (l *limitedReadCloser) Close() error {
	return l.body.Close()
}
