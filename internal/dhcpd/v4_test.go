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
		RangeStart: net.IP{192, 168, 10, 100},
		RangeEnd:   net.IP{192, 168, 10, 200},
		GatewayIP:  net.IP{192, 168, 10, 1},
		SubnetMask: net.IP{255, 255, 255, 0},
		notify:     notify4,
	}
	s, err := v4Create(conf)
	assert.Nil(t, err)

	ls := s.GetLeases(LeasesStatic)
	assert.Empty(t, ls)

	// add static lease
	l := Lease{}
	l.IP = net.IP{192, 168, 10, 150}
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.Nil(t, s.AddStaticLease(l))

	// try to add the same static lease - fail
	assert.NotNil(t, s.AddStaticLease(l))

	// check
	ls = s.GetLeases(LeasesStatic)
	assert.Len(t, ls, 1)
	assert.Equal(t, "192.168.10.150", ls[0].IP.String())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", ls[0].HWAddr.String())
	assert.EqualValues(t, leaseExpireStatic, ls[0].Expiry.Unix())

	// try to remove static lease - fail
	l.IP = net.IP{192, 168, 10, 110}
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.NotNil(t, s.RemoveStaticLease(l))

	// remove static lease
	l.IP = net.IP{192, 168, 10, 150}
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.Nil(t, s.RemoveStaticLease(l))

	// check
	ls = s.GetLeases(LeasesStatic)
	assert.Empty(t, ls)
}

func TestV4StaticLeaseAddReplaceDynamic(t *testing.T) {
	conf := V4ServerConf{
		Enabled:    true,
		RangeStart: net.IP{192, 168, 10, 100},
		RangeEnd:   net.IP{192, 168, 10, 200},
		GatewayIP:  net.IP{192, 168, 10, 1},
		SubnetMask: net.IP{255, 255, 255, 0},
		notify:     notify4,
	}
	sIface, err := v4Create(conf)
	s := sIface.(*v4Server)
	assert.Nil(t, err)

	// add dynamic lease
	ld := Lease{}
	ld.IP = net.IP{192, 168, 10, 150}
	ld.HWAddr, _ = net.ParseMAC("11:aa:aa:aa:aa:aa")
	s.addLease(&ld)

	// add dynamic lease
	{
		ld := Lease{}
		ld.IP = net.IP{192, 168, 10, 151}
		ld.HWAddr, _ = net.ParseMAC("22:aa:aa:aa:aa:aa")
		s.addLease(&ld)
	}

	// add static lease with the same IP
	l := Lease{}
	l.IP = net.IP{192, 168, 10, 150}
	l.HWAddr, _ = net.ParseMAC("33:aa:aa:aa:aa:aa")
	assert.Nil(t, s.AddStaticLease(l))

	// add static lease with the same MAC
	l = Lease{}
	l.IP = net.IP{192, 168, 10, 152}
	l.HWAddr, _ = net.ParseMAC("22:aa:aa:aa:aa:aa")
	assert.Nil(t, s.AddStaticLease(l))

	// check
	ls := s.GetLeases(LeasesStatic)
	assert.Len(t, ls, 2)

	assert.Equal(t, "192.168.10.150", ls[0].IP.String())
	assert.Equal(t, "33:aa:aa:aa:aa:aa", ls[0].HWAddr.String())
	assert.EqualValues(t, leaseExpireStatic, ls[0].Expiry.Unix())

	assert.Equal(t, "192.168.10.152", ls[1].IP.String())
	assert.Equal(t, "22:aa:aa:aa:aa:aa", ls[1].HWAddr.String())
	assert.EqualValues(t, leaseExpireStatic, ls[1].Expiry.Unix())
}

func TestV4StaticLeaseGet(t *testing.T) {
	conf := V4ServerConf{
		Enabled:    true,
		RangeStart: net.IP{192, 168, 10, 100},
		RangeEnd:   net.IP{192, 168, 10, 200},
		GatewayIP:  net.IP{192, 168, 10, 1},
		SubnetMask: net.IP{255, 255, 255, 0},
		notify:     notify4,
	}
	sIface, err := v4Create(conf)
	s := sIface.(*v4Server)
	assert.Nil(t, err)
	s.conf.dnsIPAddrs = []net.IP{{192, 168, 10, 1}}

	l := Lease{}
	l.IP = net.IP{192, 168, 10, 150}
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.Nil(t, s.AddStaticLease(l))

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
	assert.Len(t, dnsAddrs, 1)
	assert.Equal(t, "192.168.10.1", dnsAddrs[0].String())

	// check lease
	ls := s.GetLeases(LeasesStatic)
	assert.Len(t, ls, 1)
	assert.Equal(t, "192.168.10.150", ls[0].IP.String())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", ls[0].HWAddr.String())
}

func TestV4DynamicLeaseGet(t *testing.T) {
	conf := V4ServerConf{
		Enabled:    true,
		RangeStart: net.IP{192, 168, 10, 100},
		RangeEnd:   net.IP{192, 168, 10, 200},
		GatewayIP:  net.IP{192, 168, 10, 1},
		SubnetMask: net.IP{255, 255, 255, 0},
		notify:     notify4,
		Options: []string{
			"81 hex 303132",
			"82 ip 1.2.3.4",
		},
	}
	sIface, err := v4Create(conf)
	s := sIface.(*v4Server)
	assert.Nil(t, err)
	s.conf.dnsIPAddrs = []net.IP{{192, 168, 10, 1}}

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
	assert.Len(t, dnsAddrs, 1)
	assert.Equal(t, "192.168.10.1", dnsAddrs[0].String())

	// check lease
	ls := s.GetLeases(LeasesDynamic)
	assert.Len(t, ls, 1)
	assert.Equal(t, "192.168.10.100", ls[0].IP.String())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", ls[0].HWAddr.String())

	start := net.IP{192, 168, 10, 100}
	stop := net.IP{192, 168, 10, 200}
	assert.False(t, ip4InRange(start, stop, net.IP{192, 168, 10, 99}))
	assert.False(t, ip4InRange(start, stop, net.IP{192, 168, 11, 100}))
	assert.False(t, ip4InRange(start, stop, net.IP{192, 168, 11, 201}))
	assert.True(t, ip4InRange(start, stop, net.IP{192, 168, 10, 100}))
}
