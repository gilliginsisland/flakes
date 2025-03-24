package flagutil

import (
	"fmt"
	"os"
)

// File wraps os.File and implements flag.Value.
type File struct {
	*os.File
}

// Set implements flag.Value to allow parsing from the command line.
func (f *File) Set(value string) error {
	file, err := os.Open(value)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	f.File = file
	return nil
}

// String implements flag.Value to return the file name.
func (f *File) String() string {
	if f != nil && f.File != nil {
		return f.File.Name()
	}
	return ""
}
