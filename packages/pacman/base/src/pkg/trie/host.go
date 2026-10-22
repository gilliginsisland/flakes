package trie

type Host[V any] map[string]V

// Insert adds a wildcard rule (*.example.com → example.com)
func (m *Host[V]) Insert(host string, value V) {
	if *m == nil {
		*m = make(map[string]V)
	}
	(*m)[host] = value
}

// Match finds the most specific match for the given hostname.
func (m Host[V]) Match(host string) (V, bool) {
	val, ok := m[host]
	return val, ok
}
