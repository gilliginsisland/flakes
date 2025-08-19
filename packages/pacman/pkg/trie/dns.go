package trie

import (
	"strings"
)

// node represents a domain label tree; nil means wildcard match.
type node map[string]node

// Hostname stores exact and wildcard hostname rules
// All values are stored in literal or wildcard map directly, not in the tree
// Tree is used to quickly fail negative wildcard lookups

type DNS[V any] struct {
	hosts    map[string]V // exact match: example.com
	wildcard map[string]V // wildcard match: *.example.com → example.com
	root     node         // tree to fail early on wildcard lookups
}

func NewDNS[V any]() *DNS[V] {
	return &DNS[V]{
		hosts:    make(map[string]V),
		wildcard: make(map[string]V),
		root:     make(node),
	}
}

// Insert inserts a hostname rule
//
// A prefix of "." matches the base domain and all subdomains.
// A prefix of "*." matches only subdomains.
// All other hosts are treated as exact literal matches.
func (h *DNS[V]) Insert(host string, value V) {
	host = canonocalizeHost(host)

	if strings.HasPrefix(host, "*.") {
		host = strings.TrimPrefix(host, "*.")
		h.insertWildcard(host, value)
		return
	}

	if strings.HasPrefix(host, ".") {
		host = strings.TrimPrefix(host, ".")
		h.insertWildcard(host, value)
	}

	h.insertHost(host, value)
}

// insertHost adds an exact rule (e.g., example.com)
func (h *DNS[V]) insertHost(host string, value V) {
	h.hosts[host] = value
}

// insertWildcard adds a wildcard rule (*.example.com → example.com)
func (h *DNS[V]) insertWildcard(suffix string, value V) {
	h.wildcard[suffix] = value

	// Build path in reversed-label trie for fail-fast
	labels := splitHost(suffix)
	for n, i := h.root, len(labels)-1; i >= 0; i-- {
		l := labels[i]
		child, _ := n[l]
		if child != nil {
			n = child
			continue
		}
		for j := range i {
			child = node{labels[j]: child}
		}
		n[l] = child
		break
	}
}

// Match finds the most specific match for the given hostname.
func (t *DNS[V]) Match(host string) (V, bool) {
	host = canonocalizeHost(host)

	val, ok := t.hosts[host]
	if ok {
		return val, true
	}

	labels := splitHost(host)

	var j int
	for n, i := t.root, len(labels)-1; i >= 0; i-- {
		l := labels[i]

		if n, ok = n[l]; !ok {
			j = i
			break
		}

		if n == nil {
			// a nil entry means this is the most specific match
			suffix := strings.Join(labels[i:], ".")
			return t.wildcard[suffix], true
		}
	}

	// no entry means that we don't match and won't match
	// need to backtrack and see if more general matches had a wildcard
	offset := 0
	for i := 0; i <= j; i++ {
		offset += len(labels[i]) + 1 // +1 for the dot
	}

	for i := j + 1; i < len(labels); i++ {
		suffix := host[offset:]
		if v, ok := t.wildcard[suffix]; ok {
			return v, true
		}
		offset += len(labels[i]) + 1
	}

	return val, false
}

func splitHost(host string) []string {
	return strings.Split(host, ".")
}

func canonocalizeHost(host string) string {
	return strings.ToLower(strings.TrimSpace(host))
}
