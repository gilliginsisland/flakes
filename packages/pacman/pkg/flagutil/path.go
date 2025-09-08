package flagutil

import (
	"fmt"
	"os"
	"strings"

	"github.com/jessevdk/go-flags"
)

// File wraps os.File and implements flag.Value.
type Path string

var _ flags.Unmarshaler = (*Path)(nil)

// UnmarshalText allows opening from filename.
func (p *Path) UnmarshalText(text []byte) error {
	path, err := expandUser(string(text))
	if err != nil {
		return fmt.Errorf("error parsing path: %w", err)
	}
	*p = Path(path)
	return nil
}

// UnmarshalFlag allows parsing from the command line.
func (p *Path) UnmarshalFlag(value string) error {
	return p.UnmarshalText([]byte(value))
}

// expandUser expands a leading "~" to the current user's home directory.
func expandUser(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot expand %w", err)
	}
	// replace the leading '~'
	return home + path[1:], nil
}
