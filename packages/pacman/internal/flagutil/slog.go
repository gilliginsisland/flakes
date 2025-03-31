package flagutil

import (
	"log/slog"
)

// LogLevel extends slog.Level and implements encoding.TextUnmarshaler.
type LogLevel struct {
	slog.Level
}

// UnmarshalFlag calls UnmarshalText for go-flags compatibility.
func (l *LogLevel) UnmarshalFlag(value string) error {
	return l.UnmarshalText([]byte(value))
}
