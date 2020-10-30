// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"net"
	"testing"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/stretchr/testify/assert"
)

func notify4(flags uint32) {
}

func TestV4StaticLeaseAddRemove(t *testing.T) {
	conf := V4ServerConf{
		Enabled:    true,
		RangeStart: "192.168.10.100",
		RangeEnd:   "192.168.10.200",
		GatewayIP:  "192.168.10.1",
		SubnetMask: "255.255.255.0",
		notify:     notify4,
	}
	s, err := v4Create(conf)
	assert.True(t, err == nil)

	ls := s.GetLeases(LeasesStatic)
	assert.Equal(t, 0, len(ls))

	// add static lease
	l := Lease{}
	l.IP = net.ParseIP("192.168.10.150").To4()
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.True(t, s.AddStaticLease(l) == nil)

	// try to add the same static lease - fail
	assert.True(t, s.AddStaticLease(l) != nil)

	// check
	ls = s.GetLeases(LeasesStatic)
	assert.Equal(t, 1, len(ls))
	assert.Equal(t, "192.168.10.150", ls[0].IP.String())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", ls[0].HWAddr.String())
	assert.True(t, ls[0].Expiry.Unix() == leaseExpireStatic)

	// try to remove static lease - fail
	l.IP = net.ParseIP("192.168.10.110").To4()
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.True(t, s.RemoveStaticLease(l) != nil)

	// remove static lease
	l.IP = net.ParseIP("192.168.10.150").To4()
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.True(t, s.RemoveStaticLease(l) == nil)

	// check
	ls = s.GetLeases(LeasesStatic)
	assert.Equal(t, 0, len(ls))
}

func TestV4StaticLeaseAddReplaceDynamic(t *testing.T) {
	conf := V4ServerConf{
		Enabled:    true,
		RangeStart: "192.168.10.100",
		RangeEnd:   "192.168.10.200",
		GatewayIP:  "192.168.10.1",
		SubnetMask: "255.255.255.0",
		notify:     notify4,
	}
	sIface, err := v4Create(conf)
	s := sIface.(*v4Server)
	assert.True(t, err == nil)

	// add dynamic lease
	ld := Lease{}
	ld.IP = net.ParseIP("192.168.10.150").To4()
	ld.HWAddr, _ = net.ParseMAC("11:aa:aa:aa:aa:aa")
	s.addLease(&ld)

	// add dynamic lease
	{
		ld := Lease{}
		ld.IP = net.ParseIP("192.168.10.151").To4()
		ld.HWAddr, _ = net.ParseMAC("22:aa:aa:aa:aa:aa")
		s.addLease(&ld)
	}

	// add static lease with the same IP
	l := Lease{}
	l.IP = net.ParseIP("192.168.10.150").To4()
	l.HWAddr, _ = net.ParseMAC("33:aa:aa:aa:aa:aa")
	assert.True(t, s.AddStaticLease(l) == nil)

	// add static lease with the same MAC
	l = Lease{}
	l.IP = net.ParseIP("192.168.10.152").To4()
	l.HWAddr, _ = net.ParseMAC("22:aa:aa:aa:aa:aa")
	assert.True(t, s.AddStaticLease(l) == nil)

	// check
	ls := s.GetLeases(LeasesStatic)
	assert.Equal(t, 2, len(ls))

	assert.Equal(t, "192.168.10.150", ls[0].IP.String())
	assert.Equal(t, "33:aa:aa:aa:aa:aa", ls[0].HWAddr.String())
	assert.True(t, ls[0].Expiry.Unix() == leaseExpireStatic)

	assert.Equal(t, "192.168.10.152", ls[1].IP.String())
	assert.Equal(t, "22:aa:aa:aa:aa:aa", ls[1].HWAddr.String())
	assert.True(t, ls[1].Expiry.Unix() == leaseExpireStatic)
}

