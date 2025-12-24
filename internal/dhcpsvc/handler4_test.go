package dhcpsvc_test

import (
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/servicetest"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDHCPServer_ServeEther4_discover(t *testing.T) {
	t.Parallel()

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

	ifaceAddr := netip.MustParseAddr("192.168.0.2")

	// NOTE: Keep in sync with testdata.
	dynamicLeaseExpiry := time.Date(2025, 1, 1, 10, 1, 1, 0, time.UTC)
	dynamicLeaseTTL := dynamicLeaseExpiry.Sub(testCurrentTime)

	ipv4Conf := &dhcpsvc.IPv4Config{
		Clock:         testClock,
		SubnetMask:    netip.MustParseAddr("255.255.255.0"),
		GatewayIP:     netip.MustParseAddr("192.168.0.1"),
		RangeStart:    netip.MustParseAddr("192.168.0.100"),
		RangeEnd:      netip.MustParseAddr("192.168.0.200"),
		LeaseDuration: testLeaseTTL,
		Enabled:       true,
	}
	ifacesConfig := map[string]*dhcpsvc.InterfaceConfig{
		testIfaceName: {IPv4: ipv4Conf, IPv6: disabledIPv6Config},
	}

	testCases := []struct {
		name     string
		in       gopacket.Packet
		wantOpts layers.DHCPOptions
	}{{
		name: "new",
		in:   newDHCPDISCOVER(t, hwAddrUnknown),
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeOffer),
			newOptServerID(t, ifaceAddr),
			newOptLeaseTime(t, testLeaseTTL),
		},
	}, {
		name: "existing_static",
		in:   newDHCPDISCOVER(t, hwAddrStatic),
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeOffer),
			newOptServerID(t, ifaceAddr),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, leaseHostnameStatic),
		},
	}, {
		name: "existing_dynamic",
		in:   newDHCPDISCOVER(t, hwAddrDynamic),
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeOffer),
			newOptServerID(t, ifaceAddr),
			newOptLeaseTime(t, dynamicLeaseTTL),
			newOptHostname(t, leaseHostnameDynamic),
		},
	}, {
		name: "existing_dynamic_expired",
		in:   newDHCPDISCOVER(t, hwAddrExpired),
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeOffer),
			newOptServerID(t, ifaceAddr),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, leaseHostnameExpired),
		},
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv4](t, tc.in.Layer(layers.LayerTypeDHCPv4))

		ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceName, ifaceAddr)
		srv := newTestDHCPServer(t, &dhcpsvc.Config{
			Interfaces:           ifacesConfig,
			NetworkDeviceManager: ndMgr,
			DBFilePath:           newTempDB(t),
			Enabled:              true,
		})

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			servicetest.RequireRun(t, srv, testTimeout)

			testutil.RequireSend(t, inCh, tc.in, testTimeout)

			respData, ok := testutil.RequireReceive(t, outCh, testTimeout)
			require.True(t, ok)

			assertValidOffer(t, req, respData, tc.wantOpts)
		})
	}
}

