package dhcpsvc_test

import (
	"cmp"
	"context"
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(e.burkov):  Add tests for wrong packets.

// testIPv4InterfacesConf is the test interfaces configuration for the DHCPv4
// part of the [DHCPServer].
var testIPv4InterfacesConf = map[string]*dhcpsvc.InterfaceConfig{
	testIfaceName: {
		IPv4: testIPv4Conf,
		IPv6: disabledIPv6Conf,
	},
}

// testXid is a common transaction ID for DHCPv4 tests.
//
// TODO(e.burkov):  Generate unique IDs when they will be actually used.
const testXid = 1

func TestDHCPServer_ServeEther4_discover(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		in       gopacket.Packet
		wantOpts layers.DHCPOptions
	}{{
		name: "new",
		in:   newDHCPDISCOVER(t, testHWUnknown),
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeOffer),
			newOptServerID(t, testIfaceAddrV4),
			newOptLeaseTime(t, testLeaseTTL),
		},
	}, {
		name: "existing_static",
		in:   newDHCPDISCOVER(t, testHWStatic),
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeOffer),
			newOptServerID(t, testIfaceAddrV4),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, testLease4HostnameStatic),
		},
	}, {
		name: "existing_dynamic",
		in:   newDHCPDISCOVER(t, testHWDynamic),
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeOffer),
			newOptServerID(t, testIfaceAddrV4),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, testLease4HostnameDynamic),
		},
	}, {
		name: "existing_dynamic_expired",
		in:   newDHCPDISCOVER(t, testHWExpired),
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeOffer),
			newOptServerID(t, testIfaceAddrV4),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, testLease4HostnameExpired),
		},
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv4](t, tc.in.Layer(layers.LayerTypeDHCPv4))

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceAddrV4)
			startTestDHCPServer(t, &dhcpsvc.Config{
				Database:             newTestDatabase(t, testLeases),
				Interfaces:           testIPv4InterfacesConf,
				NetworkDeviceManager: ndMgr,
				Enabled:              true,
			})

			testutil.RequireSend(t, inCh, tc.in, testTimeout)

			assertValidResponse4(t, req, outCh, tc.wantOpts)
		})
	}
}

func TestDHCPServer_ServeEther4_discoverExpired(t *testing.T) {
	t.Parallel()

	pkt := newDHCPDISCOVER(t, testHWUnknown)
	req := testutil.RequireTypeAssert[*layers.DHCPv4](t, pkt.Layer(layers.LayerTypeDHCPv4))

	ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceAddrV4)

	startTestDHCPServer(t, &dhcpsvc.Config{
		Database:             newTestDatabase(t, testLeases),
		Interfaces:           testIPv4InterfacesConf,
		NetworkDeviceManager: ndMgr,
		Enabled:              true,
	})

	testutil.RequireSend(t, inCh, pkt, testTimeout)

	assertValidResponse4(t, req, outCh, layers.DHCPOptions{
		newOptMessageType(t, layers.DHCPMsgTypeOffer),
		newOptServerID(t, testIfaceAddrV4),
		newOptLeaseTime(t, testLeaseTTL),
	})
}

// TODO(e.burkov):  Test static case.
func TestDHCPServer_ServeEther4_release(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		req  gopacket.Packet
		want *dhcpsvc.Lease
		name string
	}{{
		req: newDHCPRELEASE(t, testHWDynamic, testIPv4Dynamic),
		want: &dhcpsvc.Lease{
			IP:       testIPv4Dynamic,
			HWAddr:   testHWDynamic,
			Expiry:   testExpiryDynamicLease,
			Hostname: testLease4HostnameDynamic,
			IsStatic: false,
		},
		name: "success",
	}, {
		req:  newDHCPRELEASE(t, testHWUnknown, testIPv4Dynamic),
		want: nil,
		name: "not_found",
	}, {
		req:  newDHCPRELEASE(t, testHWDynamic, testIPv4Unknown),
		want: nil,
		name: "mismatch_ip",
	}, {
		req:  newDHCPRELEASE(t, testHWDynamic, testIPv4OtherSubnet),
		want: nil,
		name: "bad_subnet",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDatabase(t, testLeases)

			onStore := func(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
				assert.NotContains(testutil.NewPanicT(t), leases, tc.want)

				return nil
			}

			if tc.want != nil {
				db.onStore = onStore
			}

			ndMgr, inCh, _ := newTestNetworkDeviceManager(t, testIfaceAddrV4)
			startTestDHCPServer(t, &dhcpsvc.Config{
				Database:             db,
				Interfaces:           testIPv4InterfacesConf,
				NetworkDeviceManager: ndMgr,
				Enabled:              true,
			})

			testutil.RequireSend(t, inCh, tc.req, testTimeout)
		})
	}
}

