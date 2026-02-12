package stackutil

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/stack"

	"github.com/gilliginsisland/pacman/pkg/netutil"
)

type Dialer struct {
	Stack    *stack.Stack
	Resolver netutil.IPLookuper
}

func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	transport, version, err := splitNetwork(network)
	if err != nil {
		return nil, err
	}

	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", address, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port %q: %w", portStr, err)
	}

	ips, err := d.Resolver.LookupIP(ctx, "ip"+version, host)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed for %q: %w", host, err)
	}

	var errs []error

	for _, ip := range ips {
		raddr := tcpip.FullAddress{
			Addr: tcpip.AddrFromSlice(ip),
			Port: uint16(port),
		}

		var conn net.Conn
		switch transport {
		case "tcp":
			conn, err = gonet.DialContextTCP(ctx, d.Stack, raddr, ipProtocolNumber(ip))
		case "udp":
			conn, err = gonet.DialUDP(d.Stack, nil, &raddr, ipProtocolNumber(ip))
		default:
			err = fmt.Errorf("invalid transport: %q", transport)
		}

		if err == nil {
			return conn, nil
		}
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		return nil, errors.New("no suitable IP addresses found")
	}
	return nil, errors.Join(errs...)
}

func ipProtocolNumber(ip net.IP) tcpip.NetworkProtocolNumber {
	if ip.To4() != nil {
		return header.IPv4ProtocolNumber
	} else if len(ip) == net.IPv6len {
		return header.IPv6ProtocolNumber
	}
	return 0
}

func splitNetwork(network string) (transport string, version string, err error) {
	switch network {
	case "tcp", "udp", "tcp4", "udp4", "tcp6", "udp6":
		return network[:3], network[3:], nil
	default:
		return "", "", net.UnknownNetworkError(network)
	}
}
