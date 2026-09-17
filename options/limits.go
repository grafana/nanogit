package options

import "fmt"

// WithLimits installs DoS-protection caps, classified by operation. For the
// four wire-response caps, embedders that don't call WithLimits (or pass a
// zero Limits) keep nanogit's historic unbounded behavior: a zero value in
// any of those four fields means "no limit".
//
// MaxObjectDecodedBytes is the exception: a zero value leaves nanogit's
// built-in decoded-object default (protocol.MaxUnpackedObjectSize) in force
// rather than disabling the cap, so decompression-bomb protection is always
// on. Set a positive value to raise or lower that ceiling.
//
// Negative values are rejected.
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
		if l.MaxObjectDecodedBytes < 0 {
			return fmt.Errorf("Limits.MaxObjectDecodedBytes is negative: %d", l.MaxObjectDecodedBytes)
		}
		o.Limits = l
		return nil
	}
}
