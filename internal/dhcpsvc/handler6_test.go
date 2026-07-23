package dhcpsvc_test

import (
	"context"
	"net"
	"net/netip"
	"slices"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(e.burkov):  Add tests for wrong packets.

// testIPv6InterfacesConf is the test interfaces configuration for the DHCPv6
// part of the [DHCPServer].
var testIPv6InterfacesConf = map[string]*dhcpsvc.InterfaceConfig{
	testIfaceName: {
		IPv4: disabledIPv4Conf,
		IPv6: testIPv6Conf,
	},
}

// testIAID is a common IAID for IANA options in tests.
const testIAID = 1

// testTransactionID is a sample transaction ID for testing.
//
// TODO(e.burkov):  Generate unique IDs when they will be actually used.
var testTransactionID = []byte{0x01, 0x02, 0x03}

func TestDHCPServer_ServeEther6_solicit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		in       gopacket.Packet
		name     string
		wantOpts layers.DHCPv6Options
	}{{
		in:   newDHCPv6SOLICIT(t, testHWUnknown, testIPv6Unknown, false),
		name: "new",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWUnknown),
			newOptIANA(t, testIAID, testIPv6Conf.RangeStart),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
		},
	}, {
		in:   newDHCPv6SOLICIT(t, testHWStatic, testIPv6Static, false),
		name: "existing_static",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWStatic),
			newOptIANA(t, testIAID, testIPv6Static),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
		},
	}, {
		in:   newDHCPv6SOLICIT(t, testHWDynamic, testIPv6Dynamic, false),
		name: "existing_dynamic",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWDynamic),
			newOptIANA(t, testIAID, testIPv6Dynamic),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
		},
	}, {
		in:   newDHCPv6SOLICIT(t, testHWExpired, testIPv6Expired, false),
		name: "existing_expired",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWExpired),
			newOptIANA(t, testIAID, testIPv6Expired),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
		},
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDatabase(t, testLeases)

			ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceAddrV6)
			startTestDHCPServer(t, &dhcpsvc.Config{
				Database:             db,
				Interfaces:           testIPv6InterfacesConf,
				Logger:               testLogger,
				NetworkDeviceManager: ndMgr,
				Enabled:              true,
			})

			testutil.RequireSend(t, inCh, tc.in, testTimeout)

			assertValidResponse6(t, req, outCh, tc.wantOpts)
		})
	}
}

func TestDHCPServer_ServeEther6_solicitRapidCommit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		in       gopacket.Packet
		want     *dhcpsvc.Lease
		name     string
		wantOpts layers.DHCPv6Options
	}{{
		in: newDHCPv6SOLICIT(t, testHWUnknown, testIPv6Unknown, true),
		want: &dhcpsvc.Lease{
			IP:       testIPv6Conf.RangeStart,
			Expiry:   testExpiryDynamicLease,
			Hostname: aghnet.GenerateHostname(testIPv6Conf.RangeStart),
			HWAddr:   testHWUnknown,
			IsStatic: false,
		},
		name: "new",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWUnknown),
			newOptIANA(t, testIAID, testIPv6Conf.RangeStart),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
			layers.NewDHCPv6Option(layers.DHCPv6OptRapidCommit, []byte{}),
		},
	}, {
		in:   newDHCPv6SOLICIT(t, testHWStatic, testIPv6Static, true),
		want: testLease6Static,
		name: "existing",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWStatic),
			newOptIANA(t, testIAID, testIPv6Static),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
			layers.NewDHCPv6Option(layers.DHCPv6OptRapidCommit, []byte{}),
		},
	}, {
		in:   newDHCPv6SOLICIT(t, testHWDynamic, testIPv6Dynamic, true),
		want: testLease6Dynamic,
		name: "existing_dynamic",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWDynamic),
			newOptIANA(t, testIAID, testIPv6Dynamic),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
			layers.NewDHCPv6Option(layers.DHCPv6OptRapidCommit, []byte{}),
		},
	}, {
		in: newDHCPv6SOLICIT(t, testHWExpired, testIPv6Expired, true),
		want: &dhcpsvc.Lease{
			IP:       testIPv6Expired,
			Expiry:   testExpiryDynamicLease,
			Hostname: testLease6HostnameExpired,
			HWAddr:   testHWExpired,
			IsStatic: false,
		},
		name: "existing_expired",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWExpired),
			newOptIANA(t, testIAID, testIPv6Expired),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
			layers.NewDHCPv6Option(layers.DHCPv6OptRapidCommit, []byte{}),
		},
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDatabase(t, testLeases)

			onStore := func(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
				assert.Contains(t, leases, tc.want)

				return nil
			}

			if tc.want != nil {
				db.onStore = onStore
			}

			ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceAddrV6)
			startTestDHCPServer(t, &dhcpsvc.Config{
				Database:             db,
				Interfaces:           testIPv6InterfacesConf,
				Logger:               testLogger,
				NetworkDeviceManager: ndMgr,
				Enabled:              true,
			})

			testutil.RequireSend(t, inCh, tc.in, testTimeout)

			assertValidResponse6(t, req, outCh, tc.wantOpts)
		})
	}
}

