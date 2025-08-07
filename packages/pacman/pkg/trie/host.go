package trie

import (
	"iter"
	"net"
)

type Host[V any] struct {
	host *Hostname[V]
	cidr CIDR[V]
}

func NewHost[V any]() *Host[V] {
	return &Host[V]{
		host: NewHostname[V](),
		cidr: NewCIDR[V](),
	}
}

func (m *Host[V]) Insert(host string, value V) {
	_, ipnet, err := net.ParseCIDR(host)
	if err != nil {
		if ip := net.ParseIP(host); ip != nil {
			// If the host is a single IP address, treat it as a CIDR with a /32 mask
			ipnet = &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(len(ip)*8, len(ip)*8),
			}
			err = nil
		}
	}

	if ipnet != nil {
		// If the host is a CIDR, insert it into the CIDR trie
		m.cidr.Insert(ipnet, value)
	} else {
		// Otherwise, insert it into the host trie
		m.host.Insert(host, value)
	}
}

func (m *Host[V]) Match(host string) (V, bool) {
	if ip := net.ParseIP(host); ip != nil {
		// If the host is an IP address, check the CIDR trie
		return m.cidr.Match(ip)
	}
	return m.host.Match(host)
}

var _ iter.Seq[struct{}] = (*Host[struct{}])(nil).Walk

func (m *Host[V]) Walk(yield func(V) bool) {
	for _, v := range m.host.wildcard {
		if !yield(v) {
			return
		}
	}
	for _, n := range m.cidr {
		if !yield(n.Value) {
			return
		}
	}
}
