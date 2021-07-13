package aghnet

import (
	"fmt"
	"net"
)

// ipArr is a representation of an IP address as an array of bytes.
type ipArr [16]byte

// String implements the fmt.Stringer interface for ipArr.
func (a ipArr) String() (s string) {
	return net.IP(a[:]).String()
}

// IPMap is a map of IP addresses.
type IPMap struct {
	m map[ipArr]interface{}
}

// NewIPMap returns a new empty IP map using hint as a size hint for the
// underlying map.
func NewIPMap(hint int) (m *IPMap) {
	return &IPMap{
		m: make(map[ipArr]interface{}, hint),
	}
}

// ipToArr converts a net.IP into an ipArr.
//
// TODO(a.garipov): Use the slice-to-array conversion in Go 1.17.
func ipToArr(ip net.IP) (a ipArr) {
	copy(a[:], ip.To16())

	return a
}

// Del deletes ip from the map.  Calling Del on a nil *IPMap has no effect, just
// like delete on an empty map doesn't.
func (m *IPMap) Del(ip net.IP) {
	if m != nil {
		delete(m.m, ipToArr(ip))
	}
}

// Get returns the value from the map.  Calling Get on a nil *IPMap returns nil
// and false, just like indexing on an empty map does.
func (m *IPMap) Get(ip net.IP) (v interface{}, ok bool) {
	if m != nil {
		v, ok = m.m[ipToArr(ip)]

		return v, ok
	}

	return nil, false
}

// Len returns the length of the map.  A nil *IPMap has a length of zero, just
// like an empty map.
func (m *IPMap) Len() (n int) {
	if m == nil {
		return 0
	}

	return len(m.m)
}

// Range calls f for each key and value present in the map in an undefined
// order.  If cont is false, range stops the iteration.  Calling Range on a nil
// *IPMap has no effect, just like ranging over a nil map.
func (m *IPMap) Range(f func(ip net.IP, v interface{}) (cont bool)) {
	if m == nil {
		return
	}

	for k, v := range m.m {
		// Array slicing produces a pointer, so copy the array here.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/3346
		// as well as https://github.com/kyoh86/looppointer/issues/9.
		k := k
		if !f(net.IP(k[:]), v) {
			break
		}
	}
}

// Set sets the value.  Set panics if the m is a nil *IPMap, just like a nil map
// does.
func (m *IPMap) Set(ip net.IP, v interface{}) {
	m.m[ipToArr(ip)] = v
}

// ShallowClone returns a shallow clone of the map.
func (m *IPMap) ShallowClone() (sclone *IPMap) {
	if m == nil {
		return nil
	}

	sclone = NewIPMap(m.Len())
	m.Range(func(ip net.IP, v interface{}) (cont bool) {
		sclone.Set(ip, v)

		return true
	})

	return sclone
}

// String implements the fmt.Stringer interface for *IPMap.
func (m *IPMap) String() (s string) {
	if m == nil {
		return "<nil>"
	}

	return fmt.Sprint(m.m)
}
