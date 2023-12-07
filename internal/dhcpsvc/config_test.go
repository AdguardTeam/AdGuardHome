package dhcpsvc_test

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
)

func TestConfig_Validate(t *testing.T) {
	testCases := []struct {
		name       string
		conf       *dhcpsvc.Config
		wantErrMsg string
	}{{
		name:       "nil_config",
		conf:       nil,
		wantErrMsg: "config is nil",
	}, {
		name:       "disabled",
		conf:       &dhcpsvc.Config{},
		wantErrMsg: "",
	}, {
		name: "empty",
		conf: &dhcpsvc.Config{
			Enabled: true,
		},
		wantErrMsg: `bad domain name "": domain name is empty`,
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			LocalDomainName: testLocalTLD,
			Interfaces:      nil,
		},
		name:       "no_interfaces",
		wantErrMsg: "no interfaces specified",
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			LocalDomainName: testLocalTLD,
			Interfaces:      nil,
		},
		name:       "no_interfaces",
		wantErrMsg: "no interfaces specified",
	}, {
		conf: &dhcpsvc.Config{
			Enabled:         true,
			LocalDomainName: testLocalTLD,
			Interfaces: map[string]*dhcpsvc.InterfaceConfig{
				"eth0": nil,
			},
		},
		name:       "nil_interface",
		wantErrMsg: `interface "eth0": config is nil`,
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
		},
		name:       "nil_ipv4",
		wantErrMsg: `interface "eth0": ipv4: config is nil`,
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
		},
		name:       "nil_ipv6",
		wantErrMsg: `interface "eth0": ipv6: config is nil`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertErrorMsg(t, tc.wantErrMsg, tc.conf.Validate())
		})
	}
}
