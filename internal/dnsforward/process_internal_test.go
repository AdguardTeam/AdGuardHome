package dnsforward

import (
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
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

const (
	ddrTestDomainName = "dns.example.net"
	ddrTestFQDN       = ddrTestDomainName + "."
)

func TestServer_ProcessDDRQuery(t *testing.T) {
	dohSVCB := &dns.SVCB{
		Priority: 1,
		Target:   ddrTestFQDN,
		Value: []dns.SVCBKeyValue{
			&dns.SVCBAlpn{Alpn: []string{"h2"}},
			&dns.SVCBPort{Port: 8044},
			&dns.SVCBDoHPath{Template: "/dns-query{?dns}"},
		},
	}

	dotSVCB := &dns.SVCB{
		Priority: 1,
		Target:   ddrTestFQDN,
		Value: []dns.SVCBKeyValue{
			&dns.SVCBAlpn{Alpn: []string{"dot"}},
			&dns.SVCBPort{Port: 8043},
		},
	}

	doqSVCB := &dns.SVCB{
		Priority: 1,
		Target:   ddrTestFQDN,
		Value: []dns.SVCBKeyValue{
			&dns.SVCBAlpn{Alpn: []string{"doq"}},
			&dns.SVCBPort{Port: 8042},
		},
	}

	testCases := []struct {
		name       string
		host       string
		want       []*dns.SVCB
		wantRes    resultCode
		portDoH    int
		portDoT    int
		portDoQ    int
		qtype      uint16
		ddrEnabled bool
	}{{
		name:       "pass_host",
		wantRes:    resultCodeSuccess,
		host:       "example.net.",
		qtype:      dns.TypeSVCB,
		ddrEnabled: true,
		portDoH:    8043,
	}, {
		name:       "pass_qtype",
		wantRes:    resultCodeFinish,
		host:       ddrHostFQDN,
		qtype:      dns.TypeA,
		ddrEnabled: true,
		portDoH:    8043,
	}, {
		name:       "pass_disabled_tls",
		wantRes:    resultCodeFinish,
		host:       ddrHostFQDN,
		qtype:      dns.TypeSVCB,
		ddrEnabled: true,
	}, {
		name:       "pass_disabled_ddr",
		wantRes:    resultCodeSuccess,
		host:       ddrHostFQDN,
		qtype:      dns.TypeSVCB,
		ddrEnabled: false,
		portDoH:    8043,
	}, {
		name:       "dot",
		wantRes:    resultCodeFinish,
		want:       []*dns.SVCB{dotSVCB},
		host:       ddrHostFQDN,
		qtype:      dns.TypeSVCB,
		ddrEnabled: true,
		portDoT:    8043,
	}, {
		name:       "doh",
		wantRes:    resultCodeFinish,
		want:       []*dns.SVCB{dohSVCB},
		host:       ddrHostFQDN,
		qtype:      dns.TypeSVCB,
		ddrEnabled: true,
		portDoH:    8044,
	}, {
		name:       "doq",
		wantRes:    resultCodeFinish,
		want:       []*dns.SVCB{doqSVCB},
		host:       ddrHostFQDN,
		qtype:      dns.TypeSVCB,
		ddrEnabled: true,
		portDoQ:    8042,
	}, {
		name:       "dot_doh",
		wantRes:    resultCodeFinish,
		want:       []*dns.SVCB{dotSVCB, dohSVCB},
		host:       ddrHostFQDN,
		qtype:      dns.TypeSVCB,
		ddrEnabled: true,
		portDoT:    8043,
		portDoH:    8044,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := prepareTestServer(t, tc.portDoH, tc.portDoT, tc.portDoQ, tc.ddrEnabled)

			req := createTestMessageWithType(tc.host, tc.qtype)

			dctx := &dnsContext{
				proxyCtx: &proxy.DNSContext{
					Req: req,
				},
			}

			res := s.processDDRQuery(dctx)
			require.Equal(t, tc.wantRes, res)

			if tc.wantRes != resultCodeFinish {
				return
			}

			msg := dctx.proxyCtx.Res
			require.NotNil(t, msg)

			for _, v := range tc.want {
				v.Hdr = s.hdr(req, dns.TypeSVCB)
			}

			assert.ElementsMatch(t, tc.want, msg.Answer)
		})
	}
}

