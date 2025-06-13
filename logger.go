package nanogit

import (
	"context"
	"errors"

	"github.com/grafana/nanogit/log"
)

// WithLogger configures a custom logger for the Git client.
// This allows integration with existing logging infrastructure and debugging.
// If not provided, a no-op logger will be used by default.
//
// Parameters:
//   - logger: Custom logger implementation
//
// Returns:
//   - Option: Configuration function for the client
//   - error: Error if the provided logger is nil
func WithLogger(logger log.Logger) Option {
	return func(c *rawClient) error {
		if logger == nil {
			return errors.New("logger cannot be nil")
		}
		c.logger = logger
		return nil
	}
}

// FIXME: this is duplicated in the client and http client
func (c *rawClient) getLogger(ctx context.Context) log.Logger {
	logger := log.GetContextLogger(ctx)
	if logger != nil {
		return logger
	}

	return c.logger
}

func (c *httpClient) getLogger(ctx context.Context) log.Logger {
	logger := log.GetContextLogger(ctx)
	if logger != nil {
		return logger
	}

	return c.logger
}
