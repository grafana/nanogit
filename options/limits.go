package options

import "fmt"

// WithLimits installs DoS-protection caps on response sizes, classified by
// operation. Embedders that don't call WithLimits (or pass a zero Limits)
// keep nanogit's historic unbounded behavior.
//
// A zero value in any field means "no limit". Negative values are rejected.
func WithLimits(l Limits) Option {
	return func(o *Options) error {
		if l.SingleObjectFetchMaxBytes < 0 {
			return fmt.Errorf("Limits.SingleObjectFetchMaxBytes is negative: %d", l.SingleObjectFetchMaxBytes)
		}
		if l.MultiObjectFetchMaxBytes < 0 {
			return fmt.Errorf("Limits.MultiObjectFetchMaxBytes is negative: %d", l.MultiObjectFetchMaxBytes)
		}
		if l.RefsMetadataMaxBytes < 0 {
			return fmt.Errorf("Limits.RefsMetadataMaxBytes is negative: %d", l.RefsMetadataMaxBytes)
		}
		if l.ReceivePackResponseMaxBytes < 0 {
			return fmt.Errorf("Limits.ReceivePackResponseMaxBytes is negative: %d", l.ReceivePackResponseMaxBytes)
		}
		o.Limits = l
		return nil
	}
}
