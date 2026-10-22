package netutil

import (
	"bufio"
	"bytes"
	"net"
	"sync"
	"time"
)

// BuffConn is a net.Conn wrapper with buffered reads.
type BuffConn struct {
	net.Conn
	Reader *bufio.Reader
}

// NewBuffConn wraps a net.Conn with buffered reads.
func NewBuffConn(conn net.Conn) *BuffConn {
	return &BuffConn{
		Conn:   conn,
		Reader: bufio.NewReader(conn),
	}
}

// Read returns buffered detection bytes before reading directly from the connection.
func (bc *BuffConn) Read(p []byte) (int, error) {
	if bc.Reader != nil {
		if bc.Reader.Buffered() == 0 {
			bc.Reader = nil
		} else {
			return bc.Reader.Read(p)
		}
	}
	return bc.Conn.Read(p)
}

// ConnHandler handles a network connection.
type ConnHandler interface {
	ServeConn(net.Conn)
}

// ConnHandlerFn adapts a function to ConnHandler.
type ConnHandlerFn func(net.Conn)

func (fn ConnHandlerFn) ServeConn(conn net.Conn) {
	fn(conn)
}

type muxHandler struct {
	ProtoMatcher
	ConnHandler
}

// MatchResult describes whether a matcher needs more input, rejects, or
// accepts a connection.
type MatchResult uint8

const (
	MatchPending MatchResult = iota
	MatchReject
	MatchAccept
)

// ProtoMatcher classifies bytes peeked from a connection.
type ProtoMatcher func([]byte) MatchResult

// ConnMux dispatches connections based on match functions.
type ConnMux struct {
	entries []muxHandler
}

const muxReadTimeout = 30 * time.Second

// Handle registers handler for connections accepted by match.
func (m *ConnMux) Handle(match ProtoMatcher, handler ConnHandler) {
	m.entries = append(m.entries, muxHandler{match, handler})
}

// ServeConn dispatches conn to the first matching handler.
func (m *ConnMux) ServeConn(conn net.Conn) {
	bc := NewBuffConn(conn)
	if err := bc.SetReadDeadline(time.Now().Add(muxReadTimeout)); err != nil {
		bc.Close()
		return
	}

	handlers := append([]muxHandler(nil), m.entries...)
	for n := 1; len(handlers) > 0; n++ {
		data, err := bc.Reader.Peek(n)
		if err != nil {
			bc.Close()
			return
		}

		for i := 0; i < len(handlers); {
			h := handlers[i]
			switch h.ProtoMatcher(data) {
			case MatchAccept:
				if err := bc.SetReadDeadline(time.Time{}); err != nil {
					bc.Close()
					return
				}
				h.ServeConn(bc)
				return
			case MatchReject:
				handlers = append(handlers[:i], handlers[i+1:]...)
				continue
			}
			i++
		}
	}

	bc.Close()
}

// MuxServer runs a set of servers over one listener selected by matchers.
type MuxServer struct {
	ConnMux
	listeners map[*net.Listener]struct{}
	closed    bool
	mu        sync.Mutex
}

// Listener returns a virtual listener for connections accepted by match.
func (s *MuxServer) Listener(match ProtoMatcher) net.Listener {
	l := newChanListener()
	s.Handle(match, l)
	if err := s.trackListener(new(net.Listener(l)), true); err != nil {
		l.Close()
	}
	return l
}

// ListenAndServe listens on addr and serves connections from it.
func (s *MuxServer) ListenAndServe(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(listener)
}

// Serve runs the registered servers and dispatches listener connections.
func (s *MuxServer) Serve(listener net.Listener) error {
	if err := s.trackListener(&listener, true); err != nil {
		listener.Close()
		return err
	}
	defer s.trackListener(&listener, false)
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go s.ServeConn(conn)
	}
}

// Close closes every listener tracked by Serve or returned by Listener.
func (s *MuxServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	for listener := range s.listeners {
		(*listener).Close()
	}
	s.listeners = nil
}

// trackListener adds or removes a net.Listener to the tracked listener set.
//
// We store a pointer to interface in the map set, in case the net.Listener is
// not comparable. This is safe because callers retain the same pointer for the
// lifetime of each tracked listener. We never need to compare a Listener from
// another caller.
func (s *MuxServer) trackListener(listener *net.Listener, add bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return net.ErrClosed
	}
	if !add {
		delete(s.listeners, listener)
		return nil
	}
	if s.listeners == nil {
		s.listeners = make(map[*net.Listener]struct{})
	}
	s.listeners[listener] = struct{}{}
	return nil
}

type chanListener struct {
	ch    chan net.Conn
	done  chan struct{}
	close func() error
}

func newChanListener() *chanListener {
	var l *chanListener
	l = &chanListener{
		ch:   make(chan net.Conn),
		done: make(chan struct{}),
		close: sync.OnceValue(func() error {
			close(l.done)
			return nil
		}),
	}
	return l
}

func (l *chanListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.ch:
		return conn, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *chanListener) Close() error {
	return l.close()
}

func (l *chanListener) Addr() net.Addr {
	return nil
}

func (l *chanListener) ServeConn(conn net.Conn) {
	select {
	case l.ch <- conn:
	case <-l.done:
		conn.Close()
	}
}

// SOCKS5Match accepts SOCKS5 connections.
func SOCKS5Match(data []byte) MatchResult {
	if data[0] == 0x05 {
		return MatchAccept
	}
	return MatchReject
}

// SSHMatch accepts SSH protocol identification strings.
func SSHMatch(data []byte) MatchResult {
	const magic = "SSH-"

	if bytes.HasPrefix(data, []byte(magic)) {
		return MatchAccept
	}
	if bytes.HasPrefix([]byte(magic), data) {
		return MatchPending
	}
	return MatchReject
}

// HTTPMatch accepts HTTP/1.1 request lines.
func HTTPMatch(data []byte) MatchResult {
	const requestSuffix10 = " HTTP/1.0"
	const requestSuffix11 = " HTTP/1.1"

	i := bytes.IndexAny(data, "\r\n")
	if i == -1 {
		return MatchPending
	}
	if data[i] != '\r' {
		return MatchReject
	}
	if len(data) == i+1 {
		return MatchPending
	}
	requestLine := data[:i]
	if bytes.HasSuffix(requestLine, []byte(requestSuffix10)) ||
		bytes.HasSuffix(requestLine, []byte(requestSuffix11)) {
		return MatchAccept
	}
	return MatchReject
}
