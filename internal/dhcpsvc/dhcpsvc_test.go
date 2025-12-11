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
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/require"
)

// testLocalTLD is a common local TLD for tests.
const testLocalTLD = "local"

// testTimeout is a common timeout for tests and contexts.
const testTimeout time.Duration = 10 * time.Second

// testLogger is a common logger for tests.
var testLogger = slogutil.NewDiscardLogger()

// testdata is a filesystem containing data for tests.
var testdata = os.DirFS("testdata")

// testInterfaceConf is a common set of interface configurations for tests.
var testInterfaceConf = map[string]*dhcpsvc.InterfaceConfig{
	"eth0": {
		IPv4: &dhcpsvc.IPv4Config{
			Enabled:       true,
			Clock:         timeutil.SystemClock{},
			GatewayIP:     netip.MustParseAddr("192.168.0.1"),
			SubnetMask:    netip.MustParseAddr("255.255.255.0"),
			RangeStart:    netip.MustParseAddr("192.168.0.2"),
			RangeEnd:      netip.MustParseAddr("192.168.0.254"),
			LeaseDuration: 1 * time.Hour,
		},
		IPv6: &dhcpsvc.IPv6Config{
			Enabled:       true,
			RangeStart:    netip.MustParseAddr("2001:db8::1"),
			LeaseDuration: 1 * time.Hour,
			RAAllowSLAAC:  true,
			RASLAACOnly:   true,
		},
	},
	"eth1": {
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

// newTempDB copies the leases database file located in the testdata FS, under
// tb.Name()/leases.json, to a temporary directory and returns the path to the
// copied file.
func newTempDB(tb testing.TB) (dst string) {
	tb.Helper()

	const filename = "leases.json"

	data, err := fs.ReadFile(testdata, path.Join(tb.Name(), filename))
	require.NoError(tb, err)

	dst = filepath.Join(tb.TempDir(), filename)

	err = os.WriteFile(dst, data, dhcpsvc.DatabasePerm)
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
	if conf.Interfaces == nil {
		conf.Interfaces = testInterfaceConf
	}
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

	srv, err := dhcpsvc.New(testutil.ContextWithTimeout(tb, testTimeout), conf)
	require.NoError(tb, err)

	return srv
}