func TestV4StaticLeaseGet(t *testing.T) {
	conf := V4ServerConf{
		Enabled:    true,
		RangeStart: "192.168.10.100",
		RangeEnd:   "192.168.10.200",
		GatewayIP:  "192.168.10.1",
		SubnetMask: "255.255.255.0",
		notify:     notify4,
	}
	sIface, err := v4Create(conf)
	s := sIface.(*v4Server)
	assert.True(t, err == nil)
	s.conf.dnsIPAddrs = []net.IP{net.ParseIP("192.168.10.1").To4()}

	l := Lease{}
	l.IP = net.ParseIP("192.168.10.150").To4()
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.True(t, s.AddStaticLease(l) == nil)

	// "Discover"
	mac, _ := net.ParseMAC("aa:aa:aa:aa:aa:aa")
	req, _ := dhcpv4.NewDiscovery(mac)
	resp, _ := dhcpv4.NewReplyFromRequest(req)
	assert.Equal(t, 1, s.process(req, resp))

	// check "Offer"
	assert.Equal(t, dhcpv4.MessageTypeOffer, resp.MessageType())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", resp.ClientHWAddr.String())
	assert.Equal(t, "192.168.10.150", resp.YourIPAddr.String())
	assert.Equal(t, "192.168.10.1", resp.Router()[0].String())
	assert.Equal(t, "192.168.10.1", resp.ServerIdentifier().String())
	assert.Equal(t, "255.255.255.0", net.IP(resp.SubnetMask()).String())
	assert.Equal(t, s.conf.leaseTime.Seconds(), resp.IPAddressLeaseTime(-1).Seconds())

	// "Request"
	req, _ = dhcpv4.NewRequestFromOffer(resp)
	resp, _ = dhcpv4.NewReplyFromRequest(req)
	assert.Equal(t, 1, s.process(req, resp))

	// check "Ack"
	assert.Equal(t, dhcpv4.MessageTypeAck, resp.MessageType())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", resp.ClientHWAddr.String())
	assert.Equal(t, "192.168.10.150", resp.YourIPAddr.String())
	assert.Equal(t, "192.168.10.1", resp.Router()[0].String())
	assert.Equal(t, "192.168.10.1", resp.ServerIdentifier().String())
	assert.Equal(t, "255.255.255.0", net.IP(resp.SubnetMask()).String())
	assert.Equal(t, s.conf.leaseTime.Seconds(), resp.IPAddressLeaseTime(-1).Seconds())

	dnsAddrs := resp.DNS()
	assert.Equal(t, 1, len(dnsAddrs))
	assert.Equal(t, "192.168.10.1", dnsAddrs[0].String())

	// check lease
	ls := s.GetLeases(LeasesStatic)
	assert.Equal(t, 1, len(ls))
	assert.Equal(t, "192.168.10.150", ls[0].IP.String())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", ls[0].HWAddr.String())
}

func TestV4DynamicLeaseGet(t *testing.T) {
	conf := V4ServerConf{
		Enabled:    true,
		RangeStart: "192.168.10.100",
		RangeEnd:   "192.168.10.200",
		GatewayIP:  "192.168.10.1",
		SubnetMask: "255.255.255.0",
		notify:     notify4,
		Options: []string{
			"81 hex 303132",
			"82 ip 1.2.3.4",
		},
	}
	sIface, err := v4Create(conf)
	s := sIface.(*v4Server)
	assert.True(t, err == nil)
	s.conf.dnsIPAddrs = []net.IP{net.ParseIP("192.168.10.1").To4()}

	// "Discover"
	mac, _ := net.ParseMAC("aa:aa:aa:aa:aa:aa")
	req, _ := dhcpv4.NewDiscovery(mac)
	resp, _ := dhcpv4.NewReplyFromRequest(req)
	assert.Equal(t, 1, s.process(req, resp))

	// check "Offer"
	assert.Equal(t, dhcpv4.MessageTypeOffer, resp.MessageType())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", resp.ClientHWAddr.String())
	assert.Equal(t, "192.168.10.100", resp.YourIPAddr.String())
	assert.Equal(t, "192.168.10.1", resp.Router()[0].String())
	assert.Equal(t, "192.168.10.1", resp.ServerIdentifier().String())
	assert.Equal(t, "255.255.255.0", net.IP(resp.SubnetMask()).String())
	assert.Equal(t, s.conf.leaseTime.Seconds(), resp.IPAddressLeaseTime(-1).Seconds())
	assert.Equal(t, []byte("012"), resp.Options[uint8(dhcpv4.OptionFQDN)])
	assert.Equal(t, "1.2.3.4", net.IP(resp.Options[uint8(dhcpv4.OptionRelayAgentInformation)]).String())

	// "Request"
	req, _ = dhcpv4.NewRequestFromOffer(resp)
	resp, _ = dhcpv4.NewReplyFromRequest(req)
	assert.Equal(t, 1, s.process(req, resp))

	// check "Ack"
	assert.Equal(t, dhcpv4.MessageTypeAck, resp.MessageType())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", resp.ClientHWAddr.String())
	assert.Equal(t, "192.168.10.100", resp.YourIPAddr.String())
	assert.Equal(t, "192.168.10.1", resp.Router()[0].String())
	assert.Equal(t, "192.168.10.1", resp.ServerIdentifier().String())
	assert.Equal(t, "255.255.255.0", net.IP(resp.SubnetMask()).String())
	assert.Equal(t, s.conf.leaseTime.Seconds(), resp.IPAddressLeaseTime(-1).Seconds())

	dnsAddrs := resp.DNS()
	assert.Equal(t, 1, len(dnsAddrs))
	assert.Equal(t, "192.168.10.1", dnsAddrs[0].String())

	// check lease
	ls := s.GetLeases(LeasesDynamic)
	assert.Equal(t, 1, len(ls))
	assert.Equal(t, "192.168.10.100", ls[0].IP.String())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", ls[0].HWAddr.String())

	start := net.ParseIP("192.168.10.100").To4()
	stop := net.ParseIP("192.168.10.200").To4()
	assert.True(t, !ip4InRange(start, stop, net.ParseIP("192.168.10.99").To4()))
	assert.True(t, !ip4InRange(start, stop, net.ParseIP("192.168.11.100").To4()))
	assert.True(t, !ip4InRange(start, stop, net.ParseIP("192.168.11.201").To4()))
	assert.True(t, ip4InRange(start, stop, net.ParseIP("192.168.10.100").To4()))
}
