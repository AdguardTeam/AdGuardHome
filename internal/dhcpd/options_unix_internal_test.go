//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/stretchr/testify/assert"
)

func TestParseOpt(t *testing.T) {
	testCases := []struct {
		name       string
		in         string
		wantCode   dhcpv4.OptionCode
		wantVal    dhcpv4.OptionValue
		wantErrMsg string
	}{{
		name:     "hex_success",
		in:       "6 hex c0a80101c0a80102",
		wantCode: dhcpv4.GenericOptionCode(6),
		wantVal: dhcpv4.OptionGeneric{Data: []byte{
			0xC0, 0xA8, 0x01, 0x01,
			0xC0, 0xA8, 0x01, 0x02,
		}},
		wantErrMsg: "",
	}, {
		name:       "ip_success",
		in:         "6 ip 1.2.3.4",
		wantCode:   dhcpv4.GenericOptionCode(6),
		wantVal:    dhcpv4.IP(net.IP{0x01, 0x02, 0x03, 0x04}),
		wantErrMsg: "",
	}, {
		name:     "ips_success",
		in:       "6 ips 192.168.1.1,192.168.1.2",
		wantCode: dhcpv4.GenericOptionCode(6),
		wantVal: dhcpv4.IPs([]net.IP{
			{0xC0, 0xA8, 0x01, 0x01},
			{0xC0, 0xA8, 0x01, 0x02},
		}),
		wantErrMsg: "",
	}, {
		name:       "text_success",
		in:         "252 text http://192.168.1.1/",
		wantCode:   dhcpv4.GenericOptionCode(252),
		wantVal:    dhcpv4.String("http://192.168.1.1/"),
		wantErrMsg: "",
	}, {
		name:       "del_success",
		in:         "61 del",
		wantCode:   dhcpv4.GenericOptionCode(dhcpv4.OptionClientIdentifier),
		wantVal:    dhcpv4.OptionGeneric{Data: nil},
		wantErrMsg: "",
	}, {
		name:       "bool_success",
		in:         "19 bool true",
		wantCode:   dhcpv4.GenericOptionCode(dhcpv4.OptionIPForwarding),
		wantVal:    dhcpv4.OptionGeneric{Data: []byte{0x01}},
		wantErrMsg: "",
	}, {
		name:       "bool_success_false",
		in:         "19 bool F",
		wantCode:   dhcpv4.GenericOptionCode(dhcpv4.OptionIPForwarding),
		wantVal:    dhcpv4.OptionGeneric{Data: []byte{0x00}},
		wantErrMsg: "",
	}, {
		name:       "dur_success",
		in:         "24 dur 2h5s",
		wantCode:   dhcpv4.GenericOptionCode(dhcpv4.OptionPathMTUAgingTimeout),
		wantVal:    dhcpv4.Duration(2*time.Hour + 5*time.Second),
		wantErrMsg: "",
	}, {
		name:       "u8_success",
		in:         "23 u8 64",
		wantCode:   dhcpv4.GenericOptionCode(dhcpv4.OptionDefaultIPTTL),
		wantVal:    dhcpv4.OptionGeneric{Data: []byte{0x40}},
		wantErrMsg: "",
	}, {
		name:       "u16_success",
		in:         "22 u16 1234",
		wantCode:   dhcpv4.GenericOptionCode(dhcpv4.OptionMaximumDatagramAssemblySize),
		wantVal:    dhcpv4.Uint16(1234),
		wantErrMsg: "",
	}, {
		name:       "bad_parts",
		in:         "6 ip",
		wantCode:   nil,
		wantVal:    nil,
		wantErrMsg: `invalid option string "6 ip": bad option format`,
	}, {
		name:     "bad_code",
		in:       "256 ip 1.1.1.1",
		wantCode: nil,
		wantVal:  nil,
		wantErrMsg: `invalid option string "256 ip 1.1.1.1": parsing option code: ` +
			`strconv.ParseUint: parsing "256": value out of range`,
	}, {
		name:       "bad_type",
		in:         "6 bad 1.1.1.1",
		wantCode:   nil,
		wantVal:    nil,
		wantErrMsg: `invalid option string "6 bad 1.1.1.1": unknown option type "bad"`,
	}, {
		name:     "hex_error",
		in:       "6 hex ZZZ",
		wantCode: nil,
		wantVal:  nil,
		wantErrMsg: `invalid option string "6 hex ZZZ": decoding hex: ` +
			`encoding/hex: invalid byte: U+005A 'Z'`,
	}, {
		name:       "ip_error",
		in:         "6 ip 1.2.3.x",
		wantCode:   nil,
		wantVal:    nil,
		wantErrMsg: "invalid option string \"6 ip 1.2.3.x\": bad ipv4 address \"1.2.3.x\"",
	}, {
		name:       "ip_error_v6",
		in:         "6 ip ::1234",
		wantCode:   nil,
		wantVal:    nil,
		wantErrMsg: "invalid option string \"6 ip ::1234\": bad ipv4 address \"::1234\"",
	}, {
		name:     "ips_error",
		in:       "6 ips 192.168.1.1,192.168.1.x",
		wantCode: nil,
		wantVal:  nil,
		wantErrMsg: "invalid option string \"6 ips 192.168.1.1,192.168.1.x\": " +
			"parsing ip at index 1: bad ipv4 address \"192.168.1.x\"",
	}, {
		name:     "bool_error",
		in:       "19 bool yes",
		wantCode: nil,
		wantVal:  nil,
		wantErrMsg: "invalid option string \"19 bool yes\": decoding bool: " +
			"strconv.ParseBool: parsing \"yes\": invalid syntax",
	}, {
		name:     "dur_error",
		in:       "24 dur 3y",
		wantCode: nil,
		wantVal:  nil,
		wantErrMsg: `invalid option string "24 dur 3y": decoding dur: time: ` +
			`unknown unit "y" in duration "3y"`,
	}, {
		name:     "u8_error",
		in:       "23 u8 256",
		wantCode: nil,
		wantVal:  nil,
		wantErrMsg: "invalid option string \"23 u8 256\": decoding u8: " +
			"strconv.ParseUint: parsing \"256\": value out of range",
	}, {
		name:     "u16_error",
		in:       "23 u16 65536",
		wantCode: nil,
		wantVal:  nil,
		wantErrMsg: "invalid option string \"23 u16 65536\": decoding u16: " +
			"strconv.ParseUint: parsing \"65536\": value out of range",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code, val, err := parseDHCPOption(tc.in)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			assert.Equal(t, tc.wantCode, code)
			assert.Equal(t, tc.wantVal, val)
		})
	}
}

