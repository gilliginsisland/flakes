package netutil

import (
	"bufio"
	"io"
	"log/slog"
	"net"
	"sync"
)

type BuffClientConn struct {
	net.Conn
	*bufio.ReadWriter
}

// Read delegates to bufio.Reader
func (b *BuffClientConn) Read(p []byte) (int, error) {
	return b.ReadWriter.Read(p)
}

// Write delegates to bufio.Writer
func (b *BuffClientConn) Write(p []byte) (int, error) {
	return b.ReadWriter.Write(p)
}

// NewBuffClientConn creates a new BuffClientConn
func NewBuffClientConn(conn net.Conn) *BuffClientConn {
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	return &BuffClientConn{
		Conn:       conn,
		ReadWriter: bufio.NewReadWriter(r, w),
	}
}

func Pipe(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	copy := func(dst, src net.Conn) {
		defer wg.Done()
		defer dst.Close()
		s, err := io.Copy(dst, src)
		slog.Debug(
			"pipe leg closed",
			slog.Any("error", err),
			slog.Int64("bytes", s),
			slog.String("remoteAddr", src.RemoteAddr().String()),
		)
	}

	go copy(b, a)
	go copy(a, b)

	wg.Wait()
}
