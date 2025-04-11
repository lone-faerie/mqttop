package log

import (
	"bytes"
	"log/slog"
	"strconv"
	"strings"
)

// A Level is the importance or severity of a log event.
// The higher the level, the more important or severe the event.
type Level slog.Level

// Names for common levels.
//
// Level numbers are inherently arbitrary,
// but we picked them to satisfy three constraints.
// Any system can map them to another numbering scheme if it wishes.
//
// First, we wanted the default level to be Info, Since Levels are ints, Info is
// the default value for int, zero.
//
// Second, we wanted to make it easy to use levels to specify logger verbosity.
// Since a larger level means a more severe event, a logger that accepts events
// with smaller (or more negative) level means a more verbose logger. Logger
// verbosity is thus the negation of event severity, and the default verbosity
// of 0 accepts all events at least as severe as INFO.
//
// Third, we wanted some room between levels to accommodate schemes with named
// levels between ours. For example, Google Cloud Logging defines a Notice level
// between Info and Warn. Since there are only a few of these intermediate
// levels, the gap between the numbers need not be large. Our gap of 4 matches
// OpenTelemetry's mapping. Subtracting 9 from an OpenTelemetry level in the
// DEBUG, INFO, WARN and ERROR ranges converts it to the corresponding slog
// Level range. OpenTelemetry also has the names TRACE and FATAL, which slog
// does not. But those OpenTelemetry levels can still be represented as slog
// Levels by using the appropriate integers.
const (
	LevelDebug    = Level(slog.LevelDebug)
	LevelInfo     = Level(slog.LevelInfo)
	LevelWarn     = Level(slog.LevelWarn)
	LevelError    = Level(slog.LevelError)
	LevelDisabled = Level(1<<31 - 1)
)

// String returns a name for the level.
// If the level has a name, then that name
// in uppercase is returned.
// If the level is between named values, then
// an integer is appended to the uppercased name.
// Examples:
//
//	LevelWarn.String() => "WARN"
func (l Level) String() string {
	if l >= LevelDisabled {
		return "DISABLED"
	}

	return slog.Level(l).String()
}

// MarshalJSON implements [encoding/json.Marshaler]
// by quoting the output of [Level.String].
func (l Level) MarshalJSON() ([]byte, error) {
	// AppendQuote is sufficient for JSON-encoding all Level strings.
	// They don't contain any runes that would produce invalid JSON
	// when escaped.
	return strconv.AppendQuote(nil, l.String()), nil
}

// UnmarshalJSON implements [encoding/json.Unmarshaler]
// It accepts any string produced by [Level.MarshalJSON],
// ignoring case.
// It also accepts numeric offsets that would result in a different string on
// output. For example, "Error-8" would marshal as "INFO".
func (l *Level) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}

	switch strings.ToLower(s) {
	case "disable", "disabled", "false":
		*l = LevelDisabled
	default:
		return (*slog.Level)(l).UnmarshalJSON(data)
	}

	return nil
}

// AppendText implements [encoding.TextAppender]
// by calling [Level.String].
func (l Level) AppendText(b []byte) ([]byte, error) {
	return append(b, l.String()...), nil
}

// MarshalText implements [encoding.TextMarshaler]
// by calling [Level.AppendText].
func (l Level) MarshalText() ([]byte, error) {
	return l.AppendText(nil)
}

// UnmarshalText implements [encoding.TextUnmarshaler].
// It accepts any string produced by [Level.MarshalText],
// ignoring case.
// It also accepts numeric offsets that would result in a different string on
// output. For example, "Error-8" would marshal as "INFO".
func (l *Level) UnmarshalText(data []byte) (err error) {
	switch string(bytes.ToLower(data)) {
	case "disable", "disabled", "false":
		*l = LevelDisabled
	default:
		err = (*slog.Level)(l).UnmarshalText(data)
	}

	return
}

// Level returns the receiver.
// It implements [slog.Leveler].
func (l Level) Level() Level { return l }

// LevelFlag implements the interfaces needed to be used as a command-line flag.
type LevelFlag Level

func (lf *LevelFlag) String() string {
	return (Level)(*lf).String()
}

func (lf *LevelFlag) Set(s string) error {
	return lf.UnmarshalText([]byte(s))
}

func (lf *LevelFlag) Get() any {
	return (Level)(*lf)
}

func (lf *LevelFlag) Type() string {
	return "level"
}

func (lf *LevelFlag) UnmarshalText(b []byte) error {
	return (*Level)(lf).UnmarshalText(b)
}
