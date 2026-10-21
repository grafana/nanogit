package log_test

import (
	"context"
	"log/slog"
	"os"

	"github.com/grafana/nanogit/log"
)

// slogAdapter adapts a *slog.Logger to nanogit's log.Logger interface.
type slogAdapter struct{ l *slog.Logger }

func (s slogAdapter) Debug(msg string, keysAndValues ...any) { s.l.Debug(msg, keysAndValues...) }
func (s slogAdapter) Info(msg string, keysAndValues ...any)  { s.l.Info(msg, keysAndValues...) }
func (s slogAdapter) Warn(msg string, keysAndValues ...any)  { s.l.Warn(msg, keysAndValues...) }
func (s slogAdapter) Error(msg string, keysAndValues ...any) { s.l.Error(msg, keysAndValues...) }

// ExampleToContext wires a logger into the context so nanogit operations
// performed with that context emit debug and progress logs. Without it,
// nanogit is silent (NoopLogger).
func ExampleToContext() {
	logger := slogAdapter{l: slog.New(slog.NewTextHandler(os.Stderr, nil))}

	ctx := log.ToContext(context.Background(), logger)

	// Pass ctx to any nanogit operation: client.GetRef(ctx, ...), etc.
	_ = ctx
}
