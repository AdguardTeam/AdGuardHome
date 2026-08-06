package aghalg

import (
	"cmp"
	"slices"
)

// SortedMap is a map that keeps elements in order with internal sorting
// function.  It must be initialized with [NewSortedMap] or [NewSortedMapFunc].
type SortedMap[K comparable, V any] struct {
	vals map[K]V
	cmp  func(a, b K) (res int)
	keys []K
}

// NewSortedMap initializes a new instance of sorted map.
func NewSortedMap[K cmp.Ordered, V any]() (m *SortedMap[K, V]) {
	return NewSortedMapFunc[K, V](cmp.Compare[K])
}

// NewSortedMapFunc initializes a new instance of sorted map.  cmpFunc is a
// comparison function to keep elements in order.  cmpFunc must not be nil.
func NewSortedMapFunc[K comparable, V any](cmpFunc func(a, b K) (res int)) (m *SortedMap[K, V]) {
	return &SortedMap[K, V]{
		vals: map[K]V{},
		cmp:  cmpFunc,
	}
}

// Set adds val with key to the sorted map.  It panics if the m is nil.
func (m *SortedMap[K, V]) Set(key K, val V) {
	m.vals[key] = val

	i, has := slices.BinarySearchFunc(m.keys, key, m.cmp)
	if has {
		m.keys[i] = key
	} else {
		m.keys = slices.Insert(m.keys, i, key)
	}
}

// Get returns val by key from the sorted map.
func (m *SortedMap[K, V]) Get(key K) (val V, ok bool) {
	if m == nil {
		var zero V

		return zero, false
	}

	val, ok = m.vals[key]

	return val, ok
}

// Del removes the value by key from the sorted map.
func (m *SortedMap[K, V]) Del(key K) {
	if m == nil {
		return
	}

	if _, has := m.vals[key]; !has {
		return
	}

	delete(m.vals, key)
	i, _ := slices.BinarySearchFunc(m.keys, key, m.cmp)
	m.keys = slices.Delete(m.keys, i, i+1)
}

// Clear removes all elements from the sorted map.
func (m *SortedMap[K, V]) Clear() {
	if m == nil {
		return
	}

	m.keys = m.keys[:0]
	clear(m.vals)
}

// Range calls cb for each element of the map, sorted by m.cmp.  If cb returns
// false it stops.
func (m *SortedMap[K, V]) Range(cb func(K, V) (cont bool)) {
	if m == nil {
		return
	}

	for _, k := range m.keys {
		if !cb(k, m.vals[k]) {
			return
		}
	}
}