func TestPrepareOptions(t *testing.T) {
	oneIP, otherIP := net.IP{1, 2, 3, 4}, net.IP{5, 6, 7, 8}

	testCases := []struct {
		name         string
		wantExplicit dhcpv4.Options
		opts         []string
	}{{
		name:         "all_default",
		wantExplicit: nil,
		opts:         nil,
	}, {
		name: "configured_ip",
		wantExplicit: dhcpv4.OptionsFromList(
			dhcpv4.OptBroadcastAddress(oneIP),
		),
		opts: []string{
			fmt.Sprintf("%d ip %s", dhcpv4.OptionBroadcastAddress, oneIP),
		},
	}, {
		name: "configured_ips",
		wantExplicit: dhcpv4.OptionsFromList(
			dhcpv4.Option{
				Code:  dhcpv4.OptionDomainNameServer,
				Value: dhcpv4.IPs{oneIP, otherIP},
			},
		),
		opts: []string{
			fmt.Sprintf("%d ips %s,%s", dhcpv4.OptionDomainNameServer, oneIP, otherIP),
		},
	}, {
		name:         "configured_bad",
		wantExplicit: nil,
		opts: []string{
			"19 bool yes",
			"24 dur 3y",
			"23 u8 256",
			"23 u16 65536",
			"20 hex",
			"23 hex abc",
			"32 ips 1,2,3,4",
			"28 256.256.256.256",
		},
	}, {
		name: "configured_del",
		wantExplicit: dhcpv4.OptionsFromList(
			dhcpv4.OptBroadcastAddress(nil),
		),
		opts: []string{
			"28 del",
		},
	}, {
		name: "rewritten_del",
		wantExplicit: dhcpv4.OptionsFromList(
			dhcpv4.OptBroadcastAddress(netutil.IPv4bcast()),
		),
		opts: []string{
			"28 del",
			"28 ip 255.255.255.255",
		},
	}, {
		name: "configured_and_del",
		wantExplicit: dhcpv4.OptionsFromList(
			dhcpv4.Option{
				Code:  dhcpv4.OptionGeoConf,
				Value: dhcpv4.String("cba"),
			},
		),
		opts: []string{
			"123 text abc",
			"123 del",
			"123 text cba",
		},
	}}

	for _, tc := range testCases {
		s := &v4Server{
			conf: &V4ServerConf{
				Options: tc.opts,
			},
		}

		t.Run(tc.name, func(t *testing.T) {
			s.prepareOptions()

			assert.Equal(t, tc.wantExplicit, s.explicitOpts)

			for c := range s.explicitOpts {
				assert.NotContains(t, s.implicitOpts, c)
			}
		})
	}
}
