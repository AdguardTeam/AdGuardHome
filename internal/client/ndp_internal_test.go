package client

import (
	"net"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNDPNeigh(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		data string
		want map[netip.Addr]net.HardwareAddr
	}{{
		name: "typical",
		data: "fe80::1 dev eth0 lladdr aa:bb:cc:dd:ee:ff REACHABLE\n" +
			"2001:db8::1 dev eth0 lladdr 11:22:33:44:55:66 STALE\n",
		want: map[netip.Addr]net.HardwareAddr{
			netip.MustParseAddr("fe80::1"):    {0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
			netip.MustParseAddr("2001:db8::1"): {0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
		},
	}, {
		name: "no_lladdr",
		data: "fe80::1 dev eth0 FAILED\n",
		want: map[netip.Addr]net.HardwareAddr{},
	}, {
		name: "short_line",
		data: "fe80::1 dev eth0\n",
		want: map[netip.Addr]net.HardwareAddr{},
	}, {
		name: "bad_ip",
		data: "not-an-ip dev eth0 lladdr aa:bb:cc:dd:ee:ff REACHABLE\n",
		want: map[netip.Addr]net.HardwareAddr{},
	}, {
		name: "bad_mac",
		data: "fe80::1 dev eth0 lladdr not-a-mac REACHABLE\n",
		want: map[netip.Addr]net.HardwareAddr{},
	}, {
		name: "router_flag",
		data: "fe80::1 dev eth0 lladdr aa:bb:cc:dd:ee:ff router REACHABLE\n",
		want: map[netip.Addr]net.HardwareAddr{
			netip.MustParseAddr("fe80::1"): {0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		},
	}, {
		name: "empty",
		data: "",
		want: map[netip.Addr]net.HardwareAddr{},
	}, {
		name: "mixed_valid_invalid",
		data: "fe80::1 dev eth0 lladdr aa:bb:cc:dd:ee:ff REACHABLE\n" +
			"bad line\n" +
			"fe80::2 dev eth0 FAILED\n" +
			"fe80::3 dev eth0 lladdr 11:22:33:44:55:66 DELAY\n",
		want: map[netip.Addr]net.HardwareAddr{
			netip.MustParseAddr("fe80::1"): {0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
			netip.MustParseAddr("fe80::3"): {0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := parseNDPNeigh([]byte(tc.data))
			require.Len(t, got, len(tc.want))

			for addr, wantMAC := range tc.want {
				gotMAC, ok := got[addr]
				require.True(t, ok, "missing address %s", addr)

				assert.Equal(t, wantMAC, gotMAC)
			}
		})
	}
}
