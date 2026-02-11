package dhcpsvc_test

import (
	"net"
	"net/netip"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDHCPServer_AddLease(t *testing.T) {
	leasesPath := filepath.Join(t.TempDir(), "leases.json")
	srv := newTestDHCPServer(t, &dhcpsvc.Config{
		DBFilePath: leasesPath,
		Enabled:    true,
	})

	// NOTE: Keep in sync with testdata.
	const (
		existHost = "host1"
		newHost   = "host2"
		ipv6Host  = "host3"
	)

	// NOTE: Keep in sync with testdata.
	var (
		existIP = netip.MustParseAddr("192.0.2.2")
		newIP   = netip.MustParseAddr("192.0.2.3")
		newIPv6 = netip.MustParseAddr("2001:db8::2")

		existMAC = errors.Must(net.ParseMAC("01:02:03:04:05:06"))
		newMAC   = errors.Must(net.ParseMAC("06:05:04:03:02:01"))
		ipv6MAC  = errors.Must(net.ParseMAC("02:03:04:05:06:07"))
	)

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	require.NoError(t, srv.AddLease(ctx, &dhcpsvc.Lease{
		Hostname: existHost,
		IP:       existIP,
		HWAddr:   existMAC,
		IsStatic: true,
	}))

	testCases := []struct {
		name       string
		lease      *dhcpsvc.Lease
		wantErrMsg string
	}{{
		name: "outside_range",
		lease: &dhcpsvc.Lease{
			Hostname: newHost,
			IP:       netip.MustParseAddr("1.2.3.4"),
			HWAddr:   newMAC,
		},
		wantErrMsg: "adding lease: no interface for ip 1.2.3.4",
	}, {
		name: "duplicate_ip",
		lease: &dhcpsvc.Lease{
			Hostname: newHost,
			IP:       existIP,
			HWAddr:   newMAC,
		},
		wantErrMsg: "adding lease: lease for ip " + existIP.String() +
			" already exists",
	}, {
		name: "duplicate_hostname",
		lease: &dhcpsvc.Lease{
			Hostname: existHost,
			IP:       newIP,
			HWAddr:   newMAC,
		},
		wantErrMsg: "adding lease: lease for hostname " + existHost +
			" already exists",
	}, {
		name: "duplicate_hostname_case",
		lease: &dhcpsvc.Lease{
			Hostname: strings.ToUpper(existHost),
			IP:       newIP,
			HWAddr:   newMAC,
		},
		wantErrMsg: "adding lease: lease for hostname " +
			strings.ToUpper(existHost) + " already exists",
	}, {
		name: "duplicate_mac",
		lease: &dhcpsvc.Lease{
			Hostname: newHost,
			IP:       newIP,
			HWAddr:   existMAC,
		},
		wantErrMsg: "adding lease: lease for mac " + existMAC.String() +
			" already exists",
	}, {
		name: "valid",
		lease: &dhcpsvc.Lease{
			Hostname: newHost,
			IP:       newIP,
			HWAddr:   newMAC,
		},
		wantErrMsg: "",
	}, {
		name: "valid_v6",
		lease: &dhcpsvc.Lease{
			Hostname: ipv6Host,
			IP:       newIPv6,
			HWAddr:   ipv6MAC,
		},
		wantErrMsg: "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx = testutil.ContextWithTimeout(t, testTimeout)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, srv.AddLease(ctx, tc.lease))
		})
	}

	assert.NotEmpty(t, srv.Leases())
	assert.FileExists(t, leasesPath)
}

func TestDHCPServer_index(t *testing.T) {
	leasesPath := newTempDB(t)
	srv := newTestDHCPServer(t, &dhcpsvc.Config{
		DBFilePath: leasesPath,
		Enabled:    true,
	})

	// NOTE: Keep in sync with testdata.
	const (
		host1 = "host1"
		host2 = "host2"
		host3 = "host3"
		host4 = "host4"
		host5 = "host5"
	)

	// NOTE: Keep in sync with testdata.
	var (
		ip1 = netip.MustParseAddr("192.0.2.2")
		ip2 = netip.MustParseAddr("192.0.2.3")
		ip3 = netip.MustParseAddr("198.51.100.3")
		ip4 = netip.MustParseAddr("198.51.100.4")

		mac1 = errors.Must(net.ParseMAC("01:02:03:04:05:06"))
		mac2 = errors.Must(net.ParseMAC("06:05:04:03:02:01"))
		mac3 = errors.Must(net.ParseMAC("02:03:04:05:06:07"))
	)

	t.Run("ip_idx", func(t *testing.T) {
		assert.Equal(t, ip1, srv.IPByHost(host1))
		assert.Equal(t, ip2, srv.IPByHost(host2))
		assert.Equal(t, ip3, srv.IPByHost(host3))
		assert.Equal(t, ip4, srv.IPByHost(host4))
		assert.Zero(t, srv.IPByHost(host5))
	})

	t.Run("name_idx", func(t *testing.T) {
		assert.Equal(t, host1, srv.HostByIP(ip1))
		assert.Equal(t, host2, srv.HostByIP(ip2))
		assert.Equal(t, host3, srv.HostByIP(ip3))
		assert.Equal(t, host4, srv.HostByIP(ip4))
		assert.Zero(t, srv.HostByIP(netip.Addr{}))
	})

	t.Run("mac_idx", func(t *testing.T) {
		assert.Equal(t, mac1, srv.MACByIP(ip1))
		assert.Equal(t, mac2, srv.MACByIP(ip2))
		assert.Equal(t, mac3, srv.MACByIP(ip3))
		assert.Equal(t, mac1, srv.MACByIP(ip4))
		assert.Zero(t, srv.MACByIP(netip.Addr{}))
	})
}

