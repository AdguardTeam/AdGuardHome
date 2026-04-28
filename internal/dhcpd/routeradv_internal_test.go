package dhcpd

import (
	"context"
	"encoding/binary"
	"net/netip"
	"slices"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateICMPv6RAPacket(t *testing.T) {
	raConf := icmpv6RA{
		managedAddressConfiguration: false,
		otherConfiguration:          true,
		mtu:                         1500,
		prefixes: []prefixPIO{{
			Prefix:       netip.MustParsePrefix("1234::/64"),
			PreferredSec: 1200,
			ValidSec:     3600,
		}, {
			Prefix:       netip.MustParsePrefix("2001:db8:abcd::/60"),
			PreferredSec: 0,
			ValidSec:     1800,
		}},
		recursiveDNSServer:     netip.MustParseAddr("fe80::800:27ff:fe00:0"),
		sourceLinkLayerAddress: []byte{0x0A, 0x00, 0x27, 0x00, 0x00, 0x00},
	}

	pkt, err := createICMPv6RAPacket(raConf)
	require.NoError(t, err)

	icmpPkt := &layers.ICMPv6{}
	err = icmpPkt.DecodeFromBytes(pkt, gopacket.NilDecodeFeedback)
	require.NoError(t, err)

	require.Equal(t, layers.LayerTypeICMPv6RouterAdvertisement, icmpPkt.NextLayerType())

	raPkt := &layers.ICMPv6RouterAdvertisement{}
	err = raPkt.DecodeFromBytes(icmpPkt.LayerPayload(), gopacket.NilDecodeFeedback)
	require.NoError(t, err)

	assert.Equal(t, raConf.managedAddressConfiguration, raPkt.ManagedAddressConfig())
	assert.Equal(t, raConf.otherConfiguration, raPkt.OtherConfig())

	require.Len(t, raPkt.Options, 5)

	opt := raPkt.Options[0]
	require.Equal(t, layers.ICMPv6OptPrefixInfo, opt.Type)
	assert.Equal(t, byte(64), opt.Data[0])
	assert.Equal(t, byte(0xc0), opt.Data[1])
	assert.Equal(t, uint32(3600), binary.BigEndian.Uint32(opt.Data[2:6]))
	assert.Equal(t, uint32(1200), binary.BigEndian.Uint32(opt.Data[6:10]))
	assert.Equal(t, netip.MustParsePrefix("1234::/64").Addr().As16(), [16]byte(opt.Data[14:30]))

	opt = raPkt.Options[1]
	require.Equal(t, layers.ICMPv6OptPrefixInfo, opt.Type)
	assert.Equal(t, byte(60), opt.Data[0])
	assert.Equal(t, byte(0xc0), opt.Data[1])
	assert.Equal(t, uint32(1800), binary.BigEndian.Uint32(opt.Data[2:6]))
	assert.Equal(t, uint32(0), binary.BigEndian.Uint32(opt.Data[6:10]))
	assert.Equal(
		t,
		netip.MustParsePrefix("2001:db8:abcd::/60").Masked().Addr().As16(),
		[16]byte(opt.Data[14:30]),
	)

	opt = raPkt.Options[2]
	require.Equal(t, layers.ICMPv6OptMTU, opt.Type)
	assert.Equal(t, uint32(1500), binary.BigEndian.Uint32(opt.Data[2:6]))

	opt = raPkt.Options[3]
	require.Equal(t, layers.ICMPv6OptSourceAddress, opt.Type)
	assert.Equal(t, []byte{0x0A, 0x00, 0x27, 0x00, 0x00, 0x00}, opt.Data)

	opt = raPkt.Options[4]
	require.Equal(t, layers.ICMPv6Opt(25), opt.Type)
	assert.Equal(t, uint32(defaultRARDNSSLifetimeSeconds), binary.BigEndian.Uint32(opt.Data[2:6]))
	assert.Equal(t, netip.MustParseAddr("fe80::800:27ff:fe00:0").As16(), [16]byte(opt.Data[6:22]))
}

func TestRACtxSyncStateChange_DeprecatedExpiry(t *testing.T) {
	now := time.Unix(100, 0)
	ra := &raCtx{
		state: newObservedRAState(),
	}

	ra.state.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 1800,
			ValidSec:     3600,
		},
		Inactive: []raPrefixSnapshot{{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 0,
			ValidSec:     1,
		}},
	}, now)

	var notifications [][]prefixPIO
	ra.onActivePrefixChange = func(_ *raPrefixSnapshot, advertised []prefixPIO) {
		notifications = append(notifications, slices.Clone(advertised))
	}
	ra.lastDigest = ra.state.digest(now)

	ra.syncStateChange(now.Add(2*time.Second), nil)

	require.Len(t, notifications, 1)
	require.Len(t, notifications[0], 1)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:1::/64"), notifications[0][0].Prefix)
}

