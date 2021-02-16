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

	srv4.addLease(&leases[0])
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
	assert.EqualValues(t, leaseExpireStatic, ll[0].Expiry.Unix())

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

func TestOptions(t *testing.T) {
	testCases := []struct {
		name     string
		optStr   string
		wantVal  []byte
		wantCode uint8
	}{{
		name:     "success_hex",
		optStr:   "12 hex abcdef",
		wantVal:  []byte{0xab, 0xcd, 0xef},
		wantCode: 12,
	}, {
		name:     "bad_hex",
		optStr:   "12 hex abcdefx",
		wantVal:  nil,
		wantCode: 0,
	}, {
		name:     "success_ip",
		optStr:   "123 ip 1.2.3.4",
		wantVal:  net.IP{1, 2, 3, 4},
		wantCode: 123,
	}, {
		name:   "success_ipv6",
		optStr: "123 ip ::1234",
		wantVal: net.IP{
			0, 0, 0, 0,
			0, 0, 0, 0,
			0, 0, 0, 0,
			0, 0, 0x12, 0x34,
		},
		wantCode: 123,
	}, {
		name:     "bad_code",
		optStr:   "256 ip 1.1.1.1",
		wantVal:  nil,
		wantCode: 0,
	}, {
		name:     "negative_code",
		optStr:   "-1 ip 1.1.1.1",
		wantVal:  nil,
		wantCode: 0,
	}, {
		name:     "bad_ip",
		optStr:   "12 ip 1.1.1.1x",
		wantVal:  nil,
		wantCode: 0,
	}, {
		name:     "bad_mode",
		wantVal:  nil,
		optStr:   "12 x 1.1.1.1",
		wantCode: 0,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code, val := parseOptionString(tc.optStr)
			require.Equal(t, tc.wantCode, code)
			if tc.wantVal != nil {
				assert.Equal(t, tc.wantVal, val)
			}
		})
	}
}
