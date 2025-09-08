package cmd

import (
	"encoding"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/jessevdk/go-flags"
	"golang.org/x/sys/unix"
	"tailscale.com/net/socks5"

	"github.com/gilliginsisland/pacman/pkg/env"
	"github.com/gilliginsisland/pacman/pkg/flagutil"
	"github.com/gilliginsisland/pacman/pkg/stackutil"
)

func init() {
	parser.AddCommand("tunsocks", "Userspace TUN to socks", "Starts a socks server passing traffic to VPNFD", &TunSocksCommand{})
}

var _ encoding.TextUnmarshaler = (*SpaceSeparated)(nil)

type SpaceSeparated []string

func (s *SpaceSeparated) UnmarshalText(text []byte) error {
	*s = strings.Fields(string(text))
	return nil
}

type TunEnv struct {
	Fd      int            `env:"VPNFD,required"`
	MTU     uint32         `env:"INTERNAL_IP4_MTU,required"`
	IP      string         `env:"INTERNAL_IP4_ADDRESS,required"`
	Netmask string         `env:"INTERNAL_IP4_NETMASK,required"`
	DNS     SpaceSeparated `env:"INTERNAL_IP4_DNS,required"`
	Domain  string         `env:"CISCO_DEF_DOMAIN"`
}

var _ flags.Commander = (*TunSocksCommand)(nil)

type TunSocksCommand struct {
	ListenAddr flagutil.HostPort `short:"l" long:"listen" default:"127.0.0.1:8080" description:"Listening address"`
}

// Execute runs the check command.
func (c *TunSocksCommand) Execute(args []string) error {
	var e TunEnv
	err := env.Unmarshal(&e, os.Environ())
	if err != nil {
		return fmt.Errorf("error parsing env: %w", err)
	}

	unix.SetNonblock(e.Fd, true)
	rwc := os.NewFile(uintptr(e.Fd), "VPNFD")
	if rwc == nil {
		return fmt.Errorf("Invalid file descriptor: %d", e.Fd)
	}

	sd, err := stackutil.NewTunDialer(rwc, &stackutil.NetOptions{
		Addr:    e.IP,
		Netmask: e.Netmask,
		DNS:     e.DNS,
		Domain:  e.Domain,
		MTU:     e.MTU,
	})
	if err != nil {
		return err
	}

	srvr := socks5.Server{
		Dialer: sd.DialContext,
		Logf: func(format string, v ...any) {
			slog.Info(fmt.Sprintf(format, v...))
		},
	}

	l, err := net.Listen("tcp", string(c.ListenAddr))
	if err != nil {
		return err
	}

	return srvr.Serve(l)
}