func prepareTestServer(t *testing.T, portDoH, portDoT, portDoQ int, ddrEnabled bool) (s *Server) {
	t.Helper()

	s = &Server{
		dnsProxy: &proxy.Proxy{
			Config: proxy.Config{},
		},
		conf: ServerConfig{
			FilteringConfig: FilteringConfig{
				HandleDDR: ddrEnabled,
			},
			TLSConfig: TLSConfig{
				ServerName: ddrTestDomainName,
			},
		},
	}

	if portDoT > 0 {
		s.dnsProxy.TLSListenAddr = []*net.TCPAddr{{Port: portDoT}}
		s.conf.hasIPAddrs = true
	}

	if portDoQ > 0 {
		s.dnsProxy.QUICListenAddr = []*net.UDPAddr{{Port: portDoQ}}
	}

	if portDoH > 0 {
		s.conf.HTTPSListenAddrs = []*net.TCPAddr{{Port: portDoH}}
	}

	return s
}

func TestServer_ProcessDetermineLocal(t *testing.T) {
	s := &Server{
		privateNets: netutil.SubnetSetFunc(netutil.IsLocallyServed),
	}

	testCases := []struct {
		want  assert.BoolAssertionFunc
		name  string
		cliIP net.IP
	}{{
		want:  assert.True,
		name:  "local",
		cliIP: net.IP{192, 168, 0, 1},
	}, {
		want:  assert.False,
		name:  "external",
		cliIP: net.IP{250, 249, 0, 1},
	}, {
		want:  assert.False,
		name:  "invalid",
		cliIP: net.IP{1, 2, 3, 4, 5},
	}, {
		want:  assert.False,
		name:  "nil",
		cliIP: nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			proxyCtx := &proxy.DNSContext{
				Addr: &net.TCPAddr{
					IP: tc.cliIP,
				},
			}
			dctx := &dnsContext{
				proxyCtx: proxyCtx,
			}
			s.processDetermineLocal(dctx)

			tc.want(t, dctx.isLocalClient)
		})
	}
}

func TestServer_ProcessDHCPHosts_localRestriction(t *testing.T) {
	knownIP := netip.MustParseAddr("1.2.3.4")
	testCases := []struct {
		name       string
		host       string
		wantIP     netip.Addr
		wantRes    resultCode
		isLocalCli bool
	}{{
		name:       "local_client_success",
		host:       "example.lan",
		wantIP:     knownIP,
		wantRes:    resultCodeSuccess,
		isLocalCli: true,
	}, {
		name:       "local_client_unknown_host",
		host:       "wronghost.lan",
		wantIP:     netip.Addr{},
		wantRes:    resultCodeSuccess,
		isLocalCli: true,
	}, {
		name:       "external_client_known_host",
		host:       "example.lan",
		wantIP:     netip.Addr{},
		wantRes:    resultCodeFinish,
		isLocalCli: false,
	}, {
		name:       "external_client_unknown_host",
		host:       "wronghost.lan",
		wantIP:     netip.Addr{},
		wantRes:    resultCodeFinish,
		isLocalCli: false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{
				dhcpServer:        testDHCP,
				localDomainSuffix: defaultLocalDomainSuffix,
				tableHostToIP: hostToIPTable{
					"example." + defaultLocalDomainSuffix: knownIP,
				},
			}

			req := &dns.Msg{
				MsgHdr: dns.MsgHdr{
					Id: dns.Id(),
				},
				Question: []dns.Question{{
					Name:   dns.Fqdn(tc.host),
					Qtype:  dns.TypeA,
					Qclass: dns.ClassINET,
				}},
			}

			dctx := &dnsContext{
				proxyCtx: &proxy.DNSContext{
					Req: req,
				},
				isLocalClient: tc.isLocalCli,
			}

			res := s.processDHCPHosts(dctx)
			require.Equal(t, tc.wantRes, res)
			pctx := dctx.proxyCtx
			if tc.wantRes == resultCodeFinish {
				require.NotNil(t, pctx.Res)

				assert.Equal(t, dns.RcodeNameError, pctx.Res.Rcode)
				assert.Len(t, pctx.Res.Answer, 0)

				return
			}

			if tc.wantIP == (netip.Addr{}) {
				assert.Nil(t, pctx.Res)
			} else {
				require.NotNil(t, pctx.Res)

				ans := pctx.Res.Answer
				require.Len(t, ans, 1)

				a := testutil.RequireTypeAssert[*dns.A](t, ans[0])

				ip, err := netutil.IPToAddr(a.A, netutil.AddrFamilyIPv4)
				require.NoError(t, err)

				assert.Equal(t, tc.wantIP, ip)
			}
		})
	}
}

