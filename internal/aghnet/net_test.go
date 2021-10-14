package aghnet

import (
	"net"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	aghtest.DiscardLogOutput(m)
}

func TestGetValidNetInterfacesForWeb(t *testing.T) {
	ifaces, err := GetValidNetInterfacesForWeb()
	require.NoErrorf(t, err, "cannot get net interfaces: %s", err)
	require.NotEmpty(t, ifaces, "no net interfaces found")
	for _, iface := range ifaces {
		require.NotEmptyf(t, iface.Addresses, "no addresses found for %s", iface.Name)
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
