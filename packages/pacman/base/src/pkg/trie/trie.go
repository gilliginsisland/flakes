package trie

import (
	"iter"
	"net"
	"strings"
)

type Trie[V any] struct {
	host Host[V]
	zone Zone[V]
	cidr CIDR[V]
}

// Insert parses a string specifying a host that should use the given proxy.
// Each value is either an IP address, a CIDR range, a zone (*.example.com) or a
// host name (example.com).
func (m *Trie[V]) Insert(host string, value V) {
	if _, ipnet, err := net.ParseCIDR(host); err == nil {
		m.cidr.Insert(ipnet, value)
	} else if base, ok := strings.CutPrefix(host, "."); ok {
		m.zone.Insert(base, value)
		m.host.Insert(base, value)
	} else if base, ok := strings.CutPrefix(host, "*."); ok {
		m.zone.Insert(base, value)
	} else {
		m.host.Insert(host, value)
	}
}

func (m *Trie[V]) Match(host string) (V, bool) {
	if v, ok := m.host.Match(host); ok {
		return v, ok
	}
	if ip := net.ParseIP(host); ip != nil {
		// If the host is an IP address, check the CIDR trie
		return m.cidr.Match(ip)
	}
	return m.zone.Match(host)
}

var _ iter.Seq2[string, struct{}] = (*Trie[struct{}])(nil).Walk

func (m *Trie[V]) Walk(yield func(string, V) bool) {
	for k, v := range m.host {
		if !yield(k, v) {
			return
		}
	}
	for k, v := range m.zone.values {
		if !yield("*."+k, v) {
			return
		}
	}
	for _, n := range m.cidr {
		if !yield(n.Network.String(), n.Value) {
			return
		}
	}
}
