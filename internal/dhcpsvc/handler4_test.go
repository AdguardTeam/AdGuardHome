package dhcpsvc_test

import (
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/servicetest"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testIPv4InterfacesConf is the test interfaces configuration for the DHCPv4
// part of the [DHCPServer].
var testIPv4InterfacesConf = map[string]*dhcpsvc.InterfaceConfig{
	testIfaceName: {
		IPv4: testIPv4Conf,
		IPv6: disabledIPv6Config,
	},
}

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

	// NOTE: Keep in sync with testdata.
	dynamicLeaseExpiry := time.Date(2025, 1, 1, 10, 1, 1, 0, time.UTC)
	dynamicLeaseTTL := dynamicLeaseExpiry.Sub(testCurrentTime)

	testCases := []struct {
		name     string
		in       gopacket.Packet
		wantOpts layers.DHCPOptions
	}{{
		name: "new",
		in:   newDHCPDISCOVER(t, hwAddrUnknown),
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeOffer),
			newOptServerID(t, testIfaceAddr),
			newOptLeaseTime(t, testLeaseTTL),
		},
	}, {
		name: "existing_static",
		in:   newDHCPDISCOVER(t, hwAddrStatic),
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeOffer),
			newOptServerID(t, testIfaceAddr),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, leaseHostnameStatic),
		},
	}, {
		name: "existing_dynamic",
		in:   newDHCPDISCOVER(t, hwAddrDynamic),
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeOffer),
			newOptServerID(t, testIfaceAddr),
			newOptLeaseTime(t, dynamicLeaseTTL),
			newOptHostname(t, leaseHostnameDynamic),
		},
	}, {
		name: "existing_dynamic_expired",
		in:   newDHCPDISCOVER(t, hwAddrExpired),
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeOffer),
			newOptServerID(t, testIfaceAddr),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, leaseHostnameExpired),
		},
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv4](t, tc.in.Layer(layers.LayerTypeDHCPv4))

		ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceName, testIfaceAddr)
		srv := newTestDHCPServer(t, &dhcpsvc.Config{
			Interfaces:           testIPv4InterfacesConf,
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

	pkt := newDHCPDISCOVER(t, hwAddrUnknown)
	req := testutil.RequireTypeAssert[*layers.DHCPv4](t, pkt.Layer(layers.LayerTypeDHCPv4))

	ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceName, testIfaceAddr)

	srv := newTestDHCPServer(t, &dhcpsvc.Config{
		Interfaces:           testIPv4InterfacesConf,
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
		newOptServerID(t, testIfaceAddr),
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
		ipSuccess = netip.MustParseAddr("192.0.2.102")

		// ipMismatch is the IP of the lease used in the mismatch cases.
		ipMismatch = netip.MustParseAddr("192.0.2.103")

		// hwAddrMismatch is the MAC address for a lease with mismatched IP.
		hwAddrMismatch = net.HardwareAddr{0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

		// ipMismatchReq is the IP requested for release, which differs from the
		// lease IP.
		ipMismatchReq = netip.MustParseAddr("192.0.2.104")

		// hwAddrUnknown is an unknown MAC.
		hwAddrUnknown = net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	)

	anotherSubnetAddr := netip.MustParseAddr(testAnotherGatewayIPv4Str)

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
		req:  newDHCPRELEASE(t, hwAddrSuccess, ipSuccess, testIfaceHWAddr, testIfaceAddr),
		wantLeases: []*dhcpsvc.Lease{
			leaseMismatch,
		},
	}, {
		name: "not_found",
		req:  newDHCPRELEASE(t, hwAddrUnknown, ipSuccess, testIfaceHWAddr, testIfaceAddr),
		wantLeases: []*dhcpsvc.Lease{
			leaseSuccess,
			leaseMismatch,
		},
	}, {
		name: "mismatch_ip",
		req:  newDHCPRELEASE(t, hwAddrMismatch, ipMismatchReq, testIfaceHWAddr, testIfaceAddr),
		wantLeases: []*dhcpsvc.Lease{
			leaseSuccess,
			leaseMismatch,
		},
	}, {
		name: "bad_subnet",
		req:  newDHCPRELEASE(t, hwAddrSuccess, anotherSubnetAddr, testIfaceHWAddr, testIfaceAddr),
		wantLeases: []*dhcpsvc.Lease{
			leaseSuccess,
			leaseMismatch,
		},
	}}

	for _, tc := range testCases {
		ndMgr, inCh, _ := newTestNetworkDeviceManager(t, testIfaceName, testIfaceAddr)
		srv := newTestDHCPServer(t, &dhcpsvc.Config{
			Interfaces:           testIPv4InterfacesConf,
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

func TestDHCPServer_ServeEther4_requestSelecting(t *testing.T) {
	t.Parallel()

	// NOTE: Keep in sync with testdata.
	var (
		hwAddrUnknown = net.HardwareAddr{0x0, 0x1, 0x2, 0x3, 0x4, 0x5}
		hwAddrStatic  = net.HardwareAddr{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}

		ipStatic = netip.MustParseAddr("192.0.2.101")
		ipWrong  = netip.MustParseAddr("192.0.2.200")

		ipOtherSubnet = netip.MustParseAddr(testAnotherGatewayIPv4Str)
	)

	testCases := []struct {
		discover     gopacket.Packet
		conf         *dhcpRequestConfig
		name         string
		wantOpts     layers.DHCPOptions
		wantResponse layers.DHCPMsgType
	}{{
		discover: newDHCPDISCOVER(t, hwAddrUnknown),
		conf: &dhcpRequestConfig{
			requestedIP:  testIPv4Conf.RangeStart,
			clientHWAddr: hwAddrUnknown,
			serverID:     testIfaceAddr,
		},
		name: "success",
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeAck),
			newOptServerID(t, testIfaceAddr),
			newOptLeaseTime(t, testLeaseTTL),
		},
		wantResponse: layers.DHCPMsgTypeAck,
	}, {
		discover: newDHCPDISCOVER(t, hwAddrStatic),
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrStatic,
			serverID:     ipOtherSubnet,
			requestedIP:  ipStatic,
		},
		name:         "wrong_server_id",
		wantOpts:     nil,
		wantResponse: layers.DHCPMsgTypeUnspecified,
	}, {
		discover: nil,
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrUnknown,
			serverID:     testIfaceAddr,
			requestedIP:  ipWrong,
		},
		name:         "no_lease",
		wantOpts:     nil,
		wantResponse: layers.DHCPMsgTypeNak,
	}, {
		discover: newDHCPDISCOVER(t, hwAddrStatic),
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrStatic,
			serverID:     testIfaceAddr,
			requestedIP:  ipWrong,
		},
		name:         "wrong_ip",
		wantOpts:     nil,
		wantResponse: layers.DHCPMsgTypeNak,
	}, {
		discover: newDHCPDISCOVER(t, hwAddrStatic),
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrStatic,
			serverID:     testIfaceAddr,
			requestedIP:  ipStatic,
			clientIP:     ipStatic,
		},
		name:         "nonzero_ciaddr",
		wantOpts:     nil,
		wantResponse: layers.DHCPMsgTypeUnspecified,
	}}

	for _, tc := range testCases {
		ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceName, testIfaceAddr)
		srv := newTestDHCPServer(t, &dhcpsvc.Config{
			Logger:               slogutil.NewDiscardLogger(),
			Interfaces:           testIPv4InterfacesConf,
			NetworkDeviceManager: ndMgr,
			DBFilePath:           newTempDB(t),
			Enabled:              true,
		})

		pkt := newDHCPREQUEST(t, tc.conf)
		req := testutil.RequireTypeAssert[*layers.DHCPv4](t, pkt.Layer(layers.LayerTypeDHCPv4))

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			servicetest.RequireRun(t, srv, testTimeout)

			if tc.discover != nil {
				testutil.RequireSend(t, inCh, tc.discover, testTimeout)

				_, ok := testutil.RequireReceive(t, outCh, testTimeout)
				require.True(t, ok)
			}

			testutil.RequireSend(t, inCh, pkt, testTimeout)

			switch tc.wantResponse {
			case layers.DHCPMsgTypeUnspecified:
				assertNoResponse(t, outCh, testTimeout/10)
			case layers.DHCPMsgTypeAck:
				assertValidACK(t, req, outCh, tc.wantOpts)
			case layers.DHCPMsgTypeNak:
				assertValidNAK(t, req, outCh, testIPv4Conf.GatewayIP)
			}
		})
	}
}

