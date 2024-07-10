package dnsforward

import (
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleDNSRequest_handleDNSRequest(t *testing.T) {
	rules := `
||blocked.domain^
@@||allowed.domain^
||cname.specific^$dnstype=~CNAME
||0.0.0.1^$dnstype=~A
||::1^$dnstype=~AAAA
0.0.0.0 duplicate.domain
0.0.0.0 duplicate.domain
0.0.0.0 blocked.by.hostrule
`

	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		Config: Config{
			UpstreamMode: UpstreamModeLoadBalance,
			EDNSClientSubnet: &EDNSClientSubnet{
				Enabled: false,
			},
		},
		ServePlainDNS: true,
	}
	filters := []filtering.Filter{{
		ID: 0, Data: []byte(rules),
	}}

	f, err := filtering.New(&filtering.Config{
		ProtectionEnabled: true,
		BlockingMode:      filtering.BlockingModeDefault,
	}, filters)
	require.NoError(t, err)
	f.SetEnabled(true)

	s, err := NewServer(DNSCreateParams{
		DHCPServer: &testDHCP{
			OnEnabled:  func() (ok bool) { return false },
			OnHostByIP: func(ip netip.Addr) (host string) { panic("not implemented") },
			OnIPByHost: func(host string) (ip netip.Addr) { panic("not implemented") },
		},
		DNSFilter:   f,
		PrivateNets: netutil.SubnetSetFunc(netutil.IsLocallyServed),
		Logger:      slogutil.NewDiscardLogger(),
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
		req       *dns.Msg
		name      string
		wantRCode int
		wantAns   []dns.RR
	}{{
		req:       createTestMessage(aghtest.ReqFQDN),
		name:      "pass",
		wantRCode: dns.RcodeNameError,
		wantAns:   nil,
	}, {
		req:       createTestMessage("cname.exception."),
		name:      "cname_exception",
		wantRCode: dns.RcodeSuccess,
		wantAns: []dns.RR{&dns.CNAME{
			Hdr: dns.RR_Header{
				Name:   "cname.exception.",
				Rrtype: dns.TypeCNAME,
			},
			Target: "cname.specific.",
		}},
	}, {
		req:       createTestMessage("should.block."),
		name:      "blocked_by_cname",
		wantRCode: dns.RcodeSuccess,
		wantAns: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   "should.block.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
			},
			A: netutil.IPv4Zero(),
		}},
	}, {
		req:       createTestMessage("a.exception."),
		name:      "a_exception",
		wantRCode: dns.RcodeSuccess,
		wantAns: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   "a.exception.",
				Rrtype: dns.TypeA,
			},
			A: net.IP{0, 0, 0, 1},
		}},
	}, {
		req:       createTestMessageWithType("aaaa.exception.", dns.TypeAAAA),
		name:      "aaaa_exception",
		wantRCode: dns.RcodeSuccess,
		wantAns: []dns.RR{&dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   "aaaa.exception.",
				Rrtype: dns.TypeAAAA,
			},
			AAAA: net.ParseIP("::1"),
		}},
	}, {
		req:       createTestMessage("allowed.first."),
		name:      "allowed_first",
		wantRCode: dns.RcodeSuccess,
		wantAns: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   "allowed.first.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
			},
			A: netutil.IPv4Zero(),
		}},
	}, {
		req:       createTestMessage("blocked.first."),
		name:      "blocked_first",
		wantRCode: dns.RcodeSuccess,
		wantAns: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   "blocked.first.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
			},
			A: netutil.IPv4Zero(),
		}},
	}, {
		req:       createTestMessage("duplicate.domain."),
		name:      "duplicate_domain",
		wantRCode: dns.RcodeSuccess,
		wantAns: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   "duplicate.domain.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
			},
			A: netutil.IPv4Zero(),
		}},
	}, {
		req:       createTestMessageWithType("blocked.domain.", dns.TypeHTTPS),
		name:      "blocked_https_req",
		wantRCode: dns.RcodeSuccess,
		wantAns:   nil,
	}, {
		req:       createTestMessageWithType("blocked.by.hostrule.", dns.TypeHTTPS),
		name:      "blocked_host_rule_https_req",
		wantRCode: dns.RcodeSuccess,
		wantAns:   nil,
	}}

	for _, tc := range testCases {
		dctx := &proxy.DNSContext{
			Proto: proxy.ProtoUDP,
			Req:   tc.req,
			Addr:  testClientAddrPort,
		}

		t.Run(tc.name, func(t *testing.T) {
			err = s.handleDNSRequest(nil, dctx)
			require.NoError(t, err)
			require.NotNil(t, dctx.Res)

			assert.Equal(t, tc.wantRCode, dctx.Res.Rcode)
			assert.Equal(t, tc.wantAns, dctx.Res.Answer)
		})
	}
}

