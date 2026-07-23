package dhcpsvc_test

import (
	"cmp"
	"context"
	"net"
	"net/netip"
	"slices"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/faketime"
	"github.com/AdguardTeam/golibs/testutil/servicetest"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/stretchr/testify/require"
)

// TODO(e.burkov):  Use addresses and prefixes from [RFC 5737].
//
// [RFC 5737]: https://datatracker.ietf.org/doc/html/rfc5737

// testLocalTLD is a common local TLD for tests.
const testLocalTLD = "local"

// testIfaceName is the name of the test network interface.
const testIfaceName = "iface0"

// testTimeout is a common timeout for tests and contexts.
const testTimeout = 10 * time.Second

// testLeaseTTL is the lease duration used in tests.
const testLeaseTTL = 24 * time.Hour

// testLogger is a common logger for tests.
var testLogger = slogutil.NewDiscardLogger()

// testCurrentTime is the fixed time returned by [testClock] to ensure
// reproducible tests.
var testCurrentTime = time.Date(2025, 1, 1, 1, 1, 1, 0, time.UTC)

// testClock is the test [timeutil.Clock] that always returns [testCurrentTime].
var testClock = &faketime.Clock{
	OnNow: func() (now time.Time) {
		return testCurrentTime
	},
}

// Lease hostnames for test cases.
//
// NOTE: Keep in sync with testdata.
const (
	// testLease4HostnameUnknown is the test hostname for an unknown DHCPv4
	// lease.
	testLease4HostnameUnknown = "unknown4"

	// testLease4HostnameStatic is the test hostname for a static DHCPv4 lease.
	testLease4HostnameStatic = "static4"

	// testLease4HostnameDynamic is the test hostname for a dynamic DHCPv4
	// lease.
	testLease4HostnameDynamic = "dynamic4"

	// testLease4HostnameExpired is the test hostname for an expired DHCPv4
	// lease.
	testLease4HostnameExpired = "expired4"

	// testLease6HostnameUnknown is the test hostname for an unknown DHCPv6
	// lease.
	testLease6HostnameUnknown = "unknown6"

	// testLease6HostnameStatic is the test hostname for a static DHCPv6 lease.
	testLease6HostnameStatic = "static6"

	// testLease6HostnameDynamic is the test hostname for a dynamic DHCPv6 lease.
	testLease6HostnameDynamic = "dynamic6"

	// testLease6HostnameExpired is the test hostname for an expired DHCPv6
	// lease.
	testLease6HostnameExpired = "expired6"
)

const (
	// testGatewayIPv4Str is the string representation of the gateway IPv4
	// address used in tests.
	testGatewayIPv4Str = "192.0.2.1"

	// testSubnetMaskV4Str is the string representation of the subnet mask for
	// the IPv4 interface used in tests.
	testSubnetMaskV4Str = "255.255.255.0"

	// testRangeStartV4Str is the string representation of the range start of
	// the IPv4 interface used in tests.
	testRangeStartV4Str = "192.0.2.100"

	// testRangeEndV4Str is the string representation of the range end of the
	// IPv4 interface used in tests.
	testRangeEndV4Str = "192.0.2.200"

	// testIfaceAddrV4Str is the string representation of the interface's IPv4
	// address used in tests.
	testIfaceAddrV4Str = "192.0.2.2"

	// testAnotherGatewayIPv4Str is the string representation of the second
	// gateway IPv4 address used in tests.
	testAnotherGatewayIPv4Str = "198.51.100.1"

	// testAnotherSubnetMaskV4Str is the string representation of the subnet
	// mask for the second IPv4 interface used in tests.
	testAnotherSubnetMaskV4Str = "255.255.255.0"

	// testAnotherRangeStartV4Str is the string representation of the range
	// start of the second IPv4 interface used in tests.
	testAnotherRangeStartV4Str = "198.51.100.100"

	// testAnotherRangeEndV4Str is the string representation of the range end
	// of the second IPv4 interface used in tests.
	testAnotherRangeEndV4Str = "198.51.100.200"
)

const (
	// testRangeStartV6Str is the string representation of the range start of
	// the IPv6 interface used in tests.
	testRangeStartV6Str = "2001:db8::2"

	// testAnotherRangeStartV6Str is the string representation of the range
	// start of the second IPv6 interface used in tests.
	testAnotherRangeStartV6Str = "2001:db9::1"

	// testIfaceAddrV6Str is the string representation of the interface's IPv6
	// address used in tests.
	testIfaceAddrV6Str = "2001:db8::1"
)