func TestDHCPServer_ServeEther4_requestSelecting(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		discover gopacket.Packet
		request  gopacket.Packet
		want     *dhcpsvc.Lease
		name     string
		wantOpts layers.DHCPOptions
	}{{
		discover: newDHCPDISCOVER(t, testHWUnknown),
		request: newDHCPREQUEST(t, &dhcpRequestConfig{
			options: layers.DHCPOptions{
				newOptRequestIP(t, testIPv4Conf.RangeStart),
				newOptServerID(t, testIfaceAddrV4),
			},
			clientHWAddr: testHWUnknown,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		want: &dhcpsvc.Lease{
			IP:       testIPv4Conf.RangeStart,
			HWAddr:   testHWUnknown,
			Expiry:   testExpiryDynamicLease,
			Hostname: "",
			IsStatic: false,
		},
		name: "success",
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeAck),
			newOptServerID(t, testIfaceAddrV4),
			newOptLeaseTime(t, testLeaseTTL),
		},
	}, {
		discover: newDHCPDISCOVER(t, testHWStatic),
		request: newDHCPREQUEST(t, &dhcpRequestConfig{
			options: layers.DHCPOptions{
				newOptRequestIP(t, testIPv4Static),
				newOptServerID(t, testIPv4OtherSubnet),
			},
			clientHWAddr: testHWStatic,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		want:     nil,
		name:     "wrong_server_id",
		wantOpts: nil,
	}, {
		discover: newDHCPDISCOVER(t, testHWStatic),
		request: newDHCPREQUEST(t, &dhcpRequestConfig{
			options: layers.DHCPOptions{
				newOptRequestIP(t, testIPv4Conf.RangeEnd.Next()),
				newOptServerID(t, testIfaceAddrV4),
			},
			clientHWAddr: testHWStatic,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		want: nil,
		name: "wrong_ip",
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeNak),
			newOptServerID(t, testIfaceAddrV4),
		},
	}, {
		discover: newDHCPDISCOVER(t, testHWStatic),
		request: newDHCPREQUEST(t, &dhcpRequestConfig{
			options: layers.DHCPOptions{
				newOptRequestIP(t, testIPv4Static),
				newOptServerID(t, testIfaceAddrV4),
			},
			clientHWAddr: testHWStatic,
			clientIP:     testIPv4Static,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		want:     nil,
		name:     "nonzero_ciaddr",
		wantOpts: nil,
	}}

	for _, tc := range testCases {
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

			ndMgr, dev, inCh, outCh := newTestNetworkDeviceAndManager(t, testIfaceAddrV4)
			startTestDHCPServer(t, &dhcpsvc.Config{
				Database:             db,
				Interfaces:           testIPv4InterfacesConf,
				Logger:               testLogger,
				NetworkDeviceManager: ndMgr,
				Enabled:              true,
			})

			testutil.RequireSend(t, inCh, tc.discover, testTimeout)

			_, ok := testutil.RequireReceive(t, outCh, testTimeout)
			require.True(t, ok)

			if tc.wantOpts == nil {
				dev.onWritePacketData = unexpectedWritePacketData
			}

			testutil.RequireSend(t, inCh, tc.request, testTimeout)

			assertValidResponse4(t, dhcpv4FromPacket(t, tc.request), outCh, tc.wantOpts)
		})
	}
}

func TestDHCPServer_ServeEther4_requestSelectingNoLease(t *testing.T) {
	t.Parallel()

	request := newDHCPREQUEST(t, &dhcpRequestConfig{
		options: layers.DHCPOptions{
			newOptRequestIP(t, testIPv4Conf.RangeEnd.Next()),
			newOptServerID(t, testIfaceAddrV4),
		},
		clientHWAddr: testHWUnknown,
		flags:        dhcpsvc.FlagsBroadcast,
	})

	db := newTestDatabase(t, testLeases)

	ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceAddrV4)
	startTestDHCPServer(t, &dhcpsvc.Config{
		Database:             db,
		Interfaces:           testIPv4InterfacesConf,
		Logger:               testLogger,
		NetworkDeviceManager: ndMgr,
		Enabled:              true,
	})

	testutil.RequireSend(t, inCh, request, testTimeout)

	assertValidResponse4(t, dhcpv4FromPacket(t, request), outCh, layers.DHCPOptions{
		newOptMessageType(t, layers.DHCPMsgTypeNak),
		newOptServerID(t, testIfaceAddrV4),
	})
}