// TODO(e.burkov):  Add tests for REQUEST causing errors.  This would require a
// custom implementation of the address checker at least.
func TestDHCPServer_ServeEther6_request(t *testing.T) {
	t.Parallel()

	notOnLinkAddr := netip.MustParseAddr(testAnotherRangeStartV6Str)

	testCases := []struct {
		in       gopacket.Packet
		want     *dhcpsvc.Lease
		name     string
		wantOpts layers.DHCPv6Options
	}{{
		in: newDHCPv6REQUEST(t, testHWUnknown, testIPv6Unknown),
		want: &dhcpsvc.Lease{
			IP:       testIPv6Conf.RangeStart,
			Expiry:   testExpiryDynamicLease,
			Hostname: aghnet.GenerateHostname(testIPv6Conf.RangeStart),
			HWAddr:   testHWUnknown,
			IsStatic: false,
		},
		name: "success",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWUnknown),
			newOptIANA(t, testIAID, testIPv6Conf.RangeStart),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
		},
	}, {
		in:   newDHCPv6REQUEST(t, testHWUnknown, notOnLinkAddr),
		want: nil,
		name: "not_on_link",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWUnknown),
			newOptIANAStatus(t, testIAID, layers.DHCPv6StatusCodeNotOnLink),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
		},
	}, {
		in:   newDHCPv6REQUEST(t, testHWStatic, testIPv6Static),
		want: testLease6Static,
		name: "existing_static",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWStatic),
			newOptIANA(t, testIAID, testIPv6Static),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
		},
	}, {
		in:   newDHCPv6REQUEST(t, testHWUnknown, netip.Addr{}),
		want: nil,
		name: "no_iana",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWUnknown),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
		},
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDatabase(t, testLeases)

			onStore := func(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
				assert.Contains(t, leases, tc.want)

				return nil
			}

			if tc.want != nil {
				db.onStore = onStore
			}

			ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceAddrV6)
			startTestDHCPServer(t, &dhcpsvc.Config{
				Database:             db,
				Interfaces:           testIPv6InterfacesConf,
				Logger:               testLogger,
				NetworkDeviceManager: ndMgr,
				Enabled:              true,
			})

			testutil.RequireSend(t, inCh, tc.in, testTimeout)

			assertValidResponse6(t, req, outCh, tc.wantOpts)
		})
	}
}

