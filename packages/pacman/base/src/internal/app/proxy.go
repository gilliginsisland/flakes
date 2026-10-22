package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	httpPprof "net/http/pprof"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
	"tailscale.com/net/socks5"

	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/httpproxy"
	"github.com/gilliginsisland/pacman/pkg/netutil"
	"github.com/gilliginsisland/pacman/pkg/sshproxy"
)

type ProxyServer struct {
	mux   netutil.MuxServer
	socks socks5.Server
	ssh   sshproxy.Server
	http  httpproxy.Server
}

func server(mux *netutil.MuxServer, match netutil.ProtoMatcher, srv interface {
	Serve(net.Listener) error
}) func() error {
	listener := mux.Listener(match)
	return func() error { return srv.Serve(listener) }
}

func NewProxyServer(pd *dialer.ByHost) *ProxyServer {
	var server ProxyServer
	server.socks = socks5.Server{
		Dialer: pd.DialContext,
		Logf: func(format string, v ...any) {
			slog.Debug(fmt.Sprintf(format, v...))
		},
	}
	server.ssh = sshproxy.Server{
		Dialer: pd.DialContext,
		HostKey: func() ssh.Signer {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				// fmt.Errorf("failed to get home directory: %w", err)
				return nil
			}
			keyPath := filepath.Join(homeDir, ".local", "state", "pacman", "ssh_host_key")
			if err = os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
				// fmt.Errorf("failed to create SSH key directory: %w", err)
				return nil
			}

			s, err := sshproxy.LoadOrGenerateHostKey(keyPath)
			if err != nil {
				return nil
			}

			return s
		}(),
	}
	mux := http.NewServeMux()
	mux.Handle("/proxy.pac", &httpproxy.PacHandler{
		Hosts: pd.Hosts,
	})
	pprofPrefix := "/debug/pprof/"
	mux.HandleFunc(pprofPrefix, httpPprof.Index)
	mux.HandleFunc(pprofPrefix+"cmdline", httpPprof.Cmdline)
	mux.HandleFunc(pprofPrefix+"profile", httpPprof.Profile)
	mux.HandleFunc(pprofPrefix+"symbol", httpPprof.Symbol)
	mux.HandleFunc(pprofPrefix+"trace", httpPprof.Trace)
	server.http = httpproxy.Server{
		Dialer:  pd.DialContext,
		Handler: mux,
	}
	return &server
}

func (s *ProxyServer) Serve(listener net.Listener) error {
	g, ctx := errgroup.WithContext(context.Background())
	context.AfterFunc(ctx, s.mux.Close)

	g.Go(server(&s.mux, netutil.SOCKS5Match, &s.socks))
	g.Go(server(&s.mux, netutil.SSHMatch, &s.ssh))
	g.Go(server(&s.mux, netutil.HTTPMatch, &s.http))
	g.Go(func() error { return s.mux.Serve(listener) })
	return g.Wait()
}
