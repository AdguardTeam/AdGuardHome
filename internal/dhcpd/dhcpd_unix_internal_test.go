//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"net"
	"net/netip"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

func testNotify(flags uint32) {
}

// Leases database store/load.
func TestDB(t *testing.T) {
	var err error
	s := server{
		conf: &ServerConfig{
			dbFilePath: filepath.Join(t.TempDir(), dataFilename),
		},
	}

	s.srv4, err = v4Create(&V4ServerConf{
		Enabled:    true,
		RangeStart: netip.MustParseAddr("192.168.10.100"),
		RangeEnd:   netip.MustParseAddr("192.168.10.200"),
		GatewayIP:  netip.MustParseAddr("192.168.10.1"),
		SubnetMask: netip.MustParseAddr("255.255.255.0"),
		notify:     testNotify,
	})
	require.NoError(t, err)

	s.srv6, err = v6Create(V6ServerConf{})
	require.NoError(t, err)

	leases := []*dhcpsvc.Lease{{
		Expiry:   time.Now().Add(time.Hour),
		Hostname: "static-1.local",
		HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		IP:       netip.MustParseAddr("192.168.10.100"),
	}, {
		Hostname: "static-2.local",
		HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xBB},
		IP:       netip.MustParseAddr("192.168.10.101"),
	}}

	srv4, ok := s.srv4.(*v4Server)
	require.True(t, ok)

	err = srv4.addLease(leases[0])
	require.NoError(t, err)

	err = s.srv4.AddStaticLease(leases[1])
	require.NoError(t, err)

	err = s.dbStore()
	require.NoError(t, err)

	err = s.srv4.ResetLeases(nil)
	require.NoError(t, err)

	err = s.dbLoad()
	require.NoError(t, err)

	ll := s.srv4.GetLeases(LeasesAll)
	require.Len(t, ll, len(leases))

	assert.Equal(t, leases[0].HWAddr, ll[0].HWAddr)
	assert.Equal(t, leases[0].IP, ll[0].IP)
	assert.Equal(t, leases[0].Expiry.Unix(), ll[0].Expiry.Unix())

	assert.Equal(t, leases[1].HWAddr, ll[1].HWAddr)
	assert.Equal(t, leases[1].IP, ll[1].IP)
	assert.True(t, ll[1].IsStatic)
}

func TestV4Server_badRange(t *testing.T) {
	testCases := []struct {
		name       string
		gatewayIP  netip.Addr
		subnetMask netip.Addr
		wantErrMsg string
	}{{
		name:       "gateway_in_range",
		gatewayIP:  netip.MustParseAddr("192.168.10.120"),
		subnetMask: netip.MustParseAddr("255.255.255.0"),
		wantErrMsg: "dhcpv4: gateway ip 192.168.10.120 in the ip range: " +
			"192.168.10.20-192.168.10.200",
	}, {
		name:       "outside_range_start",
		gatewayIP:  netip.MustParseAddr("192.168.10.1"),
		subnetMask: netip.MustParseAddr("255.255.255.240"),
		wantErrMsg: "dhcpv4: range start 192.168.10.20 is outside network " +
			"192.168.10.1/28",
	}, {
		name:       "outside_range_end",
		gatewayIP:  netip.MustParseAddr("192.168.10.1"),
		subnetMask: netip.MustParseAddr("255.255.255.224"),
		wantErrMsg: "dhcpv4: range end 192.168.10.200 is outside network " +
			"192.168.10.1/27",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conf := V4ServerConf{
				Enabled:    true,
				RangeStart: netip.MustParseAddr("192.168.10.20"),
				RangeEnd:   netip.MustParseAddr("192.168.10.200"),
				GatewayIP:  tc.gatewayIP,
				SubnetMask: tc.subnetMask,
				notify:     testNotify,
			}

			_, err := v4Create(&conf)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

// cloneUDPAddr returns a deep copy of a.
func cloneUDPAddr(a *net.UDPAddr) (clone *net.UDPAddr) {
	return &net.UDPAddr{
		IP:   slices.Clone(a.IP),
		Port: a.Port,
		Zone: a.Zone,
	}
}