func TestDHCPServer_ServeEther6_requestWithSolicit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		in       gopacket.Packet
		solicit  gopacket.Packet
		want     *dhcpsvc.Lease
		name     string
		wantOpts layers.DHCPv6Options
	}{{
		in:      newDHCPv6REQUEST(t, testHWUnknown, testIPv6Unknown),
		solicit: newDHCPv6SOLICIT(t, testHWUnknown, testIPv6Unknown, false),
		want: &dhcpsvc.Lease{
			IP:       testIPv6Conf.RangeStart,
			Expiry:   testExpiryDynamicLease,
			Hostname: aghnet.GenerateHostname(testIPv6Conf.RangeStart),
			HWAddr:   testHWUnknown,
			IsStatic: false,
		},
		name: "preceding_solicit",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWUnknown),
			newOptIANA(t, testIAID, testIPv6Conf.RangeStart),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
		},
	}, {
		in:      newDHCPv6REQUEST(t, testHWUnknown, testIPv6Unknown),
		solicit: newDHCPv6SOLICIT(t, testHWUnknown, testIPv6Unknown, true),
		want: &dhcpsvc.Lease{
			IP:       testIPv6Conf.RangeStart,
			Expiry:   testExpiryDynamicLease,
			Hostname: aghnet.GenerateHostname(testIPv6Conf.RangeStart),
			HWAddr:   testHWUnknown,
			IsStatic: false,
		},
		name: "preceding_solicit_rapid_commit",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
			newOptClientDUID(t, testHWUnknown),
			newOptIANA(t, testIAID, testIPv6Conf.RangeStart),
			newOptPreference(t, 0),
			newOptSolMaxRT(t, dhcpsvc.DefaultSolMaxRT),
		},
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDatabase(t, testLeases)

			onStore := func(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
				assert.Contains(t, leases, tc.want)

				return nil
			}

			if tc.want != nil {
				db.onStore = onStore
			}

			ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceAddrV6)
			startTestDHCPServer(t, &dhcpsvc.Config{
				Database:             db,
				Interfaces:           testIPv6InterfacesConf,
				Logger:               testLogger,
				NetworkDeviceManager: ndMgr,
				Enabled:              true,
			})

			testutil.RequireSend(t, inCh, tc.solicit, testTimeout)

			_, ok := testutil.RequireReceive(t, outCh, testTimeout)
			require.True(t, ok)

			testutil.RequireSend(t, inCh, tc.in, testTimeout)

			assertValidResponse6(t, req, outCh, tc.wantOpts)
		})
	}
}