func TestDHCPServer_ServeEther4_requestInitReboot(t *testing.T) {
	t.Parallel()

	// NOTE: Keep in sync with testdata.
	const (
		leaseHostnameStatic = "static4"
	)

	// NOTE: Keep in sync with testdata.
	var (
		hwAddrUnknown = net.HardwareAddr{0x0, 0x1, 0x2, 0x3, 0x4, 0x5}
		hwAddrStatic  = net.HardwareAddr{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}

		ipStatic  = netip.MustParseAddr("192.0.2.101")
		ipDynamic = netip.MustParseAddr("192.0.2.102")

		ipOtherSubnet = netip.MustParseAddr(testAnotherGatewayIPv4Str)
	)

	testCases := []struct {
		conf         *dhcpRequestConfig
		name         string
		wantOpts     layers.DHCPOptions
		wantResponse layers.DHCPMsgType
	}{{
		name: "success",
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrStatic,
			requestedIP:  ipStatic,
		},
		wantResponse: layers.DHCPMsgTypeAck,
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeAck),
			newOptServerID(t, testIfaceAddr),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, leaseHostnameStatic),
		},
	}, {
		name: "wrong_subnet",
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrStatic,
			requestedIP:  ipOtherSubnet,
		},
		wantResponse: layers.DHCPMsgTypeNak,
	}, {
		name: "no_lease",
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrUnknown,
			requestedIP:  ipStatic,
		},
		wantResponse: layers.DHCPMsgTypeUnspecified,
	}, {
		name: "wrong_ip",
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrStatic,
			requestedIP:  ipDynamic,
		},
		wantResponse: layers.DHCPMsgTypeNak,
	}, {
		name: "nonzero_ciaddr",
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrStatic,
			requestedIP:  ipStatic,
			clientIP:     ipStatic,
		},
		wantResponse: layers.DHCPMsgTypeUnspecified,
	}}

	for _, tc := range testCases {
		ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceName, testIfaceAddr)
		srv := newTestDHCPServer(t, &dhcpsvc.Config{
			Interfaces:           testIPv4InterfacesConf,
			NetworkDeviceManager: ndMgr,
			DBFilePath:           newTempDB(t),
			Enabled:              true,
		})

		pkt := newDHCPREQUEST(t, tc.conf)
		req := testutil.RequireTypeAssert[*layers.DHCPv4](t, pkt.Layer(layers.LayerTypeDHCPv4))

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			servicetest.RequireRun(t, srv, testTimeout)

			testutil.RequireSend(t, inCh, pkt, testTimeout)

			switch tc.wantResponse {
			case layers.DHCPMsgTypeUnspecified:
				assertNoResponse(t, outCh, testTimeout/10)
			case layers.DHCPMsgTypeAck:
				assertValidACK(t, req, outCh, tc.wantOpts)
			case layers.DHCPMsgTypeNak:
				assertValidNAK(t, req, outCh, testIPv4Conf.GatewayIP)
			}
		})
	}
}

