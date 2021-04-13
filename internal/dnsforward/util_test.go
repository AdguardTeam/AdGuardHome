package dnsforward

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeAddr is a mock implementation of net.Addr interface to simplify testing.
type fakeAddr struct {
	// Addr is embedded here simply to make fakeAddr a net.Addr without
	// actually implementing all methods.
	net.Addr
}

func TestIPFromAddr(t *testing.T) {
	supIPv4 := net.IP{1, 2, 3, 4}
	supIPv6 := net.ParseIP("2a00:1450:400c:c06::93")

	testCases := []struct {
		name string
		addr net.Addr
		want net.IP
	}{{
		name: "ipv4_tcp",
		addr: &net.TCPAddr{
			IP: supIPv4,
		},
		want: supIPv4,
	}, {
		name: "ipv6_tcp",
		addr: &net.TCPAddr{
			IP: supIPv6,
		},
		want: supIPv6,
	}, {
		name: "ipv4_udp",
		addr: &net.UDPAddr{
			IP: supIPv4,
		},
		want: supIPv4,
	}, {
		name: "ipv6_udp",
		addr: &net.UDPAddr{
			IP: supIPv6,
		},
		want: supIPv6,
	}, {
		name: "non-ip_addr",
		addr: &fakeAddr{},
		want: nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IPFromAddr(tc.addr))
		})
	}
}
