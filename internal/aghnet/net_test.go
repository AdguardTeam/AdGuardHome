package aghnet

import (
	"net"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	aghtest.DiscardLogOutput(m)
}

func TestGetInterfaceByIP(t *testing.T) {
	ifaces, err := GetValidNetInterfacesForWeb()
	require.NoError(t, err)
	require.NotEmpty(t, ifaces)

	for _, iface := range ifaces {
		t.Run(iface.Name, func(t *testing.T) {
			require.NotEmpty(t, iface.Addresses)

			for _, ip := range iface.Addresses {
				ifaceName := GetInterfaceByIP(ip)
				require.Equal(t, iface.Name, ifaceName)
			}
		})
	}
}

func TestBroadcastFromIPNet(t *testing.T) {
	known6 := net.IP{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}

	testCases := []struct {
		name   string
		subnet *net.IPNet
		want   net.IP
	}{{
		name: "full",
		subnet: &net.IPNet{
			IP:   net.IP{192, 168, 0, 1},
			Mask: net.IPMask{255, 255, 15, 0},
		},
		want: net.IP{192, 168, 240, 255},
	}, {
		name: "ipv6_no_mask",
		subnet: &net.IPNet{
			IP: known6,
		},
		want: known6,
	}, {
		name: "ipv4_no_mask",
		subnet: &net.IPNet{
			IP: net.IP{192, 168, 1, 2},
		},
		want: net.IP{192, 168, 1, 255},
	}, {
		name: "unspecified",
		subnet: &net.IPNet{
			IP:   net.IP{0, 0, 0, 0},
			Mask: net.IPMask{0, 0, 0, 0},
		},
		want: net.IPv4bcast,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bc := BroadcastFromIPNet(tc.subnet)
			assert.True(t, bc.Equal(tc.want), bc)
		})
	}
}

func TestCheckPort(t *testing.T) {
	t.Run("tcp_bound", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:")
		require.NoError(t, err)
		testutil.CleanupAndRequireSuccess(t, l.Close)

		ipp := netutil.IPPortFromAddr(l.Addr())
		require.NotNil(t, ipp)
		require.NotNil(t, ipp.IP)
		require.NotZero(t, ipp.Port)

		err = CheckPort("tcp", ipp.IP, ipp.Port)
		target := &net.OpError{}
		require.ErrorAs(t, err, &target)

		assert.Equal(t, "listen", target.Op)
	})

	t.Run("udp_bound", func(t *testing.T) {
		conn, err := net.ListenPacket("udp", "127.0.0.1:")
		require.NoError(t, err)
		testutil.CleanupAndRequireSuccess(t, conn.Close)

		ipp := netutil.IPPortFromAddr(conn.LocalAddr())
		require.NotNil(t, ipp)
		require.NotNil(t, ipp.IP)
		require.NotZero(t, ipp.Port)

		err = CheckPort("udp", ipp.IP, ipp.Port)
		target := &net.OpError{}
		require.ErrorAs(t, err, &target)

		assert.Equal(t, "listen", target.Op)
	})

	t.Run("bad_network", func(t *testing.T) {
		err := CheckPort("bad_network", nil, 0)
		assert.NoError(t, err)
	})

	t.Run("can_bind", func(t *testing.T) {
		err := CheckPort("udp", net.IP{0, 0, 0, 0}, 0)
		assert.NoError(t, err)
	})
}