func TestRACtxSyncStateChange_StableStateDoesNotFireCallback(t *testing.T) {
	// Countdown on a stable kernel-observed prefix must not trigger the
	// downstream callback: the RA loop polls every few seconds but nothing
	// is actually changing, so v6Server should not see any transitions.
	now := time.Unix(100, 0)
	ra := &raCtx{
		state: newObservedRAState(),
	}

	ra.state.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 1800,
			ValidSec:     3600,
		},
		Inactive: []raPrefixSnapshot{{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 0,
			ValidSec:     1200,
		}},
	}, now)
	ra.lastDigest = ra.state.digest(now)

	var fired int
	ra.onActivePrefixChange = func(_ *raPrefixSnapshot, _ []prefixPIO) {
		fired++
	}

	// Three ticks without any new observation: only elapsed time changed,
	// so the digest must be unchanged and the callback must not fire.
	ra.syncStateChange(now.Add(1*time.Second), nil)
	ra.syncStateChange(now.Add(3*time.Second), nil)
	ra.syncStateChange(now.Add(5*time.Second), nil)

	assert.Zero(t, fired)
}

func TestRACtxSyncStateChange_DeprecatingPrefixTriggersCallback(t *testing.T) {
	now := time.Unix(120, 0)
	ra := &raCtx{
		state: newObservedRAState(),
	}

	ra.state.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 1800,
			ValidSec:     3600,
		},
		Inactive: []raPrefixSnapshot{{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 300,
			ValidSec:     1200,
		}},
	}, now)
	ra.lastDigest = ra.state.digest(now)

	var notifications [][]prefixPIO
	ra.onActivePrefixChange = func(_ *raPrefixSnapshot, advertised []prefixPIO) {
		notifications = append(notifications, slices.Clone(advertised))
	}

	ra.state.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 1700,
			ValidSec:     3500,
		},
		Inactive: []raPrefixSnapshot{{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 0,
			ValidSec:     1100,
		}},
	}, now.Add(time.Minute))

	ra.syncStateChange(now.Add(time.Minute), nil)

	require.Len(t, notifications, 1)
	require.Len(t, notifications[0], 2)
	assert.Equal(t, uint32(0), notifications[0][1].PreferredSec)
}

// TestRACtxSyncStateChange_PreferredExpiryTriggersCallback is a regression
// test for a bug where a prefix whose preferred lifetime counts down from >0
// to 0 would not fire onActivePrefixChange.  The digest compared absolute
// deadlines, which stay constant during a natural countdown, so the
// transition from renewable to non-renewable was invisible to the downstream
// reconciliation path — v6Server.renewablePrefixes would stay stale and
// commitLease would keep refreshing leases on a deprecated prefix until the
// next kernel observation happened to report preferred=0.
//
// The digest now carries a time-derived preferredExpired boolean that flips
// exactly once at the transition, so the callback fires between polls.
func TestRACtxSyncStateChange_PreferredExpiryTriggersCallback(t *testing.T) {
	now := time.Unix(100, 0)
	ra := &raCtx{
		state: newObservedRAState(),
	}

	ra.state.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 300,
			ValidSec:     3600,
		},
	}, now)
	ra.lastDigest = ra.state.digest(now)

	var notifications [][]prefixPIO
	ra.onActivePrefixChange = func(_ *raPrefixSnapshot, advertised []prefixPIO) {
		notifications = append(notifications, slices.Clone(advertised))
	}

	// While the preferred countdown still has time remaining no callback
	// should fire.
	ra.syncStateChange(now.Add(299*time.Second), nil)
	assert.Empty(t, notifications)

	// At the moment the preferred lifetime hits zero the callback must
	// fire so v6Server can drop the prefix from its renewable set.
	ra.syncStateChange(now.Add(301*time.Second), nil)
	require.Len(t, notifications, 1)
	require.Len(t, notifications[0], 1)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), notifications[0][0].Prefix)
	assert.Equal(t, uint32(0), notifications[0][0].PreferredSec)

	// But subsequent ticks past the expiry must not re-fire: the state
	// digest has settled at preferredExpired=true and stays that way.
	ra.syncStateChange(now.Add(305*time.Second), nil)
	ra.syncStateChange(now.Add(310*time.Second), nil)
	assert.Len(t, notifications, 1)
}

