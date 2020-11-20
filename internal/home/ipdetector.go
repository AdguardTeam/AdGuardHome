package home

import "net"

// ipDetector describes IP address properties.
type ipDetector struct {
	nets []*net.IPNet
}

// newIPDetector returns a new IP detector.
func newIPDetector() (ipd *ipDetector, err error) {
	specialNetworks := []string{
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.0.0.0/29",
		"192.0.2.0/24",
		"192.88.99.0/24",
		"192.168.0.0/16",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"240.0.0.0/4",
		"255.255.255.255/32",
		"::1/128",
		"::/128",
		"64:ff9b::/96",
		// Since this network is used for mapping IPv4 addresses, we
		// don't include it.
		// "::ffff:0:0/96",
		"100::/64",
		"2001::/23",
		"2001::/32",
		"2001:2::/48",
		"2001:db8::/32",
		"2001:10::/28",
		"2002::/16",
		"fc00::/7",
		"fe80::/10",
	}

	ipd = &ipDetector{
		nets: make([]*net.IPNet, len(specialNetworks)),
	}
	for i, ipnetStr := range specialNetworks {
		_, ipnet, err := net.ParseCIDR(ipnetStr)
		if err != nil {
			return nil, err
		}

		ipd.nets[i] = ipnet
	}

	return ipd, nil
}

// detectSpecialNetwork returns true if IP address is contained by any of
// special-purpose IP address registries according to RFC-6890
// (https://tools.ietf.org/html/rfc6890).
func (ipd *ipDetector) detectSpecialNetwork(ip net.IP) bool {
	for _, ipnet := range ipd.nets {
		if ipnet.Contains(ip) {
			return true
		}
	}

	return false
}