func TestServer_ProcessDHCPHosts(t *testing.T) {
	const (
		examplecom = "example.com"
		examplelan = "example." + defaultLocalDomainSuffix
	)

	knownIP := netip.MustParseAddr("1.2.3.4")
	testCases := []struct {
		name    string
		host    string
		suffix  string
		wantIP  netip.Addr
		wantRes resultCode
		qtyp    uint16
	}{{
		name:    "success_external",
		host:    examplecom,
		suffix:  defaultLocalDomainSuffix,
		wantIP:  netip.Addr{},
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeA,
	}, {
		name:    "success_external_non_a",
		host:    examplecom,
		suffix:  defaultLocalDomainSuffix,
		wantIP:  netip.Addr{},
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeCNAME,
	}, {
		name:    "success_internal",
		host:    examplelan,
		suffix:  defaultLocalDomainSuffix,
		wantIP:  knownIP,
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeA,
	}, {
		name:    "success_internal_unknown",
		host:    "example-new.lan",
		suffix:  defaultLocalDomainSuffix,
		wantIP:  netip.Addr{},
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeA,
	}, {
		name:    "success_internal_aaaa",
		host:    examplelan,
		suffix:  defaultLocalDomainSuffix,
		wantIP:  netip.Addr{},
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeAAAA,
	}, {
		name:    "success_custom_suffix",
		host:    "example.custom",
		suffix:  "custom",
		wantIP:  knownIP,
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeA,
	}}

	for _, tc := range testCases {
		s := &Server{
			dhcpServer:        testDHCP,
			localDomainSuffix: tc.suffix,
			tableHostToIP: hostToIPTable{
				"example." + tc.suffix: knownIP,
			},
		}

		req := &dns.Msg{
			MsgHdr: dns.MsgHdr{
				Id: 1234,
			},
			Question: []dns.Question{{
				Name:   dns.Fqdn(tc.host),
				Qtype:  tc.qtyp,
				Qclass: dns.ClassINET,
			}},
		}

		dctx := &dnsContext{
			proxyCtx: &proxy.DNSContext{
				Req: req,
			},
			isLocalClient: true,
		}

		t.Run(tc.name, func(t *testing.T) {
			res := s.processDHCPHosts(dctx)
			pctx := dctx.proxyCtx
			assert.Equal(t, tc.wantRes, res)
			if tc.wantRes == resultCodeFinish {
				require.NotNil(t, pctx.Res)
				assert.Equal(t, dns.RcodeNameError, pctx.Res.Rcode)

				return
			}

			require.NoError(t, dctx.err)

			if tc.qtyp == dns.TypeAAAA {
				// TODO(a.garipov): Remove this special handling
				// when we fully support AAAA.
				require.NotNil(t, pctx.Res)

				ans := pctx.Res.Answer
				require.Len(t, ans, 0)
			} else if tc.wantIP == (netip.Addr{}) {
				assert.Nil(t, pctx.Res)
			} else {
				require.NotNil(t, pctx.Res)

				ans := pctx.Res.Answer
				require.Len(t, ans, 1)

				a := testutil.RequireTypeAssert[*dns.A](t, ans[0])

				ip, err := netutil.IPToAddr(a.A, netutil.AddrFamilyIPv4)
				require.NoError(t, err)

				assert.Equal(t, tc.wantIP, ip)
			}
		})
	}
}

