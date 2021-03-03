// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"net"
	"testing"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/iana"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func notify6(flags uint32) {
}

func TestV6_AddRemove_static(t *testing.T) {
	s, err := v6Create(V6ServerConf{
		Enabled:    true,
		RangeStart: net.ParseIP("2001::1"),
		notify:     notify6,
	})
	require.Nil(t, err)

	require.Empty(t, s.GetLeases(LeasesStatic))

	// Add static lease.
	l := Lease{
		IP:     net.ParseIP("2001::1"),
		HWAddr: net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}
	require.Nil(t, s.AddStaticLease(l))

	// Try to add the same static lease.
	require.NotNil(t, s.AddStaticLease(l))

	ls := s.GetLeases(LeasesStatic)
	require.Len(t, ls, 1)
	assert.Equal(t, l.IP, ls[0].IP)
	assert.Equal(t, l.HWAddr, ls[0].HWAddr)
	assert.EqualValues(t, leaseExpireStatic, ls[0].Expiry.Unix())

	// Try to remove non-existent static lease.
	require.NotNil(t, s.RemoveStaticLease(Lease{
		IP:     net.ParseIP("2001::2"),
		HWAddr: l.HWAddr,
	}))

	// Remove static lease.
	require.Nil(t, s.RemoveStaticLease(l))

	assert.Empty(t, s.GetLeases(LeasesStatic))
}

