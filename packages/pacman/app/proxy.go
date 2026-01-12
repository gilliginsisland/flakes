package app

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"tailscale.com/net/socks5"

	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/httpproxy"
	"github.com/gilliginsisland/pacman/pkg/netutil"
	"github.com/gilliginsisland/pacman/pkg/sshproxy"
)

func NewProxyServer(pd *dialer.ByHost) *netutil.MuxServer {
	s := netutil.NewMuxServer()
	s.HandleServer(netutil.SSHMatch, &sshproxy.Server{
		Dialer: pd.DialContext,
		HostKey: func() ssh.Signer {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				// fmt.Errorf("failed to get home directory: %w", err)
				return nil
			}
			keyPath := filepath.Join(homeDir, ".local", "state", "pacman", "ssh_host_key")
			if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
				// fmt.Errorf("failed to create SSH key directory: %w", err)
				return nil
			}

			s, err := sshproxy.LoadOrGenerateHostKey(keyPath)
			if err != nil {
				return nil
			}

			return s
		}(),
	})
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
