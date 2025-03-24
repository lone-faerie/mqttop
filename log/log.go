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

func With(args ...any) {
	defaultLogger.Logger = defaultLogger.Logger.With(args...)
	defaultLogger.with = args
}

func WithGroup(name string) {
	defaultLogger.Logger = defaultLogger.Logger.With(name)
	defaultLogger.group = name
}

func DefaultLogger() Logger {
	return defaultLogger
}

func SetOutput(w io.Writer) {
	log.SetOutput(w)
}

func Error(msg string, err error, args ...any) {
	if err != nil {
		args = append([]any{"cause", err}, args...)
	}
	defaultLogger.Error(msg, args...)
}

func Fatal(msg string, err error, args ...any) {
	Error(msg, err, args...)
	os.Exit(1)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Println(v ...any) {
	defaultLogger.Info(fmt.Sprintln(v...))
}

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

func WarnLogger() Logger {
	return warnLogger{}
}
func (warnLogger) Println(v ...any)               { Warn(fmt.Sprintln(v...)) }
func (warnLogger) Printf(format string, v ...any) { Warn(fmt.Sprintf(format, v...)) }

type errorLogger struct{}

func ErrorLogger() Logger {
	return errorLogger{}
}
func (errorLogger) Println(v ...any)               { defaultLogger.Error(fmt.Sprintln(v...)) }
func (errorLogger) Printf(format string, v ...any) { defaultLogger.Error(fmt.Sprintf(format, v...)) }

func SetJSONHandler(w io.Writer) {
	SetHandler(slog.NewJSONHandler(w, nil))
}
