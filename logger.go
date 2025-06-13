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

// loggerCtxKey is the key used to store the logger in the context.
type loggerCtxKey struct{}

// WithContextLogger adds a logger to the context that can be retrieved later.
// The logger will be used for operations performed with this context.
// If no logger is provided in the context, a no-op logger will be used.
//
// Parameters:
//   - ctx: The context to add the logger to
//   - logger: The logger to store in the context
//
// Returns:
//   - context.Context: A new context with the logger stored
func WithContextLogger(ctx context.Context, logger log.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, logger)
}

// getContextLogger retrieves the logger from the context.
// If no logger is stored in the context, nil will be returned.
//
// Parameters:
//   - ctx: The context to retrieve the logger from
//
// Returns:
//   - Logger: The logger stored in the context, or nil if none is found
func getContextLogger(ctx context.Context) log.Logger {
	logger, ok := ctx.Value(loggerCtxKey{}).(log.Logger)
	if !ok {
		return nil
	}

	return logger
}

// FIXME: this is duplicated in the client and http client
func (c *rawClient) getLogger(ctx context.Context) log.Logger {
	logger := getContextLogger(ctx)
	if logger != nil {
		return logger
	}

	return c.logger
}

func (c *httpClient) getLogger(ctx context.Context) log.Logger {
	logger := getContextLogger(ctx)
	if logger != nil {
		return logger
	}

	return c.logger
}
