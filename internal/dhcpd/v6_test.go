// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"net"
	"testing"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/iana"
	"github.com/stretchr/testify/assert"
)

func notify6(flags uint32) {
}

func TestV6StaticLeaseAddRemove(t *testing.T) {
	conf := V6ServerConf{
		Enabled:    true,
		RangeStart: "2001::1",
		notify:     notify6,
	}
	s, err := v6Create(conf)
	assert.Nil(t, err)

	ls := s.GetLeases(LeasesStatic)
	assert.Empty(t, ls)

	// add static lease
	l := Lease{}
	l.IP = net.ParseIP("2001::1")
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.Nil(t, s.AddStaticLease(l))

	// try to add static lease - fail
	assert.NotNil(t, s.AddStaticLease(l))

	// check
	ls = s.GetLeases(LeasesStatic)
	assert.Len(t, ls, 1)
	assert.Equal(t, "2001::1", ls[0].IP.String())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", ls[0].HWAddr.String())
	assert.EqualValues(t, leaseExpireStatic, ls[0].Expiry.Unix())

	// try to remove static lease - fail
	l.IP = net.ParseIP("2001::2")
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.NotNil(t, s.RemoveStaticLease(l))

	// remove static lease
	l.IP = net.ParseIP("2001::1")
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.Nil(t, s.RemoveStaticLease(l))

	// check
	ls = s.GetLeases(LeasesStatic)
	assert.Empty(t, ls)
}

func TestV6StaticLeaseAddReplaceDynamic(t *testing.T) {
	conf := V6ServerConf{
		Enabled:    true,
		RangeStart: "2001::1",
		notify:     notify6,
	}
	sIface, err := v6Create(conf)
	s := sIface.(*v6Server)
	assert.Nil(t, err)

	// add dynamic lease
	ld := Lease{}
	ld.IP = net.ParseIP("2001::1")
	ld.HWAddr, _ = net.ParseMAC("11:aa:aa:aa:aa:aa")
	s.addLease(&ld)

	// add dynamic lease
	{
		ld := Lease{}
		ld.IP = net.ParseIP("2001::2")
		ld.HWAddr, _ = net.ParseMAC("22:aa:aa:aa:aa:aa")
		s.addLease(&ld)
	}

	// add static lease with the same IP
	l := Lease{}
	l.IP = net.ParseIP("2001::1")
	l.HWAddr, _ = net.ParseMAC("33:aa:aa:aa:aa:aa")
	assert.Nil(t, s.AddStaticLease(l))

	// add static lease with the same MAC
	l = Lease{}
	l.IP = net.ParseIP("2001::3")
	l.HWAddr, _ = net.ParseMAC("22:aa:aa:aa:aa:aa")
	assert.Nil(t, s.AddStaticLease(l))

	// check
	ls := s.GetLeases(LeasesStatic)
	assert.Len(t, ls, 2)

	assert.Equal(t, "2001::1", ls[0].IP.String())
	assert.Equal(t, "33:aa:aa:aa:aa:aa", ls[0].HWAddr.String())
	assert.EqualValues(t, leaseExpireStatic, ls[0].Expiry.Unix())

	assert.Equal(t, "2001::3", ls[1].IP.String())
	assert.Equal(t, "22:aa:aa:aa:aa:aa", ls[1].HWAddr.String())
	assert.EqualValues(t, leaseExpireStatic, ls[1].Expiry.Unix())
}