func TestDHCPServer_ServeEther4_requestRenew(t *testing.T) {
	t.Parallel()

	// NOTE: Keep in sync with testdata.
	const (
		leaseHostnameStatic = "static4"
	)

	// NOTE: Keep in sync with testdata.
	var (
		hwAddrUnknown = net.HardwareAddr{0x0, 0x1, 0x2, 0x3, 0x4, 0x5}
		hwAddrStatic  = net.HardwareAddr{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
		hwAddrDynamic = net.HardwareAddr{0x2, 0x3, 0x4, 0x5, 0x6, 0x7}

		ipStatic  = netip.MustParseAddr("192.0.2.101")
		ipDynamic = netip.MustParseAddr("192.0.2.102")

		ipOtherSubnet = netip.MustParseAddr(testAnotherGatewayIPv4Str)
	)

	// NOTE: Keep in sync with testdata.
	dynamicLeaseExpiry := time.Date(2025, 1, 1, 10, 1, 1, 0, time.UTC)
	dynamicLeaseTTL := dynamicLeaseExpiry.Sub(testCurrentTime)

	testCases := []struct {
		conf         *dhcpRequestConfig
		name         string
		wantOpts     layers.DHCPOptions
		wantResponse layers.DHCPMsgType
	}{{
		name: "success",
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrDynamic,
			clientIP:     ipDynamic,
		},
		wantResponse: layers.DHCPMsgTypeAck,
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeAck),
			newOptServerID(t, testIfaceAddr),
			newOptLeaseTime(t, dynamicLeaseTTL),
			newOptHostname(t, "dynamic4"),
		},
	}, {
		name: "static",
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrStatic,
			clientIP:     ipStatic,
		},
		wantResponse: layers.DHCPMsgTypeAck,
		wantOpts: layers.DHCPOptions{
			newOptMessageType(t, layers.DHCPMsgTypeAck),
			newOptServerID(t, testIfaceAddr),
			newOptLeaseTime(t, testLeaseTTL),
			newOptHostname(t, leaseHostnameStatic),
		},
	}, {
		name: "wrong_subnet",
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrStatic,
			clientIP:     ipOtherSubnet,
		},
		wantResponse: layers.DHCPMsgTypeUnspecified,
	}, {
		name: "no_lease",
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrUnknown,
			clientIP:     ipStatic,
		},
		wantResponse: layers.DHCPMsgTypeUnspecified,
	}, {
		name: "wrong_ip",
		conf: &dhcpRequestConfig{
			clientHWAddr: hwAddrStatic,
			clientIP:     ipDynamic,
		},
		wantResponse: layers.DHCPMsgTypeNak,
	}}

	for _, tc := range testCases {
		ndMgr, inCh, outCh := newTestNetworkDeviceManager(t, testIfaceName, testIfaceAddr)
		srv := newTestDHCPServer(t, &dhcpsvc.Config{
			Interfaces:           testIPv4InterfacesConf,
			NetworkDeviceManager: ndMgr,
			DBFilePath:           newTempDB(t),
			Enabled:              true,
		})

		pkt := newDHCPREQUEST(t, tc.conf)
		req := testutil.RequireTypeAssert[*layers.DHCPv4](t, pkt.Layer(layers.LayerTypeDHCPv4))

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			servicetest.RequireRun(t, srv, testTimeout)

			testutil.RequireSend(t, inCh, pkt, testTimeout)

			switch tc.wantResponse {
			case layers.DHCPMsgTypeUnspecified:
				assertNoResponse(t, outCh, testTimeout/10)
			case layers.DHCPMsgTypeAck:
				assertValidACK(t, req, outCh, tc.wantOpts)
			case layers.DHCPMsgTypeNak:
				assertValidNAK(t, req, outCh, testIPv4Conf.GatewayIP)
			}
		})
	}
}