func TestV6_AddReplace(t *testing.T) {
	sIface, err := v6Create(V6ServerConf{
		Enabled:    true,
		RangeStart: net.ParseIP("2001::1"),
		notify:     notify6,
	})
	require.Nil(t, err)
	s, ok := sIface.(*v6Server)
	require.True(t, ok)

	// Add dynamic leases.
	dynLeases := []*Lease{{
		IP:     net.ParseIP("2001::1"),
		HWAddr: net.HardwareAddr{0x11, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}, {
		IP:     net.ParseIP("2001::2"),
		HWAddr: net.HardwareAddr{0x22, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}}

	for _, l := range dynLeases {
		s.addLease(l)
	}

	stLeases := []Lease{{
		IP:     net.ParseIP("2001::1"),
		HWAddr: net.HardwareAddr{0x33, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}, {
		IP:     net.ParseIP("2001::3"),
		HWAddr: net.HardwareAddr{0x22, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}}

	for _, l := range stLeases {
		require.Nil(t, s.AddStaticLease(l))
	}

	ls := s.GetLeases(LeasesStatic)
	require.Len(t, ls, 2)

	for i, l := range ls {
		assert.True(t, stLeases[i].IP.Equal(l.IP))
		assert.Equal(t, stLeases[i].HWAddr, l.HWAddr)
		assert.EqualValues(t, leaseExpireStatic, l.Expiry.Unix())
	}
}

func TestV6GetLease(t *testing.T) {
	var err error
	sIface, err := v6Create(V6ServerConf{
		Enabled:    true,
		RangeStart: net.ParseIP("2001::1"),
		notify:     notify6,
	})
	require.Nil(t, err)
	s, ok := sIface.(*v6Server)
	require.True(t, ok)

	dnsAddr := net.ParseIP("2000::1")
	s.conf.dnsIPAddrs = []net.IP{dnsAddr}
	s.sid = dhcpv6.Duid{
		Type:          dhcpv6.DUID_LLT,
		HwType:        iana.HWTypeEthernet,
		LinkLayerAddr: net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}

	l := Lease{
		IP:     net.ParseIP("2001::1"),
		HWAddr: net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}
	require.Nil(t, s.AddStaticLease(l))

	var req, resp, msg *dhcpv6.Message
	mac := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}
	t.Run("solicit", func(t *testing.T) {
		req, err = dhcpv6.NewSolicit(mac)
		require.Nil(t, err)

		msg, err = req.GetInnerMessage()
		require.Nil(t, err)

		resp, err = dhcpv6.NewAdvertiseFromSolicit(msg)
		require.Nil(t, err)

		assert.True(t, s.process(msg, req, resp))
	})
	require.Nil(t, err)
	resp.AddOption(dhcpv6.OptServerID(s.sid))

	var oia *dhcpv6.OptIANA
	var oiaAddr *dhcpv6.OptIAAddress
	t.Run("advertise", func(t *testing.T) {
		require.Equal(t, dhcpv6.MessageTypeAdvertise, resp.Type())
		oia = resp.Options.OneIANA()
		oiaAddr = oia.Options.OneAddress()

		assert.Equal(t, l.IP, oiaAddr.IPv6Addr)
		assert.Equal(t, s.conf.leaseTime.Seconds(), oiaAddr.ValidLifetime.Seconds())
	})

	t.Run("request", func(t *testing.T) {
		req, err = dhcpv6.NewRequestFromAdvertise(resp)
		require.Nil(t, err)

		msg, err = req.GetInnerMessage()
		require.Nil(t, err)

		resp, err = dhcpv6.NewReplyFromMessage(msg)
		require.Nil(t, err)

		assert.True(t, s.process(msg, req, resp))
	})
	require.Nil(t, err)

	t.Run("reply", func(t *testing.T) {
		require.Equal(t, dhcpv6.MessageTypeReply, resp.Type())
		oia = resp.Options.OneIANA()
		oiaAddr = oia.Options.OneAddress()

		assert.Equal(t, l.IP, oiaAddr.IPv6Addr)
		assert.Equal(t, s.conf.leaseTime.Seconds(), oiaAddr.ValidLifetime.Seconds())
	})

	dnsAddrs := resp.Options.DNS()
	require.Len(t, dnsAddrs, 1)
	assert.Equal(t, dnsAddr, dnsAddrs[0])

	t.Run("lease", func(t *testing.T) {
		ls := s.GetLeases(LeasesStatic)
		require.Len(t, ls, 1)
		assert.Equal(t, l.IP, ls[0].IP)
		assert.Equal(t, l.HWAddr, ls[0].HWAddr)
	})
}

func TestV6GetDynamicLease(t *testing.T) {
	sIface, err := v6Create(V6ServerConf{
		Enabled:    true,
		RangeStart: net.ParseIP("2001::2"),
		notify:     notify6,
	})
	require.Nil(t, err)
	s, ok := sIface.(*v6Server)
	require.True(t, ok)

	dnsAddr := net.ParseIP("2000::1")
	s.conf.dnsIPAddrs = []net.IP{dnsAddr}
	s.sid = dhcpv6.Duid{
		Type:          dhcpv6.DUID_LLT,
		HwType:        iana.HWTypeEthernet,
		LinkLayerAddr: net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}

	var req, resp, msg *dhcpv6.Message
	mac := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}
	t.Run("solicit", func(t *testing.T) {
		req, err = dhcpv6.NewSolicit(mac)
		require.Nil(t, err)

		msg, err = req.GetInnerMessage()
		require.Nil(t, err)

		resp, err = dhcpv6.NewAdvertiseFromSolicit(msg)
		require.Nil(t, err)

		assert.True(t, s.process(msg, req, resp))
	})
	require.Nil(t, err)
	resp.AddOption(dhcpv6.OptServerID(s.sid))

	var oia *dhcpv6.OptIANA
	var oiaAddr *dhcpv6.OptIAAddress
	t.Run("advertise", func(t *testing.T) {
		require.Equal(t, dhcpv6.MessageTypeAdvertise, resp.Type())
		oia = resp.Options.OneIANA()
		oiaAddr = oia.Options.OneAddress()
		assert.Equal(t, "2001::2", oiaAddr.IPv6Addr.String())
	})

	t.Run("request", func(t *testing.T) {
		req, err = dhcpv6.NewRequestFromAdvertise(resp)
		require.Nil(t, err)

		msg, err = req.GetInnerMessage()
		require.Nil(t, err)

		resp, err = dhcpv6.NewReplyFromMessage(msg)
		require.Nil(t, err)

		assert.True(t, s.process(msg, req, resp))
	})
	require.Nil(t, err)

	t.Run("reply", func(t *testing.T) {
		require.Equal(t, dhcpv6.MessageTypeReply, resp.Type())
		oia = resp.Options.OneIANA()
		oiaAddr = oia.Options.OneAddress()
		assert.Equal(t, "2001::2", oiaAddr.IPv6Addr.String())
	})

	dnsAddrs := resp.Options.DNS()
	require.Len(t, dnsAddrs, 1)
	assert.Equal(t, dnsAddr, dnsAddrs[0])

	t.Run("lease", func(t *testing.T) {
		ls := s.GetLeases(LeasesDynamic)
		require.Len(t, ls, 1)
		assert.Equal(t, "2001::2", ls[0].IP.String())
		assert.Equal(t, mac, ls[0].HWAddr)
	})
}

func TestIP6InRange(t *testing.T) {
	start := net.ParseIP("2001::2")

	testCases := []struct {
		ip   net.IP
		want bool
	}{{
		ip:   net.ParseIP("2001::1"),
		want: false,
	}, {
		ip:   net.ParseIP("2002::2"),
		want: false,
	}, {
		ip:   start,
		want: true,
	}, {
		ip:   net.ParseIP("2001::3"),
		want: true,
	}}

	for _, tc := range testCases {
		t.Run(tc.ip.String(), func(t *testing.T) {
			assert.Equal(t, tc.want, ip6InRange(start, tc.ip))
		})
	}
}
