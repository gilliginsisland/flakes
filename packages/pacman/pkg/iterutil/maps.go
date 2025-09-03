package iterutil

import (
	"cmp"
	"slices"
)

// SortedMapIter returns an iterator function that yields key-value pairs
// in sorted key order.
func SortedMapIter[K cmp.Ordered, V any](m map[K]V) func(yield func(K, V) bool) {
	return func(yield func(K, V) bool) {
		keys := make([]K, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}