func TestDHCPServer_UpdateStaticLease(t *testing.T) {
	leasesPath := newTempDB(t)
	srv := newTestDHCPServer(t, &dhcpsvc.Config{
		DBFilePath: leasesPath,
		Enabled:    true,
	})

	// NOTE: Keep in sync with testdata.
	const (
		host1 = "host1"
		host2 = "host2"
		host3 = "host3"
		host4 = "host4"
		host5 = "host5"
		host6 = "host6"
	)

	// NOTE: Keep in sync with testdata.
	var (
		ip1 = netip.MustParseAddr("192.0.2.2")
		ip2 = netip.MustParseAddr("192.0.2.3")
		ip3 = netip.MustParseAddr("192.0.2.4")
		ip4 = netip.MustParseAddr("2001:db8::2")

		mac1 = errors.Must(net.ParseMAC("01:02:03:04:05:06"))
		mac2 = errors.Must(net.ParseMAC("06:05:04:03:02:01"))
		mac3 = errors.Must(net.ParseMAC("06:05:04:03:02:02"))
	)

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
			HWAddr:   mac2,
		},
		wantErrMsg: "updating static lease: no lease for mac " + mac2.String(),
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
			IP:       ip4,
			HWAddr:   mac3,
		},
		wantErrMsg: "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testutil.ContextWithTimeout(t, testTimeout)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, srv.UpdateStaticLease(ctx, tc.lease))
		})
	}

	assert.FileExists(t, leasesPath)
}

func TestDHCPServer_RemoveLease(t *testing.T) {
	leasesPath := newTempDB(t)
	srv := newTestDHCPServer(t, &dhcpsvc.Config{
		DBFilePath: leasesPath,
		Enabled:    true,
	})

	// NOTE: Keep in sync with testdata.
	const (
		host1 = "host1"
		host2 = "host2"
		host3 = "host3"
	)

	// NOTE: Keep in sync with testdata.
	var (
		existIP = netip.MustParseAddr("192.0.2.2")
		newIP   = netip.MustParseAddr("192.0.2.3")
		newIPv6 = netip.MustParseAddr("2001:db8::2")

		existMAC = errors.Must(net.ParseMAC("01:02:03:04:05:06"))
		newMAC   = errors.Must(net.ParseMAC("02:03:04:05:06:07"))
		ipv6MAC  = errors.Must(net.ParseMAC("06:05:04:03:02:01"))
	)

	testCases := []struct {
		name       string
		lease      *dhcpsvc.Lease
		wantErrMsg string
	}{{
		name: "not_found_mac",
		lease: &dhcpsvc.Lease{
			Hostname: host1,
			IP:       existIP,
			HWAddr:   newMAC,
		},
		wantErrMsg: "removing lease: no lease for mac " + newMAC.String(),
	}, {
		name: "not_found_ip",
		lease: &dhcpsvc.Lease{
			Hostname: host1,
			IP:       newIP,
			HWAddr:   existMAC,
		},
		wantErrMsg: "removing lease: no lease for ip " + newIP.String(),
	}, {
		name: "not_found_host",
		lease: &dhcpsvc.Lease{
			Hostname: host2,
			IP:       existIP,
			HWAddr:   existMAC,
		},
		wantErrMsg: "removing lease: no lease for hostname " + host2,
	}, {
		name: "valid",
		lease: &dhcpsvc.Lease{
			Hostname: host1,
			IP:       existIP,
			HWAddr:   existMAC,
		},
		wantErrMsg: "",
	}, {
		name: "valid_v6",
		lease: &dhcpsvc.Lease{
			Hostname: host3,
			IP:       newIPv6,
			HWAddr:   ipv6MAC,
		},
		wantErrMsg: "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testutil.ContextWithTimeout(t, testTimeout)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, srv.RemoveLease(ctx, tc.lease))
		})
	}

	assert.FileExists(t, leasesPath)
	assert.Empty(t, srv.Leases())
}

func TestDHCPServer_Reset(t *testing.T) {
	leasesPath := newTempDB(t)
	srv := newTestDHCPServer(t, &dhcpsvc.Config{
		DBFilePath: leasesPath,
		Enabled:    true,
	})

	const leasesNum = 4

	require.Len(t, srv.Leases(), leasesNum)

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	require.NoError(t, srv.Reset(ctx))

	assert.FileExists(t, leasesPath)
	assert.Empty(t, srv.Leases())
}

func TestServer_Leases(t *testing.T) {
	leasesPath := newTempDB(t)
	srv := newTestDHCPServer(t, &dhcpsvc.Config{
		DBFilePath: leasesPath,
		Enabled:    true,
	})

	expiry, err := time.Parse(time.RFC3339, "2042-01-02T03:04:05Z")
	require.NoError(t, err)

	wantLeases := []*dhcpsvc.Lease{{
		Expiry:   expiry,
		IP:       netip.MustParseAddr("192.0.2.3"),
		Hostname: "example.host",
		HWAddr:   errors.Must(net.ParseMAC("AA:AA:AA:AA:AA:AA")),
		IsStatic: false,
	}, {
		Expiry:   time.Time{},
		IP:       netip.MustParseAddr("192.0.2.4"),
		Hostname: "example.static.host",
		HWAddr:   errors.Must(net.ParseMAC("BB:BB:BB:BB:BB:BB")),
		IsStatic: true,
	}}
	assert.ElementsMatch(t, wantLeases, srv.Leases())
}