func TestServer_ProcessRestrictLocal(t *testing.T) {
	const (
		extPTRQuestion = "251.252.253.254.in-addr.arpa."
		extPTRAnswer   = "host1.example.net."
		intPTRQuestion = "1.1.168.192.in-addr.arpa."
		intPTRAnswer   = "some.local-client."
	)

	ups := aghtest.NewUpstreamMock(func(req *dns.Msg) (resp *dns.Msg, err error) {
		return aghalg.Coalesce(
			aghtest.MatchedResponse(req, dns.TypePTR, extPTRQuestion, extPTRAnswer),
			aghtest.MatchedResponse(req, dns.TypePTR, intPTRQuestion, intPTRAnswer),
			new(dns.Msg).SetRcode(req, dns.RcodeNameError),
		), nil
	})

	s := createTestServer(t, &filtering.Config{}, ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		// TODO(s.chzhen):  Add tests where EDNSClientSubnet.Enabled is true.
		// Improve FilteringConfig declaration for tests.
		FilteringConfig: FilteringConfig{
			EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
		},
	}, ups)
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{ups}
	startDeferStop(t, s)

	testCases := []struct {
		name     string
		want     string
		question net.IP
		cliIP    net.IP
		wantLen  int
	}{{
		name:     "from_local_to_external",
		want:     "host1.example.net.",
		question: net.IP{254, 253, 252, 251},
		cliIP:    net.IP{192, 168, 10, 10},
		wantLen:  1,
	}, {
		name:     "from_external_for_local",
		want:     "",
		question: net.IP{192, 168, 1, 1},
		cliIP:    net.IP{254, 253, 252, 251},
		wantLen:  0,
	}, {
		name:     "from_local_for_local",
		want:     "some.local-client.",
		question: net.IP{192, 168, 1, 1},
		cliIP:    net.IP{192, 168, 1, 2},
		wantLen:  1,
	}, {
		name:     "from_external_for_external",
		want:     "host1.example.net.",
		question: net.IP{254, 253, 252, 251},
		cliIP:    net.IP{254, 253, 252, 255},
		wantLen:  1,
	}}

	for _, tc := range testCases {
		reqAddr, err := dns.ReverseAddr(tc.question.String())
		require.NoError(t, err)
		req := createTestMessageWithType(reqAddr, dns.TypePTR)

		pctx := &proxy.DNSContext{
			Proto: proxy.ProtoTCP,
			Req:   req,
			Addr: &net.TCPAddr{
				IP: tc.cliIP,
			},
		}

		t.Run(tc.name, func(t *testing.T) {
			err = s.handleDNSRequest(nil, pctx)
			require.NoError(t, err)
			require.NotNil(t, pctx.Res)
			require.Len(t, pctx.Res.Answer, tc.wantLen)

			if tc.wantLen > 0 {
				assert.Equal(t, tc.want, pctx.Res.Answer[0].(*dns.PTR).Ptr)
			}
		})
	}
}

func TestServer_ProcessLocalPTR_usingResolvers(t *testing.T) {
	const locDomain = "some.local."
	const reqAddr = "1.1.168.192.in-addr.arpa."

	s := createTestServer(
		t,
		&filtering.Config{},
		ServerConfig{
			UDPListenAddrs: []*net.UDPAddr{{}},
			TCPListenAddrs: []*net.TCPAddr{{}},
			FilteringConfig: FilteringConfig{
				EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
			},
		},
		aghtest.NewUpstreamMock(func(req *dns.Msg) (resp *dns.Msg, err error) {
			return aghalg.Coalesce(
				aghtest.MatchedResponse(req, dns.TypePTR, reqAddr, locDomain),
				new(dns.Msg).SetRcode(req, dns.RcodeNameError),
			), nil
		}),
	)

	var proxyCtx *proxy.DNSContext
	var dnsCtx *dnsContext
	setup := func(use bool) {
		proxyCtx = &proxy.DNSContext{
			Addr: &net.TCPAddr{
				IP: net.IP{127, 0, 0, 1},
			},
			Req: createTestMessageWithType(reqAddr, dns.TypePTR),
		}
		dnsCtx = &dnsContext{
			proxyCtx:        proxyCtx,
			unreversedReqIP: net.IP{192, 168, 1, 1},
		}
		s.conf.UsePrivateRDNS = use
	}

	t.Run("enabled", func(t *testing.T) {
		setup(true)

		rc := s.processLocalPTR(dnsCtx)
		require.Equal(t, resultCodeSuccess, rc)
		require.NotEmpty(t, proxyCtx.Res.Answer)

		assert.Equal(t, locDomain, proxyCtx.Res.Answer[0].(*dns.PTR).Ptr)
	})

	t.Run("disabled", func(t *testing.T) {
		setup(false)

		rc := s.processLocalPTR(dnsCtx)
		require.Equal(t, resultCodeFinish, rc)
		require.Empty(t, proxyCtx.Res.Answer)
	})
}

func TestIPStringFromAddr(t *testing.T) {
	t.Run("not_nil", func(t *testing.T) {
		addr := net.UDPAddr{
			IP:   net.ParseIP("1:2:3::4"),
			Port: 12345,
			Zone: "eth0",
		}
		assert.Equal(t, ipStringFromAddr(&addr), addr.IP.String())
	})

	t.Run("nil", func(t *testing.T) {
		assert.Empty(t, ipStringFromAddr(nil))
	})
}

