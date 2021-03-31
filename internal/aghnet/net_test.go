package aghnet

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetValidNetInterfacesForWeb(t *testing.T) {
	ifaces, err := GetValidNetInterfacesForWeb()
	require.NoErrorf(t, err, "cannot get net interfaces: %s", err)
	require.NotEmpty(t, ifaces, "no net interfaces found")
	for _, iface := range ifaces {
		require.NotEmptyf(t, iface.Addresses, "no addresses found for %s", iface.Name)
	}
}

func TestUnreverseAddr(t *testing.T) {
	testCases := []struct {
		name string
		have string
		want net.IP
	}{{
		name: "good_ipv4",
		have: "1.0.0.127.in-addr.arpa",
		want: net.IP{127, 0, 0, 1},
	}, {
		name: "good_ipv6",
		have: "4.3.2.1.d.c.b.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa",
		want: net.ParseIP("::abcd:1234"),
	}, {
		name: "good_ipv6_case",
		have: "4.3.2.1.d.c.B.A.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.iP6.ArPa",
		want: net.ParseIP("::abcd:1234"),
	}, {
		name: "good_ipv4_dot",
		have: "1.0.0.127.in-addr.arpa.",
		want: net.IP{127, 0, 0, 1},
	}, {
		name: "good_ipv4_case",
		have: "1.0.0.127.In-Addr.Arpa",
		want: net.IP{127, 0, 0, 1},
	}, {
		name: "wrong_ipv4",
		have: ".0.0.127.in-addr.arpa",
		want: nil,
	}, {
		name: "wrong_ipv6",
		have: ".3.2.1.d.c.b.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa",
		want: nil,
	}, {
		name: "bad_ipv6_dot",
		have: "4.3.2.1.d.c.b.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0..ip6.arpa",
		want: nil,
	}, {
		name: "bad_ipv6_space",
		have: "4.3.2.1.d.c.b. .0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa",
		want: nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip := UnreverseAddr(tc.have)
			assert.True(t, tc.want.Equal(ip))
		})
	}
}

func TestReverseAddr(t *testing.T) {
	testCases := []struct {
		name string
		want string
		ip   net.IP
	}{{
		name: "valid_ipv4",
		want: "4.3.2.1.in-addr.arpa",
		ip:   net.IP{1, 2, 3, 4},
	}, {
		name: "valid_ipv6",
		want: "1.3.b.5.4.1.8.6.0.0.0.0.0.0.0.0.0.0.0.0.0.1.0.0.0.0.7.4.6.0.6.2.ip6.arpa",
		ip:   net.ParseIP("2606:4700:10::6814:5b31"),
	}, {
		name: "nil_ip",
		want: "",
		ip:   nil,
	}, {
		name: "unspecified_ipv6",
		want: "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa",
		ip:   net.IPv6unspecified,
	}, {
		name: "unspecified_ipv4",
		want: "0.0.0.0.in-addr.arpa",
		ip:   net.IPv4zero,
	}, {
		name: "wrong_length_ip",
		want: "",
		ip:   net.IP{1, 2, 3, 4, 5},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ReverseAddr(tc.ip))
		})
	}
}
