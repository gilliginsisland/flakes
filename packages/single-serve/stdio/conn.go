package stdio

import (
	"io"
	"net"
	"os"
	"time"
)

// Conn wraps io.Stdin and io.Stdout to implement net.Conn interface
type readerWriterConn struct {
	Reader io.Reader
	Writer io.Writer
}

// Read implements the net.Conn Read method
func (c *readerWriterConn) Read(b []byte) (n int, err error) {
	return c.Reader.Read(b)
}

// Write implements the net.Conn Write method
func (c *readerWriterConn) Write(b []byte) (n int, err error) {
	return c.Writer.Write(b)
}

// Close implements the net.Conn Close method
func (c *readerWriterConn) Close() error {
	return nil // Stdin and Stdout cannot be closed
}

// LocalAddr implements the net.Conn LocalAddr method
func (c *readerWriterConn) LocalAddr() net.Addr {
	return nil // Not applicable
}

// RemoteAddr implements the net.Conn RemoteAddr method
func (c *readerWriterConn) RemoteAddr() net.Addr {
	return nil // Not applicable
}

// SetDeadline implements the net.Conn SetDeadline method
func (c *readerWriterConn) SetDeadline(t time.Time) error {
	return nil // Not applicable
}

// SetReadDeadline implements the net.Conn SetReadDeadline method
func (c *readerWriterConn) SetReadDeadline(t time.Time) error {
	return nil // Not applicable
}

// SetWriteDeadline implements the net.Conn SetWriteDeadline method
func (c *readerWriterConn) SetWriteDeadline(t time.Time) error {
	return nil // Not applicable
}

func Conn() net.Conn {
	return &readerWriterConn{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}
}
