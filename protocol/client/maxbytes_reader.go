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
	// err sticks once the underlying body has terminated (either via
	// its own EOF/error or via our cap). Subsequent Read calls return
	// it without touching the body, so callers that loop after a
	// terminal condition cannot trigger another underlying read.
	err error
}

// Read implements the http.MaxBytesReader pattern: read at most remaining+1
// bytes from the underlying body in a single call so the cap-vs-fits
// decision happens in one round trip. The previous implementation issued an
// explicit one-byte probe Read whenever remaining hit zero — that probe was
// a separate round trip after the caller had already read the cap's worth
// of bytes, which let a server stalling at exactly the boundary keep us
// blocked on a second read instead of failing fast. With this pattern the
// "is there more after the cap?" question is answered inside the same
// underlying Read the caller initiated, so we either get the over-cap byte
// alongside the legitimate data or get a clean EOF — no extra blocking
// probe call. Callers should still set HTTP-level read deadlines for the
// pathological "server hangs mid-chunk" case; that is unavoidable without
// timeouts at any layer, but eliminating the explicit probe removes the
// most-reachable stall surface.
func (l *limitedReadCloser) Read(p []byte) (int, error) {
	if l.err != nil {
		return 0, l.err
	}
	if len(p) == 0 {
		return 0, nil
	}

	// Cap the read at remaining+1 bytes. Getting back exactly
	// remaining+1 bytes is the unambiguous "body exceeds the cap"
	// signal; <= remaining is fine. The conversion to int is safe:
	// the predicate guarantees len(p)-1 > l.remaining, so
	// l.remaining+1 <= len(p), and len(p) fits in int by definition
	// (it came from len() on a slice).
	if int64(len(p))-1 > l.remaining {
		p = p[:int(l.remaining+1)]
	}
	n, err := l.body.Read(p)

	if int64(n) <= l.remaining {
		l.remaining -= int64(n)
		// Stick terminal errors (including io.EOF) so we never go
		// back to the body after it tells us it's done.
		if err != nil {
			l.err = err
		}
		return n, err
	}

	// Got remaining+1 bytes — body exceeds the cap. Hand the caller
	// the cap's worth and stick the typed error. No further reads
	// will reach the underlying body.
	n = int(l.remaining)
	l.remaining = 0
	l.err = &ErrResponseTooLarge{Limit: l.limit, Op: l.op}
	return n, l.err
}

func (l *limitedReadCloser) Close() error {
	return l.body.Close()
}
