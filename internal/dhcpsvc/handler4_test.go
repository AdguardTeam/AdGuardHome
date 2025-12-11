package dhcpsvc_test

import (
	"context"
	"encoding/binary"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/faketime"
	"github.com/AdguardTeam/golibs/testutil/servicetest"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDHCPServer_ServeEther4_discover(t *testing.T) {
	t.Parallel()

	// ifaceName is the name of the test network interface.
	const ifaceName = "iface0"

	// leaseTTL is the lease duration used in this test.
	const leaseTTL time.Duration = 24 * time.Hour

	// NOTE: Keep in sync with testdata.
	const (
		// leaseHostnameStatic is the hostname for the static lease.
		leaseHostnameStatic = "static4"

		// leaseHostnameDynamic is the hostname for the dynamic lease.
		leaseHostnameDynamic = "dynamic4"

		// leaseHostnameExpired is the hostname for the expired lease.
		leaseHostnameExpired = "expired4"
	)

	// NOTE: Keep in sync with testdata.
	var (
		// hwAddrUnknown is the MAC address for an unknown client.
		hwAddrUnknown = net.HardwareAddr{0x0, 0x1, 0x2, 0x3, 0x4, 0x5}

		// hwAddrStatic is the MAC address for a known static lease.
		hwAddrStatic = net.HardwareAddr{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}

		// hwAddrDynamic is the MAC address for a known dynamic lease.
		hwAddrDynamic = net.HardwareAddr{0x2, 0x3, 0x4, 0x5, 0x6, 0x7}

		// hwAddrExpired is the MAC address for a known expired lease.
		hwAddrExpired = net.HardwareAddr{0x3, 0x4, 0x5, 0x6, 0x7, 0x8}
	)

	currentTime := time.Date(2025, 1, 1, 1, 1, 1, 0, time.UTC)
	testClock := &faketime.Clock{
		OnNow: func() (now time.Time) {
			return currentTime
		},
	}
	dynamicLeaseExpiry := time.Date(2025, 1, 1, 10, 1, 1, 0, time.UTC).Sub(currentTime)

	ipv4Conf := &dhcpsvc.IPv4Config{
		Clock:         testClock,
		SubnetMask:    netip.MustParseAddr("255.255.255.0"),
		GatewayIP:     netip.MustParseAddr("192.168.0.1"),
		RangeStart:    netip.MustParseAddr("192.168.0.100"),
		RangeEnd:      netip.MustParseAddr("192.168.0.200"),
		LeaseDuration: leaseTTL,
		Enabled:       true,
	}
	ifacesConfig := map[string]*dhcpsvc.InterfaceConfig{
		ifaceName: {
			IPv4: ipv4Conf,
			IPv6: &dhcpsvc.IPv6Config{Enabled: false},
		},
	}

	// TODO(e.burkov):  Add cases for wrong packets.
	testCases := []struct {
		name string
		in   gopacket.Packet
		want layers.DHCPOptions
	}{{
		name: "new",
		in:   newDHCPDISCOVER(t, hwAddrUnknown),
		want: layers.DHCPOptions{
			layers.NewDHCPOption(
				layers.DHCPOptMessageType,
				[]byte{byte(layers.DHCPMsgTypeOffer)},
			),
			layers.NewDHCPOption(
				layers.DHCPOptServerID,
				ifacesConfig[ifaceName].IPv4.GatewayIP.AsSlice(),
			),
			layers.NewDHCPOption(
				layers.DHCPOptLeaseTime,
				binary.BigEndian.AppendUint32(nil, uint32(leaseTTL.Seconds())),
			),
		},
	}, {
		name: "existing_static",
		in:   newDHCPDISCOVER(t, hwAddrStatic),
		want: layers.DHCPOptions{
			layers.NewDHCPOption(
				layers.DHCPOptMessageType,
				[]byte{byte(layers.DHCPMsgTypeOffer)},
			),
			layers.NewDHCPOption(
				layers.DHCPOptServerID,
				ifacesConfig[ifaceName].IPv4.GatewayIP.AsSlice(),
			),
			layers.NewDHCPOption(
				layers.DHCPOptLeaseTime,
				binary.BigEndian.AppendUint32(nil, uint32(leaseTTL.Seconds())),
			),
			layers.NewDHCPOption(
				layers.DHCPOptHostname,
				[]byte(leaseHostnameStatic),
			),
		},
	}, {
		name: "existing_dynamic",
		in:   newDHCPDISCOVER(t, hwAddrDynamic),
		want: layers.DHCPOptions{
			layers.NewDHCPOption(
				layers.DHCPOptMessageType,
				[]byte{byte(layers.DHCPMsgTypeOffer)},
			),
			layers.NewDHCPOption(
				layers.DHCPOptServerID,
				ifacesConfig[ifaceName].IPv4.GatewayIP.AsSlice(),
			),
			layers.NewDHCPOption(
				layers.DHCPOptLeaseTime,
				binary.BigEndian.AppendUint32(nil, uint32((dynamicLeaseExpiry).Seconds())),
			),
			layers.NewDHCPOption(
				layers.DHCPOptHostname,
				[]byte(leaseHostnameDynamic),
			),
		},
	}, {
		name: "existing_dynamic_expired",
		in:   newDHCPDISCOVER(t, hwAddrExpired),
		want: layers.DHCPOptions{
			layers.NewDHCPOption(
				layers.DHCPOptMessageType,
				[]byte{byte(layers.DHCPMsgTypeOffer)},
			),
			layers.NewDHCPOption(
				layers.DHCPOptServerID,
				ifacesConfig[ifaceName].IPv4.GatewayIP.AsSlice(),
			),
			layers.NewDHCPOption(
				layers.DHCPOptLeaseTime,
				binary.BigEndian.AppendUint32(nil, uint32(leaseTTL.Seconds())),
			),
			layers.NewDHCPOption(
				layers.DHCPOptHostname,
				[]byte(leaseHostnameExpired),
			),
		},
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv4](t, tc.in.Layer(layers.LayerTypeDHCPv4))

		ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, ifaceName)

		dhcpConf := &dhcpsvc.Config{
			Interfaces:           ifacesConfig,
			NetworkDeviceManager: ndMgr,
			DBFilePath:           newTempDB(t),
			Enabled:              true,
		}
		srv := newTestDHCPServer(t, dhcpConf)

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			servicetest.RequireRun(t, srv, testTimeout)

			testutil.RequireSend(t, inCh, tc.in, testTimeout)

			resp, ok := testutil.RequireReceive(t, outCh, testTimeout)
			require.True(t, ok)

			var (
				eth    = &layers.Ethernet{}
				ip     = &layers.IPv4{}
				udp    = &layers.UDP{}
				dhcpv4 = &layers.DHCPv4{}
			)
			types := requireEthernet(t, resp, eth, ip, udp, dhcpv4)
			require.Equal(t, []gopacket.LayerType{
				eth.LayerType(),
				ip.LayerType(),
				udp.LayerType(),
				dhcpv4.LayerType(),
			}, types)

			assertDHCPv4Response(t, req, dhcpv4, tc.want)
		})
	}

	t.Run("new_from_expired", func(t *testing.T) {
		t.Parallel()

		pkt := newDHCPDISCOVER(t, hwAddrUnknown)
		req := testutil.RequireTypeAssert[*layers.DHCPv4](t, pkt.Layer(layers.LayerTypeDHCPv4))

		ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, ifaceName)

		narrowIPv4Conf := &dhcpsvc.IPv4Config{
			Clock:         testClock,
			SubnetMask:    netip.MustParseAddr("255.255.255.0"),
			GatewayIP:     netip.MustParseAddr("192.168.0.1"),
			RangeStart:    netip.MustParseAddr("192.168.0.100"),
			RangeEnd:      netip.MustParseAddr("192.168.0.100"),
			LeaseDuration: leaseTTL,
			Enabled:       true,
		}
		narrowIfacesConfig := map[string]*dhcpsvc.InterfaceConfig{
			ifaceName: {
				IPv4: narrowIPv4Conf,
				IPv6: &dhcpsvc.IPv6Config{Enabled: false},
			},
		}

		dhcpConf := &dhcpsvc.Config{
			Interfaces:           narrowIfacesConfig,
			NetworkDeviceManager: ndMgr,
			DBFilePath:           newTempDB(t),
			Enabled:              true,
		}
		srv := newTestDHCPServer(t, dhcpConf)

		servicetest.RequireRun(t, srv, testTimeout)

		testutil.RequireSend(t, inCh, pkt, testTimeout)

		resp, ok := testutil.RequireReceive(t, outCh, testTimeout)
		require.True(t, ok)

		var (
			eth    = &layers.Ethernet{}
			ip     = &layers.IPv4{}
			udp    = &layers.UDP{}
			dhcpv4 = &layers.DHCPv4{}
		)
		types := requireEthernet(t, resp, eth, ip, udp, dhcpv4)
		require.Equal(t, []gopacket.LayerType{
			eth.LayerType(),
			ip.LayerType(),
			udp.LayerType(),
			dhcpv4.LayerType(),
		}, types)

		assertDHCPv4Response(t, req, dhcpv4, layers.DHCPOptions{
			layers.NewDHCPOption(
				layers.DHCPOptMessageType,
				[]byte{byte(layers.DHCPMsgTypeOffer)},
			),
			layers.NewDHCPOption(
				layers.DHCPOptServerID,
				narrowIfacesConfig[ifaceName].IPv4.GatewayIP.AsSlice(),
			),
			layers.NewDHCPOption(
				layers.DHCPOptLeaseTime,
				binary.BigEndian.AppendUint32(nil, uint32(leaseTTL.Seconds())),
			),
		})
	})
}

