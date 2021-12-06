package aghnet

import (
	"net"
	"sync/atomic"
)

// IPMutFunc is the signature of a function which modifies the IP address
// instance.  It should be safe for concurrent use.
type IPMutFunc func(ip net.IP)

// nopIPMutFunc is the IPMutFunc that does nothing.
func nopIPMutFunc(net.IP) {}

// IPMut is a type-safe wrapper of atomic.Value to store the IPMutFunc.
type IPMut struct {
	f atomic.Value
}

// NewIPMut returns the new properly initialized *IPMut.  The m is guaranteed to
// always store non-nil IPMutFunc which is safe to call.
func NewIPMut(f IPMutFunc) (m *IPMut) {
	m = &IPMut{
		f: atomic.Value{},
	}
	m.Store(f)

	return m
}

// Store sets the IPMutFunc to return from Func.  It's safe for concurrent use.
// If f is nil, the stored function is the no-op one.
func (m *IPMut) Store(f IPMutFunc) {
	if f == nil {
		f = nopIPMutFunc
	}
	m.f.Store(f)
}

// Load returns the previously stored IPMutFunc.
func (m *IPMut) Load() (f IPMutFunc) {
	return m.f.Load().(IPMutFunc)
}