func TestDHCPServer_ServeEther4_decline(t *testing.T) {
	t.Parallel()

	// NOTE: Keep in sync with testdata.
	leaseExpiry := time.Date(2025, 1, 1, 10, 1, 1, 0, time.UTC)

	// NOTE: Keep in sync with testdata.
	var (
		// hwAddrSuccess is the MAC address for a lease to be declined
		// successfully.
		hwAddrSuccess = net.HardwareAddr{0x02, 0x03, 0x04, 0x05, 0x06, 0x07}

		// ipSuccess matches the lease IP.
		ipSuccess = netip.MustParseAddr("192.0.2.102")

		// ipMismatch is the IP of the lease used in the mismatch cases.
		ipMismatch = netip.MustParseAddr("192.0.2.103")

		// hwAddrMismatch is the MAC address for a lease with mismatched IP.
		hwAddrMismatch = net.HardwareAddr{0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

		// ipMismatchReq is the IP requested for decline, which differs from
		// the lease IP.
		ipMismatchReq = netip.MustParseAddr("192.0.2.104")

		// hwAddrUnknown is an unknown MAC.
		hwAddrUnknown = net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	)

	anotherSubnetAddr := netip.MustParseAddr(testAnotherGatewayIPv4Str)

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
		req:  newDHCPDECLINE(t, hwAddrSuccess, ipSuccess),
		wantLeases: []*dhcpsvc.Lease{
			leaseMismatch,
		},
	}, {
		name: "not_found",
		req:  newDHCPDECLINE(t, hwAddrUnknown, ipSuccess),
		wantLeases: []*dhcpsvc.Lease{
			leaseSuccess,
			leaseMismatch,
		},
	}, {
		name: "mismatch_ip",
		req:  newDHCPDECLINE(t, hwAddrMismatch, ipMismatchReq),
		wantLeases: []*dhcpsvc.Lease{
			leaseSuccess,
			leaseMismatch,
		},
	}, {
		name: "bad_subnet",
		req:  newDHCPDECLINE(t, hwAddrSuccess, anotherSubnetAddr),
		wantLeases: []*dhcpsvc.Lease{
			leaseSuccess,
			leaseMismatch,
		},
	}, {
		name: "no_requested_ip",
		req:  newDHCPDECLINE(t, hwAddrSuccess, netip.Addr{}),
		wantLeases: []*dhcpsvc.Lease{
			leaseSuccess,
			leaseMismatch,
		},
	}}

	for _, tc := range testCases {
		ndMgr, inCh, _ := newTestNetworkDeviceManager(t, testIfaceName, testIfaceAddr)
		srv := newTestDHCPServer(t, &dhcpsvc.Config{
			Interfaces:           testIPv4InterfacesConf,
			NetworkDeviceManager: ndMgr,
			DBFilePath:           newTempDB(t),
			Enabled:              true,
		})

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			servicetest.RequireRun(t, srv, testTimeout)

			testutil.RequireSend(t, inCh, tc.req, testTimeout)

			assert.EventuallyWithT(t, func(ct *assert.CollectT) {
				assert.Equal(ct, tc.wantLeases, srv.Leases())
			}, testTimeout/2, testTimeout/20)
		})
	}
}