func TestV6GetLease(t *testing.T) {
	conf := V6ServerConf{
		Enabled:    true,
		RangeStart: "2001::1",
		notify:     notify6,
	}
	sIface, err := v6Create(conf)
	s := sIface.(*v6Server)
	assert.Nil(t, err)
	s.conf.dnsIPAddrs = []net.IP{net.ParseIP("2000::1")}
	s.sid = dhcpv6.Duid{
		Type:   dhcpv6.DUID_LLT,
		HwType: iana.HWTypeEthernet,
	}
	s.sid.LinkLayerAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")

	l := Lease{}
	l.IP = net.ParseIP("2001::1")
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.Nil(t, s.AddStaticLease(l))

	// "Solicit"
	mac, _ := net.ParseMAC("aa:aa:aa:aa:aa:aa")
	req, _ := dhcpv6.NewSolicit(mac)
	msg, _ := req.GetInnerMessage()
	resp, _ := dhcpv6.NewAdvertiseFromSolicit(msg)
	assert.True(t, s.process(msg, req, resp))
	resp.AddOption(dhcpv6.OptServerID(s.sid))

	// check "Advertise"
	assert.Equal(t, dhcpv6.MessageTypeAdvertise, resp.Type())
	oia := resp.Options.OneIANA()
	oiaAddr := oia.Options.OneAddress()
	assert.Equal(t, "2001::1", oiaAddr.IPv6Addr.String())
	assert.Equal(t, s.conf.leaseTime.Seconds(), oiaAddr.ValidLifetime.Seconds())

	// "Request"
	req, _ = dhcpv6.NewRequestFromAdvertise(resp)
	msg, _ = req.GetInnerMessage()
	resp, _ = dhcpv6.NewReplyFromMessage(msg)
	assert.True(t, s.process(msg, req, resp))

	// check "Reply"
	assert.Equal(t, dhcpv6.MessageTypeReply, resp.Type())
	oia = resp.Options.OneIANA()
	oiaAddr = oia.Options.OneAddress()
	assert.Equal(t, "2001::1", oiaAddr.IPv6Addr.String())
	assert.Equal(t, s.conf.leaseTime.Seconds(), oiaAddr.ValidLifetime.Seconds())

	dnsAddrs := resp.Options.DNS()
	assert.Len(t, dnsAddrs, 1)
	assert.Equal(t, "2000::1", dnsAddrs[0].String())

	// check lease
	ls := s.GetLeases(LeasesStatic)
	assert.Len(t, ls, 1)
	assert.Equal(t, "2001::1", ls[0].IP.String())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", ls[0].HWAddr.String())
}

func TestV6GetDynamicLease(t *testing.T) {
	conf := V6ServerConf{
		Enabled:    true,
		RangeStart: "2001::2",
		notify:     notify6,
	}
	sIface, err := v6Create(conf)
	s := sIface.(*v6Server)
	assert.Nil(t, err)
	s.conf.dnsIPAddrs = []net.IP{net.ParseIP("2000::1")}
	s.sid = dhcpv6.Duid{
		Type:   dhcpv6.DUID_LLT,
		HwType: iana.HWTypeEthernet,
	}
	s.sid.LinkLayerAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")

	// "Solicit"
	mac, _ := net.ParseMAC("aa:aa:aa:aa:aa:aa")
	req, _ := dhcpv6.NewSolicit(mac)
	msg, _ := req.GetInnerMessage()
	resp, _ := dhcpv6.NewAdvertiseFromSolicit(msg)
	assert.True(t, s.process(msg, req, resp))
	resp.AddOption(dhcpv6.OptServerID(s.sid))

	// check "Advertise"
	assert.Equal(t, dhcpv6.MessageTypeAdvertise, resp.Type())
	oia := resp.Options.OneIANA()
	oiaAddr := oia.Options.OneAddress()
	assert.Equal(t, "2001::2", oiaAddr.IPv6Addr.String())

	// "Request"
	req, _ = dhcpv6.NewRequestFromAdvertise(resp)
	msg, _ = req.GetInnerMessage()
	resp, _ = dhcpv6.NewReplyFromMessage(msg)
	assert.True(t, s.process(msg, req, resp))

	// check "Reply"
	assert.Equal(t, dhcpv6.MessageTypeReply, resp.Type())
	oia = resp.Options.OneIANA()
	oiaAddr = oia.Options.OneAddress()
	assert.Equal(t, "2001::2", oiaAddr.IPv6Addr.String())

	dnsAddrs := resp.Options.DNS()
	assert.Len(t, dnsAddrs, 1)
	assert.Equal(t, "2000::1", dnsAddrs[0].String())

	// check lease
	ls := s.GetLeases(LeasesDynamic)
	assert.Len(t, ls, 1)
	assert.Equal(t, "2001::2", ls[0].IP.String())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", ls[0].HWAddr.String())

	assert.False(t, ip6InRange(net.ParseIP("2001::2"), net.ParseIP("2001::1")))
	assert.False(t, ip6InRange(net.ParseIP("2001::2"), net.ParseIP("2002::2")))
	assert.True(t, ip6InRange(net.ParseIP("2001::2"), net.ParseIP("2001::2")))
	assert.True(t, ip6InRange(net.ParseIP("2001::2"), net.ParseIP("2001::3")))
}
