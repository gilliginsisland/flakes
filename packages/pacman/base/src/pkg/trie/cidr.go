package trie

import (
	"net"
	"slices"
)

type cidrNode[V any] struct {
	Network *net.IPNet
	Value   V
}

type CIDR[V any] []*cidrNode[V]

func (t *CIDR[V]) Insert(cidr *net.IPNet, value V) {
	m := cidrNode[V]{Network: cidr, Value: value}
	ones, _ := cidr.Mask.Size()

	for i, existing := range *t {
		curr, _ := existing.Network.Mask.Size()
		if ones > curr {
			*t = slices.Insert(*t, i, &m)
		}
	}

	*t = append(*t, &m)
}

func (t *CIDR[V]) Match(ip net.IP) (V, bool) {
	for _, n := range *t {
		if n.Network.Contains(ip) {
			return n.Value, true
		}
	}
	var zero V
	return zero, false
}
