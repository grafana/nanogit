package options

import "fmt"

// WithLimits installs DoS-protection caps on response sizes, classified by
// operation. Embedders that don't call WithLimits (or pass a zero Limits)
// keep nanogit's historic unbounded behavior.
//
// A zero value in any field means "no limit". Negative values are rejected.
func WithLimits(l Limits) Option {
	return func(o *Options) error {
		if l.SingleObjectFetch < 0 {
			return fmt.Errorf("Limits.SingleObjectFetch is negative: %d", l.SingleObjectFetch)
		}
		if l.MultiObjectFetch < 0 {
			return fmt.Errorf("Limits.MultiObjectFetch is negative: %d", l.MultiObjectFetch)
		}
		if l.RefsMetadata < 0 {
			return fmt.Errorf("Limits.RefsMetadata is negative: %d", l.RefsMetadata)
		}
		if l.ReceivePackResponse < 0 {
			return fmt.Errorf("Limits.ReceivePackResponse is negative: %d", l.ReceivePackResponse)
		}
		o.Limits = l
		return nil
	}
}
