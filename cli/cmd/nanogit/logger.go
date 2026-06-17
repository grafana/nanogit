package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/grafana/nanogit/log"
)

// cliLogger writes structured log lines to a writer (stderr by default).
// It mirrors git's verbosity conventions:
//   - Warn/Error always emit.
//   - Info emits when verbose is true (like `git push -v`).
//   - Debug emits when the NANOGIT_TRACE env var is set (like GIT_TRACE=1).
type cliLogger struct {
	out     io.Writer
	verbose bool
	trace   bool
}

func newCLILogger(verbose bool) log.Logger {
	return &cliLogger{
		out:     os.Stderr,
		verbose: verbose,
		trace:   os.Getenv("NANOGIT_TRACE") != "",
	}
}

func (l *cliLogger) Debug(msg string, keysAndValues ...any) {
	if !l.trace {
		return
	}
	l.emit("DEBUG", msg, keysAndValues)
}

func (l *cliLogger) Info(msg string, keysAndValues ...any) {
	if !l.verbose {
		return
	}
	l.emit("INFO", msg, keysAndValues)
}

func (l *cliLogger) Warn(msg string, keysAndValues ...any) {
	l.emit("WARN", msg, keysAndValues)
}

func (l *cliLogger) Error(msg string, keysAndValues ...any) {
	l.emit("ERROR", msg, keysAndValues)
}

func (l *cliLogger) emit(level, msg string, kv []any) {
	var b strings.Builder
	b.WriteString(time.Now().UTC().Format(time.RFC3339))
	b.WriteByte(' ')
	b.WriteString(level)
	b.WriteByte(' ')
	b.WriteString(msg)
	for i := 0; i < len(kv); i += 2 {
		key := fmt.Sprintf("%v", kv[i])
		var val string
		if i+1 < len(kv) {
			val = fmt.Sprintf("%v", kv[i+1])
		}
		b.WriteByte(' ')
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(val)
	}
	b.WriteByte('\n')
	_, _ = l.out.Write([]byte(b.String()))
}
