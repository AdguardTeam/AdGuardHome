package dhcpsvc_test

import (
	"net/netip"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
)

// TODO(e.burkov):  Move string IP address representations into constants and
// use in the tests below.

func TestIPv4Config_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		conf       *dhcpsvc.IPv4Config
		wantErrMsg string
	}{{
		name:       "nil",
		conf:       nil,
		wantErrMsg: "no value",
	}, {
		name:       "disabled",
		conf:       &dhcpsvc.IPv4Config{Enabled: false},
		wantErrMsg: "",
	}, {
		name: "nil_clock",
		conf: &dhcpsvc.IPv4Config{
			Enabled:       true,
			Clock:         nil,
			GatewayIP:     testIPv4Conf.GatewayIP,
			SubnetMask:    testIPv4Conf.SubnetMask,
			RangeStart:    testIPv4Conf.RangeStart,
			RangeEnd:      testIPv4Conf.RangeEnd,
			LeaseDuration: testIPv4Conf.LeaseDuration,
		},
		wantErrMsg: "clock: no value",
	}, {
		name: "bad_lease_duration",
		conf: &dhcpsvc.IPv4Config{
			Enabled:       true,
			Clock:         testIPv4Conf.Clock,
			GatewayIP:     testIPv4Conf.GatewayIP,
			SubnetMask:    testIPv4Conf.SubnetMask,
			RangeStart:    testIPv4Conf.RangeStart,
			RangeEnd:      testIPv4Conf.RangeEnd,
			LeaseDuration: 0,
		},
		wantErrMsg: "lease duration: not positive: 0s",
	}, {
		name: "bad_gateway_ip",
		conf: &dhcpsvc.IPv4Config{
			Enabled:       true,
			Clock:         testIPv4Conf.Clock,
			GatewayIP:     netip.MustParseAddr(testRangeStartV6Str),
			SubnetMask:    testIPv4Conf.SubnetMask,
			RangeStart:    testIPv4Conf.RangeStart,
			RangeEnd:      testIPv4Conf.RangeEnd,
			LeaseDuration: testIPv4Conf.LeaseDuration,
		},
		wantErrMsg: "gateway ip " + testRangeStartV6Str + " must be a valid ipv4" + "\n" +
			"range start " + testRangeStartV4Str + " is not within " + testRangeStartV6Str + "/24",
	}, {
		name: "bad_subnet_mask",
		conf: &dhcpsvc.IPv4Config{
			Enabled:       true,
			Clock:         testIPv4Conf.Clock,
			GatewayIP:     testIPv4Conf.GatewayIP,
			SubnetMask:    netip.MustParseAddr(testRangeStartV6Str),
			RangeStart:    testIPv4Conf.RangeStart,
			RangeEnd:      testIPv4Conf.RangeEnd,
			LeaseDuration: testIPv4Conf.LeaseDuration,
		},
		wantErrMsg: "subnet mask " + testRangeStartV6Str + " must be a valid ipv4 cidr mask",
	}, {
		name: "bad_range_start",
		conf: &dhcpsvc.IPv4Config{
			Enabled:       true,
			Clock:         testIPv4Conf.Clock,
			GatewayIP:     testIPv4Conf.GatewayIP,
			SubnetMask:    testIPv4Conf.SubnetMask,
			RangeStart:    netip.MustParseAddr(testRangeStartV6Str),
			RangeEnd:      testIPv4Conf.RangeEnd,
			LeaseDuration: testIPv4Conf.LeaseDuration,
		},
		wantErrMsg: "range start " + testRangeStartV6Str + " must be a valid ipv4" + "\n" +
			"range start " + testRangeStartV6Str + " is not within " +
			testGatewayIPv4Str + "/24" + "\n" + "invalid ip range: " + testRangeStartV6Str +
			" and " + testRangeEndV4Str + " must be within the same address family",
	}, {
		name: "bad_range_end",
		conf: &dhcpsvc.IPv4Config{
			Enabled:       true,
			Clock:         testIPv4Conf.Clock,
			GatewayIP:     testIPv4Conf.GatewayIP,
			SubnetMask:    testIPv4Conf.SubnetMask,
			RangeStart:    testIPv4Conf.RangeStart,
			RangeEnd:      netip.MustParseAddr(testRangeStartV6Str),
			LeaseDuration: testIPv4Conf.LeaseDuration,
		},
		wantErrMsg: "range end " + testRangeStartV6Str + " must be a valid ipv4" + "\n" +
			"range end " + testRangeStartV6Str + " is not within " + testGatewayIPv4Str + "/24" +
			"\n" + "invalid ip range: " + testRangeStartV4Str + " and " + testRangeStartV6Str +
			" must be within the same address family",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			testutil.AssertErrorMsg(t, tc.wantErrMsg, tc.conf.Validate())
		})
	}
}

