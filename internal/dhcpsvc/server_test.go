package dhcpsvc_test

import (
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLocalTLD is a common local TLD for tests.
const testLocalTLD = "local"

func TestNew(t *testing.T) {
	validIPv4Conf := &dhcpsvc.IPv4Config{
		Enabled:       true,
		GatewayIP:     netip.MustParseAddr("192.168.0.1"),
		SubnetMask:    netip.MustParseAddr("255.255.255.0"),
		RangeStart:    netip.MustParseAddr("192.168.0.2"),
		RangeEnd:      netip.MustParseAddr("192.168.0.254"),
		LeaseDuration: 1 * time.Hour,
	}
	gwInRangeConf := &dhcpsvc.IPv4Config{
		Enabled:       true,
		GatewayIP:     netip.MustParseAddr("192.168.0.100"),
		SubnetMask:    netip.MustParseAddr("255.255.255.0"),
		RangeStart:    netip.MustParseAddr("192.168.0.1"),
		RangeEnd:      netip.MustParseAddr("192.168.0.254"),
		LeaseDuration: 1 * time.Hour,
	}
	badStartConf := &dhcpsvc.IPv4Config{
		Enabled:       true,
		GatewayIP:     netip.MustParseAddr("192.168.0.1"),
		SubnetMask:    netip.MustParseAddr("255.255.255.0"),
		RangeStart:    netip.MustParseAddr("127.0.0.1"),
		RangeEnd:      netip.MustParseAddr("192.168.0.254"),
		LeaseDuration: 1 * time.Hour,
	}

	validIPv6Conf := &dhcpsvc.IPv6Config{
		Enabled:       true,
		RangeStart:    netip.MustParseAddr("2001:db8::1"),
		LeaseDuration: 1 * time.Hour,
		RAAllowSLAAC:  true,
		RASLAACOnly:   true,
	}

	testCases := []struct {
		conf       *dhcpsvc.Config
		name       string
		wantErrMsg string
	}{{
		conf: &dhcpsvc.Config{
			Enabled:         true,
			LocalDomainName: testLocalTLD,
			Interfaces: map[string]*dhcpsvc.InterfaceConfig{
				"eth0": {
					IPv4: validIPv4Conf,
					IPv6: validIPv6Conf,
				},
			},
		},
		name:       "valid",
		wantErrMsg: "",
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			LocalDomainName: testLocalTLD,
			Interfaces: map[string]*dhcpsvc.InterfaceConfig{
				"eth0": {
					IPv4: &dhcpsvc.IPv4Config{Enabled: false},
					IPv6: &dhcpsvc.IPv6Config{Enabled: false},
				},
			},
		},
		name:       "disabled_interfaces",
		wantErrMsg: "",
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			LocalDomainName: testLocalTLD,
			Interfaces: map[string]*dhcpsvc.InterfaceConfig{
				"eth0": {
					IPv4: gwInRangeConf,
					IPv6: validIPv6Conf,
				},
			},
		},
		name: "gateway_within_range",
		wantErrMsg: `interface "eth0": ipv4: ` +
			`gateway ip 192.168.0.100 in the ip range 192.168.0.1-192.168.0.254`,
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			LocalDomainName: testLocalTLD,
			Interfaces: map[string]*dhcpsvc.InterfaceConfig{
				"eth0": {
					IPv4: badStartConf,
					IPv6: validIPv6Conf,
				},
			},
		},
		name: "bad_start",
		wantErrMsg: `interface "eth0": ipv4: ` +
			`range start 127.0.0.1 is not within 192.168.0.1/24`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := dhcpsvc.New(tc.conf)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestDHCPServer_index(t *testing.T) {
	srv, err := dhcpsvc.New(&dhcpsvc.Config{
		Enabled:         true,
		LocalDomainName: testLocalTLD,
		Interfaces: map[string]*dhcpsvc.InterfaceConfig{
			"eth0": {
				IPv4: &dhcpsvc.IPv4Config{
					Enabled:       true,
					GatewayIP:     netip.MustParseAddr("192.168.0.1"),
					SubnetMask:    netip.MustParseAddr("255.255.255.0"),
					RangeStart:    netip.MustParseAddr("192.168.0.2"),
					RangeEnd:      netip.MustParseAddr("192.168.0.254"),
					LeaseDuration: 1 * time.Hour,
				},
				IPv6: &dhcpsvc.IPv6Config{
					Enabled:       true,
					RangeStart:    netip.MustParseAddr("2001:db8::1"),
					LeaseDuration: 1 * time.Hour,
					RAAllowSLAAC:  true,
					RASLAACOnly:   true,
				},
			},
			"eth1": {
				IPv4: &dhcpsvc.IPv4Config{
					Enabled:       true,
					GatewayIP:     netip.MustParseAddr("172.16.0.1"),
					SubnetMask:    netip.MustParseAddr("255.255.255.0"),
					RangeStart:    netip.MustParseAddr("172.16.0.2"),
					RangeEnd:      netip.MustParseAddr("172.16.0.255"),
					LeaseDuration: 1 * time.Hour,
				},
				IPv6: &dhcpsvc.IPv6Config{
					Enabled:       true,
					RangeStart:    netip.MustParseAddr("2001:db9::1"),
					LeaseDuration: 1 * time.Hour,
					RAAllowSLAAC:  true,
					RASLAACOnly:   true,
				},
			},
		},
	})
	require.NoError(t, err)

	const (
		host1 = "host1"
		host2 = "host2"
		host3 = "host3"
		host4 = "host4"
		host5 = "host5"
	)

	ip1 := netip.MustParseAddr("192.168.0.2")
	ip2 := netip.MustParseAddr("192.168.0.3")
	ip3 := netip.MustParseAddr("172.16.0.3")
	ip4 := netip.MustParseAddr("172.16.0.4")

	mac1 := net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	mac2 := net.HardwareAddr{0x06, 0x05, 0x04, 0x03, 0x02, 0x01}
	mac3 := net.HardwareAddr{0x05, 0x04, 0x03, 0x02, 0x01, 0x00}

	leases := []*dhcpsvc.Lease{{
		Hostname: host1,
		IP:       ip1,
		HWAddr:   mac1,
		IsStatic: true,
	}, {
		Hostname: host2,
		IP:       ip2,
		HWAddr:   mac2,
		IsStatic: true,
	}, {
		Hostname: host3,
		IP:       ip3,
		HWAddr:   mac3,
		IsStatic: true,
	}, {
		Hostname: host4,
		IP:       ip4,
		HWAddr:   mac1,
		IsStatic: true,
	}}
	for _, l := range leases {
		require.NoError(t, srv.AddLease(l))
	}

	t.Run("ip_idx", func(t *testing.T) {
		assert.Equal(t, ip1, srv.IPByHost(host1))
		assert.Equal(t, ip2, srv.IPByHost(host2))
		assert.Equal(t, ip3, srv.IPByHost(host3))
		assert.Equal(t, ip4, srv.IPByHost(host4))
		assert.Equal(t, netip.Addr{}, srv.IPByHost(host5))
	})

	t.Run("name_idx", func(t *testing.T) {
		assert.Equal(t, host1, srv.HostByIP(ip1))
		assert.Equal(t, host2, srv.HostByIP(ip2))
		assert.Equal(t, host3, srv.HostByIP(ip3))
		assert.Equal(t, host4, srv.HostByIP(ip4))
		assert.Equal(t, "", srv.HostByIP(netip.Addr{}))
	})

	t.Run("mac_idx", func(t *testing.T) {
		assert.Equal(t, mac1, srv.MACByIP(ip1))
		assert.Equal(t, mac2, srv.MACByIP(ip2))
		assert.Equal(t, mac3, srv.MACByIP(ip3))
		assert.Equal(t, mac1, srv.MACByIP(ip4))
		assert.Nil(t, srv.MACByIP(netip.Addr{}))
	})
}
