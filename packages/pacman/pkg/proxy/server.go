package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/gilliginsisland/pacman/pkg/netutil"
)

// HTTPServer struct
type Server struct {
	Dialer  func(ctx context.Context, network, address string) (net.Conn, error)
	Handler http.Handler
	Client  http.Client
}

func (s *Server) Serve(l net.Listener) error {
	if s.Client.Transport == nil {
		s.Client.Transport = &http.Transport{
			DialContext: s.Dialer,
		}
	}

	return http.Serve(l, s)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slog.DebugContext(ctx,
		"Serving request",
		slog.String("method", r.Method),
		slog.String("uri", r.RequestURI),
	)

	var err error
	switch {
	case strings.ToUpper(r.Method) == http.MethodConnect:
		err = s.tunnel(w, r)
	case r.URL.IsAbs():
		err = s.forward(w, r)
	case s.Handler != nil:
		s.Handler.ServeHTTP(w, r)
	default:
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
	}

	if err != nil {
		slog.ErrorContext(ctx,
			"request handler failed",
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.Any("error", err),
		)
	} else {
		slog.DebugContext(ctx,
			"request handler completed",
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
		)
	}
}

func (s *Server) tunnel(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// ensure we can hijack the connection
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return fmt.Errorf("hijacking of connection not supported")
	}

	// connect to the destination (e.g. example.com:443)
	destConn, err := s.Dialer(ctx, "tcp", r.Host)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return fmt.Errorf("failed to connect to upstream %s: %w", r.Host, err)
	}
	defer destConn.Close()

	w.WriteHeader(http.StatusOK)

	// obtain underlying client TCP connection
	clientConn, bufClientConn, err := hj.Hijack()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return fmt.Errorf("failed to hijack the connection: %w", err)
	}
	defer clientConn.Close()

	netutil.Join(destConn, &netutil.BuffConn{
		Conn:       clientConn,
		ReadWriter: bufClientConn,
	})
	return nil
}

func (s *Server) forward(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, r.Method, r.RequestURI, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	req.Header = r.Header

	resp, err := s.Client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return err
	}
	defer resp.Body.Close()

	header := w.Header()
	for k, vv := range resp.Header {
		for _, v := range vv {
			header.Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
	return nil
}