var (
	// testIPv4Conf is a common valid IPv4 part of the interface configuration
	// for tests.
	testIPv4Conf = &dhcpsvc.IPv4Config{
		Clock:         testClock,
		GatewayIP:     netip.MustParseAddr(testGatewayIPv4Str),
		SubnetMask:    netip.MustParseAddr(testSubnetMaskV4Str),
		RangeStart:    netip.MustParseAddr(testRangeStartV4Str),
		RangeEnd:      netip.MustParseAddr(testRangeEndV4Str),
		LeaseDuration: testLeaseTTL,
		Enabled:       true,
	}

	// testIPv6Conf is a common valid IPv6 part of the interface configuration
	// for tests.
	testIPv6Conf = &dhcpsvc.IPv6Config{
		Enabled:       true,
		Clock:         testClock,
		RangeStart:    netip.MustParseAddr(testRangeStartV6Str),
		LeaseDuration: testLeaseTTL,
		RAAllowSLAAC:  true,
		RASLAACOnly:   true,
	}

	// disabledIPv4Conf is a configuration of IPv4 part of the interfaces
	// configuration that is disabled.
	disabledIPv4Conf = &dhcpsvc.IPv4Config{Enabled: false}

	// disabledIPv6Conf is a configuration of IPv6 part of the interfaces
	// configuration that is disabled.
	disabledIPv6Conf = &dhcpsvc.IPv6Config{Enabled: false}

	// testIfaceAddrV4 is a common valid IPv4 address of the test network
	// interface, compliant with [testIPv4Conf], i.e. outside of the range,
	// within the subnet, not equal to the gateway.
	testIfaceAddrV4 = netip.MustParseAddr(testIfaceAddrV4Str)

	// testIfaceAddrV6 is a common valid IPv6 address of the test network
	// interface, compliant with [testIPv6Conf], i.e. outside of the range,
	// within the subnet, not equal to the gateway.
	testIfaceAddrV6 = netip.MustParseAddr(testIfaceAddrV6Str)

	// testIfaceHWAddr is a common valid hardware address of the test network
	// interface.
	testIfaceHWAddr = net.HardwareAddr{0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
)

// testInterfaceConf is a common valid set of interface configurations for
// tests.
var testInterfaceConf = map[string]*dhcpsvc.InterfaceConfig{
	testIfaceName: {
		IPv4: testIPv4Conf,
		IPv6: testIPv6Conf,
	},
	"iface1": {
		IPv4: &dhcpsvc.IPv4Config{
			Enabled:       true,
			Clock:         timeutil.SystemClock{},
			GatewayIP:     netip.MustParseAddr(testAnotherGatewayIPv4Str),
			SubnetMask:    netip.MustParseAddr(testAnotherSubnetMaskV4Str),
			RangeStart:    netip.MustParseAddr(testAnotherRangeStartV4Str),
			RangeEnd:      netip.MustParseAddr(testAnotherRangeEndV4Str),
			LeaseDuration: testLeaseTTL,
		},
		IPv6: &dhcpsvc.IPv6Config{
			Enabled:       true,
			Clock:         timeutil.SystemClock{},
			RangeStart:    netip.MustParseAddr(testAnotherRangeStartV6Str),
			LeaseDuration: testLeaseTTL,
			RAAllowSLAAC:  true,
			RASLAACOnly:   true,
		},
	},
}

// Hardware addresses for test cases.
//
// NOTE: Keep in sync with testdata.
var (
	// testHWUnknown is the test MAC address for an unknown client.
	testHWUnknown = net.HardwareAddr{0x0, 0x1, 0x2, 0x3, 0x4, 0x5}

	// testHWStatic is the test MAC address for a known static lease.
	testHWStatic = net.HardwareAddr{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}

	// testHWDynamic is the test MAC address for a known dynamic lease.
	testHWDynamic = net.HardwareAddr{0x2, 0x3, 0x4, 0x5, 0x6, 0x7}

	// testHWExpired is the test MAC address for a known expired lease.
	testHWExpired = net.HardwareAddr{0x3, 0x4, 0x5, 0x6, 0x7, 0x8}

	// testHWAnother is the test MAC address for a lease with another IP.
	testHWAnother = net.HardwareAddr{0x4, 0x5, 0x6, 0x7, 0x8, 0x9}
)

// IPv4 addresses for tests.
//
// NOTE: Keep in sync with testdata.
var (
	// testIPv4Unknown is the test IP address for an unknown client.
	testIPv4Unknown = netip.MustParseAddr("192.0.2.142")

	// testIPv4Static is the test IP address for a known static lease.
	testIPv4Static = netip.MustParseAddr("192.0.2.101")

	// testIPv4Dynamic is the test IP address for a known dynamic lease.
	testIPv4Dynamic = netip.MustParseAddr("192.0.2.102")

	// testIPv4Expired is the test IP address for a known expired lease.
	testIPv4Expired = netip.MustParseAddr("192.0.2.103")

	// testIPv4OtherSubnet is the test IP address for a client on another
	// subnet.
	testIPv4OtherSubnet = netip.MustParseAddr(testAnotherGatewayIPv4Str)

	// testIPv4RelayAgent is the test IP address of the relay agent.
	testIPv4RelayAgent = netip.MustParseAddr("10.0.0.1")
)

// IPv6 addresses for tests.
//
// NOTE: Keep in sync with testdata.
var (
	// testIPv6Unknown is the test IP address for an unknown client.
	testIPv6Unknown = netip.MustParseAddr("2001:db8::64")

	// testIPv6Dynamic is the test IP address for a known dynamic lease.
	testIPv6Dynamic = netip.MustParseAddr("2001:db8::66")

	// testIPv6Expired is the test IP address for a known expired lease.
	testIPv6Expired = netip.MustParseAddr("2001:db8::67")

	// testIPv6Static is the test IP address for a known static lease.
	testIPv6Static = netip.MustParseAddr("2001:db8::65")
)

// Time-related variables for test cases.
//
// NOTE: Keep in sync with testdata.
var (
	// testExpiryDynamicLease is the test expiry time for a dynamic lease, not
	// yet expired according to [testClock].
	testExpiryDynamicLease = testCurrentTime.Add(testLeaseTTL)

	// testExpiryExpiredLease is the test expiry time for an expired lease
	// according to [testClock].
	testExpiryExpiredLease = testCurrentTime.Add(-time.Hour)
)

var (
	// testLease4Dynamic is a common valid dynamic DHCPv4 lease for tests.
	testLease4Dynamic = &dhcpsvc.Lease{
		IP:       testIPv4Dynamic,
		HWAddr:   testHWDynamic,
		Expiry:   testExpiryDynamicLease,
		Hostname: testLease4HostnameDynamic,
		IsStatic: false,
	}

	// testLease4Static is a common valid static DHCPv4 lease for tests.
	testLease4Static = &dhcpsvc.Lease{
		IP:       testIPv4Static,
		HWAddr:   testHWStatic,
		Expiry:   time.Time{},
		Hostname: testLease4HostnameStatic,
		IsStatic: true,
	}

	// testLease4Expired is a common expired DHCPv4 lease for tests.
	testLease4Expired = &dhcpsvc.Lease{
		IP:       testIPv4Expired,
		HWAddr:   testHWExpired,
		Expiry:   testExpiryExpiredLease,
		Hostname: testLease4HostnameExpired,
		IsStatic: false,
	}

	// testLease6Dynamic is a common valid dynamic DHCPv6 lease for tests.
	testLease6Dynamic = &dhcpsvc.Lease{
		IP:       testIPv6Dynamic,
		HWAddr:   testHWDynamic,
		Expiry:   testExpiryDynamicLease,
		Hostname: testLease6HostnameDynamic,
		IsStatic: false,
	}

	// testLease6Static is a common valid static DHCPv6 lease for tests.
	testLease6Static = &dhcpsvc.Lease{
		IP:       testIPv6Static,
		HWAddr:   testHWStatic,
		Expiry:   time.Time{},
		Hostname: testLease6HostnameStatic,
		IsStatic: true,
	}

	// testLease6Expired is a common expired DHCPv6 lease for tests.
	testLease6Expired = &dhcpsvc.Lease{
		IP:       testIPv6Expired,
		HWAddr:   testHWExpired,
		Expiry:   testExpiryExpiredLease,
		Hostname: testLease6HostnameExpired,
		IsStatic: false,
	}

	// testLeases4 is a common set of leases for tests, containing only IPv4
	// leases.
	testLeases4 = []*dhcpsvc.Lease{testLease4Dynamic, testLease4Expired, testLease4Static}

	// testLeases6 is a common set of leases for tests, containing only IPv6
	// leases.
	testLeases6 = []*dhcpsvc.Lease{testLease6Dynamic, testLease6Expired, testLease6Static}

	// testLeases is a set of leases for tests, containing both IPv4 and IPv6
	// leases.
	testLeases = slices.Concat(testLeases4, testLeases6)
)

// fullLayersStack4 is the complete stack of layers expected to appear in the
// DHCP response packets.
var fullLayersStack4 = []gopacket.LayerType{
	layers.LayerTypeEthernet,
	layers.LayerTypeIPv4,
	layers.LayerTypeUDP,
	layers.LayerTypeDHCPv4,
}

// fullLayersStack6 is the complete stack of layers expected to appear in the
// DHCPv6 response packets.
var fullLayersStack6 = []gopacket.LayerType{
	layers.LayerTypeEthernet,
	layers.LayerTypeIPv6,
	layers.LayerTypeUDP,
	layers.LayerTypeDHCPv6,
}

// testDatabase is a mock implementation of the [dhcpsvc.Database] interface for
// tests.
//
// TODO(e.burkov):  Consider moving to aghtest.
type testDatabase struct {
	onLoad  func(ctx context.Context) (leases []*dhcpsvc.Lease, err error)
	onStore func(ctx context.Context, leases []*dhcpsvc.Lease) (err error)
}

// type check
var _ dhcpsvc.Database = (*testDatabase)(nil)

// Load implements the [dhcpsvc.Database] interface for *testDatabase.
func (db *testDatabase) Load(ctx context.Context) (leases []*dhcpsvc.Lease, err error) {
	return db.onLoad(ctx)
}

// Store implements the [dhcpsvc.Database] interface for *testDatabase.
func (db *testDatabase) Store(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
	return db.onStore(ctx, leases)
}

// newTestDatabase creates a new *testDatabase for testing.  If initial is not
// nil, db.Load is set to return it.  By default, db.Store panics on any call.
func newTestDatabase(tb testing.TB, initial []*dhcpsvc.Lease) (db *testDatabase) {
	tb.Helper()

	db = &testDatabase{
		onLoad: func(ctx context.Context) (_ []*dhcpsvc.Lease, _ error) {
			panic(testutil.UnexpectedCall(ctx))
		},
		onStore: func(ctx context.Context, leases []*dhcpsvc.Lease) (_ error) {
			panic(testutil.UnexpectedCall(ctx, leases))
		},
	}

	if initial != nil {
		db.onLoad = func(ctx context.Context) (leases []*dhcpsvc.Lease, err error) {
			return initial, nil
		}
	}

	return db
}

// newTestDHCPServer creates a new DHCPServer for testing.  It uses the default
// values of config in case it's nil or some of its fields aren't set.
func newTestDHCPServer(tb testing.TB, conf *dhcpsvc.Config) (srv *dhcpsvc.DHCPServer) {
	tb.Helper()

	conf = cmp.Or(conf, &dhcpsvc.Config{
		Enabled: true,
	})

	conf.NetworkDeviceManager = cmp.Or[dhcpsvc.NetworkDeviceManager](
		conf.NetworkDeviceManager,
		dhcpsvc.EmptyNetworkDeviceManager{},
	)
	conf.Logger = cmp.Or(conf.Logger, testLogger)
	conf.LocalDomainName = cmp.Or(conf.LocalDomainName, testLocalTLD)
	conf.Database = cmp.Or[dhcpsvc.Database](conf.Database, dhcpsvc.EmptyDatabase{})
	conf.ICMPTimeout = cmp.Or(conf.ICMPTimeout, testTimeout)
	if conf.Interfaces == nil {
		conf.Interfaces = testInterfaceConf
	}

	srv, err := dhcpsvc.New(testutil.ContextWithTimeout(tb, testTimeout), conf)
	require.NoError(tb, err)

	return srv
}

// startTestDHCPServer creates a new DHCPServer for testing and starts it,
// adding a cleanup function to stop the server on test completion.
func startTestDHCPServer(tb testing.TB, conf *dhcpsvc.Config) {
	servicetest.RequireRun(tb, newTestDHCPServer(tb, conf), testTimeout)
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
