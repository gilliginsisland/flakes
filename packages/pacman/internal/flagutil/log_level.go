package flagutil

import (
	"log/slog"
)

// LogLevel extends slog.Level to support flag parsing.
type LogLevel struct {
	slog.Level
}

func (l *LogLevel) Set(value string) error {
	return l.UnmarshalText([]byte(value))
}
