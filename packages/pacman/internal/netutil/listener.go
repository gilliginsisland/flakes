package netutil

import (
	"fmt"
	"net"
	"time"
)

type MuxListener struct {
	net.Listener

	Http  net.Listener
	Socks net.Listener

	http  chan net.Conn
	socks chan net.Conn
}

func NewMuxListener(l net.Listener) *MuxListener {
	mux := MuxListener{
		Listener: l,
		http:     make(chan net.Conn, 100),
		socks:    make(chan net.Conn, 100),
	}
	mux.Http = &chanListener{ch: mux.http, p: &mux}
	mux.Socks = &chanListener{ch: mux.socks, p: &mux}

	go mux.loop()

	return &mux
}

func (m *MuxListener) loop() error {
	defer close(m.http)
	defer close(m.socks)

	for {
		conn, err := m.Listener.Accept()
		if err != nil {
			return err
		}

		m.handle(NewBuffConn(conn))
	}
}

func (m *MuxListener) handle(conn *BuffConn) {
	magic, err := conn.Peek(1)
	if err != nil {
		conn.Close()
		return
	}

	var ch chan<- net.Conn
	switch magic[0] {
	case 0x05:
		ch = m.socks
	default:
		ch = m.http
	}

	select {
	case ch <- conn:
	case <-time.After(5 * time.Second):
		conn.Close()
	}
}

var _ net.Listener = (*chanListener)(nil)

type chanListener struct {
	ch chan net.Conn
	p  net.Listener
}

func (l *chanListener) Accept() (net.Conn, error) {
	if conn, ok := <-l.ch; ok {
		return conn, nil
	}
	return nil, fmt.Errorf("listener closed")
}

func (l *chanListener) Close() error {
	close(l.ch)
	return nil
}

func (l *chanListener) Addr() net.Addr {
	return l.p.Addr()
}
