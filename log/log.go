// Package log provides structured logging with severity levels.
package log

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
)

type (
	Attr    = slog.Attr
	Handler = slog.Handler
)

var DiscardHandler = slog.DiscardHandler

// Logger implements the interface to provide logging for the mqtt library.
type Logger interface {
	Println(v ...any)
	Printf(format string, v ...any)
}

type logger struct {
	*slog.Logger
	with  []any
	group string
}

var defaultLogger = &logger{
	Logger: slog.Default(),
}

// With includes the given attributes in outputs from the default logger.
func With(args ...any) {
	defaultLogger.Logger = defaultLogger.Logger.With(args...)
	defaultLogger.with = args
}

// WithGroup sets the default logger to one that starts a group, if name is non-empty.
// The keys of all attributes added to the Logger will be qualified by the given name.
func WithGroup(name string) {
	defaultLogger.Logger = defaultLogger.Logger.WithGroup(name)
	defaultLogger.group = name
}

// DefaultLogger returns the default logger.
func DefaultLogger() Logger {
	return defaultLogger
}

// SetOutput sets the output of the default logger by calling [log.SetOutput]
func SetOutput(w io.Writer) {
	log.SetOutput(w)
}

// Error logs at [LevelError]
func Error(msg string, err error, args ...any) {
	if err != nil {
		args = append([]any{"cause", err}, args...)
	}
	defaultLogger.Error(msg, args...)
}

// Fatal is equivalent to [Error] followed by a call to [os.Exit](1).
func Fatal(msg string, err error, args ...any) {
	Error(msg, err, args...)
	os.Exit(1)
}

// Warn logs at [LevelWarn]
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// WarnError is equivalent to [Warn]("cause", err, args...).
func WarnError(msg string, err error, args ...any) {
	if err != nil {
		args = append([]any{"cause", err}, args...)
	}
	defaultLogger.Warn(msg, args...)
}

// Info logs at [LevelInfo]
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Println is equivalent to [Info] with a message of [fmt.Sprintln](v...).
func Println(v ...any) {
	defaultLogger.Info(fmt.Sprintln(v...))
}

// Printf is equivalent to [Info] with a message of [fmt.Sprintf](format, v...).
func Printf(format string, v ...any) {
	defaultLogger.Info(fmt.Sprintf(format, v...))
}

func (l *logger) Println(v ...any) {
	l.Info(fmt.Sprintln(v...))
}

func (l *logger) Printf(format string, v ...any) {
	l.Info(fmt.Sprintf(format, v...))
}

func (l *logger) Log(ctx context.Context, level Level, msg string, args ...any) {
	l.Logger.Log(ctx, slog.Level(level), msg, args...)
}

func (l *logger) LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr) {
	l.Logger.LogAttrs(ctx, slog.Level(level), msg, attrs...)
}

type warnLogger struct{}

// WarnLogger returns a [Logger] that logs at [LevelWarn]
func WarnLogger() Logger {
	return warnLogger{}
}
func (warnLogger) Println(v ...any)               { Warn(fmt.Sprintln(v...)) }
func (warnLogger) Printf(format string, v ...any) { Warn(fmt.Sprintf(format, v...)) }

type errorLogger struct{}

// ErrorLogger returns a [Logger] that logs at [LevelError]
func ErrorLogger() Logger {
	return errorLogger{}
}
func (errorLogger) Println(v ...any)               { defaultLogger.Error(fmt.Sprintln(v...)) }
func (errorLogger) Printf(format string, v ...any) { defaultLogger.Error(fmt.Sprintf(format, v...)) }

// SetJSONHandler sets the default logger's handler to a [slog.JSONHandler] with the given writer.
func SetJSONHandler(w io.Writer) {
	SetHandler(slog.NewJSONHandler(w, nil))
}

// SetTextHandler sets the default logger's handler to a [slog.TextHandler] with the given writer.
func SetTextHandler(w io.Writer) {
	SetHandler(slog.NewTextHandler(w, nil))
}
