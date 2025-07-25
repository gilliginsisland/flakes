package ghost

import "net"

type HostMatcher interface {
	MatchHost(host string) ([]*Proxy, bool)
}

var _ HostMatcher = (*matcher)(nil)

type matcher struct {
	host *HostTrie[[]*Proxy]
	cidr CIDRTrie[[]*Proxy]
}

func CompileRuleSet(rs RuleSet) HostMatcher {
	m := matcher{
		host: NewHostTrie[[]*Proxy](),
		cidr: NewCIDRTrie[[]*Proxy](),
	}
	for _, r := range rs {
		for _, h := range r.Hosts {
			_, ipnet, err := net.ParseCIDR(h)
			if err != nil {
				if ip := net.ParseIP(h); ip != nil {
					// If the host is a single IP address, treat it as a CIDR with a /32 mask
					ipnet = &net.IPNet{
						IP:   ip,
						Mask: net.CIDRMask(len(ip)*8, len(ip)*8),
					}
				}
			}

			if ipnet != nil {
				// If the host is a CIDR, insert it into the CIDR trie
				m.cidr.Insert(ipnet, r.Proxies)
			} else {
				// Otherwise, insert it into the host trie
				m.host.Insert(h, r.Proxies)
			}
		}
	}
	return &m
}

func (m *matcher) MatchHost(host string) ([]*Proxy, bool) {
	if ip := net.ParseIP(host); ip != nil {
		// If the host is an IP address, check the CIDR trie
		return m.cidr.Match(ip)
	}
	return m.host.Match(host)
}
