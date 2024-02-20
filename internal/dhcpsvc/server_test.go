package dhcpsvc_test

import (
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLocalTLD is a common local TLD for tests.
const testLocalTLD = "local"

// testInterfaceConf is a common set of interface configurations for tests.
var testInterfaceConf = map[string]*dhcpsvc.InterfaceConfig{
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
}

// mustParseMAC parses a hardware address from s and requires no errors.
func mustParseMAC(t require.TestingT, s string) (mac net.HardwareAddr) {
	mac, err := net.ParseMAC(s)
	require.NoError(t, err)

	return mac
}

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

func TestDHCPServer_AddLease(t *testing.T) {
	srv, err := dhcpsvc.New(&dhcpsvc.Config{
		Enabled:         true,
		LocalDomainName: testLocalTLD,
		Interfaces:      testInterfaceConf,
	})
	require.NoError(t, err)

	const (
		host1 = "host1"
		host2 = "host2"
		host3 = "host3"
	)

	ip1 := netip.MustParseAddr("192.168.0.2")
	ip2 := netip.MustParseAddr("192.168.0.3")
	ip3 := netip.MustParseAddr("2001:db8::2")

	mac1 := mustParseMAC(t, "01:02:03:04:05:06")
	mac2 := mustParseMAC(t, "06:05:04:03:02:01")
	mac3 := mustParseMAC(t, "02:03:04:05:06:07")

	require.NoError(t, srv.AddLease(&dhcpsvc.Lease{
		Hostname: host1,
		IP:       ip1,
		HWAddr:   mac1,
		IsStatic: true,
	}))

	testCases := []struct {
		name       string
		lease      *dhcpsvc.Lease
		wantErrMsg string
	}{{
		name: "outside_range",
		lease: &dhcpsvc.Lease{
			Hostname: host2,
			IP:       netip.MustParseAddr("1.2.3.4"),
			HWAddr:   mac2,
		},
		wantErrMsg: "adding lease: no interface for ip 1.2.3.4",
	}, {
		name: "duplicate_ip",
		lease: &dhcpsvc.Lease{
			Hostname: host2,
			IP:       ip1,
			HWAddr:   mac2,
		},
		wantErrMsg: "adding lease: lease for ip " + ip1.String() +
			" already exists",
	}, {
		name: "duplicate_hostname",
		lease: &dhcpsvc.Lease{
			Hostname: host1,
			IP:       ip2,
			HWAddr:   mac2,
		},
		wantErrMsg: "adding lease: lease for hostname " + host1 +
			" already exists",
	}, {
		name: "duplicate_hostname_case",
		lease: &dhcpsvc.Lease{
			Hostname: strings.ToUpper(host1),
			IP:       ip2,
			HWAddr:   mac2,
		},
		wantErrMsg: "adding lease: lease for hostname " +
			strings.ToUpper(host1) + " already exists",
	}, {
		name: "duplicate_mac",
		lease: &dhcpsvc.Lease{
			Hostname: host2,
			IP:       ip2,
			HWAddr:   mac1,
		},
		wantErrMsg: "adding lease: lease for mac " + mac1.String() +
			" already exists",
	}, {
		name: "valid",
		lease: &dhcpsvc.Lease{
			Hostname: host2,
			IP:       ip2,
			HWAddr:   mac2,
		},
		wantErrMsg: "",
	}, {
		name: "valid_v6",
		lease: &dhcpsvc.Lease{
			Hostname: host3,
			IP:       ip3,
			HWAddr:   mac3,
		},
		wantErrMsg: "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertErrorMsg(t, tc.wantErrMsg, srv.AddLease(tc.lease))
		})
	}
}

func TestDHCPServer_index(t *testing.T) {
	srv, err := dhcpsvc.New(&dhcpsvc.Config{
		Enabled:         true,
		LocalDomainName: testLocalTLD,
		Interfaces:      testInterfaceConf,
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

	mac1 := mustParseMAC(t, "01:02:03:04:05:06")
	mac2 := mustParseMAC(t, "06:05:04:03:02:01")
	mac3 := mustParseMAC(t, "02:03:04:05:06:07")

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

func TestDHCPServer_UpdateStaticLease(t *testing.T) {
	srv, err := dhcpsvc.New(&dhcpsvc.Config{
		Enabled:         true,
		LocalDomainName: testLocalTLD,
		Interfaces:      testInterfaceConf,
	})
	require.NoError(t, err)

	const (
		host1 = "host1"
		host2 = "host2"
		host3 = "host3"
		host4 = "host4"
		host5 = "host5"
		host6 = "host6"
	)

	ip1 := netip.MustParseAddr("192.168.0.2")
	ip2 := netip.MustParseAddr("192.168.0.3")
	ip3 := netip.MustParseAddr("192.168.0.4")
	ip4 := netip.MustParseAddr("2001:db8::2")
	ip5 := netip.MustParseAddr("2001:db8::3")

	mac1 := mustParseMAC(t, "01:02:03:04:05:06")
	mac2 := mustParseMAC(t, "01:02:03:04:05:07")
	mac3 := mustParseMAC(t, "06:05:04:03:02:01")
	mac4 := mustParseMAC(t, "06:05:04:03:02:02")

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
		Hostname: host4,
		IP:       ip4,
		HWAddr:   mac4,
		IsStatic: true,
	}}
	for _, l := range leases {
		require.NoError(t, srv.AddLease(l))
	}

	testCases := []struct {
		name       string
		lease      *dhcpsvc.Lease
		wantErrMsg string
	}{{
		name: "outside_range",
		lease: &dhcpsvc.Lease{
			Hostname: host1,
			IP:       netip.MustParseAddr("1.2.3.4"),
			HWAddr:   mac1,
		},
		wantErrMsg: "updating static lease: no interface for ip 1.2.3.4",
	}, {
		name: "not_found",
		lease: &dhcpsvc.Lease{
			Hostname: host3,
			IP:       ip3,
			HWAddr:   mac3,
		},
		wantErrMsg: "updating static lease: no lease for mac " + mac3.String(),
	}, {
		name: "duplicate_ip",
		lease: &dhcpsvc.Lease{
			Hostname: host1,
			IP:       ip2,
			HWAddr:   mac1,
		},
		wantErrMsg: "updating static lease: lease for ip " + ip2.String() +
			" already exists",
	}, {
		name: "duplicate_hostname",
		lease: &dhcpsvc.Lease{
			Hostname: host2,
			IP:       ip1,
			HWAddr:   mac1,
		},
		wantErrMsg: "updating static lease: lease for hostname " + host2 +
			" already exists",
	}, {
		name: "duplicate_hostname_case",
		lease: &dhcpsvc.Lease{
			Hostname: strings.ToUpper(host2),
			IP:       ip1,
			HWAddr:   mac1,
		},
		wantErrMsg: "updating static lease: lease for hostname " +
			strings.ToUpper(host2) + " already exists",
	}, {
		name: "valid",
		lease: &dhcpsvc.Lease{
			Hostname: host3,
			IP:       ip3,
			HWAddr:   mac1,
		},
		wantErrMsg: "",
	}, {
		name: "valid_v6",
		lease: &dhcpsvc.Lease{
			Hostname: host6,
			IP:       ip5,
			HWAddr:   mac4,
		},
		wantErrMsg: "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertErrorMsg(t, tc.wantErrMsg, srv.UpdateStaticLease(tc.lease))
		})
	}
}

