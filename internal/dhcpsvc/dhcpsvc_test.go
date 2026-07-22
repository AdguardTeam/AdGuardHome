package dhcpsvc_test

import (
	"cmp"
	"context"
	"io/fs"
	"net"
	"net/netip"
	"os"
	"path"
	"path/filepath"
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

// testDBLeasesFilename is the common name of a leases database file for tests.
const testDBLeasesFilename = "leases.json"

// testTimeout is a common timeout for tests and contexts.
const testTimeout = 10 * time.Second

// testLeaseTTL is the lease duration used in tests.
const testLeaseTTL = 24 * time.Hour

// testLogger is a common logger for tests.
var testLogger = slogutil.NewDiscardLogger()

// testdata is a filesystem containing data for tests.
var testdata = os.DirFS("testdata")

// testCurrentTime is the fixed time returned by [testClock] to ensure
// reproducible tests.
var testCurrentTime = time.Date(2025, 1, 1, 1, 1, 1, 0, time.UTC)

// testClock is the test [timeutil.Clock] that always returns [testCurrentTime].
var testClock = &faketime.Clock{
	OnNow: func() (now time.Time) {
		return testCurrentTime
	},
}

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
			LeaseDuration: 1 * time.Hour,
		},
		IPv6: &dhcpsvc.IPv6Config{
			Enabled:       true,
			Clock:         timeutil.SystemClock{},
			RangeStart:    netip.MustParseAddr(testAnotherRangeStartV6Str),
			LeaseDuration: 1 * time.Hour,
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

// Time-related variables for test cases.
//
// NOTE: Keep in sync with testdata.
var (
	// testExpiryDynamicLease is the test expiry time for a dynamic lease.
	testExpiryDynamicLease = time.Date(2025, 1, 1, 10, 1, 1, 0, time.UTC)

	// testTTLDynamicLease is the test TTL for the dynamic lease.
	testTTLDynamicLease = testExpiryDynamicLease.Sub(testCurrentTime)

	// TODO(e.burkov):  Add a default lease expiry time, according to
	// [testCurrentTime].
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

// newTestDatabase copies the leases database file located in the testdata FS,
// under tb.Name()/leases.json, to a temporary directory and constructs a
// [*testDatabase] that properly loads the leases from that file.
//
// TODO(e.burkov):  Move leases from testdata to the literals in tests, and
// improve this helper.
func newTestDatabase(tb testing.TB) (db *testDatabase) {
	tb.Helper()

	data, err := fs.ReadFile(testdata, path.Join(tb.Name(), testDBLeasesFilename))
	require.NoError(tb, err)

	dst := filepath.Join(tb.TempDir(), testDBLeasesFilename)
	require.NoError(tb, os.WriteFile(dst, data, dhcpsvc.JSONDatabasePerm))

	jsonDB := dhcpsvc.NewJSONDatabase(&dhcpsvc.JSONDatabaseConfig{
		Logger:   testLogger,
		FilePath: dst,
	})

	db = newPanicDatabase(tb)
	db.onLoad = jsonDB.Load

	return db
}

// newPanicDatabase returns a *testDatabase that panics on any call to its
// methods.
func newPanicDatabase(tb testing.TB) (db *testDatabase) {
	tb.Helper()

	return &testDatabase{
		onLoad: func(ctx context.Context) (leases []*dhcpsvc.Lease, err error) {
			panic(testutil.UnexpectedCall(ctx))
		},
		onStore: func(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
			panic(testutil.UnexpectedCall(ctx, leases))
		},
	}
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
