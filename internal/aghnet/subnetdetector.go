package aghnet

import (
	"net"
)

// SubnetDetector describes IP address properties.
type SubnetDetector struct {
	// spNets is the slice of special-purpose address registries as defined
	// by RFC-6890 (https://tools.ietf.org/html/rfc6890).
	spNets []*net.IPNet

	// locServedNets is the slice of locally-served networks as defined by
	// RFC-6303 (https://tools.ietf.org/html/rfc6303).
	locServedNets []*net.IPNet
}

// NewSubnetDetector returns a new IP detector.
func NewSubnetDetector() (snd *SubnetDetector, err error) {
	spNets := []string{
		// "This" network.
		"0.0.0.0/8",
		// Private-Use Networks.
		"10.0.0.0/8",
		// Shared Address Space.
		"100.64.0.0/10",
		// Loopback.
		"127.0.0.0/8",
		// Link Local.
		"169.254.0.0/16",
		// Private-Use Networks.
		"172.16.0.0/12",
		// IETF Protocol Assignments.
		"192.0.0.0/24",
		// DS-Lite.
		"192.0.0.0/29",
		// TEST-NET-1
		"192.0.2.0/24",
		// 6to4 Relay Anycast.
		"192.88.99.0/24",
		// Private-Use Networks.
		"192.168.0.0/16",
		// Network Interconnect Device Benchmark Testing.
		"198.18.0.0/15",
		// TEST-NET-2.
		"198.51.100.0/24",
		// TEST-NET-3.
		"203.0.113.0/24",
		// Reserved for Future Use.
		"240.0.0.0/4",
		// Limited Broadcast.
		"255.255.255.255/32",

		// Loopback.
		"::1/128",
		// Unspecified.
		"::/128",
		// IPv4-IPv6 Translation Address.
		"64:ff9b::/96",

		// IPv4-Mapped Address.  Since this network is used for mapping
		// IPv4 addresses, we don't include it.
		// "::ffff:0:0/96",

		// Discard-Only Prefix.
		"100::/64",
		// IETF Protocol Assignments.
		"2001::/23",
		// TEREDO.
		"2001::/32",
		// Benchmarking.
		"2001:2::/48",
		// Documentation.
		"2001:db8::/32",
		// ORCHID.
		"2001:10::/28",
		// 6to4.
		"2002::/16",
		// Unique-Local.
		"fc00::/7",
		// Linked-Scoped Unicast.
		"fe80::/10",
	}

	// TODO(e.burkov): It's a subslice of the slice above.  Should be done
	// smarter.
	locServedNets := []string{
		// IPv4.
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"192.0.2.0/24",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"255.255.255.255/32",
		// IPv6.
		"::/128",
		"::1/128",
		"fe80::/10",
		"2001:db8::/32",
	}

	snd = &SubnetDetector{
		spNets:        make([]*net.IPNet, len(spNets)),
		locServedNets: make([]*net.IPNet, len(locServedNets)),
	}
	for i, ipnetStr := range spNets {
		var ipnet *net.IPNet
		_, ipnet, err = net.ParseCIDR(ipnetStr)
		if err != nil {
			return nil, err
		}

		snd.spNets[i] = ipnet
	}
	for i, ipnetStr := range locServedNets {
		var ipnet *net.IPNet
		_, ipnet, err = net.ParseCIDR(ipnetStr)
		if err != nil {
			return nil, err
		}

		snd.locServedNets[i] = ipnet
	}

	return snd, nil
}

// anyNetContains ranges through the given ipnets slice searching for the one
// which contains the ip.  For internal use only.
//
// TODO(e.burkov): Think about memoization.
func anyNetContains(ipnets *[]*net.IPNet, ip net.IP) (is bool) {
	for _, ipnet := range *ipnets {
		if ipnet.Contains(ip) {
			return true
		}
	}

	return false
}

// IsSpecialNetwork returns true if IP address is contained by any of
// special-purpose IP address registries.  It's safe for concurrent use.
func (snd *SubnetDetector) IsSpecialNetwork(ip net.IP) (is bool) {
	return anyNetContains(&snd.spNets, ip)
}

// IsLocallyServedNetwork returns true if IP address is contained by any of
// locally-served IP address registries.  It's safe for concurrent use.
func (snd *SubnetDetector) IsLocallyServedNetwork(ip net.IP) (is bool) {
	return anyNetContains(&snd.locServedNets, ip)
}