// TODO(e.burkov):  Test expired.
func TestDHCPServer_ServeEther4_requestInitReboot(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		req      gopacket.Packet
		want     *dhcpsvc.Lease
		name     string
		wantOpts layers.DHCPOptions
	}{{
		req: newDHCPREQUEST(t, &dhcpRequestConfig{
			options:      layers.DHCPOptions{newOptRequestIP(t, testIPv4Static)},
			clientHWAddr: testHWStatic,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		want: testLease4Static,
		name: "success",
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeAck),
			newOptServerID(t, testIfaceAddrV4),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, testLease4HostnameStatic),
		},
	}, {
		req: newDHCPREQUEST(t, &dhcpRequestConfig{
			options:      layers.DHCPOptions{newOptRequestIP(t, testIPv4OtherSubnet)},
			clientHWAddr: testHWStatic,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		want: nil,
		name: "wrong_subnet",
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeNak),
			newOptServerID(t, testIfaceAddrV4),
		},
	}, {
		req: newDHCPREQUEST(t, &dhcpRequestConfig{
			options:      layers.DHCPOptions{newOptRequestIP(t, testIPv4Static)},
			clientHWAddr: testHWUnknown,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		want:     nil,
		name:     "no_lease",
		wantOpts: nil,
	}, {
		req: newDHCPREQUEST(t, &dhcpRequestConfig{
			options:      layers.DHCPOptions{newOptRequestIP(t, testIPv4Dynamic)},
			clientHWAddr: testHWStatic,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		want: nil,
		name: "wrong_ip",
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeNak),
			newOptServerID(t, testIfaceAddrV4),
		},
	}, {
		req: newDHCPREQUEST(t, &dhcpRequestConfig{
			options:      layers.DHCPOptions{newOptRequestIP(t, testIPv4Dynamic)},
			clientHWAddr: testHWStatic,
		}),
		want: nil,
		name: "wrong_ip_no_broadcast",
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeNak),
			newOptServerID(t, testIfaceAddrV4),
		},
	}, {
		req: newDHCPREQUEST(t, &dhcpRequestConfig{
			options:      layers.DHCPOptions{newOptRequestIP(t, testIPv4Static)},
			clientHWAddr: testHWStatic,
			clientIP:     testIPv4Static,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		want:     nil,
		name:     "nonzero_ciaddr",
		wantOpts: nil,
	}}

	for _, tc := range testCases {
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

			ndMgr, dev, inCh, outCh := newTestNetworkDeviceAndManager(t, testIfaceAddrV4)
			if tc.wantOpts == nil {
				dev.onWritePacketData = unexpectedWritePacketData
			}

			startTestDHCPServer(t, &dhcpsvc.Config{
				Database:             db,
				Interfaces:           testIPv4InterfacesConf,
				NetworkDeviceManager: ndMgr,
				Enabled:              true,
			})

			testutil.RequireSend(t, inCh, tc.req, testTimeout)

			assertValidResponse4(t, dhcpv4FromPacket(t, tc.req), outCh, tc.wantOpts)
		})
	}
}

