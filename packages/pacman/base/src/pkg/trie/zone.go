package trie

import (
	"strings"
)

// node represents a domain label tree; nil means wildcard match.
type node map[string]node

// Zone values are stored in the values map directly.
// The trie is used to quickly fail negative wildcard lookups.
type Zone[V any] struct {
	root   node         // tree to fail early on wildcard lookups
	values map[string]V // wildcard match: *.example.com → example.com
}

// Insert adds a wildcard rule (*.example.com → example.com)
func (w *Zone[V]) Insert(zone string, value V) {
	zone = canonocalizeHost(zone)

	if w.values == nil {
		w.values = make(map[string]V)
		w.root = make(node)
	}

	// Save the wildcard value
	w.values[zone] = value

	// Build path in reversed-label trie for fail-fast
	labels := splitHost(zone)
	for n, i := w.root, len(labels)-1; i >= 0; i-- {
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
func (w *Zone[V]) Match(host string) (V, bool) {
	host = canonocalizeHost(host)

	labels := splitHost(host)

	var j int
	for n, i := w.root, len(labels)-1; i >= 0; i-- {
		l := labels[i]

		var ok bool
		n, ok = n[l]
		if !ok {
			j = i
			break
		}

		if n == nil {
			// a nil entry means this is the most specific match
			suffix := strings.Join(labels[i:], ".")
			return w.values[suffix], true
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
		if v, ok := w.values[suffix]; ok {
			return v, true
		}
		offset += len(labels[i]) + 1
	}

	var zero V
	return zero, false
}

func splitHost(host string) []string {
	return strings.Split(host, ".")
}

func canonocalizeHost(host string) string {
	return strings.ToLower(strings.TrimSpace(host))
}
