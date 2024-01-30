package aghnet_test

import (
	"net"
	"net/netip"
	"net/url"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

func TestParseAddrPort(t *testing.T) {
	const defaultPort = 1

	v4addr := netip.MustParseAddr("1.2.3.4")

	testCases := []struct {
		name       string
		input      string
		wantErrMsg string
		want       netip.AddrPort
	}{{
		name:       "success_ip",
		input:      v4addr.String(),
		wantErrMsg: "",
		want:       netip.AddrPortFrom(v4addr, defaultPort),
	}, {
		name:       "success_ip_port",
		input:      netutil.JoinHostPort(v4addr.String(), 5),
		wantErrMsg: "",
		want:       netip.AddrPortFrom(v4addr, 5),
	}, {
		name: "success_url",
		input: (&url.URL{
			Scheme: "tcp",
			Host:   v4addr.String(),
		}).String(),
		wantErrMsg: "",
		want:       netip.AddrPortFrom(v4addr, defaultPort),
	}, {
		name: "success_url_port",
		input: (&url.URL{
			Scheme: "tcp",
			Host:   netutil.JoinHostPort(v4addr.String(), 5),
		}).String(),
		wantErrMsg: "",
		want:       netip.AddrPortFrom(v4addr, 5),
	}, {
		name:  "error_invalid_ip",
		input: "256.256.256.256",
		wantErrMsg: `not an ip:port
ParseAddr("256.256.256.256"): IPv4 field has value >255`,
		want: netip.AddrPort{},
	}, {
		name:  "error_invalid_port",
		input: net.JoinHostPort(v4addr.String(), "-5"),
		wantErrMsg: `invalid port "-5" parsing "1.2.3.4:-5"
ParseAddr("1.2.3.4:-5"): unexpected character (at ":-5")`,
		want: netip.AddrPort{},
	}, {
		name:  "error_invalid_url",
		input: "tcp:://1.2.3.4",
		wantErrMsg: `invalid port "//1.2.3.4" parsing "tcp:://1.2.3.4"
ParseAddr("tcp:://1.2.3.4"): each colon-separated field must have at least ` +
			`one digit (at "tcp:://1.2.3.4")`,
		want: netip.AddrPort{},
	}, {
		name:  "empty",
		input: "",
		want:  netip.AddrPort{},
		wantErrMsg: `not an ip:port
ParseAddr(""): unable to parse IP`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ap, err := aghnet.ParseAddrPort(tc.input, defaultPort)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			assert.Equal(t, tc.want, ap)
		})
	}
}
