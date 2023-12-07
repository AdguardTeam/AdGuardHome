//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
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
	require.NoError(t, err)

	require.Empty(t, s.GetLeases(LeasesStatic))

	// Add static lease.
	l := &dhcpsvc.Lease{
		IP:     netip.MustParseAddr("2001::1"),
		HWAddr: net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}
	err = s.AddStaticLease(l)
	require.NoError(t, err)

	// Try to add the same static lease.
	err = s.AddStaticLease(l)
	require.Error(t, err)

	ls := s.GetLeases(LeasesStatic)
	require.Len(t, ls, 1)

	assert.Equal(t, l.IP, ls[0].IP)
	assert.Equal(t, l.HWAddr, ls[0].HWAddr)
	assert.True(t, ls[0].IsStatic)

	// Try to remove non-existent static lease.
	err = s.RemoveStaticLease(&dhcpsvc.Lease{
		IP:     netip.MustParseAddr("2001::2"),
		HWAddr: l.HWAddr,
	})
	require.Error(t, err)

	// Remove static lease.
	err = s.RemoveStaticLease(l)
	require.NoError(t, err)

	assert.Empty(t, s.GetLeases(LeasesStatic))
}

func TestV6_AddReplace(t *testing.T) {
	sIface, err := v6Create(V6ServerConf{
		Enabled:    true,
		RangeStart: net.ParseIP("2001::1"),
		notify:     notify6,
	})
	require.NoError(t, err)

	s, ok := sIface.(*v6Server)
	require.True(t, ok)

	// Add dynamic leases.
	dynLeases := []*dhcpsvc.Lease{{
		IP:     netip.MustParseAddr("2001::1"),
		HWAddr: net.HardwareAddr{0x11, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}, {
		IP:     netip.MustParseAddr("2001::2"),
		HWAddr: net.HardwareAddr{0x22, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}}

	for _, l := range dynLeases {
		s.addLease(l)
	}

	stLeases := []*dhcpsvc.Lease{{
		IP:     netip.MustParseAddr("2001::1"),
		HWAddr: net.HardwareAddr{0x33, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}, {
		IP:     netip.MustParseAddr("2001::3"),
		HWAddr: net.HardwareAddr{0x22, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}}

	for _, l := range stLeases {
		err = s.AddStaticLease(l)
		require.NoError(t, err)
	}

	ls := s.GetLeases(LeasesStatic)
	require.Len(t, ls, 2)

	for i, l := range ls {
		assert.Equal(t, stLeases[i].IP, l.IP)
		assert.Equal(t, stLeases[i].HWAddr, l.HWAddr)
		assert.True(t, l.IsStatic)
	}
}

func TestV6GetLease(t *testing.T) {
	var err error
	sIface, err := v6Create(V6ServerConf{
		Enabled:    true,
		RangeStart: net.ParseIP("2001::1"),
		notify:     notify6,
	})
	require.NoError(t, err)
	s, ok := sIface.(*v6Server)

	require.True(t, ok)

	dnsAddr := net.ParseIP("2000::1")
	s.conf.dnsIPAddrs = []net.IP{dnsAddr}
	s.sid = &dhcpv6.DUIDLL{
		HWType:        iana.HWTypeEthernet,
		LinkLayerAddr: net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}

	l := &dhcpsvc.Lease{
		IP:     netip.MustParseAddr("2001::1"),
		HWAddr: net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}
	err = s.AddStaticLease(l)
	require.NoError(t, err)

	var req, resp, msg *dhcpv6.Message
	mac := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}
	t.Run("solicit", func(t *testing.T) {
		req, err = dhcpv6.NewSolicit(mac)
		require.NoError(t, err)

		msg, err = req.GetInnerMessage()
		require.NoError(t, err)

		resp, err = dhcpv6.NewAdvertiseFromSolicit(msg)
		require.NoError(t, err)

		assert.True(t, s.process(msg, req, resp))
	})
	require.NoError(t, err)

	resp.AddOption(dhcpv6.OptServerID(s.sid))

	var oia *dhcpv6.OptIANA
	var oiaAddr *dhcpv6.OptIAAddress
	t.Run("advertise", func(t *testing.T) {
		require.Equal(t, dhcpv6.MessageTypeAdvertise, resp.Type())

		oia = resp.Options.OneIANA()
		oiaAddr = oia.Options.OneAddress()

		ip := net.IP(l.IP.AsSlice())
		assert.Equal(t, ip, oiaAddr.IPv6Addr)
		assert.Equal(t, s.conf.leaseTime.Seconds(), oiaAddr.ValidLifetime.Seconds())
	})

	t.Run("request", func(t *testing.T) {
		req, err = dhcpv6.NewRequestFromAdvertise(resp)
		require.NoError(t, err)

		msg, err = req.GetInnerMessage()
		require.NoError(t, err)

		resp, err = dhcpv6.NewReplyFromMessage(msg)
		require.NoError(t, err)

		assert.True(t, s.process(msg, req, resp))
	})
	require.NoError(t, err)

	t.Run("reply", func(t *testing.T) {
		require.Equal(t, dhcpv6.MessageTypeReply, resp.Type())

		oia = resp.Options.OneIANA()
		oiaAddr = oia.Options.OneAddress()

		ip := net.IP(l.IP.AsSlice())
		assert.Equal(t, ip, oiaAddr.IPv6Addr)
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
	require.NoError(t, err)

	s, ok := sIface.(*v6Server)
	require.True(t, ok)

	dnsAddr := net.ParseIP("2000::1")
	s.conf.dnsIPAddrs = []net.IP{dnsAddr}
	s.sid = &dhcpv6.DUIDLL{
		HWType:        iana.HWTypeEthernet,
		LinkLayerAddr: net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	}

	var req, resp, msg *dhcpv6.Message
	mac := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}
	t.Run("solicit", func(t *testing.T) {
		req, err = dhcpv6.NewSolicit(mac)
		require.NoError(t, err)

		msg, err = req.GetInnerMessage()
		require.NoError(t, err)

		resp, err = dhcpv6.NewAdvertiseFromSolicit(msg)
		require.NoError(t, err)

		assert.True(t, s.process(msg, req, resp))
	})
	require.NoError(t, err)

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
		require.NoError(t, err)

		msg, err = req.GetInnerMessage()
		require.NoError(t, err)

		resp, err = dhcpv6.NewReplyFromMessage(msg)
		require.NoError(t, err)

		assert.True(t, s.process(msg, req, resp))
	})
	require.NoError(t, err)

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

func TestV6_FindMACbyIP(t *testing.T) {
	const (
		staticName  = "static-client"
		anotherName = "another-client"
	)

	staticIP := netip.MustParseAddr("2001::1")
	staticMAC := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}

	anotherIP := netip.MustParseAddr("2001::100")
	anotherMAC := net.HardwareAddr{0xBB, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB}

	s := &v6Server{
		leases: []*dhcpsvc.Lease{{
			Hostname: staticName,
			HWAddr:   staticMAC,
			IP:       staticIP,
			IsStatic: true,
		}, {
			Expiry:   time.Unix(10, 0),
			Hostname: anotherName,
			HWAddr:   anotherMAC,
			IP:       anotherIP,
		}},
	}

	s.leases = []*dhcpsvc.Lease{{
		Hostname: staticName,
		HWAddr:   staticMAC,
		IP:       staticIP,
		IsStatic: true,
	}, {
		Expiry:   time.Unix(10, 0),
		Hostname: anotherName,
		HWAddr:   anotherMAC,
		IP:       anotherIP,
	}}

	testCases := []struct {
		want net.HardwareAddr
		ip   netip.Addr
		name string
	}{{
		name: "basic",
		ip:   staticIP,
		want: staticMAC,
	}, {
		name: "not_found",
		ip:   netip.MustParseAddr("ffff::1"),
		want: nil,
	}, {
		name: "expired",
		ip:   anotherIP,
		want: nil,
	}, {
		name: "v4",
		ip:   netip.MustParseAddr("1.2.3.4"),
		want: nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mac := s.FindMACbyIP(tc.ip)

			require.Equal(t, tc.want, mac)
		})
	}
}