// TestRACtxSyncStateChange_PreferredExpiryOnDeprecatedEntryStillQuiescent
// guards against the opposite failure: entries that are *already* deprecated
// (origin raPrefixOriginDeprecated, preferred=0 from the start) must not
// cause repeated callbacks just because their preferredExpired flag is true.
func TestRACtxSyncStateChange_PreferredExpiryOnDeprecatedEntryStillQuiescent(t *testing.T) {
	now := time.Unix(100, 0)
	ra := &raCtx{
		state: newObservedRAState(),
	}

	// Inactive entry with preferred=0 starts as origin=Deprecated.
	ra.state.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 1800,
			ValidSec:     3600,
		},
		Inactive: []raPrefixSnapshot{{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 0,
			ValidSec:     1200,
		}},
	}, now)
	ra.lastDigest = ra.state.digest(now)

	var fired int
	ra.onActivePrefixChange = func(_ *raPrefixSnapshot, _ []prefixPIO) {
		fired++
	}

	// Several steady-state ticks across the valid countdown.  No kernel
	// state changed, so nothing should fire.
	ra.syncStateChange(now.Add(1*time.Second), nil)
	ra.syncStateChange(now.Add(5*time.Second), nil)
	ra.syncStateChange(now.Add(30*time.Second), nil)
	assert.Zero(t, fired)
}

func TestRACtxInit_AllowsNoSourceWhenObserving(t *testing.T) {
	ra := &raCtx{
		raAllowSLAAC:     true,
		packetSendPeriod: time.Second,
		observe:          func(context.Context) (raObservation, error) { return raObservation{}, nil },
	}

	err := ra.Init(newObservedRAState())
	require.NoError(t, err)
	require.NoError(t, ra.Close())
}

// TestRACtxEnsureConn_ClearsConnOnSetupFailure is a regression test for a bug
// where a partially-initialized icmp listener was closed by the error-path
// defer but ra.conn was left pointing to it.  The next call into ensureConn
// would then observe a non-nil but closed conn and attempt to use or re-close
// it, permanently stalling the RA loop.
//
// The scenario is hard to force without actually raising an error from a real
// icmp listener, so this test exercises the invariant directly: after an
// ensureConn failure the connection fields must be cleared so a subsequent
// call can attempt a fresh listen.
func TestRACtxEnsureConn_ClearsConnOnSetupFailure(t *testing.T) {
	ra := &raCtx{
		// No valid interface name means icmp.ListenPacket will fail when
		// ensureConn is called with a valid source address, triggering
		// the error path we care about.
		ifaceName: "",
	}

	err := ra.ensureConn(netip.MustParseAddr("fe80::1"))
	require.Error(t, err)
	assert.Nil(t, ra.conn)
	assert.False(t, ra.connSourceAddr.IsValid())

	// A subsequent call with an invalid source address must not try to
	// close a stale conn and must leave the fields cleared.
	err = ra.ensureConn(netip.Addr{})
	require.NoError(t, err)
	assert.Nil(t, ra.conn)
	assert.False(t, ra.connSourceAddr.IsValid())
}