func TestDHCPServer_ServeEther4_discoverExpired(t *testing.T) {
	t.Parallel()

	// hwAddrUnknown is the MAC address for an unknown client, not related to
	// any existing lease.
	//
	// NOTE: Keep in sync with testdata.
	hwAddrUnknown := net.HardwareAddr{0x0, 0x1, 0x2, 0x3, 0x4, 0x5}

	ifaceAddr := netip.MustParseAddr("192.168.0.2")

	pkt := newDHCPDISCOVER(t, hwAddrUnknown)
	req := testutil.RequireTypeAssert[*layers.DHCPv4](t, pkt.Layer(layers.LayerTypeDHCPv4))

	ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceName, ifaceAddr)

	ipv4Conf := &dhcpsvc.IPv4Config{
		Clock:         testClock,
		SubnetMask:    netip.MustParseAddr("255.255.255.0"),
		GatewayIP:     netip.MustParseAddr("192.168.0.1"),
		RangeStart:    netip.MustParseAddr("192.168.0.100"),
		RangeEnd:      netip.MustParseAddr("192.168.0.100"),
		LeaseDuration: testLeaseTTL,
		Enabled:       true,
	}
	srv := newTestDHCPServer(t, &dhcpsvc.Config{
		Interfaces: map[string]*dhcpsvc.InterfaceConfig{
			testIfaceName: {IPv4: ipv4Conf, IPv6: disabledIPv6Config},
		},
		NetworkDeviceManager: ndMgr,
		DBFilePath:           newTempDB(t),
		Enabled:              true,
	})
	servicetest.RequireRun(t, srv, testTimeout)

	testutil.RequireSend(t, inCh, pkt, testTimeout)

	respData, ok := testutil.RequireReceive(t, outCh, testTimeout)
	require.True(t, ok)

	assertValidOffer(t, req, respData, layers.DHCPOptions{
		newOptMessageType(t, layers.DHCPMsgTypeOffer),
		newOptServerID(t, ifaceAddr),
		newOptLeaseTime(t, testLeaseTTL),
	})
}

func TestDHCPServer_ServeEther4_release(t *testing.T) {
	t.Parallel()

	// NOTE: Keep in sync with testdata.
	leaseExpiry := time.Date(2025, 1, 1, 10, 1, 1, 0, time.UTC)

	// NOTE: Keep in sync with testdata.
	var (
		// hwAddrSuccess is the MAC address for a lease to be released
		// successfully.
		hwAddrSuccess = net.HardwareAddr{0x02, 0x03, 0x04, 0x05, 0x06, 0x07}

		// ipSuccess matches the lease IP.
		ipSuccess = netip.MustParseAddr("192.168.0.102")

		// ipMismatch is the IP of the lease used in the mismatch cases.
		ipMismatch = netip.MustParseAddr("192.168.0.103")

		// hwAddrMismatch is the MAC address for a lease with mismatched IP.
		hwAddrMismatch = net.HardwareAddr{0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

		// ipMismatchReq is the IP requested for release, which differs from the
		// lease IP.
		ipMismatchReq = netip.MustParseAddr("192.168.0.104")

		// hwAddrUnknown is an unknown MAC.
		hwAddrUnknown = net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	)

	ipv4Conf := &dhcpsvc.IPv4Config{
		Clock:         testClock,
		SubnetMask:    netip.MustParseAddr("255.255.255.0"),
		GatewayIP:     netip.MustParseAddr("192.168.0.1"),
		RangeStart:    netip.MustParseAddr("192.168.0.100"),
		RangeEnd:      netip.MustParseAddr("192.168.0.200"),
		LeaseDuration: testLeaseTTL,
		Enabled:       true,
	}
	ifacesConfig := map[string]*dhcpsvc.InterfaceConfig{
		testIfaceName: {IPv4: ipv4Conf, IPv6: disabledIPv6Config},
	}

	ifaceHWAddr := net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	ifaceAddr := netip.MustParseAddr("192.168.0.2")
	anotherSubnetAddr := netip.MustParseAddr("10.0.0.1")

	var (
		leaseSuccess = &dhcpsvc.Lease{
			Expiry:   leaseExpiry,
			IP:       ipSuccess,
			Hostname: "success",
			HWAddr:   hwAddrSuccess,
			IsStatic: false,
		}
		leaseMismatch = &dhcpsvc.Lease{
			Expiry:   leaseExpiry,
			IP:       ipMismatch,
			Hostname: "mismatch",
			HWAddr:   hwAddrMismatch,
			IsStatic: false,
		}
	)

	testCases := []struct {
		name       string
		req        gopacket.Packet
		wantLeases []*dhcpsvc.Lease
	}{{
		name: "success",
		req:  newDHCPRELEASE(t, hwAddrSuccess, ipSuccess, ifaceHWAddr, ifaceAddr),
		wantLeases: []*dhcpsvc.Lease{
			leaseMismatch,
		},
	}, {
		name: "not_found",
		req:  newDHCPRELEASE(t, hwAddrUnknown, ipSuccess, ifaceHWAddr, ifaceAddr),
		wantLeases: []*dhcpsvc.Lease{
			leaseSuccess,
			leaseMismatch,
		},
	}, {
		name: "mismatch_ip",
		req:  newDHCPRELEASE(t, hwAddrMismatch, ipMismatchReq, ifaceHWAddr, ifaceAddr),
		wantLeases: []*dhcpsvc.Lease{
			leaseSuccess,
			leaseMismatch,
		},
	}, {
		name: "bad_subnet",
		req:  newDHCPRELEASE(t, hwAddrSuccess, anotherSubnetAddr, ifaceHWAddr, ifaceAddr),
		wantLeases: []*dhcpsvc.Lease{
			leaseSuccess,
			leaseMismatch,
		},
	}}

	for _, tc := range testCases {
		ndMgr, inCh, _ := newTestNetworkDeviceManager(t, testIfaceName, ifaceAddr)
		srv := newTestDHCPServer(t, &dhcpsvc.Config{
			Interfaces:           ifacesConfig,
			NetworkDeviceManager: ndMgr,
			DBFilePath:           newTempDB(t),
			Enabled:              true,
		})

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			servicetest.RequireRun(t, srv, testTimeout)

			testutil.RequireSend(t, inCh, tc.req, testTimeout)

			// TODO(e.burkov):  Improve the test to ensure that the DHCPDISCOVER
			// actually receives the released address.
			assert.EventuallyWithT(t, func(ct *assert.CollectT) {
				assert.Equal(ct, tc.wantLeases, srv.Leases())
			}, testTimeout/2, testTimeout/20)
		})
	}
}

