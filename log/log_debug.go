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

// SetLogLevel controls the level for the bridge to the [log] package.
//
// Before [SetDefault] is called, log top-level logging functions call the default [log.Logger].
// In that mode, SetLogLevel sets the minimum level for those calls.
// By default, the minimum level is Info, so calls to [Debug]
// (as well as top-level logging calls at lower levels)
// will not be passed to the log.Logger. After calling
//
//	log.SetLogLevel(log.LevelDebug)
//
// calls to [Debug] will be passed to the log.Logger.
//
// After [SetDefault] is called, calls to the default [log.Logger] are passed to the
// slog default handler. In that mode,
// SetLogLoggerLevel sets the level at which those calls are logged.
// That is, after calling
//
//	log.SetLogLevel(slog.LevelDebug)
//
// A call to [log.Printf] will result in output at level [LevelDebug].
//
// SetLogLevel returns the previous value.
func SetLogLevel(level Level) (oldLevel Level) {
	old := slog.SetLogLoggerLevel(slog.LevelDebug)
	return Level(old)
}
