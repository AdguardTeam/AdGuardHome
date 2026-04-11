//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"math"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
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

func TestDeriveTrackedRangeStart(t *testing.T) {
	got, err := deriveTrackedRangeStart(
		net.ParseIP("fd00::1234:5678:9abc:de00"),
		netip.MustParsePrefix("2001:db8:1::/64"),
	)
	require.NoError(t, err)

	assert.Equal(t, net.ParseIP("2001:db8:1::1234:5678:9abc:de00"), got)
}

func TestV6SetTrackedRangeStart(t *testing.T) {
	var notified []uint32

	s := &v6Server{
		conf: V6ServerConf{
			notify: func(flags uint32) {
				notified = append(notified, flags)
			},
		},
		leases: []*dhcpsvc.Lease{{
			IP:     netip.MustParseAddr("2001:db8:1::10"),
			HWAddr: net.HardwareAddr{0x10, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa},
			Expiry: time.Now().Add(30 * time.Minute),
		}, {
			IP:     netip.MustParseAddr("2001:db8:2::10"),
			HWAddr: net.HardwareAddr{0x20, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa},
			Expiry: time.Now().Add(30 * time.Minute),
		}, {
			IP:       netip.MustParseAddr("2001:db8:ffff::42"),
			HWAddr:   net.HardwareAddr{0x30, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa},
			IsStatic: true,
		}},
	}

	s.setTrackedRangeStart(net.ParseIP("2001:db8:1::10"), []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
		PreferredSec: 1800,
		ValidSec:     3600,
	}, {
		Prefix:       netip.MustParsePrefix("2001:db8:2::/64"),
		PreferredSec: 1800,
		ValidSec:     3600,
	}})

	require.Len(t, s.leases, 3)
	assert.Empty(t, notified)
	assert.Equal(t, byte(1), s.ipAddrs[0x10])
	assert.Equal(t, byte(0), s.ipAddrs[0x42])

	s.setTrackedRangeStart(net.ParseIP("2001:db8:1::10"), []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
		PreferredSec: 1800,
		ValidSec:     3600,
	}})

	require.Len(t, s.leases, 2)
	assert.Equal(t, []uint32{LeaseChangedDBStore}, notified)
}

func TestV6Create_InterfacePrefixSource(t *testing.T) {
	t.Run("ra_only_without_template", func(t *testing.T) {
		srv, err := v6Create(V6ServerConf{
			Enabled:      true,
			PrefixSource: V6PrefixSourceInterface,
			RASLAACOnly:  true,
			notify:       notify6,
		})
		require.NoError(t, err)

		s, ok := srv.(*v6Server)
		require.True(t, ok)
		assert.Nil(t, s.conf.ipStart)
	})

	t.Run("dhcp_pool_requires_template", func(t *testing.T) {
		_, err := v6Create(V6ServerConf{
			Enabled:      true,
			PrefixSource: V6PrefixSourceInterface,
			notify:       notify6,
		})
		require.Error(t, err)
	})
}

func TestV6TrackedPrefixChanged_SLAACOnlyUpdatesMetadata(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			PrefixSource: V6PrefixSourceInterface,
			RASLAACOnly:  true,
			notify:       notify6,
		},
	}

	err := s.trackedPrefixChanged(&raPrefixSnapshot{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 1800,
		ValidSec:     3600,
	}, []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 1800,
		ValidSec:     3600,
	}})
	require.NoError(t, err)
	assert.Nil(t, s.conf.ipStart)
	assert.Contains(t, s.advertisedPrefixes, netip.MustParsePrefix("2001:db8::/64"))
	assert.Contains(t, s.renewablePrefixes, netip.MustParsePrefix("2001:db8::/64"))
}

// TestV6TrackedPrefixChanged_PreferredExpiredActiveDisablesPool is a
// regression test for a bug where the preferredExpired digest fix would fire
// onActivePrefixChange with an active snapshot whose preferred lifetime had
// already reached zero.  trackedPrefixChanged used to derive ipStart from
// that prefix and leave the DHCPv6 pool pointing at it, so new clients could
// reserve an address and then receive a Reply with a zero valid lifetime
// from commitLease.  The fix treats PreferredSec==0 as "no pool" even when
// the prefix is still advertised.
func TestV6TrackedPrefixChanged_PreferredExpiredActiveDisablesPool(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			PrefixSource: V6PrefixSourceInterface,
			RangeStart:   net.ParseIP("fd00::1234:5678:9abc:de00"),
			notify:       notify6,
		},
	}

	err := s.trackedPrefixChanged(&raPrefixSnapshot{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 0,
		ValidSec:     3600,
	}, []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 0,
		ValidSec:     3600,
	}})
	require.NoError(t, err)

	assert.Nil(t, s.conf.ipStart)
	assert.Empty(t, s.renewablePrefixes)
	assert.Contains(t, s.advertisedPrefixes, netip.MustParsePrefix("2001:db8::/64"))
	// The valid-lifetime deadline is still tracked so existing leases can
	// age out against it via the deprecated-lease path in commitLease.
	assert.Contains(t, s.validUntilByPrefix, netip.MustParsePrefix("2001:db8::/64"))
}

