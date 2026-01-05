package app

import (
	"fmt"
	"log/slog"

	"tailscale.com/net/socks5"

	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/httpproxy"
	"github.com/gilliginsisland/pacman/pkg/netutil"
)

func NewProxyServer(pd *dialer.ByHost) *netutil.MuxServer {
	s := netutil.NewMuxServer()
	s.HandleServer(netutil.SOCKS5Match, &socks5.Server{
		Dialer: pd.DialContext,
		Logf: func(format string, v ...any) {
			slog.Debug(fmt.Sprintf(format, v...))
		},
	})
	s.HandleServer(netutil.DefaultMatch, &httpproxy.Server{
		Dialer: pd.DialContext,
		Handler: &httpproxy.PacHandler{
			Hosts: pd.Hosts,
		},
	})
	return s
}
