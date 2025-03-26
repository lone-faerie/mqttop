//go:build !debug

package log

import "log/slog"

// Debug logs at [LevelDebug]
func Debug(_ string, _ ...any) {}

// SetHandler sets the default logger's handler to the one given.
func SetHandler(h Handler) {
	l := slog.New(h).With(defaultLogger.with...).WithGroup(defaultLogger.group)
	defaultLogger.Logger = l
}

// DebugLogger returns a [Logger] that logs at [LevelDebug]
func DebugLogger() Logger {
	return debugLogger{}
}

type debugLogger struct{}

func (debugLogger) Println(v ...any)               {}
func (debugLogger) Printf(format string, v ...any) {}