func TestV6TrackedPrefixChanged_PreferredExpiredActiveFallsBackToRenewablePrefix(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			PrefixSource: V6PrefixSourceInterface,
			RangeStart:   net.ParseIP("fd00::1234:5678:9abc:de00"),
			notify:       notify6,
		},
	}

	err := s.trackedPrefixChanged(&raPrefixSnapshot{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 0,
		ValidSec:     3600,
	}, []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 0,
		ValidSec:     3600,
	}, {
		Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
		PreferredSec: 1800,
		ValidSec:     3600,
	}})
	require.NoError(t, err)

	assert.Equal(t, net.ParseIP("2001:db8:1::1234:5678:9abc:de00"), s.conf.ipStart)
	assert.Contains(t, s.renewablePrefixes, netip.MustParsePrefix("2001:db8:1::/64"))
}

// TestV6SetTrackedRangeStart_RebuildsBitmapWhenPrefixUnchanged is a
// regression test for a bug where setTrackedRangeStart would skip the
// ipAddrs rebuild when the tracked prefix had not moved.  In that case,
// occupancy bits for dropped leases — including leases removed by a prior
// ResetLeases that also does not touch ipAddrs — stayed marked forever,
// making the pool appear exhausted.
func TestV6SetTrackedRangeStart_RebuildsBitmapWhenPrefixUnchanged(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			ipStart: net.ParseIP("2001:db8::10"),
			notify:  notify6,
		},
	}
	// Simulate stale bitmap state left behind by ResetLeases: an
	// occupancy bit is set for 0x10 (the only lease) and for 0x20 (a
	// lease that was removed from s.leases but whose bit never got
	// cleared by a previous code path).
	s.ipAddrs[0x10] = 1
	s.ipAddrs[0x20] = 1
	s.leases = []*dhcpsvc.Lease{{
		IP:     netip.MustParseAddr("2001:db8::10"),
		HWAddr: net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa},
		Expiry: time.Now().Add(time.Hour),
	}}

	// setTrackedRangeStart with the same ipStart must still rebuild the
	// bitmap from s.leases, clearing the stale 0x20 bit.
	s.setTrackedRangeStart(net.ParseIP("2001:db8::10"), []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 1800,
		ValidSec:     3600,
	}})

	assert.Equal(t, byte(1), s.ipAddrs[0x10])
	assert.Equal(t, byte(0), s.ipAddrs[0x20])
}

// TestV6ResetLeases_ClearsOccupancyBitmap guards against a latent bug where
// ResetLeases replaced s.leases but left s.ipAddrs untouched.  After this
// sequence a subsequent reserveLease call must be able to allocate the
// previously-occupied slot.
func TestV6ResetLeases_ClearsOccupancyBitmap(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			ipStart:   net.ParseIP("2001:db8::10"),
			leaseTime: time.Hour,
			notify:    notify6,
		},
	}
	// Two existing dynamic leases plus their bitmap bits.
	s.leases = []*dhcpsvc.Lease{{
		IP:     netip.MustParseAddr("2001:db8::10"),
		HWAddr: net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa},
		Expiry: time.Now().Add(time.Hour),
	}, {
		IP:     netip.MustParseAddr("2001:db8::11"),
		HWAddr: net.HardwareAddr{0xbb, 0xbb, 0xbb, 0xbb, 0xbb, 0xbb},
		Expiry: time.Now().Add(time.Hour),
	}}
	s.ipAddrs[0x10] = 1
	s.ipAddrs[0x11] = 1

	// ResetLeases replaces the set with empty.  The old occupancy bits
	// must not survive.
	err := s.ResetLeases(nil)
	require.NoError(t, err)
	assert.Empty(t, s.leases)
	assert.Equal(t, byte(0), s.ipAddrs[0x10])
	assert.Equal(t, byte(0), s.ipAddrs[0x11])

	// And a fresh reserve must succeed rather than hit NoAddrsAvail due
	// to stranded bits.
	lease := s.reserveLease(net.HardwareAddr{0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc})
	require.NotNil(t, lease)
	assert.Equal(t, netip.MustParseAddr("2001:db8::10"), lease.IP)
}

