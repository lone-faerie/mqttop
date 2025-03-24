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

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

type debugHandler struct {
	slog.Handler
}

func (h debugHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level == slog.LevelDebug || h.Handler.Enabled(ctx, level)
}

func SetHandler(h Handler) {
	l := slog.New(debugHandler{h}).With(defaultLogger.with...).WithGroup(defaultLogger.group)
	defaultLogger.Logger = l
}

type debugLogger struct{}

func DebugLogger() Logger {
	return debugLogger{}
}

func (debugLogger) Println(v ...any) {
	Debug(fmt.Sprintln(v...))
}

func (debugLogger) Printf(format string, v ...any) {
	Debug(fmt.Sprintf(format, v...))
}
