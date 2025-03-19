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
type ProxyServer struct {
	dialer proxy.Dialer
	client *http.Client
}

func NewProxyServer(dialer proxy.Dialer) *ProxyServer {
	transport := http.Transport{}
	if xd, ok := dialer.(proxy.ContextDialer); ok {
		transport.DialContext = xd.DialContext
	} else {
		transport.Dial = dialer.Dial
	}
	client := http.Client{
		Transport: &transport,
	}
	return &ProxyServer{
		dialer: dialer,
		client: &client,
	}
}

func (s *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	if strings.ToUpper(r.Method) == http.MethodConnect {
		err = s.tunnel(w, r)
	} else if r.URL.IsAbs() {
		err = s.forward(w, r)
	} else {
		err = s.handleRequest(w, r)
	}
	if err != nil {
		slog.Error(fmt.Sprintf("Error serving request: %s %s: %s", r.Method, r.RequestURI, err))
	}
}

func (s *ProxyServer) handleRequest(w http.ResponseWriter, _ *http.Request) error {
	http.Error(w, "400 Bad Request", http.StatusBadRequest)
	return nil
}

func (s *ProxyServer) tunnel(w http.ResponseWriter, r *http.Request) error {
	// ensure we can hijack the connection
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return fmt.Errorf("hijacking of connection not supported")
	}

	// connect to the destination (e.g. example.com:443)
	destConn, err := netutil.DialContext(r.Context(), s.dialer, "tcp", r.Host)
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

	netutil.Pipe(destConn, bufClientConn)
	return nil
}

func (h *ProxyServer) forward(w http.ResponseWriter, r *http.Request) error {
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