// TestV6ReserveLease_FailsWhenPreferredExpiredActiveDisabledPool pairs with
// the previous test to make sure that, after the pool is disabled,
// reserveLease refuses to hand out a fresh address (so a Solicit from a new
// client returns NoAddrsAvail instead of a zero-lifetime lease).
func TestV6ReserveLease_FailsWhenPreferredExpiredActiveDisabledPool(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			PrefixSource: V6PrefixSourceInterface,
			RangeStart:   net.ParseIP("fd00::1234:5678:9abc:de00"),
			leaseTime:    time.Hour,
			notify:       notify6,
		},
	}

	err := s.trackedPrefixChanged(&raPrefixSnapshot{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 0,
		ValidSec:     3600,
	}, []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 0,
		ValidSec:     3600,
	}})
	require.NoError(t, err)

	lease := s.reserveLease(net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff})
	assert.Nil(t, lease)
}

func TestRequiresProcessSuccess(t *testing.T) {
	assert.True(t, requiresProcessSuccess(dhcpv6.MessageTypeSolicit))
	assert.True(t, requiresProcessSuccess(dhcpv6.MessageTypeRenew))
	assert.False(t, requiresProcessSuccess(dhcpv6.MessageTypeRelease))
	assert.False(t, requiresProcessSuccess(dhcpv6.MessageTypeInformationRequest))
}

func TestReplyStatusForProcessFailure(t *testing.T) {
	code, msg, ok := replyStatusForProcessFailure(dhcpv6.MessageTypeConfirm)
	require.True(t, ok)
	assert.Equal(t, iana.StatusNotOnLink, code)
	assert.Equal(t, iana.StatusNotOnLink.String(), msg)

	code, msg, ok = replyStatusForProcessFailure(dhcpv6.MessageTypeRenew)
	require.True(t, ok)
	assert.Equal(t, iana.StatusNoBinding, code)
	assert.Equal(t, iana.StatusNoBinding.String(), msg)

	_, _, ok = replyStatusForProcessFailure(dhcpv6.MessageTypeRelease)
	assert.False(t, ok)
}

func TestV6Create_StaticPrefixSeedsRenewablePrefixes(t *testing.T) {
	srv, err := v6Create(V6ServerConf{
		Enabled:      true,
		PrefixSource: V6PrefixSourceStatic,
		RangeStart:   net.ParseIP("2001:db8::10"),
		notify:       notify6,
	})
	require.NoError(t, err)

	s, ok := srv.(*v6Server)
	require.True(t, ok)
	prefix := netip.MustParsePrefix("2001:db8::/64")
	_, ok = s.renewablePrefixes[prefix]
	assert.True(t, ok)
	_, ok = s.advertisedPrefixes[prefix]
	assert.True(t, ok)
}

func TestV6ResetLeases_PreservesAdvertisedInterfacePrefixes(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			PrefixSource: V6PrefixSourceInterface,
			ipStart:      net.ParseIP("2001:db8:1::10"),
		},
		advertisedPrefixes: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8::/64"):   {},
			netip.MustParsePrefix("2001:db8:1::/64"): {},
		},
	}

	err := s.ResetLeases([]*dhcpsvc.Lease{{
		IP:     netip.MustParseAddr("2001:db8::10"),
		HWAddr: net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa},
	}, {
		IP:     netip.MustParseAddr("2001:db8:1::10"),
		HWAddr: net.HardwareAddr{0xbb, 0xbb, 0xbb, 0xbb, 0xbb, 0xbb},
	}, {
		IP:     netip.MustParseAddr("2001:db8:2::10"),
		HWAddr: net.HardwareAddr{0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc},
	}})
	require.NoError(t, err)

	require.Len(t, s.leases, 2)
	assert.Contains(t, []netip.Addr{s.leases[0].IP, s.leases[1].IP}, netip.MustParseAddr("2001:db8::10"))
	assert.Contains(t, []netip.Addr{s.leases[0].IP, s.leases[1].IP}, netip.MustParseAddr("2001:db8:1::10"))
}

func TestObservedDNSIPAddrs(t *testing.T) {
	addrs := observedDNSIPAddrs([]aghnet.IPv6AddrState{{
		Addr:                 netip.MustParseAddr("fe80::1"),
		PreferredLifetimeSec: math.MaxUint32,
		ValidLifetimeSec:     math.MaxUint32,
	}, {
		Addr:                 netip.MustParseAddr("2001:db8::10"),
		PreferredLifetimeSec: 1800,
		ValidLifetimeSec:     3600,
	}, {
		Addr:      netip.MustParseAddr("2001:db8::20"),
		Tentative: true,
	}})

	require.Len(t, addrs, 2)
	assert.Equal(t, net.ParseIP("fe80::1"), addrs[0])
	assert.Equal(t, net.ParseIP("2001:db8::10"), addrs[1])
}