// TODO(e.burkov):  Add fuzzing when moving to golibs.
func TestExtractARPASubnet(t *testing.T) {
	const (
		v4Suf   = `in-addr.arpa.`
		v4Part  = `2.1.` + v4Suf
		v4Whole = `4.3.` + v4Part

		v6Suf   = `ip6.arpa.`
		v6Part  = `4.3.2.1.0.0.0.0.0.0.0.0.0.0.0.0.` + v6Suf
		v6Whole = `f.e.d.c.0.0.0.0.0.0.0.0.0.0.0.0.` + v6Part
	)

	v4Pref := netip.MustParsePrefix("1.2.3.4/32")
	v4PrefPart := netip.MustParsePrefix("1.2.0.0/16")
	v6Pref := netip.MustParsePrefix("::1234:0:0:0:cdef/128")
	v6PrefPart := netip.MustParsePrefix("0:0:0:1234::/64")

	testCases := []struct {
		want    netip.Prefix
		name    string
		domain  string
		wantErr string
	}{{
		want:   netip.Prefix{},
		name:   "not_an_arpa",
		domain: "some.domain.name.",
		wantErr: `bad arpa domain name "some.domain.name.": ` +
			`not a reversed ip network`,
	}, {
		want:   netip.Prefix{},
		name:   "bad_domain_name",
		domain: "abc.123.",
		wantErr: `bad domain name "abc.123": ` +
			`bad top-level domain name label "123": all octets are numeric`,
	}, {
		want:    v4Pref,
		name:    "whole_v4",
		domain:  v4Whole,
		wantErr: "",
	}, {
		want:    v4PrefPart,
		name:    "partial_v4",
		domain:  v4Part,
		wantErr: "",
	}, {
		want:    v4Pref,
		name:    "whole_v4_within_domain",
		domain:  "a." + v4Whole,
		wantErr: "",
	}, {
		want:    v4Pref,
		name:    "whole_v4_additional_label",
		domain:  "5." + v4Whole,
		wantErr: "",
	}, {
		want:    v4PrefPart,
		name:    "partial_v4_within_domain",
		domain:  "a." + v4Part,
		wantErr: "",
	}, {
		want:    v4PrefPart,
		name:    "overflow_v4",
		domain:  "256." + v4Part,
		wantErr: "",
	}, {
		want:    v4PrefPart,
		name:    "overflow_v4_within_domain",
		domain:  "a.256." + v4Part,
		wantErr: "",
	}, {
		want:   netip.Prefix{},
		name:   "empty_v4",
		domain: v4Suf,
		wantErr: `bad arpa domain name "in-addr.arpa": ` +
			`not a reversed ip network`,
	}, {
		want:   netip.Prefix{},
		name:   "empty_v4_within_domain",
		domain: "a." + v4Suf,
		wantErr: `bad arpa domain name "in-addr.arpa": ` +
			`not a reversed ip network`,
	}, {
		want:    v6Pref,
		name:    "whole_v6",
		domain:  v6Whole,
		wantErr: "",
	}, {
		want:   v6PrefPart,
		name:   "partial_v6",
		domain: v6Part,
	}, {
		want:    v6Pref,
		name:    "whole_v6_within_domain",
		domain:  "g." + v6Whole,
		wantErr: "",
	}, {
		want:    v6Pref,
		name:    "whole_v6_additional_label",
		domain:  "1." + v6Whole,
		wantErr: "",
	}, {
		want:    v6PrefPart,
		name:    "partial_v6_within_domain",
		domain:  "label." + v6Part,
		wantErr: "",
	}, {
		want:    netip.Prefix{},
		name:    "empty_v6",
		domain:  v6Suf,
		wantErr: `bad arpa domain name "ip6.arpa": not a reversed ip network`,
	}, {
		want:    netip.Prefix{},
		name:    "empty_v6_within_domain",
		domain:  "g." + v6Suf,
		wantErr: `bad arpa domain name "ip6.arpa": not a reversed ip network`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			subnet, err := extractARPASubnet(tc.domain)
			testutil.AssertErrorMsg(t, tc.wantErr, err)
			assert.Equal(t, tc.want, subnet)
		})
	}
}
