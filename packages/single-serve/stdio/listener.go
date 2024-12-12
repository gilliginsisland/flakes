package stdio

import (
	"net"
)

type singleConnListener struct {
	conn net.Conn
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	if l.conn == nil {
		return nil, net.ErrClosed
	}
	c := l.conn
	l.conn = nil // Ensure it only accepts once
	return c, nil
}

func (l *singleConnListener) Close() error {
	return nil // No-op, since we don't manage the underlying connection
}

func (l *singleConnListener) Addr() net.Addr {
	return l.conn.RemoteAddr()
}

func Listener() net.Listener {
	return &singleConnListener{
		conn: Conn(),
	}
}
