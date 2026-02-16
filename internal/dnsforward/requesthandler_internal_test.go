package dnsforward

import (
	"cmp"
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_ServeDNS(t *testing.T) {
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
		TLSConf:        &TLSConfig{},
		Config: Config{
			UpstreamMode: UpstreamModeLoadBalance,
			EDNSClientSubnet: &EDNSClientSubnet{
				Enabled: false,
			},
			ClientsContainer: EmptyClientsContainer{},
		},
		ServePlainDNS: true,
	}
	filters := []filtering.Filter{{
		ID: 0, Data: []byte(rules),
	}}

	f, err := filtering.New(&filtering.Config{
		Logger:               testLogger,
		ProtectionEnabled:    true,
		ApplyClientFiltering: applyEmptyClientFiltering,
		BlockedServices:      emptyFilteringBlockedServices(),
		BlockingMode:         filtering.BlockingModeDefault,
	}, filters)
	require.NoError(t, err)
	f.SetEnabled(true)

	s, err := NewServer(DNSCreateParams{
		DHCPServer: &testDHCP{
			OnEnabled:  func() (ok bool) { return false },
			OnHostByIP: func(ip netip.Addr) (_ string) { panic(testutil.UnexpectedCall(ip)) },
			OnIPByHost: func(host string) (_ netip.Addr) { panic(testutil.UnexpectedCall(host)) },
		},
		DNSFilter:   f,
		PrivateNets: netutil.SubnetSetFunc(netutil.IsLocallyServed),
		Logger:      testLogger,
	})
	require.NoError(t, err)

	err = s.Prepare(testutil.ContextWithTimeout(t, testTimeout), &forwardConf)
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
			err = s.ServeDNS(nil, dctx)
			require.NoError(t, err)
			require.NotNil(t, dctx.Res)

			assert.Equal(t, tc.wantRCode, dctx.Res.Rcode)
			assert.Equal(t, tc.wantAns, dctx.Res.Answer)
		})
	}
}

// TODO(e.burkov):  Rewrite this test to use the whole server instead of just
// testing the [Handle] method.  See comment on "from_external_for_local" test
// case.
func TestServer_ServeDNS_restrictLocal(t *testing.T) {
	intAddr := netip.MustParseAddr("192.168.1.1")
	intPTRQuestion, err := netutil.IPToReversedAddr(intAddr.AsSlice())
	require.NoError(t, err)

	extAddr := netip.MustParseAddr("254.253.252.1")
	extPTRQuestion, err := netutil.IPToReversedAddr(extAddr.AsSlice())
	require.NoError(t, err)

	const (
		extPTRAnswer = "host1.example.net."
		intPTRAnswer = "some.local-client."
	)

	localUpsHdlr := dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		resp := cmp.Or(
			aghtest.MatchedResponse(req, dns.TypePTR, extPTRQuestion, extPTRAnswer),
			aghtest.MatchedResponse(req, dns.TypePTR, intPTRQuestion, intPTRAnswer),
			(&dns.Msg{}).SetRcode(req, dns.RcodeNameError),
		)

		require.NoError(testutil.PanicT{}, w.WriteMsg(resp))
	})
	localUpsAddr := aghtest.StartLocalhostUpstream(t, localUpsHdlr).String()

	s := createTestServer(t, &filtering.Config{
		BlockingMode: filtering.BlockingModeDefault,
	}, ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		TLSConf:        &TLSConfig{},
		// TODO(s.chzhen):  Add tests where EDNSClientSubnet.Enabled is true.
		// Improve Config declaration for tests.
		Config: Config{
			UpstreamDNS:      []string{localUpsAddr},
			UpstreamMode:     UpstreamModeLoadBalance,
			EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
			ClientsContainer: EmptyClientsContainer{},
		},
		UsePrivateRDNS:    true,
		LocalPTRResolvers: []string{localUpsAddr},
		ServePlainDNS:     true,
	})
	startDeferStop(t, s)

	testCases := []struct {
		name      string
		question  string
		wantErr   error
		wantAns   []dns.RR
		isPrivate bool
	}{{
		name:     "from_local_for_external",
		question: extPTRQuestion,
		wantErr:  nil,
		wantAns: []dns.RR{&dns.PTR{
			Hdr: dns.RR_Header{
				Name:     dns.Fqdn(extPTRQuestion),
				Rrtype:   dns.TypePTR,
				Class:    dns.ClassINET,
				Ttl:      60,
				Rdlength: uint16(len(extPTRAnswer) + 1),
			},
			Ptr: dns.Fqdn(extPTRAnswer),
		}},
		isPrivate: true,
	}, {
		// In theory this case is not reproducible because [proxy.Proxy] should
		// respond to such queries with NXDOMAIN before they reach
		// [Server.Handle].
		name:      "from_external_for_local",
		question:  intPTRQuestion,
		wantErr:   upstream.ErrNoUpstreams,
		wantAns:   nil,
		isPrivate: false,
	}, {
		name:     "from_local_for_local",
		question: intPTRQuestion,
		wantErr:  nil,
		wantAns: []dns.RR{&dns.PTR{
			Hdr: dns.RR_Header{
				Name:     dns.Fqdn(intPTRQuestion),
				Rrtype:   dns.TypePTR,
				Class:    dns.ClassINET,
				Ttl:      60,
				Rdlength: uint16(len(intPTRAnswer) + 1),
			},
			Ptr: dns.Fqdn(intPTRAnswer),
		}},
		isPrivate: true,
	}, {
		name:     "from_external_for_external",
		question: extPTRQuestion,
		wantErr:  nil,
		wantAns: []dns.RR{&dns.PTR{
			Hdr: dns.RR_Header{
				Name:     dns.Fqdn(extPTRQuestion),
				Rrtype:   dns.TypePTR,
				Class:    dns.ClassINET,
				Ttl:      60,
				Rdlength: uint16(len(extPTRAnswer) + 1),
			},
			Ptr: dns.Fqdn(extPTRAnswer),
		}},
		isPrivate: false,
	}}

	for _, tc := range testCases {
		pref, extErr := netutil.ExtractReversedAddr(tc.question)
		require.NoError(t, extErr)

		req := createTestMessageWithType(dns.Fqdn(tc.question), dns.TypePTR)
		pctx := &proxy.DNSContext{
			Req:             req,
			IsPrivateClient: tc.isPrivate,
		}
		// TODO(e.burkov):  Configure the subnet set properly.
		if netutil.IsLocallyServed(pref.Addr()) {
			pctx.RequestedPrivateRDNS = pref
		}

		t.Run(tc.name, func(t *testing.T) {
			err = s.ServeDNS(s.dnsProxy, pctx)
			require.ErrorIs(t, err, tc.wantErr)

			require.NotNil(t, pctx.Res)
			assert.Equal(t, tc.wantAns, pctx.Res.Answer)
		})
	}
}
