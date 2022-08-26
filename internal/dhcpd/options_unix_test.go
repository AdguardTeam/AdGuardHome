//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"fmt"
	"net"
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/stretchr/testify/assert"
)

func TestParseOpt(t *testing.T) {
	testCases := []struct {
		name       string
		in         string
		wantOpt    dhcpv4.Option
		wantErrMsg string
	}{{
		name: "hex_success",
		in:   "6 hex c0a80101c0a80102",
		wantOpt: dhcpv4.OptGeneric(
			dhcpv4.GenericOptionCode(6),
			[]byte{
				0xC0, 0xA8, 0x01, 0x01,
				0xC0, 0xA8, 0x01, 0x02,
			},
		),
		wantErrMsg: "",
	}, {
		name: "ip_success",
		in:   "6 ip 1.2.3.4",
		wantOpt: dhcpv4.Option{
			Code:  dhcpv4.GenericOptionCode(6),
			Value: dhcpv4.IP(net.IP{0x01, 0x02, 0x03, 0x04}),
		},
		wantErrMsg: "",
	}, {
		name:       "ip_fail_v6",
		in:         "6 ip ::1234",
		wantOpt:    dhcpv4.Option{},
		wantErrMsg: "invalid option string \"6 ip ::1234\": bad ipv4 address \"::1234\"",
	}, {
		name: "ips_success",
		in:   "6 ips 192.168.1.1,192.168.1.2",
		wantOpt: dhcpv4.Option{
			Code: dhcpv4.GenericOptionCode(6),
			Value: dhcpv4.IPs([]net.IP{
				{0xC0, 0xA8, 0x01, 0x01},
				{0xC0, 0xA8, 0x01, 0x02},
			}),
		},
		wantErrMsg: "",
	}, {
		name: "text_success",
		in:   "252 text http://192.168.1.1/",
		wantOpt: dhcpv4.OptGeneric(
			dhcpv4.GenericOptionCode(252),
			[]byte("http://192.168.1.1/"),
		),
		wantErrMsg: "",
	}, {
		name: "del_success",
		in:   "61 del",
		wantOpt: dhcpv4.Option{
			Code:  dhcpv4.GenericOptionCode(dhcpv4.OptionClientIdentifier),
			Value: dhcpv4.OptionGeneric{Data: nil},
		},
		wantErrMsg: "",
	}, {
		name:       "bad_parts",
		in:         "6 ip",
		wantOpt:    dhcpv4.Option{},
		wantErrMsg: `invalid option string "6 ip": bad option format`,
	}, {
		name:    "bad_code",
		in:      "256 ip 1.1.1.1",
		wantOpt: dhcpv4.Option{},
		wantErrMsg: `invalid option string "256 ip 1.1.1.1": parsing option code: ` +
			`strconv.ParseUint: parsing "256": value out of range`,
	}, {
		name:       "bad_type",
		in:         "6 bad 1.1.1.1",
		wantOpt:    dhcpv4.Option{},
		wantErrMsg: `invalid option string "6 bad 1.1.1.1": unknown option type "bad"`,
	}, {
		name:    "hex_error",
		in:      "6 hex ZZZ",
		wantOpt: dhcpv4.Option{},
		wantErrMsg: `invalid option string "6 hex ZZZ": decoding hex: ` +
			`encoding/hex: invalid byte: U+005A 'Z'`,
	}, {
		name:       "ip_error",
		in:         "6 ip 1.2.3.x",
		wantOpt:    dhcpv4.Option{},
		wantErrMsg: "invalid option string \"6 ip 1.2.3.x\": bad ipv4 address \"1.2.3.x\"",
	}, {
		name:    "ips_error",
		in:      "6 ips 192.168.1.1,192.168.1.x",
		wantOpt: dhcpv4.Option{},
		wantErrMsg: "invalid option string \"6 ips 192.168.1.1,192.168.1.x\": " +
			"parsing ip at index 1: bad ipv4 address \"192.168.1.x\"",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opt, err := parseDHCPOption(tc.in)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			// assert.Equal(t, tc.wantOpt.Code.Code(), opt.Code.Code())
			// assert.Equal(t, tc.wantOpt.Value.ToBytes(), opt.Value.ToBytes())
			assert.Equal(t, tc.wantOpt, opt)
		})
	}
}

func TestPrepareOptions(t *testing.T) {
	allDefault := dhcpv4.Options{
		dhcpv4.OptionNonLocalSourceRouting.Code():     []byte{0},
		dhcpv4.OptionDefaultIPTTL.Code():              []byte{64},
		dhcpv4.OptionPerformMaskDiscovery.Code():      []byte{0},
		dhcpv4.OptionMaskSupplier.Code():              []byte{0},
		dhcpv4.OptionPerformRouterDiscovery.Code():    []byte{1},
		dhcpv4.OptionRouterSolicitationAddress.Code(): []byte{224, 0, 0, 2},
		dhcpv4.OptionBroadcastAddress.Code():          []byte{255, 255, 255, 255},
		dhcpv4.OptionTrailerEncapsulation.Code():      []byte{0},
		dhcpv4.OptionEthernetEncapsulation.Code():     []byte{0},
		dhcpv4.OptionTCPKeepaliveInterval.Code():      []byte{0, 0, 0, 0},
		dhcpv4.OptionTCPKeepaliveGarbage.Code():       []byte{0},
	}
	oneIP, otherIP := net.IP{1, 2, 3, 4}, net.IP{5, 6, 7, 8}

	testCases := []struct {
		name   string
		checks dhcpv4.Options
		opts   []string
	}{{
		name:   "all_default",
		checks: allDefault,
		opts:   nil,
	}, {
		name: "configured_ip",
		checks: dhcpv4.Options{
			dhcpv4.OptionBroadcastAddress.Code(): oneIP,
		},
		opts: []string{
			fmt.Sprintf("%d ip %s", dhcpv4.OptionBroadcastAddress, oneIP),
		},
	}, {
		name: "configured_ips",
		checks: dhcpv4.Options{
			dhcpv4.OptionDomainNameServer.Code(): append(oneIP, otherIP...),
		},
		opts: []string{
			fmt.Sprintf("%d ips %s,%s", dhcpv4.OptionDomainNameServer, oneIP, otherIP),
		},
	}, {
		name:   "configured_bad",
		checks: allDefault,
		opts: []string{
			"20 hex",
			"23 hex abc",
			"32 ips 1,2,3,4",
			"28 256.256.256.256",
		},
	}, {
		name: "configured_del",
		checks: dhcpv4.Options{
			dhcpv4.OptionBroadcastAddress.Code(): nil,
		},
		opts: []string{
			"28 del",
		},
	}, {
		name: "rewritten_del",
		checks: dhcpv4.Options{
			dhcpv4.OptionBroadcastAddress.Code(): []byte{255, 255, 255, 255},
		},
		opts: []string{
			"28 del",
			"28 ip 255.255.255.255",
		},
	}, {
		name: "configured_and_del",
		checks: dhcpv4.Options{
			123: []byte("cba"),
		},
		opts: []string{
			"123 text abc",
			"123 del",
			"123 text cba",
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "configured_del" {
				assert.True(t, true)
			}
			opts := prepareOptions(V4ServerConf{
				// Just to avoid nil pointer dereference.
				subnet:  &net.IPNet{},
				Options: tc.opts,
			})
			for c, v := range tc.checks {
				val := opts.Get(dhcpv4.GenericOptionCode(c))
				assert.Lenf(t, val, len(v), "Code: %v", c)
				assert.Equal(t, v, val)
			}
		})
	}
}
