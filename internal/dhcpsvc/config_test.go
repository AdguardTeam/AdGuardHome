package dhcpsvc_test

import (
	"net/netip"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
)

func TestConfig_Validate(t *testing.T) {
	validIPv4Conf := &dhcpsvc.IPv4Config{
		Enabled:       true,
		Clock:         timeutil.SystemClock{},
		GatewayIP:     netip.MustParseAddr("192.168.0.1"),
		SubnetMask:    netip.MustParseAddr("255.255.255.0"),
		RangeStart:    netip.MustParseAddr("192.168.0.2"),
		RangeEnd:      netip.MustParseAddr("192.168.0.254"),
		LeaseDuration: 1 * time.Hour,
	}
	gwInRangeConf := &dhcpsvc.IPv4Config{
		Enabled:       true,
		Clock:         timeutil.SystemClock{},
		GatewayIP:     netip.MustParseAddr("192.168.0.100"),
		SubnetMask:    netip.MustParseAddr("255.255.255.0"),
		RangeStart:    netip.MustParseAddr("192.168.0.1"),
		RangeEnd:      netip.MustParseAddr("192.168.0.254"),
		LeaseDuration: 1 * time.Hour,
	}
	badStartConf := &dhcpsvc.IPv4Config{
		Enabled:       true,
		Clock:         timeutil.SystemClock{},
		GatewayIP:     netip.MustParseAddr("192.168.0.1"),
		SubnetMask:    netip.MustParseAddr("255.255.255.0"),
		RangeStart:    netip.MustParseAddr("127.0.0.1"),
		RangeEnd:      netip.MustParseAddr("192.168.0.254"),
		LeaseDuration: 1 * time.Hour,
	}

	validIPv6Conf := &dhcpsvc.IPv6Config{
		Enabled:       true,
		RangeStart:    netip.MustParseAddr("2001:db8::1"),
		LeaseDuration: 1 * time.Hour,
		RAAllowSLAAC:  true,
		RASLAACOnly:   true,
	}

	leasesPath := filepath.Join(t.TempDir(), "leases.json")

	testCases := []struct {
		name       string
		conf       *dhcpsvc.Config
		wantErrMsg string
	}{{
		name:       "nil_config",
		conf:       nil,
		wantErrMsg: "no value",
	}, {
		name:       "disabled",
		conf:       &dhcpsvc.Config{},
		wantErrMsg: "",
	}, {
		name: "empty",
		conf: &dhcpsvc.Config{
			Enabled:    true,
			Interfaces: testInterfaceConf,
			DBFilePath: leasesPath,
		},
		wantErrMsg: `LocalDomainName: bad domain name "": domain name is empty`,
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			LocalDomainName: testLocalTLD,
			Interfaces:      nil,
			DBFilePath:      leasesPath,
		},
		name:       "no_interfaces",
		wantErrMsg: "interfaces: empty value",
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			LocalDomainName: testLocalTLD,
			Interfaces: map[string]*dhcpsvc.InterfaceConfig{
				"eth0": nil,
			},
			DBFilePath: leasesPath,
		},
		name:       "nil_interface",
		wantErrMsg: `eth0: no value`,
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			LocalDomainName: testLocalTLD,
			Interfaces: map[string]*dhcpsvc.InterfaceConfig{
				"eth0": {
					IPv4: nil,
					IPv6: &dhcpsvc.IPv6Config{Enabled: false},
				},
			},
			DBFilePath: leasesPath,
		},
		name:       "nil_ipv4",
		wantErrMsg: `eth0: ipv4: no value`,
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			LocalDomainName: testLocalTLD,
			Interfaces: map[string]*dhcpsvc.InterfaceConfig{
				"eth0": {
					IPv4: &dhcpsvc.IPv4Config{Enabled: false},
					IPv6: nil,
				},
			},
			DBFilePath: leasesPath,
		},
		name:       "nil_ipv6",
		wantErrMsg: `eth0: ipv6: no value`,
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			Logger:          discardLog,
			LocalDomainName: testLocalTLD,
			Interfaces: map[string]*dhcpsvc.InterfaceConfig{
				"eth0": {
					IPv4: validIPv4Conf,
					IPv6: validIPv6Conf,
				},
			},
			DBFilePath: leasesPath,
		},
		name:       "valid",
		wantErrMsg: "",
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			Logger:          discardLog,
			LocalDomainName: testLocalTLD,
			Interfaces: map[string]*dhcpsvc.InterfaceConfig{
				"eth0": {
					IPv4: &dhcpsvc.IPv4Config{Enabled: false},
					IPv6: &dhcpsvc.IPv6Config{Enabled: false},
				},
			},
			DBFilePath: leasesPath,
		},
		name:       "disabled_interfaces",
		wantErrMsg: "",
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			Logger:          discardLog,
			LocalDomainName: testLocalTLD,
			Interfaces: map[string]*dhcpsvc.InterfaceConfig{
				"eth0": {
					IPv4: gwInRangeConf,
					IPv6: validIPv6Conf,
				},
			},
			DBFilePath: leasesPath,
		},
		name: "gateway_within_range",
		wantErrMsg: "eth0: ipv4: gateway ip 192.168.0.100 in the ip range " +
			"192.168.0.1-192.168.0.254",
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			Logger:          discardLog,
			LocalDomainName: testLocalTLD,
			Interfaces: map[string]*dhcpsvc.InterfaceConfig{
				"eth0": {
					IPv4: badStartConf,
					IPv6: validIPv6Conf,
				},
			},
			DBFilePath: leasesPath,
		},
		name: "bad_start",
		wantErrMsg: "eth0: ipv4: range start 127.0.0.1 is not within 192.168.0.1/24" + "\n" +
			"gateway ip 192.168.0.1 in the ip range 127.0.0.1-192.168.0.254",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertErrorMsg(t, tc.wantErrMsg, tc.conf.Validate())
		})
	}
}
