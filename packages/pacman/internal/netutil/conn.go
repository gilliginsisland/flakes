package netutil

import (
	"bufio"
	"net"
)

var _ net.Conn = (*BuffConn)(nil)

// BuffConn is a net.Conn wrapper with buffered reads and writes.
type BuffConn struct {
	net.Conn
	*bufio.ReadWriter
}

// NewBuffConn wraps a net.Conn with buffered read and write.
func NewBuffConn(conn net.Conn) *BuffConn {
	return &BuffConn{
		Conn: conn,
		ReadWriter: bufio.NewReadWriter(
			bufio.NewReader(conn),
			bufio.NewWriter(conn),
		),
	}
}

// Read reads from the buffered reader.
func (b *BuffConn) Read(p []byte) (int, error) {
	return b.ReadWriter.Read(p)
}

// Write writes to the buffered writer and flushes immediately.
func (b *BuffConn) Write(p []byte) (int, error) {
	n, err := b.ReadWriter.Write(p)
	if err == nil {
		err = b.Flush()
	}
	return n, err
}
