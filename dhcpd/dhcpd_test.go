package dhcpd

import (
	"bytes"
	"net"
	"os"
	"testing"
	"time"

	"github.com/krolaw/dhcp4"
	"github.com/stretchr/testify/assert"
)

func check(t *testing.T, result bool, msg string) {
	if !result {
		t.Fatal(msg)
	}
}

// Tests performed:
// . Handle Discover message (lease reserve)
// . Handle Request message (lease commit)
// . Static leases
func TestDHCP(t *testing.T) {
	var s = Server{}
	s.conf.DBFilePath = dbFilename
	defer func() { _ = os.Remove(dbFilename) }()
	var p, p2 dhcp4.Packet
	var hw net.HardwareAddr
	var lease *Lease
	var opt dhcp4.Options

	s.reset()
	s.leaseStart = []byte{1, 1, 1, 1}
	s.leaseStop = []byte{1, 1, 1, 2}
	s.leaseTime = 5 * time.Second
	s.leaseOptions = dhcp4.Options{}
	s.ipnet = &net.IPNet{
		IP:   []byte{1, 2, 3, 4},
		Mask: []byte{0xff, 0xff, 0xff, 0xff},
	}

	p = make(dhcp4.Packet, 241)

	// Discover and reserve an IP
	hw = []byte{3, 2, 3, 4, 5, 6}
	p.SetCHAddr(hw)
	p.SetCIAddr([]byte{0, 0, 0, 0})
	opt = make(dhcp4.Options, 10)
	p2 = s.handleDiscover(p, opt)
	opt = p2.ParseOptions()
	check(t, bytes.Equal(opt[dhcp4.OptionDHCPMessageType], []byte{byte(dhcp4.Offer)}), "dhcp4.Offer")
	check(t, bytes.Equal(p2.YIAddr(), []byte{1, 1, 1, 1}), "p2.YIAddr")
	check(t, bytes.Equal(p2.CHAddr(), hw), "p2.CHAddr")
	check(t, bytes.Equal(opt[dhcp4.OptionIPAddressLeaseTime], dhcp4.OptionsLeaseTime(5*time.Second)), "OptionIPAddressLeaseTime")
	check(t, bytes.Equal(opt[dhcp4.OptionServerIdentifier], s.ipnet.IP), "OptionServerIdentifier")

	lease = s.findLease(p)
	check(t, bytes.Equal(lease.HWAddr, hw), "lease.HWAddr")
	check(t, bytes.Equal(lease.IP, []byte{1, 1, 1, 1}), "lease.IP")

	// Reserve an IP - the next IP from the range
	hw = []byte{2, 2, 3, 4, 5, 6}
	p.SetCHAddr(hw)
	lease, _ = s.reserveLease(p)
	check(t, bytes.Equal(lease.HWAddr, hw), "lease.HWAddr")
	check(t, bytes.Equal(lease.IP, []byte{1, 1, 1, 2}), "lease.IP")

	// Reserve an IP - we have no more available IPs,
	//  so the first expired (or, in our case, not yet committed) lease is returned
	hw = []byte{1, 2, 3, 4, 5, 6}
	p.SetCHAddr(hw)
	lease, _ = s.reserveLease(p)
	check(t, bytes.Equal(lease.HWAddr, hw), "lease.HWAddr")
	check(t, bytes.Equal(lease.IP, []byte{1, 1, 1, 1}), "lease.IP")

	// Decline request for a lease which doesn't match our internal state
	hw = []byte{1, 2, 3, 4, 5, 6}
	p.SetCHAddr(hw)
	p.SetCIAddr([]byte{0, 0, 0, 0})
	opt = make(dhcp4.Options, 10)
	// ask a different IP
	opt[dhcp4.OptionRequestedIPAddress] = []byte{1, 1, 1, 2}
	p2 = s.handleDHCP4Request(p, opt)
	opt = p2.ParseOptions()
	check(t, bytes.Equal(opt[dhcp4.OptionDHCPMessageType], []byte{byte(dhcp4.NAK)}), "dhcp4.NAK")

	// Commit the previously reserved lease
	hw = []byte{1, 2, 3, 4, 5, 6}
	p.SetCHAddr(hw)
	p.SetCIAddr([]byte{0, 0, 0, 0})
	opt = make(dhcp4.Options, 10)
	opt[dhcp4.OptionRequestedIPAddress] = []byte{1, 1, 1, 1}
	p2 = s.handleDHCP4Request(p, opt)
	opt = p2.ParseOptions()
	check(t, bytes.Equal(opt[dhcp4.OptionDHCPMessageType], []byte{byte(dhcp4.ACK)}), "dhcp4.ACK")
	check(t, bytes.Equal(p2.YIAddr(), []byte{1, 1, 1, 1}), "p2.YIAddr")
	check(t, bytes.Equal(p2.CHAddr(), hw), "p2.CHAddr")
	check(t, bytes.Equal(opt[dhcp4.OptionIPAddressLeaseTime], dhcp4.OptionsLeaseTime(5*time.Second)), "OptionIPAddressLeaseTime")
	check(t, bytes.Equal(opt[dhcp4.OptionServerIdentifier], s.ipnet.IP), "OptionServerIdentifier")

	check(t, bytes.Equal(s.FindIPbyMAC(hw), []byte{1, 1, 1, 1}), "FindIPbyMAC")

	// Commit the previously reserved lease #2
	hw = []byte{2, 2, 3, 4, 5, 6}
	p.SetCHAddr(hw)
	p.SetCIAddr([]byte{0, 0, 0, 0})
	opt = make(dhcp4.Options, 10)
	opt[dhcp4.OptionRequestedIPAddress] = []byte{1, 1, 1, 2}
	p2 = s.handleDHCP4Request(p, opt)
	check(t, bytes.Equal(p2.YIAddr(), []byte{1, 1, 1, 2}), "p2.YIAddr")

	// Reserve an IP - we have no more available IPs
	hw = []byte{3, 2, 3, 4, 5, 6}
	p.SetCHAddr(hw)
	lease, _ = s.reserveLease(p)
	check(t, lease == nil, "lease == nil")

	s.reset()
	testStaticLeases(t, &s)
	testStaticLeaseReplaceByMAC(t, &s)

	s.reset()
	misc(t, &s)
}

