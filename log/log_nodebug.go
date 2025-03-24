//go:build !debug

package log

import "log/slog"

func Debug(_ string, _ ...any) {}

func SetHandler(h Handler) {
	l := slog.New(h).With(defaultLogger.with...).WithGroup(defaultLogger.group)
	defaultLogger.Logger = l
}

func DebugLogger() Logger {
	return debugLogger{}
}

type debugLogger struct{}

func (debugLogger) Println(v ...any)               {}
func (debugLogger) Printf(format string, v ...any) {}
