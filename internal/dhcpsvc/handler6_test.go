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
		in:   newDHCPv6Solicit(t, testHWUnknown, testIPv6Unknown, false),
		name: "new",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWUnknown,
			newDefaultOptIANA(t, testIPv6Conf.RangeStart),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:   newDHCPv6Solicit(t, testHWStatic, testIPv6Static, false),
		name: "existing_static",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWStatic,
			newDefaultOptIANA(t, testIPv6Static),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:   newDHCPv6Solicit(t, testHWDynamic, testIPv6Dynamic, false),
		name: "existing_dynamic",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWDynamic,
			newDefaultOptIANA(t, testIPv6Dynamic),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:   newDHCPv6Solicit(t, testHWExpired, testIPv6Expired, false),
		name: "existing_expired",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWExpired,
			newDefaultOptIANA(t, testIPv6Expired),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDatabase(t)

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

	rapidCommitOpt := layers.NewDHCPv6Option(layers.DHCPv6OptRapidCommit, []byte{})

	testCases := []struct {
		in       gopacket.Packet
		want     *dhcpsvc.Lease
		name     string
		wantOpts layers.DHCPv6Options
	}{{
		in: newDHCPv6Solicit(t, testHWUnknown, testIPv6Unknown, true),
		want: &dhcpsvc.Lease{
			IP:       testIPv6Conf.RangeStart,
			Expiry:   testExpiryDynamicLease,
			Hostname: aghnet.GenerateHostname(testIPv6Conf.RangeStart),
			HWAddr:   testHWUnknown,
			IsStatic: false,
		},
		name: "new",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWUnknown,
			newDefaultOptIANA(t, testIPv6Conf.RangeStart),
			defaultOptPref,
			defaultOptSolMaxRT,
			rapidCommitOpt,
		),
	}, {
		in:   newDHCPv6Solicit(t, testHWStatic, testIPv6Static, true),
		want: testLease6Static,
		name: "existing",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWStatic,
			newDefaultOptIANA(t, testIPv6Static),
			defaultOptPref,
			defaultOptSolMaxRT,
			rapidCommitOpt,
		),
	}, {
		in:   newDHCPv6Solicit(t, testHWDynamic, testIPv6Dynamic, true),
		want: testLease6Dynamic,
		name: "existing_dynamic",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWDynamic,
			newDefaultOptIANA(t, testIPv6Dynamic),
			defaultOptPref,
			defaultOptSolMaxRT,
			rapidCommitOpt,
		),
	}, {
		in: newDHCPv6Solicit(t, testHWExpired, testIPv6Expired, true),
		want: &dhcpsvc.Lease{
			IP:       testIPv6Expired,
			Expiry:   testExpiryDynamicLease,
			Hostname: testLease6HostnameExpired,
			HWAddr:   testHWExpired,
			IsStatic: false,
		},
		name: "existing_expired",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWExpired,
			newDefaultOptIANA(t, testIPv6Expired),
			defaultOptPref,
			defaultOptSolMaxRT,
			rapidCommitOpt,
		),
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDatabase(t)

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

	testCases := []struct {
		in       gopacket.Packet
		want     *dhcpsvc.Lease
		name     string
		wantOpts layers.DHCPv6Options
	}{{
		in: newDHCPv6Request(t, testHWUnknown, testIPv6Unknown),
		want: &dhcpsvc.Lease{
			IP:       testIPv6Conf.RangeStart,
			Expiry:   testExpiryDynamicLease,
			Hostname: aghnet.GenerateHostname(testIPv6Conf.RangeStart),
			HWAddr:   testHWUnknown,
			IsStatic: false,
		},
		name: "success",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWUnknown,
			newDefaultOptIANA(t, testIPv6Conf.RangeStart),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:   newDHCPv6Request(t, testHWUnknown, testIPv6OtherSubnet),
		want: nil,
		name: "not_on_link",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWUnknown,
			newOptIANAStatus(t, testIAID, layers.DHCPv6StatusCodeNotOnLink),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:   newDHCPv6Request(t, testHWStatic, testIPv6Static),
		want: testLease6Static,
		name: "existing_static",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWStatic,
			newDefaultOptIANA(t, testIPv6Static),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:       newDHCPv6Request(t, testHWUnknown, netip.Addr{}),
		want:     nil,
		name:     "no_iana",
		wantOpts: newWantDHCPv6Opts(t, testHWUnknown, defaultOptPref, defaultOptSolMaxRT),
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDatabase(t)

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

	wantOpts := newWantDHCPv6Opts(
		t,
		testHWUnknown,
		newDefaultOptIANA(t, testIPv6Conf.RangeStart),
		defaultOptPref,
		defaultOptSolMaxRT,
	)

	testCases := []struct {
		in      gopacket.Packet
		solicit gopacket.Packet
		want    *dhcpsvc.Lease
		name    string
	}{{
		in:      newDHCPv6Request(t, testHWUnknown, testIPv6Unknown),
		solicit: newDHCPv6Solicit(t, testHWUnknown, testIPv6Unknown, false),
		want: &dhcpsvc.Lease{
			IP:       testIPv6Conf.RangeStart,
			Expiry:   testExpiryDynamicLease,
			Hostname: aghnet.GenerateHostname(testIPv6Conf.RangeStart),
			HWAddr:   testHWUnknown,
			IsStatic: false,
		},
		name: "preceding_solicit",
	}, {
		in:      newDHCPv6Request(t, testHWUnknown, testIPv6Unknown),
		solicit: newDHCPv6Solicit(t, testHWUnknown, testIPv6Unknown, true),
		want: &dhcpsvc.Lease{
			IP:       testIPv6Conf.RangeStart,
			Expiry:   testExpiryDynamicLease,
			Hostname: aghnet.GenerateHostname(testIPv6Conf.RangeStart),
			HWAddr:   testHWUnknown,
			IsStatic: false,
		},
		name: "preceding_solicit_rapid_commit",
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDatabase(t)

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

			assertValidResponse6(t, req, outCh, wantOpts)
		})
	}
}