func TestDHCPServer_RemoveLease(t *testing.T) {
	srv, err := dhcpsvc.New(&dhcpsvc.Config{
		Enabled:         true,
		LocalDomainName: testLocalTLD,
		Interfaces:      testInterfaceConf,
	})
	require.NoError(t, err)

	const (
		host1 = "host1"
		host2 = "host2"
		host3 = "host3"
	)

	ip1 := netip.MustParseAddr("192.168.0.2")
	ip2 := netip.MustParseAddr("192.168.0.3")
	ip3 := netip.MustParseAddr("2001:db8::2")

	mac1 := mustParseMAC(t, "01:02:03:04:05:06")
	mac2 := mustParseMAC(t, "02:03:04:05:06:07")
	mac3 := mustParseMAC(t, "06:05:04:03:02:01")

	leases := []*dhcpsvc.Lease{{
		Hostname: host1,
		IP:       ip1,
		HWAddr:   mac1,
		IsStatic: true,
	}, {
		Hostname: host3,
		IP:       ip3,
		HWAddr:   mac3,
		IsStatic: true,
	}}
	for _, l := range leases {
		require.NoError(t, srv.AddLease(l))
	}

	testCases := []struct {
		name       string
		lease      *dhcpsvc.Lease
		wantErrMsg string
	}{{
		name: "not_found_mac",
		lease: &dhcpsvc.Lease{
			Hostname: host1,
			IP:       ip1,
			HWAddr:   mac2,
		},
		wantErrMsg: "removing lease: no lease for mac " + mac2.String(),
	}, {
		name: "not_found_ip",
		lease: &dhcpsvc.Lease{
			Hostname: host1,
			IP:       ip2,
			HWAddr:   mac1,
		},
		wantErrMsg: "removing lease: no lease for ip " + ip2.String(),
	}, {
		name: "not_found_host",
		lease: &dhcpsvc.Lease{
			Hostname: host2,
			IP:       ip1,
			HWAddr:   mac1,
		},
		wantErrMsg: "removing lease: no lease for hostname " + host2,
	}, {
		name: "valid",
		lease: &dhcpsvc.Lease{
			Hostname: host1,
			IP:       ip1,
			HWAddr:   mac1,
		},
		wantErrMsg: "",
	}, {
		name: "valid_v6",
		lease: &dhcpsvc.Lease{
			Hostname: host3,
			IP:       ip3,
			HWAddr:   mac3,
		},
		wantErrMsg: "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertErrorMsg(t, tc.wantErrMsg, srv.RemoveLease(tc.lease))
		})
	}

	assert.Empty(t, srv.Leases())
}

func TestDHCPServer_Reset(t *testing.T) {
	srv, err := dhcpsvc.New(&dhcpsvc.Config{
		Enabled:         true,
		LocalDomainName: testLocalTLD,
		Interfaces:      testInterfaceConf,
	})
	require.NoError(t, err)

	leases := []*dhcpsvc.Lease{{
		Hostname: "host1",
		IP:       netip.MustParseAddr("192.168.0.2"),
		HWAddr:   mustParseMAC(t, "01:02:03:04:05:06"),
		IsStatic: true,
	}, {
		Hostname: "host2",
		IP:       netip.MustParseAddr("192.168.0.3"),
		HWAddr:   mustParseMAC(t, "06:05:04:03:02:01"),
		IsStatic: true,
	}, {
		Hostname: "host3",
		IP:       netip.MustParseAddr("2001:db8::2"),
		HWAddr:   mustParseMAC(t, "02:03:04:05:06:07"),
		IsStatic: true,
	}, {
		Hostname: "host4",
		IP:       netip.MustParseAddr("2001:db8::3"),
		HWAddr:   mustParseMAC(t, "06:05:04:03:02:02"),
		IsStatic: true,
	}}

	for _, l := range leases {
		require.NoError(t, srv.AddLease(l))
	}

	require.Len(t, srv.Leases(), len(leases))

	require.NoError(t, srv.Reset())

	assert.Empty(t, srv.Leases())
}
