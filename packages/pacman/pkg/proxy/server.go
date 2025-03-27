package proxy

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gilliginsisland/pacman/internal/netutil"
	"golang.org/x/net/proxy"
)

// HTTPServer struct
type Server struct {
	dialer proxy.ContextDialer
	client *http.Client
	hndlr  http.Handler
}

func NewServer(dialer proxy.ContextDialer, hndlr http.Handler) *Server {
	return &Server{
		dialer: dialer,
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: dialer.DialContext,
			},
		},
		hndlr: hndlr,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slog.DebugContext(ctx,
		"Serving request",
		slog.String("method", r.Method),
		slog.String("uri", r.RequestURI),
	)

	var err error
	if strings.ToUpper(r.Method) == http.MethodConnect {
		err = s.tunnel(w, r)
	} else if r.URL.IsAbs() {
		err = s.forward(w, r)
	} else {
		s.handleRequest(w, r)
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

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	if s.hndlr == nil {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
	s.hndlr.ServeHTTP(w, r)
}

func (s *Server) tunnel(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	// ensure we can hijack the connection
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return fmt.Errorf("hijacking of connection not supported")
	}

	// connect to the destination (e.g. example.com:443)
	destConn, err := s.dialer.DialContext(ctx, "tcp", r.Host)
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

	netutil.Pipe(destConn, &netutil.BuffClientConn{
		Conn:       clientConn,
		ReadWriter: bufClientConn,
	})
	return nil
}

func (h *Server) forward(w http.ResponseWriter, r *http.Request) error {
	req, err := http.NewRequest(r.Method, r.RequestURI, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	req.Header = r.Header

	resp, err := h.client.Do(req)
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