// TODO(e.burkov):  Add tests for wrong packets.

// dhcpRequestConfig contains the configuration for creating a DHCPREQUEST
// packet.
type dhcpRequestConfig struct {
	// serverID is the server identifier option value.  If zero, the option is
	// not included.
	serverID netip.Addr

	// requestedIP is the requested IP address option value.  If zero, the
	// option is not included.
	requestedIP netip.Addr

	// clientIP is the ciaddr field value.  If zero, it's set to 0.0.0.0.
	clientIP netip.Addr

	// hostname is the hostname option value.  If empty, the option is not
	// included.
	hostname string

	// clientHWAddr is the MAC address of the client.
	clientHWAddr net.HardwareAddr
}

// newDHCPREQUEST creates a new DHCPREQUEST packet for testing.
func newDHCPREQUEST(tb testing.TB, conf *dhcpRequestConfig) (pkt gopacket.Packet) {
	tb.Helper()

	eth := &layers.Ethernet{
		SrcMAC:       conf.clientHWAddr,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}

	srcIP := net.IPv4zero.To4()
	if conf.clientIP.IsValid() {
		srcIP = conf.clientIP.AsSlice()
	}

	ip := &layers.IPv4{
		Version:  4,
		TTL:      dhcpsvc.IPv4DefaultTTL,
		SrcIP:    srcIP,
		DstIP:    net.IPv4bcast.To4(),
		Protocol: layers.IPProtocolUDP,
	}
	udp := &layers.UDP{
		SrcPort: dhcpsvc.ClientPortV4,
		DstPort: dhcpsvc.ServerPortV4,
	}
	_ = udp.SetNetworkLayerForChecksum(ip)

	opts := layers.DHCPOptions{
		layers.NewDHCPOption(
			layers.DHCPOptMessageType,
			[]byte{byte(layers.DHCPMsgTypeRequest)},
		),
	}

	if conf.serverID.IsValid() {
		opts = append(opts, layers.NewDHCPOption(
			layers.DHCPOptServerID,
			conf.serverID.AsSlice(),
		))
	}

	if conf.requestedIP.IsValid() {
		opts = append(opts, layers.NewDHCPOption(
			layers.DHCPOptRequestIP,
			conf.requestedIP.AsSlice(),
		))
	}

	if conf.hostname != "" {
		opts = append(opts, layers.NewDHCPOption(
			layers.DHCPOptHostname,
			[]byte(conf.hostname),
		))
	}

	ciaddr := net.IPv4zero.To4()
	if conf.clientIP.IsValid() {
		ciaddr = conf.clientIP.AsSlice()
	}

	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  dhcpsvc.EUI48AddrLen,
		Xid:          testXid,
		ClientIP:     ciaddr,
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

// newDHCPDECLINE creates a new DHCPDECLINE packet for testing.
func newDHCPDECLINE(
	tb testing.TB,
	clientHWAddr net.HardwareAddr,
	requestedIP netip.Addr,
) (pkt gopacket.Packet) {
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

// assertValidACK asserts that respData contains a complete DHCPACK response
// with the expected options, wrapped with all layers down to Ethernet.
func assertValidACK(
	tb testing.TB,
	request *layers.DHCPv4,
	outCh <-chan []byte,
	wantOpts layers.DHCPOptions,
) {
	tb.Helper()

	respData, ok := testutil.RequireReceive(tb, outCh, testTimeout)
	require.True(tb, ok)

	resp := &layers.DHCPv4{}
	types := requireEthernet(tb, respData, &layers.Ethernet{}, &layers.IPv4{}, &layers.UDP{}, resp)
	require.Equal(tb, fullLayersStack, types)

	assert.Equal(tb, layers.DHCPOpReply, resp.Operation, "operation")
	assert.Equal(tb, request.HardwareType, resp.HardwareType, "hardware type")
	assert.Equal(tb, request.HardwareLen, resp.HardwareLen, "hardware length")
	assert.Equal(tb, request.Xid, resp.Xid, "xid")
	assert.Equal(tb, request.ClientHWAddr, resp.ClientHWAddr, "client hardware address")
	assert.Equal(tb, wantOpts, resp.Options, "options")
}

// assertValidNAK asserts that respData contains a complete DHCPNAK response
// wrapped with all layers down to Ethernet.
func assertValidNAK(
	tb testing.TB,
	request *layers.DHCPv4,
	outCh <-chan []byte,
	serverIP netip.Addr,
) {
	tb.Helper()

	respData, ok := testutil.RequireReceive(tb, outCh, testTimeout)
	require.True(tb, ok)

	resp := &layers.DHCPv4{}
	types := requireEthernet(tb, respData, &layers.Ethernet{}, &layers.IPv4{}, &layers.UDP{}, resp)
	require.Equal(tb, fullLayersStack, types)

	assert.Equal(tb, layers.DHCPOpReply, resp.Operation, "operation")
	assert.Equal(tb, request.HardwareType, resp.HardwareType, "hardware type")
	assert.Equal(tb, request.HardwareLen, resp.HardwareLen, "hardware length")
	assert.Equal(tb, request.Xid, resp.Xid, "xid")
	assert.Equal(tb, request.ClientHWAddr, resp.ClientHWAddr, "client hardware address")

	wantOpts := layers.DHCPOptions{
		newOptMessageType(tb, layers.DHCPMsgTypeNak),
		newOptServerID(tb, serverIP),
	}
	assert.Equal(tb, wantOpts, resp.Options, "options")
}

// assertNoResponse asserts that no response is received on the channel within
// the timeout.
func assertNoResponse(tb testing.TB, outCh <-chan []byte, timeout time.Duration) {
	tb.Helper()

	var resp []byte
	require.Panics(tb, func() {
		resp, _ = testutil.RequireReceive(testutil.PanicT{}, outCh, timeout)
	})

	require.Nil(tb, resp)
}
