package log

import (
	"bytes"
	"log/slog"
	"testing"
)

func TestLevelString(t *testing.T) {
	var tests = []struct {
		in   Level
		want string
	}{
		{LevelDisabled, "DISABLED"},
		{LevelDisabled + 1, "DISABLED"},
		{LevelError, slog.LevelError.String()},
		{LevelError + 2, (slog.LevelError + 2).String()},
		{LevelError - 2, (slog.LevelError - 2).String()},
		{LevelWarn, slog.LevelWarn.String()},
		{LevelWarn - 1, (slog.LevelWarn - 1).String()},
		{LevelInfo, slog.LevelInfo.String()},
		{LevelInfo + 1, (slog.LevelInfo + 1).String()},
		{LevelInfo - 3, (slog.LevelInfo - 3).String()},
		{LevelDebug, slog.LevelDebug.String()},
	}
	for _, tt := range tests {
		got := tt.in.String()
		if got != tt.want {
			t.Errorf("%d: Wanted %s, got %s", tt.in, tt.want, got)
		}
	}
}

func TestLevelUnmarshalText(t *testing.T) {
	var tests = []struct {
		in   []byte
		want Level
	}{
		{[]byte("DISABLED"), LevelDisabled},
		{[]byte("DiSaBlE"), LevelDisabled},
		{[]byte("false"), LevelDisabled},
		{[]byte("ERROR"), LevelError},
		{[]byte("Error+1"), LevelError + 1},
	}
	for _, tt := range tests {
		var got Level
		if err := got.UnmarshalText(tt.in); err != nil {
			t.Fatalf("%s: %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("%s: Wanted %s, got %s", tt.in, tt.want, got)
		}
	}
}

func TestLevelUnmarshalJSON(t *testing.T) {
	var tests = []struct {
		in   []byte
		want Level
	}{
		{[]byte("\"DISABLED\""), LevelDisabled},
		{[]byte("\"DiSaBlE\""), LevelDisabled},
		{[]byte("\"false\""), LevelDisabled},
		{[]byte("\"ERROR\""), LevelError},
		{[]byte("\"Error+1\""), LevelError + 1},
	}
	for _, tt := range tests {
		var got Level
		if err := got.UnmarshalJSON(tt.in); err != nil {
			t.Fatalf("%s: %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("%s: Wanted %s, got %s", tt.in, tt.want, got)
		}
	}
}

func TestLevelAppendText(t *testing.T) {
	buf := make([]byte, 4, 16)
	want := LevelDisabled
	wantData := []byte("\x00\x00\x00\x00DISABLED")
	data, err := want.AppendText(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, wantData) {
		t.Errorf("%s: Wanted %s, got %s", want, string(wantData), string(data))
	}
}

func TestLevelMarshalText(t *testing.T) {
	var tests = []struct {
		in   Level
		want string
	}{
		{LevelDisabled, "DISABLED"},
		{LevelDisabled + 1, "DISABLED"},
		{LevelError, slog.LevelError.String()},
		{LevelError + 2, (slog.LevelError + 2).String()},
		{LevelError - 2, (slog.LevelError - 2).String()},
		{LevelWarn, slog.LevelWarn.String()},
		{LevelWarn - 1, (slog.LevelWarn - 1).String()},
		{LevelInfo, slog.LevelInfo.String()},
		{LevelInfo + 1, (slog.LevelInfo + 1).String()},
		{LevelInfo - 3, (slog.LevelInfo - 3).String()},
		{LevelDebug, slog.LevelDebug.String()},
	}
	for _, tt := range tests {
		got, err := tt.in.MarshalText()
		if err != nil {
			t.Fatalf("%s: %v", tt.in, err)
		}
		if string(got) != tt.want {
			t.Errorf("%d: Wanted %s, got %s", tt.in, tt.want, got)
		}
	}

}

func TestLevelMarshalJSON(t *testing.T) {
	var tests = []struct {
		in   Level
		want string
	}{
		{LevelDisabled, "DISABLED"},
		{LevelDisabled + 1, "DISABLED"},
		{LevelError, slog.LevelError.String()},
		{LevelError + 2, (slog.LevelError + 2).String()},
		{LevelError - 2, (slog.LevelError - 2).String()},
		{LevelWarn, slog.LevelWarn.String()},
		{LevelWarn - 1, (slog.LevelWarn - 1).String()},
		{LevelInfo, slog.LevelInfo.String()},
		{LevelInfo + 1, (slog.LevelInfo + 1).String()},
		{LevelInfo - 3, (slog.LevelInfo - 3).String()},
		{LevelDebug, slog.LevelDebug.String()},
	}
	for _, tt := range tests {
		got, err := tt.in.MarshalJSON()
		if err != nil {
			t.Fatalf("%s: %v", tt.in, err)
		}
		if string(got) != "\""+tt.want+"\"" {
			t.Errorf("%d: Wanted %s, got %s", tt.in, tt.want, got)
		}
	}

}
