package dnsforward

import (
	"net"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleDNSRequest_filterDNSResponse(t *testing.T) {
	rules := `
||blocked.domain^
@@||allowed.domain^
||cname.specific^$dnstype=~CNAME
||0.0.0.1^$dnstype=~A
||::1^$dnstype=~AAAA
0.0.0.0 duplicate.domain
0.0.0.0 duplicate.domain
`

	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			BlockingMode:      BlockingModeDefault,
			EDNSClientSubnet: &EDNSClientSubnet{
				Enabled: false,
			},
		},
	}
	filters := []filtering.Filter{{
		ID: 0, Data: []byte(rules),
	}}

	f, err := filtering.New(&filtering.Config{}, filters)
	require.NoError(t, err)
	f.SetEnabled(true)

	s, err := NewServer(DNSCreateParams{
		DHCPServer:  testDHCP,
		DNSFilter:   f,
		PrivateNets: netutil.SubnetSetFunc(netutil.IsLocallyServed),
	})
	require.NoError(t, err)

	err = s.Prepare(&forwardConf)
	require.NoError(t, err)

	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&aghtest.Upstream{
			CName: map[string][]string{
				"cname.exception.": {"cname.specific."},
				"should.block.":    {"blocked.domain."},
				"allowed.first.":   {"allowed.domain.", "blocked.domain."},
				"blocked.first.":   {"blocked.domain.", "allowed.domain."},
			},
			IPv4: map[string][]net.IP{
				"a.exception.": {{0, 0, 0, 1}},
			},
			IPv6: map[string][]net.IP{
				"aaaa.exception.": {net.ParseIP("::1")},
			},
		},
	}
	startDeferStop(t, s)

	testCases := []struct {
		req     *dns.Msg
		name    string
		wantAns []dns.RR
	}{{
		req:  createTestMessage("cname.exception."),
		name: "cname_exception",
		wantAns: []dns.RR{&dns.CNAME{
			Hdr: dns.RR_Header{
				Name:   "cname.exception.",
				Rrtype: dns.TypeCNAME,
			},
			Target: "cname.specific.",
		}},
	}, {
		req:  createTestMessage("should.block."),
		name: "blocked_by_cname",
		wantAns: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   "should.block.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
			},
			A: netutil.IPv4Zero(),
		}},
	}, {
		req:  createTestMessage("a.exception."),
		name: "a_exception",
		wantAns: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   "a.exception.",
				Rrtype: dns.TypeA,
			},
			A: net.IP{0, 0, 0, 1},
		}},
	}, {
		req:  createTestMessageWithType("aaaa.exception.", dns.TypeAAAA),
		name: "aaaa_exception",
		wantAns: []dns.RR{&dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   "aaaa.exception.",
				Rrtype: dns.TypeAAAA,
			},
			AAAA: net.ParseIP("::1"),
		}},
	}, {
		req:  createTestMessage("allowed.first."),
		name: "allowed_first",
		wantAns: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   "allowed.first.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
			},
			A: netutil.IPv4Zero(),
		}},
	}, {
		req:  createTestMessage("blocked.first."),
		name: "blocked_first",
		wantAns: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   "blocked.first.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
			},
			A: netutil.IPv4Zero(),
		}},
	}, {
		req:  createTestMessage("duplicate.domain."),
		name: "duplicate_domain",
		wantAns: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   "duplicate.domain.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
			},
			A: netutil.IPv4Zero(),
		}},
	}}

	for _, tc := range testCases {
		dctx := &proxy.DNSContext{
			Proto: proxy.ProtoUDP,
			Req:   tc.req,
			Addr:  &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 1},
		}

		t.Run(tc.name, func(t *testing.T) {
			err = s.handleDNSRequest(nil, dctx)
			require.NoError(t, err)
			require.NotNil(t, dctx.Res)

			assert.Equal(t, tc.wantAns, dctx.Res.Answer)
		})
	}
}