func TestDHCPServer_ServeEther6_confirm(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		in       gopacket.Packet
		name     string
		wantOpts layers.DHCPv6Options
	}{{
		in: newDHCPv6Confirm(
			t,
			testHWUnknown,
			newOptIANA(t, testIAID, testIPv6Unknown, 0),
		),
		name:     "success",
		wantOpts: newWantDHCPv6Opts(t, testHWUnknown),
	}, {
		in: newDHCPv6Confirm(
			t,
			testHWDynamic,
			newOptIANA(t, testIAID, testIPv6Dynamic, 0),
			newOptIANA(t, testIAID+1, testIPv6Static, 0),
		),
		name:     "success_multiple",
		wantOpts: newWantDHCPv6Opts(t, testHWDynamic),
	}, {
		in: newDHCPv6Confirm(
			t,
			testHWUnknown,
			newOptIANA(t, testIAID, testIPv6OtherSubnet, 0),
		),
		name: "not_on_link",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWUnknown,
			newOptStatusCode(t, layers.DHCPv6StatusCodeNotOnLink),
		),
	}, {
		in: newDHCPv6Confirm(
			t,
			testHWUnknown,
			newOptIANA(t, testIAID, testIPv6Unknown, 0),
			newOptIANA(t, testIAID+1, testIPv6OtherSubnet, 0),
		),
		name: "mixed",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWUnknown,
			newOptStatusCode(t, layers.DHCPv6StatusCodeNotOnLink),
		),
	}, {
		in:       newDHCPv6Confirm(t, testHWUnknown),
		name:     "no_iana",
		wantOpts: nil,
	}, {
		in: newDHCPv6Confirm(
			t,
			testHWUnknown,
			newOptIANAStatus(t, testIAID, layers.DHCPv6StatusCodeSuccess),
		),
		name:     "no_addrs",
		wantOpts: nil,
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := newTestDatabase(t)

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

func TestDHCPServer_ServeEther6_renew(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		in       gopacket.Packet
		want     *dhcpsvc.Lease
		name     string
		wantOpts layers.DHCPv6Options
	}{{
		in:   newDHCPv6Renew(t, testHWDynamic, testIPv6Dynamic),
		name: "success",
		want: testLease6Dynamic,
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWDynamic,
			newDefaultOptIANA(t, testIPv6Dynamic),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:   newDHCPv6Renew(t, testHWStatic, testIPv6Static),
		name: "success_static",
		want: testLease6Static,
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWStatic,
			newDefaultOptIANA(t, testIPv6Static),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:   newDHCPv6Renew(t, testHWUnknown, testIPv6Unknown),
		name: "no_binding",
		want: nil,
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWUnknown,
			newOptIANAStatus(t, testIAID, layers.DHCPv6StatusCodeNoBinding),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:       newDHCPv6Renew(t, testHWUnknown, netip.Addr{}),
		name:     "no_iana",
		want:     nil,
		wantOpts: newWantDHCPv6Opts(t, testHWUnknown, defaultOptPref, defaultOptSolMaxRT),
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		db := newTestDatabase(t)

		onStore := func(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
			assert.Contains(t, leases, tc.want)

			return nil
		}

		if tc.want != nil {
			db.onStore = onStore
		}

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

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

func TestDHCPServer_ServeEther6_rebind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		in       gopacket.Packet
		want     *dhcpsvc.Lease
		name     string
		wantOpts layers.DHCPv6Options
	}{{
		in:   newDHCPv6Rebind(t, testHWDynamic, testIPv6Dynamic),
		name: "success",
		want: testLease6Dynamic,
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWDynamic,
			newDefaultOptIANA(t, testIPv6Dynamic),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:   newDHCPv6Rebind(t, testHWStatic, testIPv6Static),
		name: "success_static",
		want: testLease6Static,
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWStatic,
			newDefaultOptIANA(t, testIPv6Static),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:   newDHCPv6Rebind(t, testHWUnknown, testIPv6Unknown),
		name: "no_binding",
		want: nil,
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWUnknown,
			newOptIANAStatus(t, testIAID, layers.DHCPv6StatusCodeNoBinding),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:       newDHCPv6Rebind(t, testHWUnknown, netip.Addr{}),
		name:     "no_iana",
		want:     nil,
		wantOpts: newWantDHCPv6Opts(t, testHWUnknown, defaultOptPref, defaultOptSolMaxRT),
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		db := newTestDatabase(t)

		onStore := func(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
			assert.Contains(t, leases, tc.want)

			return nil
		}

		if tc.want != nil {
			db.onStore = onStore
		}

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

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

func TestDHCPServer_ServeEther6_info(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		in       gopacket.Packet
		name     string
		wantOpts layers.DHCPv6Options
	}{{
		in:       newDHCPv6Info(t, testHWUnknown, true, true),
		name:     "cli_and_srv",
		wantOpts: newWantDHCPv6Opts(t, testHWUnknown),
	}, {
		in:   newDHCPv6Info(t, testHWUnknown, false, true),
		name: "srv_only",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
		},
	}, {
		in:       newDHCPv6Info(t, testHWUnknown, true, false),
		name:     "cli_only",
		wantOpts: newWantDHCPv6Opts(t, testHWUnknown),
	}, {
		in:   newDHCPv6Info(t, testHWUnknown, false, false),
		name: "no_opts",
		wantOpts: layers.DHCPv6Options{
			newOptServerDUID(t, testIfaceHWAddr),
		},
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		db := newTestDatabase(t)

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

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

func TestDHCPServer_ServeEther6_release(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		in       gopacket.Packet
		want     *dhcpsvc.Lease
		name     string
		wantOpts layers.DHCPv6Options
	}{{
		in:   newDHCPv6Release(t, testHWDynamic, testIPv6Dynamic),
		want: testLease6Dynamic,
		name: "success",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWDynamic,
			newOptIANAStatus(t, testIAID, layers.DHCPv6StatusCodeSuccess),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:   newDHCPv6Release(t, testHWStatic, testIPv6Static),
		want: testLease6Static,
		name: "success_static",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWStatic,
			newOptIANAStatus(t, testIAID, layers.DHCPv6StatusCodeSuccess),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:   newDHCPv6Release(t, testHWUnknown, testIPv6Unknown),
		want: nil,
		name: "no_binding",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWUnknown,
			newOptIANAStatus(t, testIAID, layers.DHCPv6StatusCodeNoBinding),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:       newDHCPv6Release(t, testHWDynamic, netip.Addr{}),
		want:     nil,
		name:     "no_iana",
		wantOpts: newWantDHCPv6Opts(t, testHWDynamic, defaultOptPref, defaultOptSolMaxRT),
	}, {
		in:   newDHCPv6Release(t, testHWDynamic, testIPv6Unknown),
		want: nil,
		name: "ip_mismatch",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWDynamic,
			newOptIANAStatus(t, testIAID, layers.DHCPv6StatusCodeNoBinding),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		db := newTestDatabase(t)

		onStore := func(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
			assert.NotContains(t, leases, tc.want)

			return nil
		}

		if tc.want != nil {
			db.onStore = onStore
		}

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

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

func TestDHCPServer_ServeEther6_decline(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		in       gopacket.Packet
		want     *dhcpsvc.Lease
		name     string
		wantOpts layers.DHCPv6Options
	}{{
		in: newDHCPv6Decline(t, testHWDynamic, testIPv6Dynamic),
		want: &dhcpsvc.Lease{
			IP:       testIPv6Dynamic,
			Expiry:   testExpiryDynamicLease,
			Hostname: "",
			HWAddr:   dhcpsvc.BlockedHardwareAddr,
			IsStatic: false,
		},
		name: "success",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWDynamic,
			newOptIANAStatus(t, testIAID, layers.DHCPv6StatusCodeSuccess),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:   newDHCPv6Decline(t, testHWUnknown, testIPv6Unknown),
		want: nil,
		name: "no_binding",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWUnknown,
			newOptIANAStatus(t, testIAID, layers.DHCPv6StatusCodeNoBinding),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:   newDHCPv6Decline(t, testHWDynamic, testIPv6Unknown),
		want: nil,
		name: "ip_mismatch",
		wantOpts: newWantDHCPv6Opts(
			t,
			testHWDynamic,
			newOptIANAStatus(t, testIAID, layers.DHCPv6StatusCodeNoBinding),
			defaultOptPref,
			defaultOptSolMaxRT,
		),
	}, {
		in:       newDHCPv6Decline(t, testHWDynamic, netip.Addr{}),
		want:     nil,
		name:     "no_iana",
		wantOpts: newWantDHCPv6Opts(t, testHWDynamic, defaultOptPref, defaultOptSolMaxRT),
	}}

	for _, tc := range testCases {
		req := testutil.RequireTypeAssert[*layers.DHCPv6](t, tc.in.Layer(layers.LayerTypeDHCPv6))

		db := newTestDatabase(t)

		onStore := func(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
			assert.Contains(t, leases, tc.want)

			return nil
		}

		if tc.want != nil {
			db.onStore = onStore
		}

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

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

// newDHCPv6Solicit creates a new DHCPv6 SOLICIT packet for testing.
func newDHCPv6Solicit(
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
		dhcp.Options = append(dhcp.Options, newOptIANA(tb, testIAID, reqIP, testLeaseTTL))
	}

	if rapidCommit {
		o := layers.NewDHCPv6Option(layers.DHCPv6OptRapidCommit, nil)
		dhcp.Options = append(dhcp.Options, o)
	}

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newDHCPv6Request creates a new DHCPv6 REQUEST packet for testing.
func newDHCPv6Request(tb testing.TB, mac net.HardwareAddr, reqIP netip.Addr) (pkt gopacket.Packet) {
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
		dhcp.Options = append(dhcp.Options, newOptIANA(tb, testIAID, reqIP, testLeaseTTL))
	}

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newDHCPv6Confirm creates a new DHCPv6 CONFIRM packet for testing.  addrs
// provides the addresses included within IA_NA options in the packet.  If addrs
// is empty, the packet contains no IA_NA options.
func newDHCPv6Confirm(
	tb testing.TB,
	mac net.HardwareAddr,
	ianas ...layers.DHCPv6Option,
) (pkt gopacket.Packet) {
	tb.Helper()

	eth := newEthernetLayer(tb, mac, testIfaceHWAddr, layers.EthernetTypeIPv6)
	ip, udp := newIPv6UDPLayer(tb, netip.AddrPort{}, netip.AddrPort{})

	dhcp := &layers.DHCPv6{
		MsgType:       layers.DHCPv6MsgTypeConfirm,
		HopCount:      0,
		LinkAddr:      nil,
		PeerAddr:      nil,
		TransactionID: testTransactionID,
		Options: layers.DHCPv6Options{
			newOptClientDUID(tb, mac),
		},
	}

	dhcp.Options = append(dhcp.Options, ianas...)

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newDHCPv6Renew creates a new DHCPv6 RENEW packet for testing.
func newDHCPv6Renew(tb testing.TB, mac net.HardwareAddr, reqIP netip.Addr) (pkt gopacket.Packet) {
	tb.Helper()

	opts := layers.DHCPv6Options{
		newOptClientDUID(tb, mac),
		newOptServerDUID(tb, testIfaceHWAddr),
	}

	if reqIP.Is6() {
		opts = append(opts, newOptIANA(tb, testIAID, reqIP, testLeaseTTL))
	}

	eth := newEthernetLayer(tb, mac, testIfaceHWAddr, layers.EthernetTypeIPv6)
	ip, udp := newIPv6UDPLayer(tb, netip.AddrPort{}, netip.AddrPort{})
	dhcp := newTestDHCPv6(tb, layers.DHCPv6MsgTypeRenew, opts...)

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newDHCPv6Rebind creates a new DHCPv6 REBIND packet for testing.
func newDHCPv6Rebind(tb testing.TB, mac net.HardwareAddr, reqIP netip.Addr) (pkt gopacket.Packet) {
	tb.Helper()

	opts := layers.DHCPv6Options{
		newOptClientDUID(tb, mac),
		// REBIND must not contain a Server ID option.
	}

	if reqIP.IsValid() && reqIP.Is6() {
		opts = append(opts, newOptIANA(tb, testIAID, reqIP, testLeaseTTL))
	}

	// REBIND is sent to any available server, so the destination is the
	// multicast address, not a specific server's unicast.
	eth := newEthernetLayer(tb, mac, nil, layers.EthernetTypeIPv6)
	ip, udp := newIPv6UDPLayer(tb, netip.AddrPort{}, netip.AddrPort{})
	dhcp := newTestDHCPv6(tb, layers.DHCPv6MsgTypeRebind, opts...)

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newDHCPv6Info creates a new DHCPv6 INFORMATION-REQUEST packet for testing.
// withClientID controls whether the packet includes a Client Identifier option.
func newDHCPv6Info(
	tb testing.TB,
	mac net.HardwareAddr,
	addClientID bool,
	addServerID bool,
) (pkt gopacket.Packet) {
	tb.Helper()

	var opts layers.DHCPv6Options

	if addClientID {
		opts = append(opts, newOptClientDUID(tb, mac))
	}

	if addServerID {
		opts = append(opts, newOptServerDUID(tb, testIfaceHWAddr))
	}

	eth := newEthernetLayer(tb, mac, nil, layers.EthernetTypeIPv6)
	ip, udp := newIPv6UDPLayer(tb, netip.AddrPort{}, netip.AddrPort{})
	dhcp := newTestDHCPv6(tb, layers.DHCPv6MsgTypeInformationRequest, opts...)

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newDHCPv6Release creates a new DHCPv6 RELEASE packet for testing.
func newDHCPv6Release(tb testing.TB, mac net.HardwareAddr, reqIP netip.Addr) (pkt gopacket.Packet) {
	tb.Helper()

	opts := layers.DHCPv6Options{
		newOptClientDUID(tb, mac),
		newOptServerDUID(tb, testIfaceHWAddr),
	}

	if reqIP.Is6() {
		opts = append(opts, newOptIANA(tb, testIAID, reqIP, testLeaseTTL))
	}

	eth := newEthernetLayer(tb, mac, testIfaceHWAddr, layers.EthernetTypeIPv6)
	ip, udp := newIPv6UDPLayer(tb, netip.AddrPort{}, netip.AddrPort{})
	dhcp := newTestDHCPv6(tb, layers.DHCPv6MsgTypeRelease, opts...)

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newDHCPv6Decline creates a new DHCPv6 DECLINE packet for testing.
func newDHCPv6Decline(tb testing.TB, mac net.HardwareAddr, reqIP netip.Addr) (pkt gopacket.Packet) {
	tb.Helper()

	opts := layers.DHCPv6Options{
		newOptClientDUID(tb, mac),
		newOptServerDUID(tb, testIfaceHWAddr),
	}

	if reqIP.Is6() {
		opts = append(opts, newOptIANA(tb, testIAID, reqIP, testLeaseTTL))
	}

	eth := newEthernetLayer(tb, mac, testIfaceHWAddr, layers.EthernetTypeIPv6)
	ip, udp := newIPv6UDPLayer(tb, netip.AddrPort{}, netip.AddrPort{})
	dhcp := newTestDHCPv6(tb, layers.DHCPv6MsgTypeDecline, opts...)

	return newTestPacket(tb, layers.LinkTypeEthernet, eth, ip, udp, dhcp)
}

// newTestDHCPv6 creates a new DHCPv6 message for testing with the specified
// type and options.  The link and peer addresses are not set, as they are
// intended for relay messages.
func newTestDHCPv6(
	tb testing.TB,
	msgType layers.DHCPv6MsgType,
	opts ...layers.DHCPv6Option,
) (dhcp *layers.DHCPv6) {
	tb.Helper()

	return &layers.DHCPv6{
		MsgType:  msgType,
		HopCount: 0,
		// Don't specify link and peer addresses, as they are intended for relay
		// messages.
		LinkAddr:      nil,
		PeerAddr:      nil,
		TransactionID: testTransactionID,
		Options:       opts,
	}
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