func TestHandleDNSRequest_filterDNSResponse(t *testing.T) {
	const (
		passedIPv4Str  = "1.1.1.1"
		blockedIPv4Str = "1.2.3.4"
		blockedIPv6Str = "1234::cdef"
		blockRules     = blockedIPv4Str + "\n" + blockedIPv6Str + "\n"
	)

	var (
		passedIPv4  net.IP = netip.MustParseAddr(passedIPv4Str).AsSlice()
		blockedIPv4 net.IP = netip.MustParseAddr(blockedIPv4Str).AsSlice()
		blockedIPv6 net.IP = netip.MustParseAddr(blockedIPv6Str).AsSlice()
	)

	filters := []filtering.Filter{{
		ID: 0, Data: []byte(blockRules),
	}}

	f, err := filtering.New(&filtering.Config{}, filters)
	require.NoError(t, err)

	f.SetEnabled(true)

	s, err := NewServer(DNSCreateParams{
		DHCPServer:  &testDHCP{},
		DNSFilter:   f,
		PrivateNets: netutil.SubnetSetFunc(netutil.IsLocallyServed),
		Logger:      slogutil.NewDiscardLogger(),
	})
	require.NoError(t, err)

	testCases := []struct {
		req      *dns.Msg
		name     string
		wantRule string
		respAns  []dns.RR
	}{{
		name:     "pass",
		req:      createTestMessageWithType(aghtest.ReqFQDN, dns.TypeA),
		wantRule: "",
		respAns: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   aghtest.ReqFQDN,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
			},
			A: passedIPv4,
		}},
	}, {
		name:     "ipv4",
		req:      createTestMessageWithType(aghtest.ReqFQDN, dns.TypeA),
		wantRule: blockedIPv4Str,
		respAns: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   aghtest.ReqFQDN,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
			},
			A: blockedIPv4,
		}},
	}, {
		name:     "ipv6",
		req:      createTestMessageWithType(aghtest.ReqFQDN, dns.TypeAAAA),
		wantRule: blockedIPv6Str,
		respAns: []dns.RR{&dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   aghtest.ReqFQDN,
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
			},
			AAAA: blockedIPv6,
		}},
	}, {
		name:     "ipv4hint",
		req:      createTestMessageWithType(aghtest.ReqFQDN, dns.TypeHTTPS),
		wantRule: blockedIPv4Str,
		respAns: newSVCBHintsAnswer(
			aghtest.ReqFQDN,
			[]dns.SVCBKeyValue{
				&dns.SVCBIPv4Hint{Hint: []net.IP{blockedIPv4}},
				&dns.SVCBIPv6Hint{Hint: []net.IP{}},
			},
		),
	}, {
		name:     "ipv6hint",
		req:      createTestMessageWithType(aghtest.ReqFQDN, dns.TypeHTTPS),
		wantRule: blockedIPv6Str,
		respAns: newSVCBHintsAnswer(
			aghtest.ReqFQDN,
			[]dns.SVCBKeyValue{
				&dns.SVCBIPv4Hint{Hint: []net.IP{}},
				&dns.SVCBIPv6Hint{Hint: []net.IP{blockedIPv6}},
			},
		),
	}, {
		name:     "ipv4_ipv6_hints",
		req:      createTestMessageWithType(aghtest.ReqFQDN, dns.TypeHTTPS),
		wantRule: blockedIPv4Str,
		respAns: newSVCBHintsAnswer(
			aghtest.ReqFQDN,
			[]dns.SVCBKeyValue{
				&dns.SVCBIPv4Hint{Hint: []net.IP{blockedIPv4}},
				&dns.SVCBIPv6Hint{Hint: []net.IP{blockedIPv6}},
			},
		),
	}, {
		name:     "pass_hints",
		req:      createTestMessageWithType(aghtest.ReqFQDN, dns.TypeHTTPS),
		wantRule: "",
		respAns: newSVCBHintsAnswer(
			aghtest.ReqFQDN,
			[]dns.SVCBKeyValue{
				&dns.SVCBIPv4Hint{Hint: []net.IP{passedIPv4}},
				&dns.SVCBIPv6Hint{Hint: []net.IP{}},
			},
		),
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := newResp(dns.RcodeSuccess, tc.req, tc.respAns)

			pctx := &proxy.DNSContext{
				Proto: proxy.ProtoUDP,
				Req:   tc.req,
				Res:   resp,
				Addr:  testClientAddrPort,
			}

			dctx := &dnsContext{
				proxyCtx: pctx,
				setts: &filtering.Settings{
					ProtectionEnabled: true,
					FilteringEnabled:  true,
				},
			}

			fltErr := s.filterDNSResponse(dctx)
			require.NoError(t, fltErr)

			res := dctx.result
			if tc.wantRule == "" {
				assert.Nil(t, res)

				return
			}

			wantResult := &filtering.Result{
				IsFiltered: true,
				Reason:     filtering.FilteredBlockList,
				Rules: []*filtering.ResultRule{{
					Text: tc.wantRule,
				}},
			}

			assert.Equal(t, wantResult, res)
			assert.Equal(t, resp, dctx.origResp)
		})
	}
}

// newSVCBHintsAnswer returns a test HTTPS answer RRs with SVCB hints.
func newSVCBHintsAnswer(target string, hints []dns.SVCBKeyValue) (rrs []dns.RR) {
	return []dns.RR{&dns.HTTPS{
		SVCB: dns.SVCB{
			Hdr: dns.RR_Header{
				Name:   target,
				Rrtype: dns.TypeHTTPS,
				Class:  dns.ClassINET,
			},
			Target: target,
			Value:  hints,
		},
	}}
}
