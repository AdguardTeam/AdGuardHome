package dhcpsvc_test

import (
	"cmp"
	"io/fs"
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

// testLocalTLD is a common local TLD for tests.
const testLocalTLD = "local"

// testIfaceName is the name of the test network interface.
const testIfaceName = "iface0"

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

// testIPv4Conf is a common valid IPv4 part of the interface configuration for
// tests.
var testIPv4Conf = &dhcpsvc.IPv4Config{
	Enabled:       true,
	Clock:         timeutil.SystemClock{},
	GatewayIP:     netip.MustParseAddr("192.168.0.1"),
	SubnetMask:    netip.MustParseAddr("255.255.255.0"),
	RangeStart:    netip.MustParseAddr("192.168.0.2"),
	RangeEnd:      netip.MustParseAddr("192.168.0.254"),
	LeaseDuration: testLeaseTTL,
}

// testIPv6Conf is a common valid IPv6 part of the interface configuration for
// tests.
var testIPv6Conf = &dhcpsvc.IPv6Config{
	Enabled:       true,
	RangeStart:    netip.MustParseAddr("2001:db8::1"),
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
			GatewayIP:     netip.MustParseAddr("172.16.0.1"),
			SubnetMask:    netip.MustParseAddr("255.255.255.0"),
			RangeStart:    netip.MustParseAddr("172.16.0.2"),
			RangeEnd:      netip.MustParseAddr("172.16.0.255"),
			LeaseDuration: 1 * time.Hour,
		},
		IPv6: &dhcpsvc.IPv6Config{
			Enabled:       true,
			RangeStart:    netip.MustParseAddr("2001:db9::1"),
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

	const filename = "leases.json"

	data, err := fs.ReadFile(testdata, path.Join(tb.Name(), filename))
	require.NoError(tb, err)

	dst = filepath.Join(tb.TempDir(), filename)

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
