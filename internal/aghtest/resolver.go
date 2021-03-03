package aghtest

import (
	"context"
	"crypto/sha256"
	"net"
	"sync"
)

// TestResolver is a Resolver for tests.
type TestResolver struct {
	counter     int
	counterLock sync.Mutex
}

// HostToIPs generates IPv4 and IPv6 from host.
//
// TODO(e.burkov): Replace with LookupIP after upgrading go to v1.15.
func (r *TestResolver) HostToIPs(host string) (ipv4, ipv6 net.IP) {
	hash := sha256.Sum256([]byte(host))

	return net.IP(hash[:4]), net.IP(hash[4:20])
}

// LookupIPAddr implements Resolver interface for *testResolver. It returns the
// slice of net.IPAddr with IPv4 and IPv6 instances.
func (r *TestResolver) LookupIPAddr(_ context.Context, host string) (ips []net.IPAddr, err error) {
	ipv4, ipv6 := r.HostToIPs(host)
	addrs := []net.IPAddr{{
		IP: ipv4,
	}, {
		IP: ipv6,
	}}

	r.counterLock.Lock()
	defer r.counterLock.Unlock()
	r.counter++

	return addrs, nil
}

// LookupHost implements Resolver interface for *testResolver. It returns the
// slice of IPv4 and IPv6 instances converted to strings.
func (r *TestResolver) LookupHost(host string) (addrs []string, err error) {
	ipv4, ipv6 := r.HostToIPs(host)

	r.counterLock.Lock()
	defer r.counterLock.Unlock()
	r.counter++

	return []string{
		ipv4.String(),
		ipv6.String(),
	}, nil
}

// Counter returns the number of requests handled.
func (r *TestResolver) Counter() int {
	r.counterLock.Lock()
	defer r.counterLock.Unlock()

	return r.counter
}
