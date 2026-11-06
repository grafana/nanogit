package options

import (
	"net/http"
	"time"
)

// RateLimitSignal is what a provider's response says about retrying.
type RateLimitSignal struct {
	// RateLimited reports that the response is a transient rate-limit
	// rejection rather than a permanent failure, so the request may be retried.
	RateLimited bool
	// RetryAfter is how long the server asked the client to wait. Zero leaves
	// the retrier's own backoff schedule in charge.
	RetryAfter time.Duration
}

// RateLimitDetector reads a provider's rate-limit signalling off a response.
//
// nanogit recognises the common cases without one: 5xx, 429, a 403 carrying a
// rate-limit header, and the Retry-After header in both RFC 9110 forms. A
// detector covers a provider that signals differently — a reset timestamp
// rather than a delay, a vendor header, or a status nanogit treats as
// permanent.
//
// It is called for every response and its result is merged with the built-in
// rules: it can mark a response rate limited that nanogit would not, and it can
// supply a delay for one nanogit already classified. It cannot un-classify a
// response nanogit considers transient, and it does not control whether the
// retry is actually attempted — a request whose body cannot be rewound is still
// never retried.
//
// Implementations must not read or close res.Body.
type RateLimitDetector func(res *http.Response) RateLimitSignal

// WithRateLimitDetector installs a detector for providers whose rate-limit
// signalling nanogit does not recognise out of the box. Passing nil restores
// the built-in rules alone.
//
// Example, for a provider that reports the reset time as a Unix timestamp:
//
//	options.WithRateLimitDetector(func(res *http.Response) options.RateLimitSignal {
//	    if res.StatusCode != http.StatusTooManyRequests {
//	        return options.RateLimitSignal{}
//	    }
//	    epoch, err := strconv.ParseInt(res.Header.Get("X-RateLimit-Reset"), 10, 64)
//	    if err != nil {
//	        return options.RateLimitSignal{RateLimited: true}
//	    }
//	    return options.RateLimitSignal{
//	        RateLimited: true,
//	        RetryAfter:  time.Until(time.Unix(epoch, 0)),
//	    }
//	})
func WithRateLimitDetector(d RateLimitDetector) Option {
	return func(o *Options) error {
		o.RateLimitDetector = d
		return nil
	}
}