func testStaticLeases(t *testing.T, s *Server) {
	var err error
	var l Lease
	l.IP = []byte{1, 1, 1, 1}

	l.HWAddr = []byte{1, 2, 3, 4, 5, 6}
	s.leases = append(s.leases, &l)

	// replace dynamic lease with a static (same IP)
	l.HWAddr = []byte{2, 2, 3, 4, 5, 6}
	err = s.AddStaticLease(l)
	check(t, err == nil, "AddStaticLease")

	ll := s.Leases(LeasesAll)
	assert.True(t, len(ll) == 1)
	assert.True(t, bytes.Equal(ll[0].IP, []byte{1, 1, 1, 1}))
	assert.True(t, bytes.Equal(ll[0].HWAddr, []byte{2, 2, 3, 4, 5, 6}))
	assert.True(t, ll[0].Expiry.Unix() == leaseExpireStatic)

	err = s.RemoveStaticLease(l)
	assert.True(t, err == nil)

	ll = s.Leases(LeasesAll)
	assert.True(t, len(ll) == 0)
}

func testStaticLeaseReplaceByMAC(t *testing.T, s *Server) {
	var err error
	var l Lease
	l.HWAddr = []byte{1, 2, 3, 4, 5, 6}

	l.IP = []byte{1, 1, 1, 1}
	l.Expiry = time.Now().Add(time.Hour)
	s.leases = append(s.leases, &l)

	// replace dynamic lease with a static (same MAC)
	l.IP = []byte{2, 1, 1, 1}
	err = s.AddStaticLease(l)
	assert.True(t, err == nil)

	ll := s.Leases(LeasesAll)
	assert.True(t, len(ll) == 1)
	assert.True(t, bytes.Equal(ll[0].IP, []byte{2, 1, 1, 1}))
	assert.True(t, bytes.Equal(ll[0].HWAddr, []byte{1, 2, 3, 4, 5, 6}))
}

