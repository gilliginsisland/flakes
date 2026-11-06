package stackutil

import (
	"errors"
	"fmt"
	"net"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
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

// NewEndpointDialer configures a stack and resolver for the supplied link
// endpoint. On setup failure it destroys only the stack; the endpoint keeps
// ownership of its transport.
func NewEndpointDialer(ep stack.LinkEndpoint, opts *NetOptions) (*Dialer, error) {
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

	nicID := s.NextNICID()
	if iperr := s.CreateNIC(nicID, ep); iperr != nil {
		s.Destroy()
		return nil, fmt.Errorf("failed to create NIC: %s", iperr.String())
	}

	for _, a := range addrs {
		if iperr := s.AddProtocolAddress(nicID, a, stack.AddressProperties{}); iperr != nil {
			s.Destroy()
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

	dns := make([]string, len(opts.DNS))
	for i, addr := range opts.DNS {
		dns[i] = net.JoinHostPort(addr, "53")
	}

	d := &Dialer{Stack: s}
	d.Resolver = netutil.NewResolver(dns, d.DialContext)
	return d, nil
}
