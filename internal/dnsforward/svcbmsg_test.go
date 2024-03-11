package dnsforward

import (
	"net"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestGenAnswerHTTPS_andSVCB(t *testing.T) {
	// Preconditions.

	s := createTestServer(t, &filtering.Config{
		BlockingMode: filtering.BlockingModeDefault,
	}, ServerConfig{
		Config: Config{
			UpstreamMode:     UpstreamModeLoadBalance,
			EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
		},
		ServePlainDNS: true,
	})

	req := &dns.Msg{
		Question: []dns.Question{{
			Name: "abcd",
		}},
	}

	// Constants and helper values.

	const host = "example.com"
	const prio = 32

	ip4 := net.IPv4(127, 0, 0, 1)
	ip6 := net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	// Helper functions.

	dnssvcb := func(key, value string) (svcb *rules.DNSSVCB) {
		svcb = &rules.DNSSVCB{
			Target:   host,
			Priority: prio,
		}

		if key == "" {
			return svcb
		}

		svcb.Params = map[string]string{
			key: value,
		}

		return svcb
	}

	wantsvcb := func(kv dns.SVCBKeyValue) (want *dns.SVCB) {
		want = &dns.SVCB{
			Hdr:      s.hdr(req, dns.TypeSVCB),
			Priority: prio,
			Target:   dns.Fqdn(host),
		}

		if kv == nil {
			return want
		}

		want.Value = []dns.SVCBKeyValue{kv}

		return want
	}

	// Tests.

	testCases := []struct {
		svcb *rules.DNSSVCB
		want *dns.SVCB
		name string
	}{{
		svcb: dnssvcb("", ""),
		want: wantsvcb(nil),
		name: "no_params",
	}, {
		svcb: dnssvcb("foo", "bar"),
		want: wantsvcb(nil),
		name: "invalid",
	}, {
		svcb: dnssvcb("alpn", "h3"),
		want: wantsvcb(&dns.SVCBAlpn{Alpn: []string{"h3"}}),
		name: "alpn",
	}, {
		svcb: dnssvcb("ech", "AAAA"),
		want: wantsvcb(&dns.SVCBECHConfig{ECH: []byte{0, 0, 0}}),
		name: "ech",
	}, {
		svcb: dnssvcb("echconfig", "AAAA"),
		want: wantsvcb(&dns.SVCBECHConfig{ECH: []byte{0, 0, 0}}),
		name: "ech_deprecated",
	}, {
		svcb: dnssvcb("echconfig", "%BAD%"),
		want: wantsvcb(nil),
		name: "ech_invalid",
	}, {
		svcb: dnssvcb("ipv4hint", "127.0.0.1"),
		want: wantsvcb(&dns.SVCBIPv4Hint{Hint: []net.IP{ip4}}),
		name: "ipv4hint",
	}, {
		svcb: dnssvcb("ipv4hint", "127.0.01"),
		want: wantsvcb(nil),
		name: "ipv4hint_invalid",
	}, {
		svcb: dnssvcb("ipv6hint", "::1"),
		want: wantsvcb(&dns.SVCBIPv6Hint{Hint: []net.IP{ip6}}),
		name: "ipv6hint",
	}, {
		svcb: dnssvcb("ipv6hint", ":::1"),
		want: wantsvcb(nil),
		name: "ipv6hint_invalid",
	}, {
		svcb: dnssvcb("mandatory", "alpn"),
		want: wantsvcb(&dns.SVCBMandatory{Code: []dns.SVCBKey{dns.SVCB_ALPN}}),
		name: "mandatory",
	}, {
		svcb: dnssvcb("mandatory", "alpnn"),
		want: wantsvcb(nil),
		name: "mandatory_invalid",
	}, {
		svcb: dnssvcb("no-default-alpn", ""),
		want: wantsvcb(&dns.SVCBNoDefaultAlpn{}),
		name: "no_default_alpn",
	}, {
		svcb: dnssvcb("dohpath", "/dns-query"),
		want: wantsvcb(&dns.SVCBDoHPath{Template: "/dns-query"}),
		name: "dohpath",
	}, {
		svcb: dnssvcb("port", "8080"),
		want: wantsvcb(&dns.SVCBPort{Port: 8080}),
		name: "port",
	}, {
		svcb: dnssvcb("port", "1005008080"),
		want: wantsvcb(nil),
		name: "bad_port",
	}}

	for _, tc := range testCases {
		t.Run("https", func(t *testing.T) {
			t.Run(tc.name, func(t *testing.T) {
				want := &dns.HTTPS{SVCB: *tc.want}
				want.Hdr.Rrtype = dns.TypeHTTPS

				got := s.genAnswerHTTPS(req, tc.svcb)
				assert.Equal(t, want, got)
			})
		})

		t.Run("svcb", func(t *testing.T) {
			t.Run(tc.name, func(t *testing.T) {
				got := s.genAnswerSVCB(req, tc.svcb)
				assert.Equal(t, tc.want, got)
			})
		})
	}
}
