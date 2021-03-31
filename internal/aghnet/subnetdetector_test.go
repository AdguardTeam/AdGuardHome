package aghnet

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubnetDetector_DetectSpecialNetwork(t *testing.T) {
	snd, err := NewSubnetDetector()
	require.NoError(t, err)

	testCases := []struct {
		name string
		ip   net.IP
		want bool
	}{{
		name: "not_specific",
		ip:   net.ParseIP("8.8.8.8"),
		want: false,
	}, {
		name: "this_host_on_this_network",
		ip:   net.ParseIP("0.0.0.0"),
		want: true,
	}, {
		name: "private-Use",
		ip:   net.ParseIP("10.0.0.0"),
		want: true,
	}, {
		name: "shared_address_space",
		ip:   net.ParseIP("100.64.0.0"),
		want: true,
	}, {
		name: "loopback",
		ip:   net.ParseIP("127.0.0.0"),
		want: true,
	}, {
		name: "link_local",
		ip:   net.ParseIP("169.254.0.0"),
		want: true,
	}, {
		name: "private-use",
		ip:   net.ParseIP("172.16.0.0"),
		want: true,
	}, {
		name: "ietf_protocol_assignments",
		ip:   net.ParseIP("192.0.0.0"),
		want: true,
	}, {
		name: "ds-lite",
		ip:   net.ParseIP("192.0.0.0"),
		want: true,
	}, {
		name: "documentation_(test-net-1)",
		ip:   net.ParseIP("192.0.2.0"),
		want: true,
	}, {
		name: "6to4_relay_anycast",
		ip:   net.ParseIP("192.88.99.0"),
		want: true,
	}, {
		name: "private-use",
		ip:   net.ParseIP("192.168.0.0"),
		want: true,
	}, {
		name: "benchmarking",
		ip:   net.ParseIP("198.18.0.0"),
		want: true,
	}, {
		name: "documentation_(test-net-2)",
		ip:   net.ParseIP("198.51.100.0"),
		want: true,
	}, {
		name: "documentation_(test-net-3)",
		ip:   net.ParseIP("203.0.113.0"),
		want: true,
	}, {
		name: "reserved",
		ip:   net.ParseIP("240.0.0.0"),
		want: true,
	}, {
		name: "limited_broadcast",
		ip:   net.ParseIP("255.255.255.255"),
		want: true,
	}, {
		name: "loopback_address",
		ip:   net.ParseIP("::1"),
		want: true,
	}, {
		name: "unspecified_address",
		ip:   net.ParseIP("::"),
		want: true,
	}, {
		name: "ipv4-ipv6_translation",
		ip:   net.ParseIP("64:ff9b::"),
		want: true,
	}, {
		name: "discard-only_address_block",
		ip:   net.ParseIP("100::"),
		want: true,
	}, {
		name: "ietf_protocol_assignments",
		ip:   net.ParseIP("2001::"),
		want: true,
	}, {
		name: "teredo",
		ip:   net.ParseIP("2001::"),
		want: true,
	}, {
		name: "benchmarking",
		ip:   net.ParseIP("2001:2::"),
		want: true,
	}, {
		name: "documentation",
		ip:   net.ParseIP("2001:db8::"),
		want: true,
	}, {
		name: "orchid",
		ip:   net.ParseIP("2001:10::"),
		want: true,
	}, {
		name: "6to4",
		ip:   net.ParseIP("2002::"),
		want: true,
	}, {
		name: "unique-local",
		ip:   net.ParseIP("fc00::"),
		want: true,
	}, {
		name: "linked-scoped_unicast",
		ip:   net.ParseIP("fe80::"),
		want: true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, snd.IsSpecialNetwork(tc.ip))
		})
	}
}

func TestSubnetDetector_DetectLocallyServedNetwork(t *testing.T) {
	snd, err := NewSubnetDetector()
	require.NoError(t, err)

	testCases := []struct {
		name string
		ip   net.IP
		want bool
	}{{
		name: "not_specific",
		ip:   net.ParseIP("8.8.8.8"),
		want: false,
	}, {
		name: "private-Use",
		ip:   net.ParseIP("10.0.0.0"),
		want: true,
	}, {
		name: "loopback",
		ip:   net.ParseIP("127.0.0.0"),
		want: true,
	}, {
		name: "link_local",
		ip:   net.ParseIP("169.254.0.0"),
		want: true,
	}, {
		name: "private-use",
		ip:   net.ParseIP("172.16.0.0"),
		want: true,
	}, {
		name: "documentation_(test-net-1)",
		ip:   net.ParseIP("192.0.2.0"),
		want: true,
	}, {
		name: "private-use",
		ip:   net.ParseIP("192.168.0.0"),
		want: true,
	}, {
		name: "documentation_(test-net-2)",
		ip:   net.ParseIP("198.51.100.0"),
		want: true,
	}, {
		name: "documentation_(test-net-3)",
		ip:   net.ParseIP("203.0.113.0"),
		want: true,
	}, {
		name: "limited_broadcast",
		ip:   net.ParseIP("255.255.255.255"),
		want: true,
	}, {
		name: "loopback_address",
		ip:   net.ParseIP("::1"),
		want: true,
	}, {
		name: "unspecified_address",
		ip:   net.ParseIP("::"),
		want: true,
	}, {
		name: "documentation",
		ip:   net.ParseIP("2001:db8::"),
		want: true,
	}, {
		name: "linked-scoped_unicast",
		ip:   net.ParseIP("fe80::"),
		want: true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, snd.IsLocallyServedNetwork(tc.ip))
		})
	}
}

func TestSubnetDetector_Detect_parallel(t *testing.T) {
	t.Parallel()

	snd, err := NewSubnetDetector()
	require.NoError(t, err)

	testFunc := func() {
		for _, ip := range []net.IP{
			net.IPv4allrouter,
			net.IPv4allsys,
			net.IPv4bcast,
			net.IPv4zero,
			net.IPv6interfacelocalallnodes,
			net.IPv6linklocalallnodes,
			net.IPv6linklocalallrouters,
			net.IPv6loopback,
			net.IPv6unspecified,
		} {
			_ = snd.IsSpecialNetwork(ip)
			_ = snd.IsLocallyServedNetwork(ip)
		}
	}

	const goroutinesNum = 50
	for i := 0; i < goroutinesNum; i++ {
		go testFunc()
	}
}