func TestIPv6Config_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		conf       *dhcpsvc.IPv6Config
		wantErrMsg string
	}{{
		name:       "nil",
		conf:       nil,
		wantErrMsg: "no value",
	}, {
		name:       "disabled",
		conf:       &dhcpsvc.IPv6Config{Enabled: false},
		wantErrMsg: "",
	}, {
		name: "bad_range_start",
		conf: &dhcpsvc.IPv6Config{
			Enabled:       true,
			RangeStart:    testIPv4Conf.GatewayIP,
			LeaseDuration: 1 * time.Hour,
		},
		wantErrMsg: "range start " + testGatewayIPv4Str + " should be a valid ipv6",
	}, {
		name: "bad_lease_duration",
		conf: &dhcpsvc.IPv6Config{
			Enabled:       true,
			RangeStart:    netip.MustParseAddr(testRangeStartV6Str),
			LeaseDuration: 0,
		},
		wantErrMsg: "lease duration 0s must be positive",
	}, {
		name: "valid",
		conf: &dhcpsvc.IPv6Config{
			Enabled:       true,
			RangeStart:    netip.MustParseAddr(testRangeStartV6Str),
			LeaseDuration: 1 * time.Hour,
		},
		wantErrMsg: "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			testutil.AssertErrorMsg(t, tc.wantErrMsg, tc.conf.Validate())
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	valid := &dhcpsvc.Config{
		Interfaces:           testInterfaceConf,
		NetworkDeviceManager: dhcpsvc.EmptyNetworkDeviceManager{},
		Logger:               testLogger,
		LocalDomainName:      testLocalTLD,
		DBFilePath:           filepath.Join(t.TempDir(), testDBLeasesFilename),
		ICMPTimeout:          1 * time.Second,
		Enabled:              true,
	}

	testCases := []struct {
		name       string
		conf       *dhcpsvc.Config
		wantErrMsg string
	}{{
		name:       "disabled",
		conf:       &dhcpsvc.Config{Enabled: false},
		wantErrMsg: "",
	}, {
		name:       "nil",
		conf:       nil,
		wantErrMsg: "no value",
	}, {
		name:       "valid",
		conf:       valid,
		wantErrMsg: "",
	}, {
		name: "bad_icmp_timeout",
		conf: &dhcpsvc.Config{
			Interfaces:           valid.Interfaces,
			NetworkDeviceManager: valid.NetworkDeviceManager,
			Logger:               valid.Logger,
			LocalDomainName:      valid.LocalDomainName,
			DBFilePath:           valid.DBFilePath,
			ICMPTimeout:          -1 * time.Second,
			Enabled:              valid.Enabled,
		},
		wantErrMsg: "conf.ICMPTimeout: negative value: -1s",
	}, {
		name: "bad_db_filepath",
		conf: &dhcpsvc.Config{
			Interfaces:           valid.Interfaces,
			NetworkDeviceManager: valid.NetworkDeviceManager,
			Logger:               valid.Logger,
			LocalDomainName:      valid.LocalDomainName,
			DBFilePath:           "",
			ICMPTimeout:          valid.ICMPTimeout,
			Enabled:              valid.Enabled,
		},
		wantErrMsg: "conf.DBFilePath: empty value",
	}, {
		name: "no_interfaces",
		conf: &dhcpsvc.Config{
			Interfaces:           nil,
			NetworkDeviceManager: valid.NetworkDeviceManager,
			Logger:               valid.Logger,
			LocalDomainName:      valid.LocalDomainName,
			DBFilePath:           valid.DBFilePath,
			ICMPTimeout:          valid.ICMPTimeout,
			Enabled:              valid.Enabled,
		},
		wantErrMsg: "conf.Interfaces: empty value",
	}, {
		name: "nil_network_manager",
		conf: &dhcpsvc.Config{
			Interfaces:           valid.Interfaces,
			NetworkDeviceManager: nil,
			Logger:               valid.Logger,
			LocalDomainName:      valid.LocalDomainName,
			DBFilePath:           valid.DBFilePath,
			ICMPTimeout:          valid.ICMPTimeout,
			Enabled:              valid.Enabled,
		},
		wantErrMsg: "conf.NetworkDeviceManager: no value",
	}, {
		name: "no_logger",
		conf: &dhcpsvc.Config{
			Interfaces:           valid.Interfaces,
			NetworkDeviceManager: valid.NetworkDeviceManager,
			Logger:               nil,
			LocalDomainName:      valid.LocalDomainName,
			DBFilePath:           valid.DBFilePath,
			ICMPTimeout:          valid.ICMPTimeout,
			Enabled:              valid.Enabled,
		},
		wantErrMsg: "conf.Logger: no value",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			testutil.AssertErrorMsg(t, tc.wantErrMsg, tc.conf.Validate())
		})
	}
}
