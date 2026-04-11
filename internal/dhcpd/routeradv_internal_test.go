package dhcpd

import (
	"context"
	"encoding/binary"
	"net/netip"
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
	assert.Equal(t, netip.MustParsePrefix("2001:db8:abcd::/60").Masked().Addr().As16(), [16]byte(opt.Data[14:30]))

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
		notifications = append(notifications, clonePIOs(advertised))
	}
	ra.lastActiveSnapshot = clonePrefixSnapshot(ra.state.activeSnapshot(now))
	ra.lastAdvertised = clonePIOs(ra.state.pios(now))

	ra.syncStateChange(now.Add(2*time.Second), false)

	require.Len(t, notifications, 1)
	require.Len(t, notifications[0], 1)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:1::/64"), notifications[0][0].Prefix)
}

func TestRACtxSyncStateChange_RefreshesCachedLifetimesWithoutCallback(t *testing.T) {
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
			ValidSec:     10,
		}},
	}, now)
	ra.lastActiveSnapshot = clonePrefixSnapshot(ra.state.activeSnapshot(now))
	ra.lastAdvertised = clonePIOs(ra.state.pios(now))
	ra.onActivePrefixChange = nil

	ra.syncStateChange(now.Add(3*time.Second), false)

	require.Len(t, ra.lastAdvertised, 2)
	assert.Equal(t, uint32(7), ra.lastAdvertised[1].ValidSec)
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
	ra.lastActiveSnapshot = clonePrefixSnapshot(ra.state.activeSnapshot(now))
	ra.lastAdvertised = clonePIOs(ra.state.pios(now))

	var notifications [][]prefixPIO
	ra.onActivePrefixChange = func(_ *raPrefixSnapshot, advertised []prefixPIO) {
		notifications = append(notifications, clonePIOs(advertised))
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

	ra.syncStateChange(now.Add(time.Minute), true)

	require.Len(t, notifications, 1)
	require.Len(t, notifications[0], 2)
	assert.Equal(t, uint32(0), notifications[0][1].PreferredSec)
}

func TestSameAdvertisedPIOSet_PreferredLifetimeChange(t *testing.T) {
	a := []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 1800,
		ValidSec:     3600,
	}}
	b := []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 900,
		ValidSec:     3600,
	}}

	assert.False(t, sameAdvertisedPIOSet(a, b, true))
	assert.True(t, sameAdvertisedPIOSet(a, b, false))
}

func TestSameAdvertisedPIOSet_ValidLifetimeChange(t *testing.T) {
	a := []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 0,
		ValidSec:     1200,
	}}
	b := []prefixPIO{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 0,
		ValidSec:     900,
	}}

	assert.False(t, sameAdvertisedPIOSet(a, b, true))
	assert.True(t, sameAdvertisedPIOSet(a, b, false))
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
