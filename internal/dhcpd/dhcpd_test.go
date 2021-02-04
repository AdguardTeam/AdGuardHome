// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"bytes"
	"net"
	"os"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	aghtest.DiscardLogOutput(m)
}

func testNotify(flags uint32) {
}

// Leases database store/load
func TestDB(t *testing.T) {
	var err error
	s := Server{}
	s.conf.DBFilePath = dbFilename

	conf := V4ServerConf{
		Enabled:    true,
		RangeStart: net.IP{192, 168, 10, 100},
		RangeEnd:   net.IP{192, 168, 10, 200},
		GatewayIP:  net.IP{192, 168, 10, 1},
		SubnetMask: net.IP{255, 255, 255, 0},
		notify:     testNotify,
	}
	s.srv4, err = v4Create(conf)
	assert.Nil(t, err)

	s.srv6, err = v6Create(V6ServerConf{})
	assert.Nil(t, err)

	l := Lease{}
	l.IP = net.IP{192, 168, 10, 100}
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	exp1 := time.Now().Add(time.Hour)
	l.Expiry = exp1

	srv4, ok := s.srv4.(*v4Server)
	assert.True(t, ok)

	srv4.addLease(&l)

	l2 := Lease{}
	l2.IP = net.IP{192, 168, 10, 101}
	l2.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:bb")
	err = s.srv4.AddStaticLease(l2)
	assert.Nil(t, err)

	_ = os.Remove("leases.db")
	s.dbStore()
	s.srv4.ResetLeases(nil)

	s.dbLoad()

	ll := s.srv4.GetLeases(LeasesAll)

	assert.Equal(t, "aa:aa:aa:aa:aa:bb", ll[0].HWAddr.String())
	assert.True(t, net.IP{192, 168, 10, 101}.Equal(ll[0].IP))
	assert.EqualValues(t, leaseExpireStatic, ll[0].Expiry.Unix())

	assert.Equal(t, "aa:aa:aa:aa:aa:aa", ll[1].HWAddr.String())
	assert.True(t, net.IP{192, 168, 10, 100}.Equal(ll[1].IP))
	assert.Equal(t, exp1.Unix(), ll[1].Expiry.Unix())

	_ = os.Remove("leases.db")
}

func TestIsValidSubnetMask(t *testing.T) {
	assert.True(t, isValidSubnetMask([]byte{255, 255, 255, 0}))
	assert.True(t, isValidSubnetMask([]byte{255, 255, 254, 0}))
	assert.True(t, isValidSubnetMask([]byte{255, 255, 252, 0}))
	assert.False(t, isValidSubnetMask([]byte{255, 255, 253, 0}))
	assert.False(t, isValidSubnetMask([]byte{255, 255, 255, 1}))
}

func TestNormalizeLeases(t *testing.T) {
	dynLeases := []*Lease{}
	staticLeases := []*Lease{}

	lease := &Lease{}
	lease.HWAddr = []byte{1, 2, 3, 4}
	dynLeases = append(dynLeases, lease)
	lease = new(Lease)
	lease.HWAddr = []byte{1, 2, 3, 5}
	dynLeases = append(dynLeases, lease)

	lease = new(Lease)
	lease.HWAddr = []byte{1, 2, 3, 4}
	lease.IP = []byte{0, 2, 3, 4}
	staticLeases = append(staticLeases, lease)
	lease = new(Lease)
	lease.HWAddr = []byte{2, 2, 3, 4}
	staticLeases = append(staticLeases, lease)

	leases := normalizeLeases(staticLeases, dynLeases)

	assert.Len(t, leases, 3)
	assert.True(t, bytes.Equal(leases[0].HWAddr, []byte{1, 2, 3, 4}))
	assert.True(t, bytes.Equal(leases[0].IP, []byte{0, 2, 3, 4}))
	assert.True(t, bytes.Equal(leases[1].HWAddr, []byte{2, 2, 3, 4}))
	assert.True(t, bytes.Equal(leases[2].HWAddr, []byte{1, 2, 3, 5}))
}

func TestOptions(t *testing.T) {
	code, val := parseOptionString(" 12  hex  abcdef ")
	assert.EqualValues(t, 12, code)
	assert.True(t, bytes.Equal([]byte{0xab, 0xcd, 0xef}, val))

	code, _ = parseOptionString(" 12  hex  abcdef1 ")
	assert.EqualValues(t, 0, code)

	code, val = parseOptionString("123 ip 1.2.3.4")
	assert.EqualValues(t, 123, code)
	assert.True(t, net.IP{1, 2, 3, 4}.Equal(net.IP(val)))

	code, _ = parseOptionString("256 ip 1.1.1.1")
	assert.EqualValues(t, 0, code)
	code, _ = parseOptionString("-1 ip 1.1.1.1")
	assert.EqualValues(t, 0, code)
	code, _ = parseOptionString("12 ip 1.1.1.1x")
	assert.EqualValues(t, 0, code)
	code, _ = parseOptionString("12 x 1.1.1.1")
	assert.EqualValues(t, 0, code)
}
