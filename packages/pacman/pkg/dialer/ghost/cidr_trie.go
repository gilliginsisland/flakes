package ghost

import (
	"net"
	"slices"
)

type CIDRMatcher[V any] struct {
	Network *net.IPNet
	Value   V
}

type CIDRTrie[V any] []*CIDRMatcher[V]

func NewCIDRTrie[V any]() CIDRTrie[V] {
	return make(CIDRTrie[V], 0)
}

func (t *CIDRTrie[V]) Insert(cidr *net.IPNet, value V) {
	m := CIDRMatcher[V]{Network: cidr, Value: value}
	ones, _ := cidr.Mask.Size()

	for i, existing := range *t {
		curr, _ := existing.Network.Mask.Size()
		if ones > curr {
			*t = slices.Insert(*t, i, &m)
		}
	}

	*t = append(*t, &m)
}

func (t *CIDRTrie[V]) Match(ip net.IP) (V, bool) {
	for _, matcher := range *t {
		if matcher.Network.Contains(ip) {
			return matcher.Value, true
		}
	}
	var zero V
	return zero, false
}