func TestObservedDNSIPAddrs_FiltersDeprecatedGlobals(t *testing.T) {
	addrs := observedDNSIPAddrs([]aghnet.IPv6AddrState{{
		Addr:                 netip.MustParseAddr("fe80::1"),
		PreferredLifetimeSec: math.MaxUint32,
		ValidLifetimeSec:     math.MaxUint32,
	}, {
		Addr:                 netip.MustParseAddr("2001:db8::10"),
		PreferredLifetimeSec: 0,
		ValidLifetimeSec:     3600,
	}, {
		Addr:                 netip.MustParseAddr("2001:db8::20"),
		PreferredLifetimeSec: 1200,
		ValidLifetimeSec:     3600,
	}})

	require.Len(t, addrs, 2)
	assert.Equal(t, net.ParseIP("fe80::1"), addrs[0])
	assert.Equal(t, net.ParseIP("2001:db8::20"), addrs[1])
}

func TestV6FindUsableLease_PrefersCurrentPoolWithoutRequestedDeprecatedIP(t *testing.T) {
	mac := net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa}
	s := &v6Server{
		conf: V6ServerConf{
			ipStart: net.ParseIP("2001:db8:1::10"),
		},
		advertisedPrefixes: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8::/64"):   {},
			netip.MustParsePrefix("2001:db8:1::/64"): {},
		},
		leases: []*dhcpsvc.Lease{{
			IP:     netip.MustParseAddr("2001:db8::10"),
			HWAddr: mac,
		}, {
			IP:     netip.MustParseAddr("2001:db8:1::10"),
			HWAddr: mac,
		}},
	}

	msg, err := dhcpv6.NewMessage()
	require.NoError(t, err)
	msg.MessageType = dhcpv6.MessageTypeRenew

	lease := s.findUsableLease(msg, mac)
	require.NotNil(t, lease)
	assert.Equal(t, netip.MustParseAddr("2001:db8:1::10"), lease.IP)
}

func TestV6FindUsableLease_MatchesRequestedDeprecatedLease(t *testing.T) {
	mac := net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa}
	s := &v6Server{
		conf: V6ServerConf{
			ipStart: net.ParseIP("2001:db8:1::10"),
		},
		advertisedPrefixes: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8::/64"):   {},
			netip.MustParsePrefix("2001:db8:1::/64"): {},
		},
		leases: []*dhcpsvc.Lease{{
			IP:     netip.MustParseAddr("2001:db8::10"),
			HWAddr: mac,
		}, {
			IP:     netip.MustParseAddr("2001:db8:1::10"),
			HWAddr: mac,
		}},
	}

	msg, err := dhcpv6.NewMessage()
	require.NoError(t, err)
	msg.MessageType = dhcpv6.MessageTypeRenew
	msg.AddOption(&dhcpv6.OptIANA{
		Options: dhcpv6.IdentityOptions{
			Options: []dhcpv6.Option{&dhcpv6.OptIAAddress{
				IPv6Addr: net.ParseIP("2001:db8::10"),
			}},
		},
	})

	lease := s.findUsableLease(msg, mac)
	require.NotNil(t, lease)
	assert.Equal(t, netip.MustParseAddr("2001:db8::10"), lease.IP)
}

func TestV6FindUsableLease_MatchesRequestedDeprecatedLeaseOnRequest(t *testing.T) {
	mac := net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa}
	s := &v6Server{
		conf: V6ServerConf{
			ipStart: net.ParseIP("2001:db8:1::10"),
		},
		advertisedPrefixes: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8::/64"):   {},
			netip.MustParsePrefix("2001:db8:1::/64"): {},
		},
		leases: []*dhcpsvc.Lease{{
			IP:     netip.MustParseAddr("2001:db8::10"),
			HWAddr: mac,
			Expiry: time.Now().Add(10 * time.Minute),
		}},
	}

	msg, err := dhcpv6.NewMessage()
	require.NoError(t, err)
	msg.MessageType = dhcpv6.MessageTypeRequest
	msg.AddOption(&dhcpv6.OptIANA{
		Options: dhcpv6.IdentityOptions{
			Options: []dhcpv6.Option{&dhcpv6.OptIAAddress{
				IPv6Addr: net.ParseIP("2001:db8::10"),
			}},
		},
	})

	lease := s.findUsableLease(msg, mac)
	require.NotNil(t, lease)
	assert.Equal(t, netip.MustParseAddr("2001:db8::10"), lease.IP)
}

func TestV6FindUsableLease_SkipsExpiredDeprecatedLease(t *testing.T) {
	mac := net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa}
	s := &v6Server{
		conf: V6ServerConf{
			ipStart: net.ParseIP("2001:db8:1::10"),
		},
		advertisedPrefixes: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8::/64"):   {},
			netip.MustParsePrefix("2001:db8:1::/64"): {},
		},
		leases: []*dhcpsvc.Lease{{
			IP:     netip.MustParseAddr("2001:db8::10"),
			HWAddr: mac,
			Expiry: time.Now().Add(-time.Minute),
		}, {
			IP:     netip.MustParseAddr("2001:db8:1::10"),
			HWAddr: mac,
			Expiry: time.Now().Add(10 * time.Minute),
		}},
	}

	msg, err := dhcpv6.NewMessage()
	require.NoError(t, err)
	msg.MessageType = dhcpv6.MessageTypeRenew
	msg.AddOption(&dhcpv6.OptIANA{
		Options: dhcpv6.IdentityOptions{
			Options: []dhcpv6.Option{&dhcpv6.OptIAAddress{
				IPv6Addr: net.ParseIP("2001:db8::10"),
			}},
		},
	})

	lease := s.findUsableLease(msg, mac)
	require.NotNil(t, lease)
	assert.Equal(t, netip.MustParseAddr("2001:db8:1::10"), lease.IP)
}