func TestDHCPServer_ServeEther4_requestRenewSuccess(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		req      gopacket.Packet
		want     *dhcpsvc.Lease
		name     string
		wantOpts layers.DHCPOptions
	}{{
		req: newDHCPREQUEST(t, &dhcpRequestConfig{
			clientHWAddr: testHWDynamic,
			clientIP:     testIPv4Dynamic,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		want: testLease4Dynamic,
		name: "success",
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeAck),
			newOptServerID(t, testIfaceAddrV4),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, testLease4HostnameDynamic),
		},
	}, {
		req: newDHCPREQUEST(t, &dhcpRequestConfig{
			clientHWAddr: testHWStatic,
			clientIP:     testIPv4Static,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		want: testLease4Static,
		name: "static",
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeAck),
			newOptServerID(t, testIfaceAddrV4),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, testLease4HostnameStatic),
		},
	}, {
		req: newDHCPREQUEST(t, &dhcpRequestConfig{
			clientHWAddr: testHWDynamic,
			clientIP:     testIPv4Dynamic,
			relayAgentIP: testIPv4RelayAgent,
		}),
		want: testLease4Dynamic,
		name: "relay_agent",
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeAck),
			newOptServerID(t, testIfaceAddrV4),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, testLease4HostnameDynamic),
		},
	}, {
		req: newDHCPREQUEST(t, &dhcpRequestConfig{
			clientHWAddr: testHWDynamic,
			clientIP:     testIPv4Dynamic,
		}),
		want: testLease4Dynamic,
		name: "ciaddr_unicast",
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeAck),
			newOptServerID(t, testIfaceAddrV4),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, testLease4HostnameDynamic),
		},
	}}

	for _, tc := range testCases {
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

			ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceAddrV4)
			startTestDHCPServer(t, &dhcpsvc.Config{
				Database:             db,
				Interfaces:           testIPv4InterfacesConf,
				NetworkDeviceManager: ndMgr,
				Enabled:              true,
			})

			testutil.RequireSend(t, inCh, tc.req, testTimeout)

			assertValidResponse4(t, dhcpv4FromPacket(t, tc.req), outCh, tc.wantOpts)
		})
	}
}

func TestDHCPServer_ServeEther4_requestRenewFail(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		req      gopacket.Packet
		wantOpts layers.DHCPOptions
	}{{
		name: "wrong_subnet",
		req: newDHCPREQUEST(t, &dhcpRequestConfig{
			clientHWAddr: testHWStatic,
			clientIP:     testIPv4OtherSubnet,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		wantOpts: nil,
	}, {
		name: "no_lease",
		req: newDHCPREQUEST(t, &dhcpRequestConfig{
			clientHWAddr: testHWUnknown,
			clientIP:     testIPv4Static,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		wantOpts: nil,
	}, {
		name: "wrong_ip",
		req: newDHCPREQUEST(t, &dhcpRequestConfig{
			clientHWAddr: testHWStatic,
			clientIP:     testIPv4Dynamic,
			flags:        dhcpsvc.FlagsBroadcast,
		}),
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeNak),
			newOptServerID(t, testIfaceAddrV4),
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ndMgr, dev, inCh, outCh := newTestNetworkDeviceAndManager(t, testIfaceAddrV4)
			if tc.wantOpts == nil {
				dev.onWritePacketData = unexpectedWritePacketData
			}

			startTestDHCPServer(t, &dhcpsvc.Config{
				Database:             newTestDatabase(t, testLeases),
				Interfaces:           testIPv4InterfacesConf,
				NetworkDeviceManager: ndMgr,
				Enabled:              true,
			})

			testutil.RequireSend(t, inCh, tc.req, testTimeout)

			assertValidResponse4(t, dhcpv4FromPacket(t, tc.req), outCh, tc.wantOpts)
		})
	}
}

func TestDHCPServer_ServeEther4_decline(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		want netip.Addr
		req  gopacket.Packet
		name string
	}{{
		want: testIPv4Dynamic,
		req:  newDHCPDECLINE(t, testHWDynamic, testIPv4Dynamic),
		name: "success",
	}, {
		want: netip.Addr{},
		req:  newDHCPDECLINE(t, testHWUnknown, testIPv4Dynamic),
		name: "not_found",
	}, {
		want: netip.Addr{},
		req:  newDHCPDECLINE(t, testHWAnother, testIPv4Unknown),
		name: "mismatch_ip",
	}, {
		want: netip.Addr{},
		req:  newDHCPDECLINE(t, testHWDynamic, testIPv4OtherSubnet),
		name: "bad_subnet",
	}, {
		want: netip.Addr{},
		req:  newDHCPDECLINE(t, testHWDynamic, netip.Addr{}),
		name: "no_requested_ip",
	}}

	for _, tc := range testCases {
		// The lease should be blocked.
		wantLease := &dhcpsvc.Lease{
			IP:       tc.want,
			Expiry:   testExpiryDynamicLease,
			Hostname: "",
			HWAddr:   dhcpsvc.BlockedHardwareAddr,
			IsStatic: false,
		}

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDatabase(t, testLeases)

			onStore := func(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
				assert.Contains(t, leases, wantLease)

				return nil
			}

			if tc.want != (netip.Addr{}) {
				db.onStore = onStore
			}

			ndMgr, inCh, _ := newTestNetworkDeviceManager(t, testIfaceAddrV4)
			startTestDHCPServer(t, &dhcpsvc.Config{
				Database:             db,
				Interfaces:           testIPv4InterfacesConf,
				NetworkDeviceManager: ndMgr,
				Enabled:              true,
			})

			testutil.RequireSend(t, inCh, tc.req, testTimeout)
		})
	}
}