// Small tests that don't require a static server's state
func misc(t *testing.T, s *Server) {
	var p, p2 dhcp4.Packet
	var hw net.HardwareAddr
	var opt dhcp4.Options

	p = make(dhcp4.Packet, 241)

	// Try to commit a lease for an IP without prior Discover-Offer packets
	hw = []byte{2, 2, 3, 4, 5, 6}
	p.SetCHAddr(hw)
	p.SetCIAddr([]byte{0, 0, 0, 0})
	opt = make(dhcp4.Options, 10)
	opt[dhcp4.OptionRequestedIPAddress] = []byte{1, 1, 1, 1}
	p2 = s.handleDHCP4Request(p, opt)
	opt = p2.ParseOptions()
	check(t, bytes.Equal(opt[dhcp4.OptionDHCPMessageType], []byte{byte(dhcp4.NAK)}), "dhcp4.NAK")
}

// Leases database store/load
func TestDB(t *testing.T) {
	var s = Server{}
	s.conf.DBFilePath = dbFilename
	var p dhcp4.Packet
	var hw1, hw2 net.HardwareAddr
	var lease *Lease

	s.reset()
	s.leaseStart = []byte{1, 1, 1, 1}
	s.leaseStop = []byte{1, 1, 1, 2}
	s.leaseTime = 5 * time.Second
	s.leaseOptions = dhcp4.Options{}
	s.ipnet = &net.IPNet{
		IP:   []byte{1, 2, 3, 4},
		Mask: []byte{0xff, 0xff, 0xff, 0xff},
	}

	p = make(dhcp4.Packet, 241)

	hw1 = []byte{1, 2, 3, 4, 5, 6}
	p.SetCHAddr(hw1)
	lease, _ = s.reserveLease(p)
	lease.Expiry = time.Unix(4000000001, 0)

	hw2 = []byte{2, 2, 3, 4, 5, 6}
	p.SetCHAddr(hw2)
	lease, _ = s.reserveLease(p)
	lease.Expiry = time.Unix(4000000002, 0)

	_ = os.Remove("leases.db")
	s.dbStore()
	s.reset()

	s.dbLoad()
	check(t, bytes.Equal(s.leases[0].HWAddr, hw1), "leases[0].HWAddr")
	check(t, bytes.Equal(s.leases[0].IP, []byte{1, 1, 1, 1}), "leases[0].IP")
	check(t, s.leases[0].Expiry.Unix() == 4000000001, "leases[0].Expiry")

	check(t, bytes.Equal(s.leases[1].HWAddr, hw2), "leases[1].HWAddr")
	check(t, bytes.Equal(s.leases[1].IP, []byte{1, 1, 1, 2}), "leases[1].IP")
	check(t, s.leases[1].Expiry.Unix() == 4000000002, "leases[1].Expiry")

	_ = os.Remove("leases.db")
}

func TestIsValidSubnetMask(t *testing.T) {
	if !isValidSubnetMask([]byte{255, 255, 255, 0}) {
		t.Fatalf("isValidSubnetMask([]byte{255,255,255,0})")
	}
	if isValidSubnetMask([]byte{255, 255, 253, 0}) {
		t.Fatalf("isValidSubnetMask([]byte{255,255,253,0})")
	}
	if isValidSubnetMask([]byte{0, 255, 255, 255}) {
		t.Fatalf("isValidSubnetMask([]byte{255,255,253,0})")
	}
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
