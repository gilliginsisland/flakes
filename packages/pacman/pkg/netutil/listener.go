package netutil

import (
	"context"
	"fmt"
	"net"

	"golang.org/x/sync/errgroup"
)

// ChanListener implements a net.Listener backed by a channel.
type ChanListener chan net.Conn

var (
	_ net.Listener = (ChanListener)(nil)
	_ ConnHandler  = (ChanListener)(nil)
)

func (c ChanListener) Accept() (net.Conn, error) {
	if conn, ok := <-c; ok {
		return conn, nil
	}
	return nil, fmt.Errorf("listener closed")
}

func (c ChanListener) Close() error {
	close(c)
	return nil
}

func (c ChanListener) Addr() net.Addr {
	return nil
}

func (c ChanListener) ServeConn(conn net.Conn) {
	c <- conn
}

type ConnHandler interface {
	ServeConn(net.Conn)
}

type ConnHandlerFn func(net.Conn)

func (handle ConnHandlerFn) Handle(conn net.Conn) {
	handle(conn)
}

type muxEntry struct {
	h     ConnHandler
	match func(*BuffConn) bool
}

// ListenMux wraps a net.Listener and dispatches connections based on match functions.
type ListenMux struct {
	entries []*muxEntry
}

func (m *ListenMux) Handle(match func(*BuffConn) bool, handler ConnHandler) {
	m.entries = append(m.entries, &muxEntry{
		h:     handler,
		match: match,
	})
}

func (m *ListenMux) Listener(match func(*BuffConn) bool) net.Listener {
	l := make(ChanListener)
	m.Handle(match, l)
	return l
}

func (m *ListenMux) ServeConn(conn net.Conn) {
	bc := NewBuffConn(conn)
	for _, e := range m.entries {
		if e.match(bc) {
			e.h.ServeConn(bc)
			return
		}
	}
	bc.Close()
}

func Serve(l net.Listener, h ConnHandler) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go h.ServeConn(conn)
	}
}

type Server interface {
	Serve(l net.Listener) error
}

type MuxServer struct {
	mux *ListenMux
	g   *errgroup.Group
	ctx context.Context
}

func NewMuxServer() *MuxServer {
	g, ctx := errgroup.WithContext(context.Background())
	return &MuxServer{mux: &ListenMux{}, g: g, ctx: ctx}
}

func (s *MuxServer) HandleServer(match func(*BuffConn) bool, srv Server) {
	l := s.mux.Listener(match)
	context.AfterFunc(s.ctx, func() { l.Close() })
	s.g.Go(func() error {
		return srv.Serve(l)
	})
}

func (s *MuxServer) Serve(l net.Listener) error {
	s.g.Go(func() error {
		return Serve(l, s.mux)
	})
	return s.g.Wait()
}

func SOCKS5Match(conn *BuffConn) bool {
	magic, err := conn.Peek(1)
	if err != nil {
		return false
	}
	return magic[0] == 0x05
}

func DefaultMatch(conn *BuffConn) bool {
	return true
}
