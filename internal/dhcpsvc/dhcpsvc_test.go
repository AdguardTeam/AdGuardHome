package dhcpsvc_test

import (
	"cmp"
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
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
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

// testXid is a common transaction ID for DHCPv4 tests.
const testXid = 1

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
	testRangeStartV6Str = "2001:db8::1"

	// testAnotherRangeStartV6Str is the string representation of the range
	// start of the second IPv6 interface used in tests.
	testAnotherRangeStartV6Str = "2001:db9::1"
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

	// testIfaceAddr is a common valid IPv4 address of the test network
	// interface, compliant with [testIPv4Conf], i.e. outside of the range,
	// within the subnet, not equal to the gateway.
	testIfaceAddr = netip.MustParseAddr(testIfaceAddrV4Str)

	// testIfaceHWAddr is a common valid hardware address of the test network
	// interface.
	testIfaceHWAddr = net.HardwareAddr{0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
)

// testIPv6Conf is a common valid IPv6 part of the interface configuration for
// tests.
var testIPv6Conf = &dhcpsvc.IPv6Config{
	Enabled:       true,
	RangeStart:    netip.MustParseAddr(testRangeStartV6Str),
	LeaseDuration: testLeaseTTL,
	RAAllowSLAAC:  true,
	RASLAACOnly:   true,
}

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
			RangeStart:    netip.MustParseAddr(testAnotherRangeStartV6Str),
			LeaseDuration: 1 * time.Hour,
			RAAllowSLAAC:  true,
			RASLAACOnly:   true,
		},
	},
}

// disabledIPv6Config is a configuration of IPv6 part of the interfaces
// configuration that is disabled.
var disabledIPv6Config = &dhcpsvc.IPv6Config{Enabled: false}

// fullLayersStack is the complete stack of layers expected to appear in the
// DHCP response packets.
var fullLayersStack = []gopacket.LayerType{
	layers.LayerTypeEthernet,
	layers.LayerTypeIPv4,
	layers.LayerTypeUDP,
	layers.LayerTypeDHCPv4,
}

// newTempDB copies the leases database file located in the testdata FS, under
// tb.Name()/leases.json, to a temporary directory and returns the path to the
// copied file.
func newTempDB(tb testing.TB) (dst string) {
	tb.Helper()

	data, err := fs.ReadFile(testdata, path.Join(tb.Name(), testDBLeasesFilename))
	require.NoError(tb, err)

	dst = filepath.Join(tb.TempDir(), testDBLeasesFilename)

	err = os.WriteFile(dst, data, 0o640)
	require.NoError(tb, err)

	return dst
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
	if conf.DBFilePath == "" {
		conf.DBFilePath = filepath.Join(tb.TempDir(), "leases.json")
	}
	conf.ICMPTimeout = cmp.Or(conf.ICMPTimeout, testTimeout)
	if conf.Interfaces == nil {
		conf.Interfaces = testInterfaceConf
	}

	srv, err := dhcpsvc.New(testutil.ContextWithTimeout(tb, testTimeout), conf)
	require.NoError(tb, err)

	return srv
}