func TestV6ReserveLease_PreservesDeprecatedLeaseForSameMAC(t *testing.T) {
	mac := net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa}
	s := &v6Server{
		conf: V6ServerConf{
			ipStart: net.ParseIP("2001:db8:1::10"),
		},
		leases: []*dhcpsvc.Lease{{
			IP:     netip.MustParseAddr("2001:db8::10"),
			HWAddr: mac,
			Expiry: time.Now().Add(10 * time.Minute),
		}},
	}

	lease := s.reserveLease(mac)
	require.NotNil(t, lease)
	assert.Equal(t, netip.MustParseAddr("2001:db8:1::10"), lease.IP)
	require.Len(t, s.leases, 2)
	assert.Contains(t, []netip.Addr{s.leases[0].IP, s.leases[1].IP}, netip.MustParseAddr("2001:db8::10"))
	assert.Contains(t, []netip.Addr{s.leases[0].IP, s.leases[1].IP}, netip.MustParseAddr("2001:db8:1::10"))
}

func TestV6CommitLease_DeprecatedDynamicLeaseKeepsRemainingLifetime(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			ipStart:   net.ParseIP("2001:db8:1::10"),
			leaseTime: time.Hour,
		},
		validUntilByPrefix: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"): time.Now().Add(10 * time.Minute),
		},
	}

	lease := &dhcpsvc.Lease{
		IP:     netip.MustParseAddr("2001:db8::10"),
		Expiry: time.Now().Add(10 * time.Minute),
	}

	msg, err := dhcpv6.NewMessage()
	require.NoError(t, err)
	msg.MessageType = dhcpv6.MessageTypeRenew

	lifetime, preferred := s.commitLease(msg, lease)
	assert.Greater(t, lifetime, 9*time.Minute)
	assert.LessOrEqual(t, lifetime, 10*time.Minute)
	assert.Equal(t, time.Duration(0), preferred)
}

func TestV6CommitLease_DeprecatedLeaseCappedByPrefixValidLifetime(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			ipStart:   net.ParseIP("2001:db8:1::10"),
			leaseTime: time.Hour,
		},
		validUntilByPrefix: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"): time.Now().Add(2 * time.Minute),
		},
	}

	lease := &dhcpsvc.Lease{
		IP:     netip.MustParseAddr("2001:db8::10"),
		Expiry: time.Now().Add(24 * time.Hour),
	}

	msg, err := dhcpv6.NewMessage()
	require.NoError(t, err)
	msg.MessageType = dhcpv6.MessageTypeRenew

	lifetime, preferred := s.commitLease(msg, lease)
	assert.Greater(t, lifetime, time.Minute)
	assert.LessOrEqual(t, lifetime, 2*time.Minute)
	assert.Equal(t, time.Duration(0), preferred)
}

func TestV6CommitLease_DeprecatedConfirmCappedByPrefixValidLifetime(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			ipStart:   net.ParseIP("2001:db8:1::10"),
			leaseTime: time.Hour,
		},
		validUntilByPrefix: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"): time.Now().Add(90 * time.Second),
		},
	}

	lease := &dhcpsvc.Lease{
		IP:     netip.MustParseAddr("2001:db8::10"),
		Expiry: time.Now().Add(24 * time.Hour),
	}

	msg, err := dhcpv6.NewMessage()
	require.NoError(t, err)
	msg.MessageType = dhcpv6.MessageTypeConfirm

	lifetime, preferred := s.commitLease(msg, lease)
	assert.Greater(t, lifetime, time.Minute)
	assert.LessOrEqual(t, lifetime, 90*time.Second)
	assert.Equal(t, time.Duration(0), preferred)
}

func TestV6CommitLease_StaticConfirmUsesConfiguredLeaseTime(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			leaseTime: 24 * time.Hour,
		},
	}

	lease := &dhcpsvc.Lease{
		IP:       netip.MustParseAddr("2001:db8::10"),
		HWAddr:   net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa},
		IsStatic: true,
	}

	msg, err := dhcpv6.NewMessage()
	require.NoError(t, err)
	msg.MessageType = dhcpv6.MessageTypeConfirm

	lifetime, preferred := s.commitLease(msg, lease)
	assert.Equal(t, 24*time.Hour, lifetime)
	assert.Equal(t, 24*time.Hour, preferred)
}