// TODO(e.burkov):  Add tests for DHCPREQUEST, DHCPDECLINE.

// TODO(e.burkov):  Add tests for wrong packets.

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
			newOptMessageType(tb, layers.DHCPMsgTypeDiscover),
		},
	}

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newDHCPRELEASE creates a new DHCPRELEASE packet for testing.
func newDHCPRELEASE(
	tb testing.TB,
	clientHWAddr net.HardwareAddr,
	clientIP netip.Addr,
	serverHWAddr net.HardwareAddr,
	serverIP netip.Addr,
) (pkt gopacket.Packet) {
	tb.Helper()

	eth := &layers.Ethernet{
		SrcMAC:       clientHWAddr,
		DstMAC:       serverHWAddr,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version:  4,
		TTL:      dhcpsvc.IPv4DefaultTTL,
		SrcIP:    clientIP.AsSlice(),
		DstIP:    serverIP.AsSlice(),
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
		ClientIP:     clientIP.AsSlice(),
		Options: layers.DHCPOptions{
			newOptMessageType(tb, layers.DHCPMsgTypeRelease),
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

// assertValidOffer asserts that respData contains a complete DHCPOFFER response
// with the expected options, wrapped with all layers down to Ethernet.
func assertValidOffer(
	tb testing.TB,
	discover *layers.DHCPv4,
	respData []byte,
	wantOpts layers.DHCPOptions,
) {
	tb.Helper()

	resp := &layers.DHCPv4{}
	types := requireEthernet(tb, respData, &layers.Ethernet{}, &layers.IPv4{}, &layers.UDP{}, resp)
	require.Equal(tb, fullLayersStack, types)

	assert.Equal(tb, layers.DHCPOpReply, resp.Operation, "operation")
	assert.Equal(tb, discover.HardwareType, resp.HardwareType, "hardware type")
	assert.Equal(tb, discover.HardwareLen, resp.HardwareLen, "hardware length")
	assert.Equal(tb, discover.Xid, resp.Xid, "xid")
	assert.Equal(tb, discover.ClientHWAddr, resp.ClientHWAddr, "client hardware address")
	assert.Equal(tb, wantOpts, resp.Options, "options")
}
