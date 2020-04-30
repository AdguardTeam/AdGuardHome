package dhcpd

import (
	"net"
	"testing"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/iana"
	"github.com/stretchr/testify/assert"
)

func notify(flags uint32) {
}

func TestV6StaticLease(t *testing.T) {
	conf := V6ServerConf{
		Enabled:    true,
		RangeStart: "2001::1",
		notify:     notify,
	}
	s, err := v6Create(conf)
	assert.True(t, err == nil)

	ls := s.GetLeases(LeasesStatic)
	assert.Equal(t, 0, len(ls))

	// add static lease
	l := Lease{}
	l.IP = net.ParseIP("2001::1")
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.True(t, s.AddStaticLease(l) == nil)

	// try to add static lease - fail
	assert.True(t, s.AddStaticLease(l) != nil)

	// check
	ls = s.GetLeases(LeasesStatic)
	assert.Equal(t, 1, len(ls))
	assert.Equal(t, "2001::1", ls[0].IP.String())
	assert.Equal(t, "aa:aa:aa:aa:aa:aa", ls[0].HWAddr.String())
	assert.True(t, ls[0].Expiry.Unix() == leaseExpireStatic)

	// try to remove static lease - fail
	l.IP = net.ParseIP("2001::2")
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.True(t, s.RemoveStaticLease(l) != nil)

	// remove static lease
	l.IP = net.ParseIP("2001::1")
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.True(t, s.RemoveStaticLease(l) == nil)

	// check
	ls = s.GetLeases(LeasesStatic)
	assert.Equal(t, 0, len(ls))

	s.Stop()
}

func TestV6GetLease(t *testing.T) {
	conf := V6ServerConf{
		Enabled:    true,
		RangeStart: "2001::1",
		notify:     notify,
	}
	s, err := v6Create(conf)
	assert.True(t, err == nil)
	s.conf.dnsIPAddrs = []net.IP{net.ParseIP("2000::1")}
	s.conf.sid = dhcpv6.Duid{
		Type:   dhcpv6.DUID_LLT,
		HwType: iana.HWTypeEthernet,
	}
	s.conf.sid.LinkLayerAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")

	l := Lease{}
	l.IP = net.ParseIP("2001::1")
	l.HWAddr, _ = net.ParseMAC("aa:aa:aa:aa:aa:aa")
	assert.True(t, s.AddStaticLease(l) == nil)

	// "Solicit"
	mac, _ := net.ParseMAC("aa:aa:aa:aa:aa:aa")
	req, _ := dhcpv6.NewSolicit(mac)
	msg, _ := req.GetInnerMessage()
	resp, _ := dhcpv6.NewAdvertiseFromSolicit(msg)
	assert.True(t, s.process(msg, req, resp))
	resp.AddOption(dhcpv6.OptServerID(s.conf.sid))

	// check "Advertise"
	oia := resp.Options.OneIANA()
	oiaAddr := oia.Options.OneAddress()
	assert.Equal(t, "2001::1", oiaAddr.IPv6Addr.String())

	// "Request"
	req, _ = dhcpv6.NewRequestFromAdvertise(resp)
	msg, _ = req.GetInnerMessage()
	resp, _ = dhcpv6.NewReplyFromMessage(msg)
	assert.True(t, s.process(msg, req, resp))

	// check "Reply"
	oia = resp.Options.OneIANA()
	oiaAddr = oia.Options.OneAddress()
	assert.Equal(t, "2001::1", oiaAddr.IPv6Addr.String())
	dnsAddrs := resp.Options.DNS()
	assert.Equal(t, 1, len(dnsAddrs))
	assert.Equal(t, "2000::1", dnsAddrs[0].String())

	s.Stop()
}