func TestV6CommitLease_DeprecatedLeaseUsesZeroPreferredLifetime(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			ipStart:   net.ParseIP("2001:db8:1::10"),
			leaseTime: time.Hour,
		},
		advertisedPrefixes: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8::/64"): {},
		},
		validUntilByPrefix: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"): time.Now().Add(10 * time.Minute),
		},
	}

	lease := &dhcpsvc.Lease{
		IP:     netip.MustParseAddr("2001:db8::10"),
		HWAddr: net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa},
		Expiry: time.Now().Add(10 * time.Minute),
	}

	msg, err := dhcpv6.NewMessage()
	require.NoError(t, err)
	msg.MessageType = dhcpv6.MessageTypeRenew

	_, preferred := s.commitLease(msg, lease)
	assert.Equal(t, time.Duration(0), preferred)
}

func TestV6CommitLease_SecondaryRenewablePrefixGetsFullLifetime(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			leaseTime: time.Hour,
			notify:    notify6,
		},
		validUntilByPrefix: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8:2::/64"): time.Now().Add(30 * time.Minute),
		},
		renewablePrefixes: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8:2::/64"): {},
		},
	}

	lease := &dhcpsvc.Lease{
		IP:     netip.MustParseAddr("2001:db8:2::10"),
		Expiry: time.Now().Add(10 * time.Minute),
	}

	msg, err := dhcpv6.NewMessage()
	require.NoError(t, err)
	msg.MessageType = dhcpv6.MessageTypeRenew

	lifetime, preferred := s.commitLease(msg, lease)
	assert.Greater(t, lifetime, 29*time.Minute)
	assert.LessOrEqual(t, lifetime, 30*time.Minute)
	assert.Equal(t, lifetime, preferred)
}

func TestV6RestoreDeprecatedPrefixes(t *testing.T) {
	now := time.Unix(400, 0)
	s := &v6Server{
		restoredRenewable: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8:1::/64"): {},
		},
		restoredDeprecated: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"): now.Add(90 * time.Minute),
		},
	}

	st := newObservedRAState()
	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 1800,
			ValidSec:     7200,
		},
	}, now)

	s.restoreDeprecatedPrefixes(now, &st)

	pios := st.pios(now)
	require.Len(t, pios, 2)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:1::/64"), pios[0].Prefix)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), pios[1].Prefix)
	assert.Equal(t, uint32(0), pios[1].PreferredSec)
	assert.Equal(t, uint32(5400), pios[1].ValidSec)
}

func TestV6RestoreDeprecatedPrefixes_IgnoresUnrelatedMetadata(t *testing.T) {
	now := time.Unix(400, 0)
	s := &v6Server{
		restoredRenewable: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8:1::/64"): {},
		},
		restoredDeprecated: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"): now.Add(90 * time.Minute),
		},
	}

	st := newObservedRAState()
	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:2::/64"),
			PreferredSec: 1800,
			ValidSec:     7200,
		},
	}, now)

	s.restoreDeprecatedPrefixes(now, &st)

	pios := st.pios(now)
	require.Len(t, pios, 1)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:2::/64"), pios[0].Prefix)
}

func TestV6RestoreDeprecatedPrefixes_RestoresAfterDelayedOverlap(t *testing.T) {
	now := time.Unix(400, 0)
	s := &v6Server{
		restoredRenewable: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8:1::/64"): {},
		},
		restoredDeprecated: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"): now.Add(90 * time.Minute),
		},
	}

	st := newObservedRAState()
	st.merge(raObservation{}, now)
	s.restoreDeprecatedPrefixes(now, &st)
	require.Empty(t, st.pios(now))

	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 1800,
			ValidSec:     7200,
		},
	}, now.Add(time.Minute))
	s.restoreDeprecatedPrefixes(now.Add(time.Minute), &st)

	pios := st.pios(now.Add(time.Minute))
	require.Len(t, pios, 2)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:1::/64"), pios[0].Prefix)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), pios[1].Prefix)
}

func TestV6RestoreDeprecatedPrefixes_WithoutRenewablePrefixesNeedsObservedPrefixState(t *testing.T) {
	now := time.Unix(400, 0)
	s := &v6Server{
		restoredDeprecated: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"): now.Add(30 * time.Minute),
		},
	}

	st := newObservedRAState()
	st.merge(raObservation{}, now)

	s.restoreDeprecatedPrefixes(now, &st)

	pios := st.pios(now)
	require.Empty(t, pios)
}

