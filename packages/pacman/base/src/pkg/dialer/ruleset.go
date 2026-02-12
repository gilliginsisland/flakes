package dialer

import (
	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/trie"
)

// RuleSet wraps the trie of host → dialer mappings.
type RuleSet struct {
	trie trie.Trie[proxy.ContextDialer]
}

// Add parses a string specifying a host that should use the given proxy.
// Each value is either an IP address, a CIDR range, a zone (*.example.com) or a
// host name (example.com).
func (rs *RuleSet) Add(host string, p proxy.ContextDialer) {
	rs.trie.Insert(host, p)
}

// Hosts iterates over all hosts in the ruleset.
func (rs *RuleSet) Hosts(yield func(string) bool) {
	if rs == nil {
		return
	}
	for k := range rs.trie.Walk {
		if !yield(k) {
			return
		}
	}
}

// Match finds the dialer for a host, if any.
func (rs *RuleSet) Match(host string) (proxy.ContextDialer, bool) {
	if rs == nil {
		return nil, false
	}
	return rs.trie.Match(host)
}
