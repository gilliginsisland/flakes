package stackutil

import (
	"errors"
	"fmt"
	"io"
	"net"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"

	"github.com/gilliginsisland/pacman/pkg/netutil"
)

type NetOptions struct {
	Addr     string
	Netmask  string
	Addr6    string
	Netmask6 string
	DNS      []string
	Domain   string
	MTU      uint32
}

func (opts *NetOptions) protocolAddrs() ([]tcpip.ProtocolAddress, error) {
	var addrs []tcpip.ProtocolAddress
	var errs []error

	if opts.Addr != "" {
		ip := net.ParseIP(opts.Addr).To4()
		if ip == nil {
			errs = append(errs, fmt.Errorf("invalid IPv4 address: %q", opts.Addr))
		}

		mask := net.ParseIP(opts.Netmask).To4()
		if mask == nil {
			errs = append(errs, fmt.Errorf("invalid IPv4 netmask: %q", opts.Netmask))
		}

		if ip != nil && mask != nil {
			ones, bits := net.IPv4Mask(mask[0], mask[1], mask[2], mask[3]).Size()
			if ones == 0 && bits == 0 {
				errs = append(errs, fmt.Errorf("non-contiguous or invalid IPv4 netmask: %q", opts.Netmask))
			} else {
				addrs = append(addrs, tcpip.ProtocolAddress{
					Protocol: ipv4.ProtocolNumber,
					AddressWithPrefix: tcpip.AddressWithPrefix{
						Address:   tcpip.AddrFrom4Slice(ip),
						PrefixLen: ones,
					},
				})
			}
		}
	}

	if opts.Netmask6 != "" {
		ip, ipNet, err := net.ParseCIDR(opts.Netmask6)

		if err != nil {
			errs = append(errs, fmt.Errorf("invalid IPv6 CIDR netmask: %q: %w", opts.Netmask6, err))
		} else {
			ip = ip.To16()
			if ip == nil {
				errs = append(errs, fmt.Errorf("invalid IPv6 address: %q", opts.Netmask6))
			}

			ones, _ := ipNet.Mask.Size()
			addrs = append(addrs, tcpip.ProtocolAddress{
				Protocol: ipv6.ProtocolNumber,
				AddressWithPrefix: tcpip.AddressWithPrefix{
					Address:   tcpip.AddrFrom16Slice(ip),
					PrefixLen: ones,
				},
			})
		}
	}

	return addrs, errors.Join(errs...)
}

func NewTunDialer(rwc io.ReadWriteCloser, opts *NetOptions) (*Dialer, error) {
	dns := make([]string, len(opts.DNS))
	for i, addr := range opts.DNS {
		dns[i] = net.JoinHostPort(addr, "53")
	}

	addrs, err := opts.protocolAddrs()
	if err != nil {
		return nil, err
	}

	s := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
		},
	})

	var ep stack.LinkEndpoint
	ch := channel.New(1024, opts.MTU, "")
	// if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
	// 	ep = &DumpingLinkEndpoint{
	// 		LinkEndpoint: ch,
	// 		Dumper: PacketDumperFunc(func(pkt *stack.PacketBuffer, chain string) {
	// 			b := pkt.ToBuffer()
	// 			netutil.DumpPacket(b.Flatten(), chain)
	// 		}),
	// 	}
	// } else {
	ep = ch
	// }

	cleanup := func() {
		ep.Close()
		s.Close()
	}

	nicID := s.NextNICID()
	iperr := s.CreateNIC(nicID, ep)
	if iperr != nil {
		cleanup()
		return nil, fmt.Errorf("Failed to create nic: %s", iperr.String())
	}

	for _, a := range addrs {
		iperr := s.AddProtocolAddress(nicID, a, stack.AddressProperties{})
		if iperr != nil {
			cleanup()
			return nil, fmt.Errorf("failed to add protocol address: %s", iperr.String())
		}

		var subnet tcpip.Subnet
		switch a.Protocol {
		case ipv4.ProtocolNumber:
			subnet = header.IPv4EmptySubnet
		case ipv6.ProtocolNumber:
			subnet = header.IPv6EmptySubnet
		}

		s.AddRoute(tcpip.Route{
			Destination: subnet,
			NIC:         nicID,
		})
	}

	sd := Dialer{Stack: s}
	sd.Resolver = netutil.NewResolver(dns, sd.DialContext)

	go func() {
		netutil.JoinBuffer(rwc, WrapChannel(ch), int(opts.MTU))
		cleanup()
	}()

	return &sd, nil
}
