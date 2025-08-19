package netutil

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
)

// ChanListener implements a net.Listener backed by a channel.
type ChanListener struct {
	c    chan net.Conn
	addr net.Addr
}

var _ net.Listener = (*ChanListener)(nil)

func (l *ChanListener) Accept() (net.Conn, error) {
	if conn, ok := <-l.c; ok {
		return conn, nil
	}
	return nil, fmt.Errorf("listener closed")
}

func (l *ChanListener) Close() error {
	close(l.c)
	return nil
}

func (l *ChanListener) Addr() net.Addr {
	return l.addr
}

func (l *ChanListener) ServeConn(conn net.Conn) {
	l.c <- conn
}

type ConnHandler interface {
	ServeConn(net.Conn)
}

type ConnHandlerFn func(net.Conn)

func (handle ConnHandlerFn) Handle(conn net.Conn) {
	handle(conn)
}

type muxConnHandler struct {
	ConnHandler
	match func(*BuffConn) bool
}

// ConnMux wraps a net.Listener and dispatches connections based on match functions.
type ConnMux struct {
	handlers []*muxConnHandler
}

func (m *ConnMux) Handle(match func(*BuffConn) bool, handler ConnHandler) {
	m.handlers = append(m.handlers, &muxConnHandler{
		ConnHandler: handler,
		match:       match,
	})
}

func (m *ConnMux) ServeConn(conn net.Conn) {
	bc := NewBuffConn(conn)
	for _, h := range m.handlers {
		if h.match(bc) {
			h.ServeConn(bc)
			return
		}
	}
	bc.Close()
}

func (m *ConnMux) Serve(l net.Listener) error {
	return Serve(l, m)
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

type muxServer struct {
	Server
	match func(*BuffConn) bool
}

// ServeMux wraps a net.Listener and dispatches connections based on match functions.
type ServeMux struct {
	servers []*muxServer
}

// Handle registers a Server with a match function.
// Returns a channel that will receive at most one error from the server.
func (m *ServeMux) Handle(match func(*BuffConn) bool, srv Server) {
	m.servers = append(m.servers, &muxServer{
		Server: srv,
		match:  match,
	})
}

// Serve runs the connection accept loop until an error occurs or the listener is closed.
func (m *ServeMux) Serve(l net.Listener) error {
	g, ctx := errgroup.WithContext(context.Background())

	mux := ConnMux{}
	for _, srv := range m.servers {
		ch := ChanListener{
			c:    make(chan net.Conn),
			addr: l.Addr(),
		}
		mux.Handle(srv.match, &ch)

		cancelFunc := context.AfterFunc(ctx, func() { ch.Close() })
		defer cancelFunc()

		g.Go(func() error {
			return srv.Serve(&ch)
		})
	}

	g.Go(func() error {
		for {
			if err := ctx.Err(); err != nil {
				return err
			}

			conn, err := l.Accept()
			if err != nil {
				return err
			}

			go mux.ServeConn(conn)
		}
	})

	return g.Wait()
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

// FreePort asks the kernel for a free open port that is ready to use.
func FreePort(network string) (int, error) {
	l, err := net.Listen(network, ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()

	addr := l.Addr().String()
	colon := strings.LastIndexByte(addr, ':')
	if colon < 0 || colon == len(addr)-1 {
		return 0, fmt.Errorf("unexpected address format: %q", addr)
	}

	portStr := addr[colon+1:]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port number %q: %w", portStr, err)
	}

	return port, nil
}