func TestV6RestoreDeprecatedPrefixes_WithoutRenewablePrefixesNeedsDeprecatedOverlap(t *testing.T) {
	now := time.Unix(400, 0)
	s := &v6Server{
		restoredDeprecated: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"):   now.Add(30 * time.Minute),
			netip.MustParsePrefix("2001:db8:1::/64"): now.Add(20 * time.Minute),
		},
	}

	st := newObservedRAState()
	st.merge(raObservation{
		Inactive: []raPrefixSnapshot{{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 0,
			ValidSec:     1200,
		}},
	}, now)

	s.restoreDeprecatedPrefixes(now, &st)

	pios := st.pios(now)
	require.Len(t, pios, 2)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), pios[0].Prefix)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:1::/64"), pios[1].Prefix)
}

func TestV6RestoreDeprecatedPrefixes_RequiresFullRenewableMatch(t *testing.T) {
	now := time.Unix(400, 0)
	s := &v6Server{
		restoredRenewable: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("fd00::/64"):       {},
			netip.MustParsePrefix("2001:db8:1::/64"): {},
		},
		restoredDeprecated: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"): now.Add(30 * time.Minute),
		},
	}

	st := newObservedRAState()
	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("fd00::/64"),
			PreferredSec: 1800,
			ValidSec:     7200,
		},
	}, now)

	s.restoreDeprecatedPrefixes(now, &st)

	pios := st.pios(now)
	require.Len(t, pios, 1)
	assert.Equal(t, netip.MustParsePrefix("fd00::/64"), pios[0].Prefix)
}

func TestV6DeprecatedPrefixMeta_FallsBackToRestoredMetadata(t *testing.T) {
	now := time.Unix(500, 0)
	s := &v6Server{
		conf: V6ServerConf{
			PrefixSource: V6PrefixSourceInterface,
		},
		restoredRenewable: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8:1::/64"): {},
		},
		restoredDeprecated: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"): now.Add(30 * time.Minute),
		},
		persistRestoredMeta: true,
	}

	renewable, deprecated := s.deprecatedPrefixMeta(now)
	assert.Equal(t, map[netip.Prefix]struct{}{
		netip.MustParsePrefix("2001:db8:1::/64"): {},
	}, renewable)
	assert.Equal(t, map[netip.Prefix]time.Time{
		netip.MustParsePrefix("2001:db8::/64"): now.Add(30 * time.Minute),
	}, deprecated)
}

func TestV6DeprecatedPrefixMeta_DropsFallbackAfterObservation(t *testing.T) {
	now := time.Unix(500, 0)
	s := &v6Server{
		conf: V6ServerConf{
			PrefixSource: V6PrefixSourceInterface,
		},
		restoredRenewable: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8:1::/64"): {},
		},
		restoredDeprecated: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"): now.Add(30 * time.Minute),
		},
		persistRestoredMeta: true,
	}

	st := newObservedRAState()
	st.merge(raObservation{}, now)
	s.restoreDeprecatedPrefixes(now, &st)

	renewable, deprecated := s.deprecatedPrefixMeta(now)
	assert.Empty(t, renewable)
	assert.Empty(t, deprecated)
}

func TestV6SetTrackedRangeStart_RefreshesValidUntilForSamePrefixes(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			ipStart: net.ParseIP("2001:db8:1::10"),
		},
		advertisedPrefixes: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8::/64"): {},
		},
		renewablePrefixes: map[netip.Prefix]struct{}{},
	}

	s.setTrackedRangeStart(net.ParseIP("2001:db8:1::10"), []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 0,
		ValidSec:     600,
	}})
	firstDeadline := s.validUntilByPrefix[netip.MustParsePrefix("2001:db8::/64")]

	s.setTrackedRangeStart(net.ParseIP("2001:db8:1::10"), []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 0,
		ValidSec:     120,
	}})
	secondDeadline := s.validUntilByPrefix[netip.MustParsePrefix("2001:db8::/64")]

	assert.True(t, secondDeadline.Before(firstDeadline.Add(-4*time.Minute)))
}

func TestV6SetTrackedRangeStart_ClampsDeprecatedLeaseExpiry(t *testing.T) {
	now := time.Now()
	s := &v6Server{
		conf: V6ServerConf{
			ipStart: net.ParseIP("2001:db8:1::10"),
			notify:  notify6,
		},
		leases: []*dhcpsvc.Lease{{
			IP:     netip.MustParseAddr("2001:db8::10"),
			HWAddr: net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa},
			Expiry: now.Add(24 * time.Hour),
		}},
	}

	s.setTrackedRangeStart(net.ParseIP("2001:db8:1::10"), []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 0,
		ValidSec:     120,
	}, {
		Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
		PreferredSec: 1800,
		ValidSec:     3600,
	}})

	require.Len(t, s.leases, 1)
	assert.LessOrEqual(t, time.Until(s.leases[0].Expiry), 2*time.Minute)
	assert.Greater(t, time.Until(s.leases[0].Expiry), time.Minute)
}

