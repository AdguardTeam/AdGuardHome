package dnsforward

import (
	"net"
	"net/netip"

	"github.com/AdguardTeam/dnsproxy/proxy"
)

// setupDNS64 initializes DNS64 settings, the NAT64 prefixes in particular.  If
// the DNS64 feature is enabled and no prefixes are configured, the default
// Well-Known Prefix is used, just like Section 5.2 of RFC 6147 prescribes.  Any
// configured set of prefixes discards the default Well-Known prefix unless it
// is specified explicitly.  Each prefix also validated to be a valid IPv6
// CIDR with a maximum length of 96 bits.  The first specified prefix is then
// used to synthesize AAAA records.
func (s *Server) setupDNS64() {
	if !s.conf.UseDNS64 {
		return
	}

	if len(s.conf.DNS64Prefixes) == 0 {
		// dns64WellKnownPref is the default prefix to use in an algorithmic
		// mapping for DNS64.
		//
		// See https://datatracker.ietf.org/doc/html/rfc6052#section-2.1.
		dns64WellKnownPref := netip.MustParsePrefix("64:ff9b::/96")

		s.dns64Pref = dns64WellKnownPref
	} else {
		s.dns64Pref = s.conf.DNS64Prefixes[0]
	}
}

// mapDNS64 maps ip to IPv6 address using configured DNS64 prefix.  ip must be a
// valid IPv4.  It panics, if there are no configured DNS64 prefixes, because
// synthesis should not be performed unless DNS64 function enabled.
func (s *Server) mapDNS64(ip netip.Addr) (mapped net.IP) {
	pref := s.dns64Pref.Masked().Addr().As16()
	ipData := ip.As4()

	mapped = make(net.IP, net.IPv6len)
	copy(mapped[:proxy.NAT64PrefixLength], pref[:])
	copy(mapped[proxy.NAT64PrefixLength:], ipData[:])

	return mapped
}
