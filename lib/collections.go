package lib

import (
	"sort"
)

// GetSortedKeys returns the keys of the map in sorted order.
func GetSortedKeys[R interface {
	~[]K
	sort.Interface
}, K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Sort(R(keys))
	return keys
}
