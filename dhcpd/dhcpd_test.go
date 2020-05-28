package dhcpd

import (
	"bytes"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func check(t *testing.T, result bool, msg string) {
	if !result {
		t.Fatal(msg)
	}
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
	leases := []*Lease{}

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

	leases = normalizeLeases(staticLeases, dynLeases)

	assert.True(t, len(leases) == 3)
	assert.True(t, bytes.Equal(leases[0].HWAddr, []byte{1, 2, 3, 4}))
	assert.True(t, bytes.Equal(leases[0].IP, []byte{0, 2, 3, 4}))
	assert.True(t, bytes.Equal(leases[1].HWAddr, []byte{2, 2, 3, 4}))
	assert.True(t, bytes.Equal(leases[2].HWAddr, []byte{1, 2, 3, 5}))
}