// dhcpRequestConfig contains the configuration for creating a DHCPREQUEST
// packet.
type dhcpRequestConfig struct {
	// options are additional DHCP options to include in the packet, excluding
	// the message type.
	options layers.DHCPOptions

	// clientIP is the ciaddr field value.  If zero, it's set to unspecified.
	clientIP netip.Addr

	// relayAgentIP is the giaddr field value.  If zero, it's set to
	// unspecified.
	relayAgentIP netip.Addr

	// clientHWAddr is the MAC address of the client.  It must be set.
	clientHWAddr net.HardwareAddr

	// flags is the DHCP message flags field value.
	flags uint16
}

// newDHCPREQUEST creates a new DHCPREQUEST packet for testing.
func newDHCPREQUEST(tb testing.TB, conf *dhcpRequestConfig) (pkt gopacket.Packet) {
	tb.Helper()

	eth := newEthernetLayer(tb, conf.clientHWAddr, nil, layers.EthernetTypeIPv4)

	ip, udp := newIPv4UDPLayer(
		tb,
		netip.AddrPortFrom(conf.clientIP, uint16(dhcpsvc.ClientPortV4)),
		netip.AddrPort{},
	)

	opts := append(layers.DHCPOptions{
		layers.NewDHCPOption(
			layers.DHCPOptMessageType,
			[]byte{byte(layers.DHCPMsgTypeRequest)},
		),
	}, conf.options...)

	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  uint8(len(conf.clientHWAddr)),
		Xid:          testXid,
		Flags:        conf.flags,
		ClientIP:     cmp.Or(conf.clientIP, netip.IPv4Unspecified()).AsSlice(),
		RelayAgentIP: cmp.Or(conf.relayAgentIP, netip.IPv4Unspecified()).AsSlice(),
		ClientHWAddr: conf.clientHWAddr,
		Options:      opts,
	}

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newDHCPDISCOVER creates a new DHCPDISCOVER packet for testing.
//
// TODO(e.burkov):  Add parameters.
func newDHCPDISCOVER(tb testing.TB, clientHWAddr net.HardwareAddr) (pkt gopacket.Packet) {
	tb.Helper()

	eth := newEthernetLayer(tb, clientHWAddr, nil, layers.EthernetTypeIPv4)

	ip, udp := newIPv4UDPLayer(tb, netip.AddrPort{}, netip.AddrPort{})

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
) (pkt gopacket.Packet) {
	tb.Helper()

	eth := newEthernetLayer(tb, clientHWAddr, testIfaceHWAddr, layers.EthernetTypeIPv4)

	ip, udp := newIPv4UDPLayer(
		tb,
		netip.AddrPortFrom(clientIP, uint16(dhcpsvc.ClientPortV4)),
		netip.AddrPortFrom(testIfaceAddrV4, uint16(dhcpsvc.ServerPortV4)),
	)

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

// newDHCPDECLINE creates a new DHCPDECLINE packet for testing.
func newDHCPDECLINE(
	tb testing.TB,
	clientHWAddr net.HardwareAddr,
	requestedIP netip.Addr,
) (pkt gopacket.Packet) {
	tb.Helper()

	eth := newEthernetLayer(tb, clientHWAddr, nil, layers.EthernetTypeIPv4)

	ip, udp := newIPv4UDPLayer(tb, netip.AddrPort{}, netip.AddrPort{})

	opts := layers.DHCPOptions{
		newOptMessageType(tb, layers.DHCPMsgTypeDecline),
	}

	if requestedIP.IsValid() {
		opts = append(opts, layers.NewDHCPOption(
			layers.DHCPOptRequestIP,
			requestedIP.AsSlice(),
		))
	}

	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  dhcpsvc.EUI48AddrLen,
		Xid:          testXid,
		ClientHWAddr: clientHWAddr,
		Options:      opts,
	}

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newIPv4UDPLayer creates IPv4 and UDP layers for testing.  Invalid src is
// replaced with an unspecified address and client DHCPv4 port, invalid dst is
// replaced with the broadcast address and server DHCPv4 port.
func newIPv4UDPLayer(tb testing.TB, src, dst netip.AddrPort) (ip *layers.IPv4, udp *layers.UDP) {
	tb.Helper()

	if !src.IsValid() {
		src = netip.AddrPortFrom(netip.IPv4Unspecified(), uint16(dhcpsvc.ClientPortV4))
	}

	if !dst.IsValid() {
		bcastAddr, ok := netip.AddrFromSlice(net.IPv4bcast)
		require.True(tb, ok)

		dst = netip.AddrPortFrom(bcastAddr, uint16(dhcpsvc.ServerPortV4))
	}

	ip = &layers.IPv4{
		Version:  4,
		TTL:      dhcpsvc.IPv4DefaultTTL,
		SrcIP:    src.Addr().AsSlice(),
		DstIP:    dst.Addr().AsSlice(),
		Protocol: layers.IPProtocolUDP,
	}
	udp = &layers.UDP{
		SrcPort: layers.UDPPort(src.Port()),
		DstPort: layers.UDPPort(dst.Port()),
	}
	require.NoError(tb, udp.SetNetworkLayerForChecksum(ip))

	return ip, udp
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

// assertValidResponse4 asserts that recvCh eventually gets the response with
// wantOpts for request.  It does nothing if wantOpts is nil, which should be
// used in case no response is expected.  request and recvCh must not be nil.
func assertValidResponse4(
	tb testing.TB,
	request *layers.DHCPv4,
	recvCh <-chan []byte,
	wantOpts layers.DHCPOptions,
) {
	tb.Helper()

	if wantOpts == nil {
		return
	}

	respData, ok := testutil.RequireReceive(tb, recvCh, testTimeout)
	require.True(tb, ok)

	ip := &layers.IPv4{}
	udp := &layers.UDP{}
	resp := &layers.DHCPv4{}
	types := requireEthernet(tb, respData, &layers.Ethernet{}, ip, udp, resp)
	require.Equal(tb, fullLayersStack4, types)

	assertValidDHCPv4(tb, request, resp, ip, udp)

	// TODO(e.burkov):  Consider comparing the whole message instead of separate
	// fields.
	assert.Equal(tb, layers.DHCPOpReply, resp.Operation, "operation")
	assert.Equal(tb, request.HardwareType, resp.HardwareType, "hardware type")
	assert.Equal(tb, request.HardwareLen, resp.HardwareLen, "hardware length")
	assert.Equal(tb, request.Xid, resp.Xid, "xid")
	assert.Equal(tb, request.ClientHWAddr, resp.ClientHWAddr, "client hardware address")
	assert.Equal(tb, wantOpts, resp.Options, "options")
}

// assertValidDHCPv4 asserts that the response is valid for the given request
// according to RFC 2131.
func assertValidDHCPv4(tb testing.TB, req, resp *layers.DHCPv4, ip *layers.IPv4, udp *layers.UDP) {
	tb.Helper()

	switch {
	case !req.RelayAgentIP.IsUnspecified():
		assert.Equal(tb, req.RelayAgentIP.To4(), ip.DstIP)
		assert.Equal(tb, dhcpsvc.ServerPortV4, udp.DstPort)
	case !req.ClientIP.IsUnspecified():
		assert.Equal(tb, req.ClientIP.To4(), ip.DstIP)
	case req.Flags&dhcpsvc.FlagsBroadcast != 0:
		assert.Equal(tb, net.IPv4bcast.To4(), ip.DstIP)
		assert.Equal(tb, dhcpsvc.ClientPortV4, udp.DstPort)
	case !resp.YourClientIP.IsUnspecified():
		assert.Equal(tb, resp.YourClientIP.To4(), ip.DstIP)
		assert.Equal(tb, dhcpsvc.ClientPortV4, udp.DstPort)
	default:
		assert.Equal(tb, net.IPv4zero.To4(), ip.DstIP)
		assert.Equal(tb, dhcpsvc.ClientPortV4, udp.DstPort)
	}
}

// dhcpv4FromPacket extracts the DHCPv4 layer from pkt, which is required to
// contain one.
func dhcpv4FromPacket(tb testing.TB, pkt gopacket.Packet) (msg *layers.DHCPv4) {
	tb.Helper()

	return testutil.RequireTypeAssert[*layers.DHCPv4](tb, pkt.Layer(layers.LayerTypeDHCPv4))
}
