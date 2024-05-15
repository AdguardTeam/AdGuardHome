package dnsforward

import (
	"net"
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

// maxDNS64SynTTL is the maximum TTL for synthesized DNS64 responses with no SOA
// records in seconds.
//
// If the SOA RR was not delivered with the negative response to the AAAA query,
// then the DNS64 SHOULD use the TTL of the original A RR or 600 seconds,
// whichever is shorter.
//
// See https://datatracker.ietf.org/doc/html/rfc6147#section-5.1.7.
const maxDNS64SynTTL uint32 = 600

// newRR is a helper that creates a new dns.RR with the given name, qtype, ttl
// and value.  It fails the test if the qtype is not supported or the type of
// value doesn't match the qtype.
func newRR(t *testing.T, name string, qtype uint16, ttl uint32, val any) (rr dns.RR) {
	t.Helper()

	switch qtype {
	case dns.TypeA:
		rr = &dns.A{A: testutil.RequireTypeAssert[net.IP](t, val)}
	case dns.TypeAAAA:
		rr = &dns.AAAA{AAAA: testutil.RequireTypeAssert[net.IP](t, val)}
	case dns.TypeCNAME:
		rr = &dns.CNAME{Target: testutil.RequireTypeAssert[string](t, val)}
	case dns.TypeSOA:
		rr = &dns.SOA{
			Ns:      "ns." + name,
			Mbox:    "hostmaster." + name,
			Serial:  1,
			Refresh: 1,
			Retry:   1,
			Expire:  1,
			Minttl:  1,
		}
	case dns.TypePTR:
		rr = &dns.PTR{Ptr: testutil.RequireTypeAssert[string](t, val)}
	default:
		t.Fatalf("unsupported qtype: %d", qtype)
	}

	*rr.Header() = dns.RR_Header{
		Name:   name,
		Rrtype: qtype,
		Class:  dns.ClassINET,
		Ttl:    ttl,
	}

	return rr
}

func TestServer_HandleDNSRequest_dns64(t *testing.T) {
	t.Parallel()

	const (
		ipv4Domain    = "ipv4.only."
		ipv6Domain    = "ipv6.only."
		soaDomain     = "ipv4.soa."
		mappedDomain  = "filterable.ipv6."
		anotherDomain = "another.domain."

		pointedDomain = "local1234.ipv4."
		globDomain    = "real1234.ipv4."
	)

	someIPv4 := net.IP{1, 2, 3, 4}
	someIPv6 := net.IP{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	mappedIPv6 := net.ParseIP("64:ff9b::102:304")

	ptr64Domain, err := netutil.IPToReversedAddr(mappedIPv6)
	require.NoError(t, err)
	ptr64Domain = dns.Fqdn(ptr64Domain)

	ptrGlobDomain, err := netutil.IPToReversedAddr(someIPv4)
	require.NoError(t, err)
	ptrGlobDomain = dns.Fqdn(ptrGlobDomain)

	const (
		sectionAnswer = iota
		sectionAuthority
		sectionAdditional

		sectionsNum
	)

	// answerMap is a convenience alias for describing the upstream response for
	// a given question type.
	type answerMap = map[uint16][sectionsNum][]dns.RR

	pt := testutil.PanicT{}

	testCases := []struct {
		name    string
		qname   string
		upsAns  answerMap
		wantAns []dns.RR
		qtype   uint16
	}{{
		name:  "simple_a",
		qname: ipv4Domain,
		upsAns: answerMap{
			dns.TypeA: {
				sectionAnswer: {newRR(t, ipv4Domain, dns.TypeA, 3600, someIPv4)},
			},
			dns.TypeAAAA: {},
		},
		wantAns: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:     ipv4Domain,
				Rrtype:   dns.TypeA,
				Class:    dns.ClassINET,
				Ttl:      3600,
				Rdlength: 4,
			},
			A: someIPv4,
		}},
		qtype: dns.TypeA,
	}, {
		name:  "simple_aaaa",
		qname: ipv6Domain,
		upsAns: answerMap{
			dns.TypeA: {},
			dns.TypeAAAA: {
				sectionAnswer: {newRR(t, ipv6Domain, dns.TypeAAAA, 3600, someIPv6)},
			},
		},
		wantAns: []dns.RR{&dns.AAAA{
			Hdr: dns.RR_Header{
				Name:     ipv6Domain,
				Rrtype:   dns.TypeAAAA,
				Class:    dns.ClassINET,
				Ttl:      3600,
				Rdlength: 16,
			},
			AAAA: someIPv6,
		}},
		qtype: dns.TypeAAAA,
	}, {
		name:  "actual_dns64",
		qname: ipv4Domain,
		upsAns: answerMap{
			dns.TypeA: {
				sectionAnswer: {newRR(t, ipv4Domain, dns.TypeA, 3600, someIPv4)},
			},
			dns.TypeAAAA: {},
		},
		wantAns: []dns.RR{&dns.AAAA{
			Hdr: dns.RR_Header{
				Name:     ipv4Domain,
				Rrtype:   dns.TypeAAAA,
				Class:    dns.ClassINET,
				Ttl:      maxDNS64SynTTL,
				Rdlength: 16,
			},
			AAAA: mappedIPv6,
		}},
		qtype: dns.TypeAAAA,
	}, {
		name:  "actual_dns64_soattl",
		qname: soaDomain,
		upsAns: answerMap{
			dns.TypeA: {
				sectionAnswer: {newRR(t, soaDomain, dns.TypeA, 3600, someIPv4)},
			},
			dns.TypeAAAA: {
				sectionAuthority: {newRR(t, soaDomain, dns.TypeSOA, maxDNS64SynTTL+50, nil)},
			},
		},
		wantAns: []dns.RR{&dns.AAAA{
			Hdr: dns.RR_Header{
				Name:     soaDomain,
				Rrtype:   dns.TypeAAAA,
				Class:    dns.ClassINET,
				Ttl:      maxDNS64SynTTL + 50,
				Rdlength: 16,
			},
			AAAA: mappedIPv6,
		}},
		qtype: dns.TypeAAAA,
	}, {
		name:  "filtered",
		qname: mappedDomain,
		upsAns: answerMap{
			dns.TypeA: {},
			dns.TypeAAAA: {
				sectionAnswer: {
					newRR(t, mappedDomain, dns.TypeAAAA, 3600, net.ParseIP("64:ff9b::506:708")),
					newRR(t, mappedDomain, dns.TypeCNAME, 3600, anotherDomain),
				},
			},
		},
		wantAns: []dns.RR{&dns.CNAME{
			Hdr: dns.RR_Header{
				Name:     mappedDomain,
				Rrtype:   dns.TypeCNAME,
				Class:    dns.ClassINET,
				Ttl:      3600,
				Rdlength: 16,
			},
			Target: anotherDomain,
		}},
		qtype: dns.TypeAAAA,
	}, {
		name:   "ptr",
		qname:  ptr64Domain,
		upsAns: nil,
		wantAns: []dns.RR{&dns.PTR{
			Hdr: dns.RR_Header{
				Name:     ptr64Domain,
				Rrtype:   dns.TypePTR,
				Class:    dns.ClassINET,
				Ttl:      3600,
				Rdlength: 16,
			},
			Ptr: pointedDomain,
		}},
		qtype: dns.TypePTR,
	}, {
		name:  "ptr_glob",
		qname: ptrGlobDomain,
		upsAns: answerMap{
			dns.TypePTR: {
				sectionAnswer: {newRR(t, ptrGlobDomain, dns.TypePTR, 3600, globDomain)},
			},
		},
		wantAns: []dns.RR{&dns.PTR{
			Hdr: dns.RR_Header{
				Name:     ptrGlobDomain,
				Rrtype:   dns.TypePTR,
				Class:    dns.ClassINET,
				Ttl:      3600,
				Rdlength: 15,
			},
			Ptr: globDomain,
		}},
		qtype: dns.TypePTR,
	}}

	localRR := newRR(t, ptr64Domain, dns.TypePTR, 3600, pointedDomain)
	localUpsHdlr := dns.HandlerFunc(func(w dns.ResponseWriter, m *dns.Msg) {
		require.Len(pt, m.Question, 1)
		require.Equal(pt, m.Question[0].Name, ptr64Domain)

		resp := (&dns.Msg{}).SetReply(m)
		resp.Answer = []dns.RR{localRR}

		require.NoError(t, w.WriteMsg(resp))
	})
	localUpsAddr := aghtest.StartLocalhostUpstream(t, localUpsHdlr).String()

	client := &dns.Client{
		Net:     string(proxy.ProtoTCP),
		Timeout: testTimeout,
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			upsHdlr := dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
				q := req.Question[0]

				require.Contains(pt, tc.upsAns, q.Qtype)
				answer := tc.upsAns[q.Qtype]

				resp := (&dns.Msg{}).SetReply(req)
				resp.Answer = answer[sectionAnswer]
				resp.Ns = answer[sectionAuthority]
				resp.Extra = answer[sectionAdditional]

				require.NoError(pt, w.WriteMsg(resp))
			})
			upsAddr := aghtest.StartLocalhostUpstream(t, upsHdlr).String()

			// TODO(e.burkov):  It seems [proxy.Proxy] isn't intended to be
			// reused right after stop, due to a data race in [proxy.Proxy.Init]
			// method when setting an OOB size.  As a temporary workaround,
			// recreate the whole server for each test case.
			s := createTestServer(t, &filtering.Config{
				BlockingMode: filtering.BlockingModeDefault,
			}, ServerConfig{
				UDPListenAddrs: []*net.UDPAddr{{}},
				TCPListenAddrs: []*net.TCPAddr{{}},
				UseDNS64:       true,
				Config: Config{
					UpstreamMode:     UpstreamModeLoadBalance,
					EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
					UpstreamDNS:      []string{upsAddr},
				},
				UsePrivateRDNS:    true,
				LocalPTRResolvers: []string{localUpsAddr},
				ServePlainDNS:     true,
			})

			startDeferStop(t, s)

			req := (&dns.Msg{}).SetQuestion(tc.qname, tc.qtype)

			resp, _, excErr := client.Exchange(req, s.proxy().Addr(proxy.ProtoTCP).String())
			require.NoError(t, excErr)

			require.Equal(t, tc.wantAns, resp.Answer)
		})
	}
}

