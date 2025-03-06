package dhcpd

import (
	"net"
	"testing"

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
		prefix:                      net.ParseIP("1234::"),
		prefixLen:                   64,
		recursiveDNSServer:          net.ParseIP("fe80::800:27ff:fe00:0"),
		sourceLinkLayerAddress:      []byte{0x0A, 0x00, 0x27, 0x00, 0x00, 0x00},
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

	wantOpts := layers.ICMPv6Options{{
		Type: layers.ICMPv6OptPrefixInfo,
		Data: []uint8{
			0x40, 0xC0, 0x00, 0x00, 0x0E, 0x10, 0x00, 0x00,
			0x0E, 0x10, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		},
	}, {
		Type: layers.ICMPv6OptMTU,
		Data: []uint8{0x00, 0x00, 0x00, 0x00, 0x05, 0xDC},
	}, {
		Type: layers.ICMPv6OptSourceAddress,
		Data: []uint8{0x0A, 0x00, 0x27, 0x00, 0x00, 0x0},
	}, {
		// Package layers declares no constant for Recursive DNS Server option.
		Type: layers.ICMPv6Opt(25),
		Data: []uint8{
			0x00, 0x00, 0x00, 0x00, 0x0E, 0x10, 0xFE, 0x80,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x00,
			0x27, 0xFF, 0xFE, 0x00, 0x00, 0x00,
		},
	}}
	assert.Equal(t, wantOpts, raPkt.Options)
}
