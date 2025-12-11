package dhcpsvc_test

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/servicetest"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/require"
)

func TestDHCPServer_ServeEther4_discover(t *testing.T) {
	t.Parallel()

	ipv4Conf := &dhcpsvc.IPv4Config{
		Clock:         timeutil.SystemClock{},
		SubnetMask:    netip.MustParseAddr("255.255.255.0"),
		GatewayIP:     netip.MustParseAddr("192.168.0.1"),
		RangeStart:    netip.MustParseAddr("192.168.0.100"),
		RangeEnd:      netip.MustParseAddr("192.168.0.200"),
		LeaseDuration: 24 * time.Hour,
		Enabled:       true,
	}
	ifacesConfig := map[string]*dhcpsvc.InterfaceConfig{
		"iface": {
			IPv4: ipv4Conf,
			IPv6: &dhcpsvc.IPv6Config{Enabled: false},
		},
	}

	// TODO(e.burkov):  !! add cases for known lease and wrong packets.
	testCases := []struct {
		name string
		in   gopacket.Packet
		want []byte
	}{{
		name: "new",
		in:   newDHCPDISCOVER(t),
		want: nil,
	}}

	for _, tc := range testCases {
		ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, "iface")

		dhcpConf := &dhcpsvc.Config{
			Interfaces:           ifacesConfig,
			NetworkDeviceManager: ndMgr,
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
			requireEthernet(t, resp, eth, ip, udp, dhcpv4)

			// TODO(e.burkov):  !! assert layers
		})
	}
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
// TODO(e.burkov):  !! add parameters.
func newDHCPDISCOVER(tb testing.TB) (pkt gopacket.Packet) {
	tb.Helper()

	clientHWAddr := net.HardwareAddr{0x0, 0x1, 0x2, 0x3, 0x4, 0x5}

	etherLayer := &layers.Ethernet{
		SrcMAC:       clientHWAddr,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ipLayer := &layers.IPv4{
		Version:  4,
		TTL:      dhcpsvc.IPv4DefaultTTL,
		SrcIP:    net.IPv4zero.To4(),
		DstIP:    net.IPv4bcast.To4(),
		Protocol: layers.IPProtocolUDP,
	}
	udpLayer := &layers.UDP{
		SrcPort: dhcpsvc.ClientPort,
		DstPort: dhcpsvc.ServerPort,
	}
	_ = udpLayer.SetNetworkLayerForChecksum(ipLayer)

	dhcpLayer := &layers.DHCPv4{
		Operation:    layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  dhcpsvc.EUI48AddrLen,
		Xid:          1,
		ClientHWAddr: clientHWAddr,
		Options: layers.DHCPOptions{
			layers.NewDHCPOption(layers.DHCPOptMessageType, []byte{
				byte(layers.DHCPMsgTypeDiscover),
			}),
		},
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	err := gopacket.SerializeLayers(
		buf,
		opts,
		etherLayer,
		ipLayer,
		udpLayer,
		dhcpLayer,
	)
	require.NoError(tb, err)

	return gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
}

// requireEthernet requires data to contain an Ethernet layer and all layers
// from ls.
func requireEthernet(tb testing.TB, data []byte, ls ...gopacket.DecodingLayer) {
	tb.Helper()

	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, ls...)

	layerTypes := make([]gopacket.LayerType, 0, len(ls))

	err := parser.DecodeLayers(data, &layerTypes)
	require.NoError(tb, err)
}