// newDHCPv6SOLICIT creates a new DHCPv6 SOLICIT packet for testing.
func newDHCPv6SOLICIT(
	tb testing.TB,
	hwAddr net.HardwareAddr,
	reqIP netip.Addr,
	rapidCommit bool,
) (pkt gopacket.Packet) {
	tb.Helper()

	eth := newEthernetLayer(tb, hwAddr, nil, layers.EthernetTypeIPv6)

	ip, udp := newIPv6UDPLayer(tb, netip.AddrPort{}, netip.AddrPort{})

	dhcp := &layers.DHCPv6{
		MsgType:  layers.DHCPv6MsgTypeSolicit,
		HopCount: 0,
		// Don't specify link and peer addresses, as they are intended for relay
		// messages.
		LinkAddr:      nil,
		PeerAddr:      nil,
		TransactionID: testTransactionID,
		Options: layers.DHCPv6Options{
			newOptClientDUID(tb, hwAddr),
		},
	}

	if reqIP.IsValid() && reqIP.Is6() {
		dhcp.Options = append(dhcp.Options, newOptIANA(tb, testIAID, reqIP))
	}

	if rapidCommit {
		o := layers.NewDHCPv6Option(layers.DHCPv6OptRapidCommit, nil)
		dhcp.Options = append(dhcp.Options, o)
	}

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newDHCPv6REQUEST creates a new DHCPv6 REQUEST packet for testing.
func newDHCPv6REQUEST(tb testing.TB, mac net.HardwareAddr, reqIP netip.Addr) (pkt gopacket.Packet) {
	tb.Helper()

	eth := newEthernetLayer(tb, mac, testIfaceHWAddr, layers.EthernetTypeIPv6)
	ip, udp := newIPv6UDPLayer(tb, netip.AddrPort{}, netip.AddrPort{})

	dhcp := &layers.DHCPv6{
		MsgType:  layers.DHCPv6MsgTypeRequest,
		HopCount: 0,
		// Don't specify link and peer addresses, as they are intended for relay
		// messages.
		LinkAddr:      nil,
		PeerAddr:      nil,
		TransactionID: testTransactionID,
		Options: layers.DHCPv6Options{
			newOptClientDUID(tb, mac),
			newOptServerDUID(tb, testIfaceHWAddr),
		},
	}

	if reqIP.IsValid() && reqIP.Is6() {
		dhcp.Options = append(dhcp.Options, newOptIANA(tb, testIAID, reqIP))
	}

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newIPv6UDPLayer creates IPv6 and UDP layers for testing.  Invalid src is
// replaced with an unspecified address and client DHCPv6 port, invalid dst is
// replaced with the broadcast address and server DHCPv6 port.
func newIPv6UDPLayer(tb testing.TB, src, dst netip.AddrPort) (ip *layers.IPv6, udp *layers.UDP) {
	tb.Helper()

	if !src.IsValid() {
		src = netip.AddrPortFrom(netip.IPv6Unspecified(), uint16(dhcpsvc.ClientPortV6))
	}

	if !dst.IsValid() {
		bcastAddr, ok := netip.AddrFromSlice(net.IPv6linklocalallnodes)
		require.True(tb, ok)

		dst = netip.AddrPortFrom(bcastAddr, uint16(dhcpsvc.ServerPortV6))
	}

	ip = &layers.IPv6{
		Version:    6,
		HopLimit:   dhcpsvc.IPv6DefaultHopLimit,
		SrcIP:      src.Addr().AsSlice(),
		DstIP:      dst.Addr().AsSlice(),
		NextHeader: layers.IPProtocolUDP,
	}
	udp = &layers.UDP{
		SrcPort: layers.UDPPort(src.Port()),
		DstPort: layers.UDPPort(dst.Port()),
	}
	require.NoError(tb, udp.SetNetworkLayerForChecksum(ip))

	return ip, udp
}

// newEthernetLayer creates a new Ethernet layer for IP packets of the specified
// type.  Nil src is replaced with an unspecified MAC address, nil dst is
// replaced with a broadcast MAC address, typ must be [layers.EthernetTypeIPv4]
// or [layers.EthernetTypeIPv6].
func newEthernetLayer(
	tb testing.TB,
	src net.HardwareAddr,
	dst net.HardwareAddr,
	typ layers.EthernetType,
) (eth *layers.Ethernet) {
	tb.Helper()

	if src == nil {
		src = net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	}
	if dst == nil {
		dst = net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	}

	return &layers.Ethernet{
		SrcMAC:       src,
		DstMAC:       dst,
		EthernetType: typ,
	}
}

// assertValidResponse6 asserts that the response received on recvCh is a valid
// DHCPv6 response for the given request and contains the expected options.  It
// does nothing if wantOpts is nil, which should be used in case no response is
// expected.  req and recvCh must not be nil.
func assertValidResponse6(
	tb testing.TB,
	req *layers.DHCPv6,
	recvCh <-chan []byte,
	wantOpts layers.DHCPv6Options,
) {
	tb.Helper()

	if wantOpts == nil {
		return
	}

	respData, ok := testutil.RequireReceive(tb, recvCh, testTimeout)
	require.True(tb, ok)

	ip := &layers.IPv6{}
	udp := &layers.UDP{}
	resp := &layers.DHCPv6{}
	types := requireEthernet(tb, respData, &layers.Ethernet{}, ip, udp, resp)
	require.Equal(tb, fullLayersStack6, types)

	assertValidDHCPv6(tb, req, resp)

	// TODO(e.burkov):  Consider comparing the whole message instead of separate
	// fields.
	assert.Equal(tb, req.LinkAddr, resp.LinkAddr, "link address")
	assert.Equal(tb, req.PeerAddr, resp.PeerAddr, "peer address")
	assert.Equal(tb, req.TransactionID, resp.TransactionID, "transaction id")
	assert.Equal(tb, wantOpts, resp.Options, "options")
}

// assertValidDHCPv6 asserts that the response is valid for the given request
// according to RFC 9915.
//
// TODO(e.burkov):  Add more checks involving other network layers.
func assertValidDHCPv6(
	tb testing.TB,
	req *layers.DHCPv6,
	resp *layers.DHCPv6,
) {
	tb.Helper()

	switch req.MsgType {
	case
		layers.DHCPv6MsgTypeRequest,
		layers.DHCPv6MsgTypeConfirm,
		layers.DHCPv6MsgTypeRenew,
		layers.DHCPv6MsgTypeRebind,
		layers.DHCPv6MsgTypeRelease,
		layers.DHCPv6MsgTypeDecline,
		layers.DHCPv6MsgTypeInformationRequest:
		assert.Equal(tb, layers.DHCPv6MsgTypeReply, resp.MsgType)
	case layers.DHCPv6MsgTypeSolicit:
		isRapidCommit := slices.ContainsFunc(resp.Options, func(o layers.DHCPv6Option) (ok bool) {
			return o.Code == layers.DHCPv6OptRapidCommit
		})

		if isRapidCommit {
			assert.Equal(tb, layers.DHCPv6MsgTypeReply, resp.MsgType)
		} else {
			assert.Equal(tb, layers.DHCPv6MsgTypeAdvertise, resp.MsgType)
		}
	default:
		tb.Errorf("request message type: %v: %s", errors.ErrUnexpectedValue, req.MsgType)
	}
}
