package netutil

import (
	"io"
)

// Join uses the default buffer size (32 KiB) for copying.
func Join(a, b io.ReadWriteCloser) [2]error {
	return JoinBuffer(a, b, 32*1024)
}

// JoinBuffer copies between a and b using the provided buffer size.
func JoinBuffer(a, b io.ReadWriteCloser, bufSize int) [2]error {
	if bufSize <= 0 {
		panic("JoinBuffer: buffer size must be > 0")
	}

	pipe := func(dst, src io.ReadWriteCloser) <-chan error {
		ch := make(chan error, 1)
		go func() {
			defer dst.Close()
			_, err := io.CopyBuffer(dst, src, make([]byte, bufSize))
			ch <- err
			close(ch)
		}()
		return ch
	}

	chans := [2]<-chan error{
		pipe(b, a), // a → b
		pipe(a, b), // b → a
	}

	var errs [2]error
	for i, ch := range chans {
		errs[i] = <-ch
	}

	return errs
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
