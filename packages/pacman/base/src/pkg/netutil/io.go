package netutil

import (
	"errors"
	"io"
)

// Join copies between a and b with the default io.CopyBuffer behavior.
func Join(a, b io.ReadWriteCloser) error {
	return JoinBuffer(a, b, 0)
}

// JoinBuffer copies between a and b using the provided buffer size.
// A bufSize of 0 lets io.CopyBuffer choose its default behavior.
func JoinBuffer(a, b io.ReadWriteCloser, bufSize uint32) error {
	pipe := func(dst, src io.ReadWriteCloser) <-chan error {
		ch := make(chan error, 1)
		go func() {
			var buf []byte
			if bufSize != 0 {
				buf = make([]byte, bufSize)
			}
			defer dst.Close()
			_, err := io.CopyBuffer(dst, src, buf)
			ch <- err
			close(ch)
		}()
		return ch
	}

	chans := [2]<-chan error{
		pipe(b, a), // a → b
		pipe(a, b), // b → a
	}

	return errors.Join(<-chans[0], <-chans[1])
}

// RWCDumper wraps an io.ReadWriteCloser and calls the Dumper function
// on each Read and Write, logging the data and label.
type RWCDumper struct {
	io.ReadWriteCloser
	Label string
	Dump  func([]byte, string)
}

func (r *RWCDumper) Read(p []byte) (int, error) {
	n, err := r.ReadWriteCloser.Read(p)
	if r.Dump != nil {
		r.Dump(p[:n], r.Label+" read")
	}
	return n, err
}

func (r *RWCDumper) Write(p []byte) (int, error) {
	n, err := r.ReadWriteCloser.Write(p)
	if r.Dump != nil {
		r.Dump(p[:n], r.Label+" write")
	}
	return n, err
}
