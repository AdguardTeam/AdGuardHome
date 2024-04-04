package client

import "net/netip"

// RuntimeIndex stores information about runtime clients.
type RuntimeIndex struct {
	// index maps IP address to runtime client.
	index map[netip.Addr]*Runtime
}

// NewRuntimeIndex returns initialized runtime index.
func NewRuntimeIndex() (ri *RuntimeIndex) {
	return &RuntimeIndex{
		index: map[netip.Addr]*Runtime{},
	}
}

// Client returns the saved runtime client by ip.  If no such client exists,
// returns nil.
func (ri *RuntimeIndex) Client(ip netip.Addr) (rc *Runtime) {
	return ri.index[ip]
}

// Add saves the runtime client in the index.  IP address of a client must be
// unique.  See [Runtime.Client].  rc must not be nil.
func (ri *RuntimeIndex) Add(rc *Runtime) {
	ip := rc.Addr()
	ri.index[ip] = rc
}

// Size returns the number of the runtime clients.
func (ri *RuntimeIndex) Size() (n int) {
	return len(ri.index)
}

// Range calls f for each runtime client in an undefined order.
func (ri *RuntimeIndex) Range(f func(rc *Runtime) (cont bool)) {
	for _, rc := range ri.index {
		if !f(rc) {
			return
		}
	}
}

// Delete removes the runtime client by ip.
func (ri *RuntimeIndex) Delete(ip netip.Addr) {
	delete(ri.index, ip)
}

// DeleteBySource removes all runtime clients that have information only from
// the specified source and returns the number of removed clients.
func (ri *RuntimeIndex) DeleteBySource(src Source) (n int) {
	for ip, rc := range ri.index {
		rc.unset(src)

		if rc.isEmpty() {
			delete(ri.index, ip)
			n++
		}
	}

	return n
}