// newTestNetworkDeviceManager creates a network device manager for testing.  It
// requires that device opened have a deviceName.  The device itself has a link
// type [layers.LinkTypeEthernet].  Incoming packets are received from inCh and
// outgoing packets are sent to outCh.
func newTestNetworkDeviceManager(
	tb testing.TB,
	deviceName string,
) (ndMgr dhcpsvc.NetworkDeviceManager, inCh chan gopacket.Packet, outCh chan []byte) {
	tb.Helper()

	inCh = make(chan gopacket.Packet)
	outCh = make(chan []byte)

	pt := testutil.PanicT{}

	dev := &testNetworkDevice{
		onReadPacketData: func() (data []byte, ci gopacket.CaptureInfo, err error) {
			pkt, ok := testutil.RequireReceive(pt, inCh, testTimeout)
			require.True(pt, ok)

			data = pkt.Data()
			ci = gopacket.CaptureInfo{
				Length:        len(data),
				CaptureLength: len(data),
			}

			return data, ci, nil
		},
		onLinkType: func() (lt layers.LinkType) {
			return layers.LinkTypeEthernet
		},
		onWritePacketData: func(data []byte) (err error) {
			testutil.RequireSend(pt, outCh, data, testTimeout)

			return nil
		},
	}

	ndMgr = &testNetworkDeviceManager{
		onOpen: func(
			_ context.Context,
			conf *dhcpsvc.NetworkDeviceConfig,
		) (nd dhcpsvc.NetworkDevice, err error) {
			require.Equal(pt, deviceName, conf.Name)

			return dev, nil
		},
	}

	return ndMgr, inCh, outCh
}

