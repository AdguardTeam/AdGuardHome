package dnsforward

import (
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_filterDNSResponse(t *testing.T) {
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

	f, err := filtering.New(&filtering.Config{
		Logger: testLogger,
	}, filters)
	require.NoError(t, err)

	f.SetEnabled(true)

	s, err := NewServer(DNSCreateParams{
		DHCPServer:  &testDHCP{},
		DNSFilter:   f,
		PrivateNets: netutil.SubnetSetFunc(netutil.IsLocallyServed),
		Logger:      testLogger,
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

			fltErr := s.filterDNSResponse(testutil.ContextWithTimeout(t, testTimeout), dctx)
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
