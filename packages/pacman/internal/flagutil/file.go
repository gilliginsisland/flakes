package flagutil

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

// File wraps os.File and implements flag.Value.
type File struct {
	os.File
}

var _ flags.Unmarshaler = (*File)(nil)

// UnmarshalText allows opening from filename.
func (f *File) UnmarshalText(text []byte) error {
	file, err := os.Open(string(text))
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	f.File = *file
	return nil
}

// UnmarshalFlag allows parsing from the command line.
func (f *File) UnmarshalFlag(value string) error {
	return f.UnmarshalText([]byte(value))
}