// newDHCPDISCOVER creates a new DHCPDISCOVER packet for testing.
//
// TODO(e.burkov):  Add parameters.
func newDHCPDISCOVER(tb testing.TB, clientHWAddr net.HardwareAddr) (pkt gopacket.Packet) {
	tb.Helper()

	eth := &layers.Ethernet{
		SrcMAC:       clientHWAddr,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version:  4,
		TTL:      dhcpsvc.IPv4DefaultTTL,
		SrcIP:    net.IPv4zero.To4(),
		DstIP:    net.IPv4bcast.To4(),
		Protocol: layers.IPProtocolUDP,
	}
	udp := &layers.UDP{
		SrcPort: dhcpsvc.ClientPortV4,
		DstPort: dhcpsvc.ServerPortV4,
	}
	_ = udp.SetNetworkLayerForChecksum(ip)

	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  dhcpsvc.EUI48AddrLen,
		Xid:          testXid,
		ClientHWAddr: clientHWAddr,
		Options: layers.DHCPOptions{
			layers.NewDHCPOption(layers.DHCPOptMessageType, []byte{
				byte(layers.DHCPMsgTypeDiscover),
			}),
		},
	}

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newTestPacket creates a valid packet from ls using first as first layer
// decoder.
func newTestPacket(
	tb testing.TB,
	first gopacket.Decoder,
	ls ...gopacket.SerializableLayer,
) (pkg gopacket.Packet) {
	tb.Helper()

	buf := gopacket.NewSerializeBuffer()

	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	err := gopacket.SerializeLayers(buf, opts, ls...)
	require.NoError(tb, err)

	return gopacket.NewPacket(buf.Bytes(), first, gopacket.Default)
}

// requireEthernet requires data to contain an Ethernet layer and all layers
// from ls.  First of ls must be of type [layers.LayerTypeEthernet].
func requireEthernet(
	tb testing.TB,
	data []byte,
	ls ...gopacket.DecodingLayer,
) (types []gopacket.LayerType) {
	tb.Helper()

	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, ls...)

	err := parser.DecodeLayers(data, &types)
	require.NoError(tb, err)

	return types
}

// assertDHCPv4Response asserts that the DHCPv4 response matches the expected
// values.
func assertDHCPv4Response(tb testing.TB, req, resp *layers.DHCPv4, wantOpts layers.DHCPOptions) {
	tb.Helper()

	assert.Equal(tb, layers.DHCPOpReply, resp.Operation, "operation")
	assert.Equal(tb, req.HardwareType, resp.HardwareType, "hardware type")
	assert.Equal(tb, req.HardwareLen, resp.HardwareLen, "hardware length")
	assert.Equal(tb, req.Xid, resp.Xid, "xid")
	assert.Equal(tb, req.ClientHWAddr, resp.ClientHWAddr, "client hardware address")
	assert.Equal(tb, wantOpts, resp.Options, "options")
}