func TestServer_dns64WithDisabledRDNS(t *testing.T) {
	t.Parallel()

	// Shouldn't go to upstream at all.
	panicHdlr := dns.HandlerFunc(func(w dns.ResponseWriter, m *dns.Msg) {
		panic("not implemented")
	})
	upsAddr := aghtest.StartLocalhostUpstream(t, panicHdlr).String()
	localUpsAddr := aghtest.StartLocalhostUpstream(t, panicHdlr).String()

	s := createTestServer(t, &filtering.Config{
		BlockingMode: filtering.BlockingModeDefault,
	}, ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		UseDNS64:       true,
		Config: Config{
			UpstreamMode:     UpstreamModeLoadBalance,
			EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
			UpstreamDNS:      []string{upsAddr},
		},
		UsePrivateRDNS:    false,
		LocalPTRResolvers: []string{localUpsAddr},
		ServePlainDNS:     true,
	})
	startDeferStop(t, s)

	mappedIPv6 := net.ParseIP("64:ff9b::102:304")
	arpa, err := netutil.IPToReversedAddr(mappedIPv6)
	require.NoError(t, err)

	req := (&dns.Msg{}).SetQuestion(dns.Fqdn(arpa), dns.TypePTR)

	cli := &dns.Client{
		Net:     string(proxy.ProtoTCP),
		Timeout: testTimeout,
	}

	resp, _, err := cli.Exchange(req, s.proxy().Addr(proxy.ProtoTCP).String())
	require.NoError(t, err)

	assert.Equal(t, dns.RcodeNameError, resp.Rcode)
}
