//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"fmt"
	"net"
	"testing"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOpt(t *testing.T) {
	testCases := []struct {
		name       string
		in         string
		wantErrMsg string
		wantOpt    dhcpv4.Option
	}{{
		name:       "hex_success",
		in:         "6 hex c0a80101c0a80102",
		wantErrMsg: "",
		wantOpt: dhcpv4.OptDNS(
			net.IP{0xC0, 0xA8, 0x01, 0x01},
			net.IP{0xC0, 0xA8, 0x01, 0x02},
		),
	}, {
		name:       "ip_success",
		in:         "6 ip 1.2.3.4",
		wantErrMsg: "",
		wantOpt: dhcpv4.OptDNS(
			net.IP{0x01, 0x02, 0x03, 0x04},
		),
	}, {
		name:       "ip_fail_v6",
		in:         "6 ip ::1234",
		wantErrMsg: "invalid option string \"6 ip ::1234\": bad ipv4 address \"::1234\"",
		wantOpt:    dhcpv4.Option{},
	}, {
		name:       "ips_success",
		in:         "6 ips 192.168.1.1,192.168.1.2",
		wantErrMsg: "",
		wantOpt: dhcpv4.OptDNS(
			net.IP{0xC0, 0xA8, 0x01, 0x01},
			net.IP{0xC0, 0xA8, 0x01, 0x02},
		),
	}, {
		name:       "text_success",
		in:         "252 text http://192.168.1.1/",
		wantErrMsg: "",
		wantOpt: dhcpv4.OptGeneric(
			dhcpv4.GenericOptionCode(252),
			[]byte("http://192.168.1.1/"),
		),
	}, {
		name:       "bad_parts",
		in:         "6 ip",
		wantErrMsg: `invalid option string "6 ip": need at least three fields`,
		wantOpt:    dhcpv4.Option{},
	}, {
		name: "bad_code",
		in:   "256 ip 1.1.1.1",
		wantErrMsg: `invalid option string "256 ip 1.1.1.1": parsing option code: ` +
			`strconv.ParseUint: parsing "256": value out of range`,
		wantOpt: dhcpv4.Option{},
	}, {
		name:       "bad_type",
		in:         "6 bad 1.1.1.1",
		wantErrMsg: `invalid option string "6 bad 1.1.1.1": unknown option type "bad"`,
		wantOpt:    dhcpv4.Option{},
	}, {
		name: "hex_error",
		in:   "6 hex ZZZ",
		wantErrMsg: `invalid option string "6 hex ZZZ": decoding hex: ` +
			`encoding/hex: invalid byte: U+005A 'Z'`,
		wantOpt: dhcpv4.Option{},
	}, {
		name:       "ip_error",
		in:         "6 ip 1.2.3.x",
		wantErrMsg: "invalid option string \"6 ip 1.2.3.x\": bad ipv4 address \"1.2.3.x\"",
		wantOpt:    dhcpv4.Option{},
	}, {
		name: "ips_error",
		in:   "6 ips 192.168.1.1,192.168.1.x",
		wantErrMsg: "invalid option string \"6 ips 192.168.1.1,192.168.1.x\": " +
			"parsing ip at index 1: bad ipv4 address \"192.168.1.x\"",
		wantOpt: dhcpv4.Option{},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opt, err := parseDHCPOption(tc.in)
			if tc.wantErrMsg != "" {
				require.Error(t, err)

				assert.Equal(t, tc.wantErrMsg, err.Error())

				return
			}

			require.NoError(t, err)

			assert.Equal(t, tc.wantOpt.Code.Code(), opt.Code.Code())
			assert.Equal(t, tc.wantOpt.Value.ToBytes(), opt.Value.ToBytes())
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
		opts   []string
		checks dhcpv4.Options
	}{{
		name:   "all_default",
		checks: allDefault,
	}, {
		name: "configured_ip",
		opts: []string{
			fmt.Sprintf("%d ip %s", dhcpv4.OptionBroadcastAddress, oneIP),
		},
		checks: dhcpv4.Options{
			dhcpv4.OptionBroadcastAddress.Code(): oneIP,
		},
	}, {
		name: "configured_ips",
		opts: []string{
			fmt.Sprintf("%d ips %s,%s", dhcpv4.OptionDomainNameServer, oneIP, otherIP),
		},
		checks: dhcpv4.Options{
			dhcpv4.OptionDomainNameServer.Code(): append(oneIP, otherIP...),
		},
	}, {
		name: "configured_bad",
		opts: []string{
			"20 hex",
			"23 hex abc",
			"32 ips 1,2,3,4",
			"28 256.256.256.256",
		},
		checks: allDefault,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := prepareOptions(V4ServerConf{
				// Just to avoid nil pointer dereference.
				subnet:  &net.IPNet{},
				Options: tc.opts,
			})
			for c, v := range tc.checks {
				optVal := opts.Get(dhcpv4.GenericOptionCode(c))
				require.NotNil(t, optVal)

				assert.Len(t, optVal, len(v))
				assert.Equal(t, v, optVal)
			}
		})
	}
}
