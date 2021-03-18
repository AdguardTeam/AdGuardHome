// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	aghtest.DiscardLogOutput(m)
}

func testNotify(flags uint32) {
}

// Leases database store/load.
func TestDB(t *testing.T) {
	var err error
	s := Server{
		conf: ServerConfig{
			DBFilePath: dbFilename,
		},
	}

	s.srv4, err = v4Create(V4ServerConf{
		Enabled:    true,
		RangeStart: net.IP{192, 168, 10, 100},
		RangeEnd:   net.IP{192, 168, 10, 200},
		GatewayIP:  net.IP{192, 168, 10, 1},
		SubnetMask: net.IP{255, 255, 255, 0},
		notify:     testNotify,
	})
	require.Nil(t, err)

	s.srv6, err = v6Create(V6ServerConf{})
	require.Nil(t, err)

	leases := []Lease{{
		IP:     net.IP{192, 168, 10, 100},
		HWAddr: net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		Expiry: time.Now().Add(time.Hour),
	}, {
		IP:     net.IP{192, 168, 10, 101},
		HWAddr: net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xBB},
	}}

	srv4, ok := s.srv4.(*v4Server)
	require.True(t, ok)

	err = srv4.addLease(&leases[0])
	require.Nil(t, err)
	require.Nil(t, s.srv4.AddStaticLease(leases[1]))

	s.dbStore()
	t.Cleanup(func() {
		assert.Nil(t, os.Remove(dbFilename))
	})
	s.srv4.ResetLeases(nil)
	s.dbLoad()

	ll := s.srv4.GetLeases(LeasesAll)
	require.Len(t, ll, len(leases))

	assert.Equal(t, leases[1].HWAddr, ll[0].HWAddr)
	assert.Equal(t, leases[1].IP, ll[0].IP)
	assert.True(t, ll[0].IsStatic())

	assert.Equal(t, leases[0].HWAddr, ll[1].HWAddr)
	assert.Equal(t, leases[0].IP, ll[1].IP)
	assert.Equal(t, leases[0].Expiry.Unix(), ll[1].Expiry.Unix())
}

func TestIsValidSubnetMask(t *testing.T) {
	testCases := []struct {
		mask net.IP
		want bool
	}{{
		mask: net.IP{255, 255, 255, 0},
		want: true,
	}, {
		mask: net.IP{255, 255, 254, 0},
		want: true,
	}, {
		mask: net.IP{255, 255, 252, 0},
		want: true,
	}, {
		mask: net.IP{255, 255, 253, 0},
	}, {
		mask: net.IP{255, 255, 255, 1},
	}}

	for _, tc := range testCases {
		t.Run(tc.mask.String(), func(t *testing.T) {
			assert.Equal(t, tc.want, isValidSubnetMask(tc.mask))
		})
	}
}

func TestNormalizeLeases(t *testing.T) {
	dynLeases := []*Lease{{
		HWAddr: net.HardwareAddr{1, 2, 3, 4},
	}, {
		HWAddr: net.HardwareAddr{1, 2, 3, 5},
	}}

	staticLeases := []*Lease{{
		HWAddr: net.HardwareAddr{1, 2, 3, 4},
		IP:     net.IP{0, 2, 3, 4},
	}, {
		HWAddr: net.HardwareAddr{2, 2, 3, 4},
	}}

	leases := normalizeLeases(staticLeases, dynLeases)
	require.Len(t, leases, 3)

	assert.Equal(t, leases[0].HWAddr, dynLeases[0].HWAddr)
	assert.Equal(t, leases[0].IP, staticLeases[0].IP)
	assert.Equal(t, leases[1].HWAddr, staticLeases[1].HWAddr)
	assert.Equal(t, leases[2].HWAddr, dynLeases[1].HWAddr)
}
