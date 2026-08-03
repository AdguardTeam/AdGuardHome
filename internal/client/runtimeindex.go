package client

import "net/netip"

// runtimeIndex stores information about runtime clients.
type runtimeIndex struct {
	// index maps IP address to runtime client.
	index map[netip.Addr]*Runtime
}

// newRuntimeIndex returns initialized runtime index.
func newRuntimeIndex() (ri *runtimeIndex) {
	return &runtimeIndex{
		index: map[netip.Addr]*Runtime{},
	}
}

// client returns the saved runtime client by ip.  If no such client exists,
// returns nil.
func (ri *runtimeIndex) client(ip netip.Addr) (rc *Runtime) {
	return ri.index[ip]
}

// clientByIP returns the exact runtime client for ip.  If there is no
// non-empty exact client, it returns the only client whose address differs
// from ip only by its IPv6 zone.
func (ri *runtimeIndex) clientByIP(ip netip.Addr) (rc *Runtime) {
	rc = ri.client(ip)
	if rc != nil && !rc.isEmpty() {
		return rc
	}

	return ri.clientByIPWithoutZone(ip)
}

// clientByIPWithoutZone returns the only saved runtime client whose IP address
// equals ip after removing its IPv6 zone.  It returns nil if ip is not an
// unzoned IPv6 address, no client matches, or multiple clients match.
func (ri *runtimeIndex) clientByIPWithoutZone(ip netip.Addr) (rc *Runtime) {
	if !ip.Is6() || ip.Zone() != "" {
		return nil
	}

	for addr, c := range ri.index {
		if c.isEmpty() || addr.WithZone("") != ip {
			continue
		}

		if rc != nil {
			return nil
		}

		rc = c
	}

	return rc
}

// add saves the runtime client in the index.  IP address of a client must be
// unique.  See [Runtime.Client].  rc must not be nil.
func (ri *runtimeIndex) add(rc *Runtime) {
	ip := rc.Addr()
	ri.index[ip] = rc
}

// rangeClients calls f for each runtime client in an undefined order.
func (ri *runtimeIndex) rangeClients(f func(rc *Runtime) (cont bool)) {
	for _, rc := range ri.index {
		if !f(rc) {
			return
		}
	}
}

// setInfo sets the client information from cs for runtime client stored by ip.
// If no such client exists, it creates one.
func (ri *runtimeIndex) setInfo(ip netip.Addr, cs Source, hosts []string) (rc *Runtime) {
	rc = ri.index[ip]
	if rc == nil {
		rc = NewRuntime(ip)
		ri.add(rc)
	}

	rc.setInfo(cs, hosts)

	return rc
}

// clearSource removes information from the specified source from all clients.
func (ri *runtimeIndex) clearSource(src Source) {
	for _, rc := range ri.index {
		rc.unset(src)
	}
}

// removeEmpty removes empty runtime clients and returns the number of removed
// clients.
func (ri *runtimeIndex) removeEmpty() (n int) {
	for ip, rc := range ri.index {
		if rc.isEmpty() {
			delete(ri.index, ip)
			n++
		}
	}

	return n
}
