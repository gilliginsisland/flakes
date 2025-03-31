package flagutil

import (
	"log/slog"

	"github.com/jessevdk/go-flags"
)

// LogLevel extends slog.Level and implements encoding.TextUnmarshaler.
type LogLevel struct {
	slog.Level
}

var _ flags.Unmarshaler = (*LogLevel)(nil)

// UnmarshalFlag calls UnmarshalText for go-flags compatibility.
func (l *LogLevel) UnmarshalFlag(value string) error {
	return l.Level.UnmarshalText([]byte(value))
}
