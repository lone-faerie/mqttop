//go:build debug

package log

import (
	"context"
	"fmt"
	"log/slog"
)

func init() {
	SetLogLevel(LevelDebug)
	defaultLogger.Warn("DEBUG")
}

// Debug logs at [LevelDebug]
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

type debugHandler struct {
	slog.Handler
}

func (h debugHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level == slog.LevelDebug || h.Handler.Enabled(ctx, level)
}

// SetHandler sets the default logger's handler to the one given.
func SetHandler(h Handler) {
	l := slog.New(debugHandler{h}).With(defaultLogger.with...).WithGroup(defaultLogger.group)
	defaultLogger.Logger = l
}

type debugLogger struct{}

// DebugLogger returns a [Logger] that logs at [LevelDebug]
func DebugLogger() Logger {
	return debugLogger{}
}

func (debugLogger) Println(v ...any) {
	Debug(fmt.Sprintln(v...))
}

func (debugLogger) Printf(format string, v ...any) {
	Debug(fmt.Sprintf(format, v...))
}
