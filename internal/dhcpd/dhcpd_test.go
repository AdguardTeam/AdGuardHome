// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"bytes"
	"net"
	"os"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
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
		RangeStart: "192.168.10.100",
		RangeEnd:   "192.168.10.200",
		GatewayIP:  "192.168.10.1",
		SubnetMask: "255.255.255.0",
		notify:     testNotify,
	}
	s.srv4, err = v4Create(conf)
	assert.True(t, err == nil)

	s.srv6, err = v6Create(V6ServerConf{})
	assert.True(t, err == nil)

	l := Lease{}
	l.IP = net.ParseIP("192.168.10.100").To4()
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	exp1 := time.Now().Add(time.Hour)
	l.Expiry = exp1
	s.srv4.(*v4Server).addLease(&l)

	l2 := Lease{}
	l2.IP = net.ParseIP("192.168.10.101").To4()
	l2.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:bb")
	s.srv4.AddStaticLease(l2)

	_ = os.Remove("leases.db")
	s.dbStore()
	s.srv4.ResetLeases(nil)

	s.dbLoad()

	ll := s.srv4.GetLeases(LeasesAll)

	assert.Equal(t, "aa:aa:aa:aa:aa:bb", ll[0].HWAddr.String())
	assert.Equal(t, "192.168.10.101", ll[0].IP.String())
	assert.Equal(t, int64(leaseExpireStatic), ll[0].Expiry.Unix())

	assert.Equal(t, "aa:aa:aa:aa:aa:aa", ll[1].HWAddr.String())
	assert.Equal(t, "192.168.10.100", ll[1].IP.String())
	assert.Equal(t, exp1.Unix(), ll[1].Expiry.Unix())

	_ = os.Remove("leases.db")
}

func TestIsValidSubnetMask(t *testing.T) {
	assert.True(t, isValidSubnetMask([]byte{255, 255, 255, 0}))
	assert.True(t, isValidSubnetMask([]byte{255, 255, 254, 0}))
	assert.True(t, isValidSubnetMask([]byte{255, 255, 252, 0}))
	assert.True(t, !isValidSubnetMask([]byte{255, 255, 253, 0}))
	assert.True(t, !isValidSubnetMask([]byte{255, 255, 255, 1}))
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

	assert.True(t, len(leases) == 3)
	assert.True(t, bytes.Equal(leases[0].HWAddr, []byte{1, 2, 3, 4}))
	assert.True(t, bytes.Equal(leases[0].IP, []byte{0, 2, 3, 4}))
	assert.True(t, bytes.Equal(leases[1].HWAddr, []byte{2, 2, 3, 4}))
	assert.True(t, bytes.Equal(leases[2].HWAddr, []byte{1, 2, 3, 5}))
}

func TestOptions(t *testing.T) {
	code, val := parseOptionString(" 12  hex  abcdef ")
	assert.Equal(t, uint8(12), code)
	assert.True(t, bytes.Equal([]byte{0xab, 0xcd, 0xef}, val))

	code, _ = parseOptionString(" 12  hex  abcdef1 ")
	assert.Equal(t, uint8(0), code)

	code, val = parseOptionString("123 ip 1.2.3.4")
	assert.Equal(t, uint8(123), code)
	assert.Equal(t, "1.2.3.4", net.IP(string(val)).String())

	code, _ = parseOptionString("256 ip 1.1.1.1")
	assert.Equal(t, uint8(0), code)
	code, _ = parseOptionString("-1 ip 1.1.1.1")
	assert.Equal(t, uint8(0), code)
	code, _ = parseOptionString("12 ip 1.1.1.1x")
	assert.Equal(t, uint8(0), code)
	code, _ = parseOptionString("12 x 1.1.1.1")
	assert.Equal(t, uint8(0), code)
}
