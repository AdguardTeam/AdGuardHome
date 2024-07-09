package dhcpsvc_test

import (
	"net"
	"net/netip"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/stretchr/testify/require"
)

// testLocalTLD is a common local TLD for tests.
const testLocalTLD = "local"

// testTimeout is a common timeout for tests and contexts.
const testTimeout time.Duration = 10 * time.Second

// discardLog is a logger to discard test output.
var discardLog = slogutil.NewDiscardLogger()

// testInterfaceConf is a common set of interface configurations for tests.
var testInterfaceConf = map[string]*dhcpsvc.InterfaceConfig{
	"eth0": {
		IPv4: &dhcpsvc.IPv4Config{
			Enabled:       true,
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

// mustParseMAC parses a hardware address from s and requires no errors.
func mustParseMAC(t require.TestingT, s string) (mac net.HardwareAddr) {
	mac, err := net.ParseMAC(s)
	require.NoError(t, err)

	return mac
}
