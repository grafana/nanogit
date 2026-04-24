package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestLogger(verbose, trace bool) (*cliLogger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return &cliLogger{out: buf, verbose: verbose, trace: trace}, buf
}

func TestCLILoggerLevelGating(t *testing.T) {
	tests := []struct {
		name      string
		verbose   bool
		trace     bool
		wantDebug bool
		wantInfo  bool
		wantWarn  bool
		wantError bool
	}{
		{"quiet: only Warn/Error", false, false, false, false, true, true},
		{"verbose: adds Info", true, false, false, true, true, true},
		{"trace only: adds Debug", false, true, true, false, true, true},
		{"verbose + trace: all levels", true, true, true, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, buf := newTestLogger(tt.verbose, tt.trace)
			l.Debug("dbg")
			l.Info("inf")
			l.Warn("wrn")
			l.Error("err")
			out := buf.String()
			assert.Equal(t, tt.wantDebug, strings.Contains(out, "DEBUG dbg"), "debug emission")
			assert.Equal(t, tt.wantInfo, strings.Contains(out, "INFO inf"), "info emission")
			assert.Equal(t, tt.wantWarn, strings.Contains(out, "WARN wrn"), "warn emission")
			assert.Equal(t, tt.wantError, strings.Contains(out, "ERROR err"), "error emission")
		})
	}
}

func TestCLILoggerKeyValueFormatting(t *testing.T) {
	l, buf := newTestLogger(false, false)
	l.Warn("operation", "path", "docs/note.md", "size", 42)
	out := buf.String()
	assert.Contains(t, out, "WARN operation")
	assert.Contains(t, out, "path=docs/note.md")
	assert.Contains(t, out, "size=42")
}

func TestCLILoggerOddKeyValueDoesNotPanic(t *testing.T) {
	l, buf := newTestLogger(false, false)
	assert.NotPanics(t, func() {
		l.Warn("msg", "lonely-key")
	})
	assert.Contains(t, buf.String(), "lonely-key=")
}

func TestNewCLILoggerReadsTraceEnv(t *testing.T) {
	t.Setenv("NANOGIT_TRACE", "1")
	l := newCLILogger(false).(*cliLogger)
	assert.True(t, l.trace)
	assert.False(t, l.verbose)
}

func TestNewCLILoggerNoTrace(t *testing.T) {
	t.Setenv("NANOGIT_TRACE", "")
	l := newCLILogger(true).(*cliLogger)
	assert.False(t, l.trace)
	assert.True(t, l.verbose)
}