func TestV6SetTrackedRangeStart_MetadataOnlyChangeNotifiesDBStore(t *testing.T) {
	var notified []uint32

	s := &v6Server{
		conf: V6ServerConf{
			PrefixSource: V6PrefixSourceInterface,
			RASLAACOnly:  true,
			notify: func(flags uint32) {
				notified = append(notified, flags)
			},
		},
	}

	s.setTrackedRangeStart(nil, []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 0,
		ValidSec:     300,
	}})

	assert.Equal(t, []uint32{LeaseChangedDBStore}, notified)
}

func TestV6SetTrackedRangeStart_ClampsExpiryWhenLifetimesShrinkInPlace(t *testing.T) {
	now := time.Now()
	s := &v6Server{
		conf: V6ServerConf{
			ipStart: net.ParseIP("2001:db8:1::10"),
			notify:  notify6,
		},
		advertisedPrefixes: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8::/64"): {},
		},
		renewablePrefixes: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8::/64"): {},
		},
		leases: []*dhcpsvc.Lease{{
			IP:     netip.MustParseAddr("2001:db8::10"),
			HWAddr: net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa},
			Expiry: now.Add(24 * time.Hour),
		}},
	}

	s.setTrackedRangeStart(net.ParseIP("2001:db8:1::10"), []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 300,
		ValidSec:     300,
	}})

	require.Len(t, s.leases, 1)
	assert.LessOrEqual(t, time.Until(s.leases[0].Expiry), 5*time.Minute)
	assert.Greater(t, time.Until(s.leases[0].Expiry), 4*time.Minute)
}

func TestV6SetTrackedRangeStart_FiltersLeasesBelowNewHostTemplate(t *testing.T) {
	s := &v6Server{
		conf: V6ServerConf{
			ipStart: net.ParseIP("2001:db8::10"),
			notify:  notify6,
		},
		leases: []*dhcpsvc.Lease{{
			IP:     netip.MustParseAddr("2001:db8::20"),
			HWAddr: net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa},
			Expiry: time.Now().Add(time.Hour),
		}, {
			IP:     netip.MustParseAddr("2001:db8::90"),
			HWAddr: net.HardwareAddr{0xbb, 0xbb, 0xbb, 0xbb, 0xbb, 0xbb},
			Expiry: time.Now().Add(time.Hour),
		}},
	}

	s.setTrackedRangeStart(net.ParseIP("2001:db8::80"), []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 1800,
		ValidSec:     3600,
	}})

	require.Len(t, s.leases, 1)
	assert.Equal(t, netip.MustParseAddr("2001:db8::90"), s.leases[0].IP)
}

// TestV6CommitLease_SnapshotAfterLockDrop reproduces the TOCTOU window between
// commitLease's renewable-check and the subsequent lifetime computation that
// used to re-acquire the lock: a concurrent setTrackedRangeStart that removes
// the prefix before the second lookup must not see the lease handed a
// "renewable" lifetime for a no-longer-renewable prefix.
func TestV6CommitLease_SnapshotAfterLockDrop(t *testing.T) {
	now := time.Now()
	s := &v6Server{
		conf: V6ServerConf{
			ipStart:   net.ParseIP("2001:db8::10"),
			leaseTime: time.Hour,
			notify:    notify6,
		},
		advertisedPrefixes: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8::/64"): {},
		},
		renewablePrefixes: map[netip.Prefix]struct{}{
			netip.MustParsePrefix("2001:db8::/64"): {},
		},
		validUntilByPrefix: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"): now.Add(45 * time.Minute),
		},
		preferredUntilByPrefix: map[netip.Prefix]time.Time{
			netip.MustParsePrefix("2001:db8::/64"): now.Add(30 * time.Minute),
		},
	}

	lease := &dhcpsvc.Lease{
		IP:     netip.MustParseAddr("2001:db8::10"),
		HWAddr: net.HardwareAddr{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa},
		Expiry: now.Add(10 * time.Minute),
	}

	msg, err := dhcpv6.NewMessage()
	require.NoError(t, err)
	msg.MessageType = dhcpv6.MessageTypeRenew

	lifetime, preferred := s.commitLease(msg, lease)
	// Capped by remaining valid lifetime of the prefix (~45m), not leaseTime (1h).
	assert.Greater(t, lifetime, 44*time.Minute)
	assert.LessOrEqual(t, lifetime, 45*time.Minute)
	// Preferred lifetime comes from the same snapshot and is capped by the
	// prefix's remaining preferred lifetime (~30m).
	assert.Greater(t, preferred, 29*time.Minute)
	assert.LessOrEqual(t, preferred, 30*time.Minute)
	// The lease expiry must have been updated consistently with the
	// returned lifetime (data race regression test for l.Expiry mutation
	// without the lock).
	assert.InDelta(t, lifetime.Seconds(), time.Until(lease.Expiry).Seconds(), 2.0)
}
