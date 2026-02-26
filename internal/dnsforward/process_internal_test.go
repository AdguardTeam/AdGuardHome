package dnsforward

import (
	"cmp"
	"context"
	"crypto/tls"
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ddrTestDomainName = "dns.example.net"
	ddrTestFQDN       = ddrTestDomainName + "."
)

func TestServer_ProcessInitial(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		target       string
		wantRCode    rules.RCode
		qType        rules.RRType
		aaaaDisabled bool
		wantRC       resultCode
	}{{
		name:         "success",
		target:       testQuestionTarget,
		wantRCode:    -1,
		qType:        dns.TypeA,
		aaaaDisabled: false,
		wantRC:       resultCodeSuccess,
	}, {
		name:         "aaaa_disabled",
		target:       testQuestionTarget,
		wantRCode:    dns.RcodeSuccess,
		qType:        dns.TypeAAAA,
		aaaaDisabled: true,
		wantRC:       resultCodeFinish,
	}, {
		name:         "aaaa_disabled_a",
		target:       testQuestionTarget,
		wantRCode:    -1,
		qType:        dns.TypeA,
		aaaaDisabled: true,
		wantRC:       resultCodeSuccess,
	}, {
		name:         "mozilla_canary",
		target:       mozillaFQDN,
		wantRCode:    dns.RcodeNameError,
		qType:        dns.TypeA,
		aaaaDisabled: false,
		wantRC:       resultCodeFinish,
	}, {
		name:         "adguardhome_healthcheck",
		target:       healthcheckFQDN,
		wantRCode:    dns.RcodeSuccess,
		qType:        dns.TypeA,
		aaaaDisabled: false,
		wantRC:       resultCodeFinish,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := ServerConfig{
				TLSConf: &TLSConfig{},
				Config: Config{
					AAAADisabled:     tc.aaaaDisabled,
					UpstreamMode:     UpstreamModeLoadBalance,
					EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
					ClientsContainer: EmptyClientsContainer{},
				},
				ServePlainDNS: true,
			}

			s := createTestServer(t, &filtering.Config{
				BlockingMode: filtering.BlockingModeDefault,
			}, c)

			var gotAddr netip.Addr
			s.addrProc = &aghtest.AddressProcessor{
				OnProcess: func(ctx context.Context, ip netip.Addr) { gotAddr = ip },
				OnClose:   func() (_ error) { panic(testutil.UnexpectedCall()) },
			}

			dctx := &dnsContext{
				proxyCtx: &proxy.DNSContext{
					Req:       createTestMessageWithType(tc.target, tc.qType),
					Addr:      testClientAddrPort,
					RequestID: 1234,
				},
			}

			gotRC := s.processInitial(testutil.ContextWithTimeout(t, testTimeout), dctx)
			assert.Equal(t, tc.wantRC, gotRC)
			assert.Equal(t, testClientAddrPort.Addr(), gotAddr)

			if tc.wantRCode > 0 {
				gotResp := dctx.proxyCtx.Res
				require.NotNil(t, gotResp)

				assert.Equal(t, tc.wantRCode, gotResp.Rcode)
			}
		})
	}
}

func TestServer_ProcessFilteringAfterResponse(t *testing.T) {
	t.Parallel()

	var (
		testIPv4 net.IP = netip.MustParseAddr("1.1.1.1").AsSlice()
		testIPv6 net.IP = netip.MustParseAddr("1234::cdef").AsSlice()
	)

	testCases := []struct {
		name         string
		req          *dns.Msg
		aaaaDisabled bool
		respAns      []dns.RR
		wantRC       resultCode
		wantRespAns  []dns.RR
	}{{
		name:         "pass",
		req:          createTestMessageWithType(aghtest.ReqFQDN, dns.TypeHTTPS),
		aaaaDisabled: false,
		respAns: newSVCBHintsAnswer(
			aghtest.ReqFQDN,
			[]dns.SVCBKeyValue{
				&dns.SVCBIPv4Hint{Hint: []net.IP{testIPv4}},
				&dns.SVCBIPv6Hint{Hint: []net.IP{testIPv6}},
			},
		),
		wantRespAns: newSVCBHintsAnswer(
			aghtest.ReqFQDN,
			[]dns.SVCBKeyValue{
				&dns.SVCBIPv4Hint{Hint: []net.IP{testIPv4}},
				&dns.SVCBIPv6Hint{Hint: []net.IP{testIPv6}},
			},
		),
		wantRC: resultCodeSuccess,
	}, {
		name:         "filter",
		req:          createTestMessageWithType(aghtest.ReqFQDN, dns.TypeHTTPS),
		aaaaDisabled: true,
		respAns: newSVCBHintsAnswer(
			aghtest.ReqFQDN,
			[]dns.SVCBKeyValue{
				&dns.SVCBIPv4Hint{Hint: []net.IP{testIPv4}},
				&dns.SVCBIPv6Hint{Hint: []net.IP{testIPv6}},
			},
		),
		wantRespAns: newSVCBHintsAnswer(
			aghtest.ReqFQDN,
			[]dns.SVCBKeyValue{
				&dns.SVCBIPv4Hint{Hint: []net.IP{testIPv4}},
			},
		),
		wantRC: resultCodeSuccess,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := ServerConfig{
				TLSConf: &TLSConfig{},
				Config: Config{
					AAAADisabled:     tc.aaaaDisabled,
					UpstreamMode:     UpstreamModeLoadBalance,
					EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
					ClientsContainer: EmptyClientsContainer{},
				},
				ServePlainDNS: true,
			}

			s := createTestServer(t, &filtering.Config{
				BlockingMode: filtering.BlockingModeDefault,
			}, c)

			resp := newResp(dns.RcodeSuccess, tc.req, tc.respAns)
			dctx := &dnsContext{
				setts: &filtering.Settings{
					FilteringEnabled:  true,
					ProtectionEnabled: true,
				},
				protectionEnabled:    true,
				responseFromUpstream: true,
				result:               &filtering.Result{},
				proxyCtx: &proxy.DNSContext{
					Proto: proxy.ProtoUDP,
					Req:   tc.req,
					Res:   resp,
					Addr:  testClientAddrPort,
				},
			}
			ctx := testutil.ContextWithTimeout(t, testTimeout)
			gotRC := s.processFilteringAfterResponse(ctx, dctx)
			assert.Equal(t, tc.wantRC, gotRC)
			assert.Equal(t, newResp(dns.RcodeSuccess, tc.req, tc.wantRespAns), dctx.proxyCtx.Res)
		})
	}
}

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
		addrsDoH   []*net.TCPAddr
		addrsDoT   []*net.TCPAddr
		addrsDoQ   []*net.UDPAddr
		qtype      uint16
		ddrEnabled bool
	}{{
		name:       "pass_host",
		wantRes:    resultCodeSuccess,
		host:       testQuestionTarget,
		qtype:      dns.TypeSVCB,
		ddrEnabled: true,
		addrsDoH:   []*net.TCPAddr{{Port: 8043}},
	}, {
		name:       "pass_qtype",
		wantRes:    resultCodeFinish,
		host:       ddrHostFQDN,
		qtype:      dns.TypeA,
		ddrEnabled: true,
		addrsDoH:   []*net.TCPAddr{{Port: 8043}},
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
		addrsDoH:   []*net.TCPAddr{{Port: 8043}},
	}, {
		name:       "dot",
		wantRes:    resultCodeFinish,
		want:       []*dns.SVCB{dotSVCB},
		host:       ddrHostFQDN,
		qtype:      dns.TypeSVCB,
		ddrEnabled: true,
		addrsDoT:   []*net.TCPAddr{{Port: 8043}},
	}, {
		name:       "doh",
		wantRes:    resultCodeFinish,
		want:       []*dns.SVCB{dohSVCB},
		host:       ddrHostFQDN,
		qtype:      dns.TypeSVCB,
		ddrEnabled: true,
		addrsDoH:   []*net.TCPAddr{{Port: 8044}},
	}, {
		name:       "doq",
		wantRes:    resultCodeFinish,
		want:       []*dns.SVCB{doqSVCB},
		host:       ddrHostFQDN,
		qtype:      dns.TypeSVCB,
		ddrEnabled: true,
		addrsDoQ:   []*net.UDPAddr{{Port: 8042}},
	}, {
		name:       "dot_doh",
		wantRes:    resultCodeFinish,
		want:       []*dns.SVCB{dotSVCB, dohSVCB},
		host:       ddrHostFQDN,
		qtype:      dns.TypeSVCB,
		ddrEnabled: true,
		addrsDoT:   []*net.TCPAddr{{Port: 8043}},
		addrsDoH:   []*net.TCPAddr{{Port: 8044}},
	}}

	_, certPem, keyPem := createServerTLSConfig(t)
	cert, err := tls.X509KeyPair(certPem, keyPem)
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := createTestServer(t, &filtering.Config{
				BlockingMode: filtering.BlockingModeDefault,
			}, ServerConfig{
				Config: Config{
					HandleDDR:        tc.ddrEnabled,
					UpstreamMode:     UpstreamModeLoadBalance,
					EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
					ClientsContainer: EmptyClientsContainer{},
				},
				TLSConf: &TLSConfig{
					ServerName:       ddrTestDomainName,
					Cert:             &cert,
					TLSListenAddrs:   tc.addrsDoT,
					HTTPSListenAddrs: tc.addrsDoH,
					QUICListenAddrs:  tc.addrsDoQ,
				},
				ServePlainDNS: true,
			})
			// TODO(e.burkov):  Generate a certificate actually containing the
			// IP addresses.
			s.hasIPAddrs = true

			req := createTestMessageWithType(tc.host, tc.qtype)

			dctx := &dnsContext{
				proxyCtx: &proxy.DNSContext{
					Req: req,
				},
			}

			res := s.processDDRQuery(testutil.ContextWithTimeout(t, testTimeout), dctx)
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

// createTestDNSFilter returns the minimum valid DNSFilter.
func createTestDNSFilter(tb testing.TB) (f *filtering.DNSFilter) {
	tb.Helper()

	f, err := filtering.New(&filtering.Config{
		Logger:       testLogger,
		BlockingMode: filtering.BlockingModeDefault,
	}, []filtering.Filter{})
	require.NoError(tb, err)

	return f
}

func TestServer_ProcessDHCPHosts_localRestriction(t *testing.T) {
	const (
		localDomainSuffix = "lan"
		dhcpClient        = "example"

		knownHost   = dhcpClient + "." + localDomainSuffix
		unknownHost = "wronghost." + localDomainSuffix
	)

	knownIP := netip.MustParseAddr("1.2.3.4")
	dhcp := &testDHCP{
		OnEnabled: func() (_ bool) { return true },
		OnIPByHost: func(host string) (ip netip.Addr) {
			if host == dhcpClient {
				ip = knownIP
			}

			return ip
		},
	}

	testCases := []struct {
		wantIP     netip.Addr
		name       string
		host       string
		isLocalCli bool
	}{{
		wantIP:     knownIP,
		name:       "local_client_success",
		host:       knownHost,
		isLocalCli: true,
	}, {
		wantIP:     netip.Addr{},
		name:       "local_client_unknown_host",
		host:       unknownHost,
		isLocalCli: true,
	}, {
		wantIP:     netip.Addr{},
		name:       "external_client_known_host",
		host:       knownHost,
		isLocalCli: false,
	}, {
		wantIP:     netip.Addr{},
		name:       "external_client_unknown_host",
		host:       unknownHost,
		isLocalCli: false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{
				dnsFilter:         createTestDNSFilter(t),
				dhcpServer:        dhcp,
				localDomainSuffix: localDomainSuffix,
				baseLogger:        testLogger,
				logger:            testLogger,
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
					Req:             req,
					IsPrivateClient: tc.isLocalCli,
				},
			}

			res := s.processDHCPHosts(testutil.ContextWithTimeout(t, testTimeout), dctx)

			pctx := dctx.proxyCtx
			if !tc.isLocalCli {
				require.Equal(t, resultCodeFinish, res)
				require.NotNil(t, pctx.Res)

				assert.Equal(t, dns.RcodeNameError, pctx.Res.Rcode)
				assert.Empty(t, pctx.Res.Answer)

				return
			}

			require.Equal(t, resultCodeSuccess, res)

			if tc.wantIP == (netip.Addr{}) {
				assert.Nil(t, pctx.Res)

				return
			}

			require.NotNil(t, pctx.Res)

			ans := pctx.Res.Answer
			require.Len(t, ans, 1)

			a := testutil.RequireTypeAssert[*dns.A](t, ans[0])

			ip, err := netutil.IPToAddr(a.A, netutil.AddrFamilyIPv4)
			require.NoError(t, err)

			assert.Equal(t, tc.wantIP, ip)
		})
	}
}

func TestServer_ProcessDHCPHosts(t *testing.T) {
	const (
		localTLD  = "lan"
		customTLD = "custom"

		oneLabelClient     = "example"
		twoLabelClient     = "sub.example"
		externalHost       = oneLabelClient + ".com"
		oneLabelClientHost = oneLabelClient + "." + localTLD
		twoLabelClientHost = twoLabelClient + "." + localTLD
	)

	knownClients := map[string]netip.Addr{
		oneLabelClient: netip.MustParseAddr("1.2.3.4"),
		twoLabelClient: netip.MustParseAddr("1.2.3.5"),
	}

	testDHCP := &testDHCP{
		OnEnabled: func() (ok bool) { return true },
		OnIPByHost: func(host string) (ip netip.Addr) {
			return knownClients[host]
		},
		OnHostByIP: func(ip netip.Addr) (_ string) { panic(testutil.UnexpectedCall(ip)) },
	}

	testCases := []struct {
		wantIP  netip.Addr
		name    string
		host    string
		suffix  string
		wantRes resultCode
		qtyp    uint16
	}{{
		wantIP:  netip.Addr{},
		name:    "external",
		host:    externalHost,
		suffix:  localTLD,
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeA,
	}, {
		wantIP:  netip.Addr{},
		name:    "external_non_a",
		host:    externalHost,
		suffix:  localTLD,
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeCNAME,
	}, {
		wantIP:  knownClients[oneLabelClient],
		name:    "internal",
		host:    oneLabelClientHost,
		suffix:  localTLD,
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeA,
	}, {
		wantIP:  knownClients[twoLabelClient],
		name:    "internal_two_label",
		host:    twoLabelClientHost,
		suffix:  localTLD,
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeA,
	}, {
		wantIP:  netip.Addr{},
		name:    "internal_unknown",
		host:    "example-new.lan",
		suffix:  localTLD,
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeA,
	}, {
		wantIP:  netip.Addr{},
		name:    "internal_aaaa",
		host:    oneLabelClientHost,
		suffix:  localTLD,
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeAAAA,
	}, {
		wantIP:  knownClients[oneLabelClient],
		name:    "custom_suffix",
		host:    oneLabelClient + "." + customTLD,
		suffix:  customTLD,
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeA,
	}, {
		wantIP:  knownClients[twoLabelClient],
		name:    "custom_suffix_two_label",
		host:    twoLabelClient + "." + customTLD,
		suffix:  customTLD,
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeA,
	}}

	for _, tc := range testCases {
		s := &Server{
			dnsFilter:         createTestDNSFilter(t),
			dhcpServer:        testDHCP,
			localDomainSuffix: tc.suffix,
			baseLogger:        testLogger,
			logger:            testLogger,
		}

		req := (&dns.Msg{}).SetQuestion(dns.Fqdn(tc.host), tc.qtyp)

		dctx := &dnsContext{
			proxyCtx: &proxy.DNSContext{
				Req:             req,
				IsPrivateClient: true,
			},
		}

		t.Run(tc.name, func(t *testing.T) {
			res := s.processDHCPHosts(testutil.ContextWithTimeout(t, testTimeout), dctx)
			pctx := dctx.proxyCtx
			assert.Equal(t, tc.wantRes, res)
			require.NoError(t, dctx.err)

			if tc.qtyp == dns.TypeAAAA {
				// TODO(a.garipov): Remove this special handling when we fully
				// support AAAA.
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

func TestServer_ProcessUpstream_localPTR(t *testing.T) {
	const locDomain = "some.local."
	const reqAddr = "1.1.168.192.in-addr.arpa."

	localUpsHdlr := dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		resp := cmp.Or(
			aghtest.MatchedResponse(req, dns.TypePTR, reqAddr, locDomain),
			(&dns.Msg{}).SetRcode(req, dns.RcodeNameError),
		)

		require.NoError(testutil.PanicT{}, w.WriteMsg(resp))
	})
	localUpsAddr := aghtest.StartLocalhostUpstream(t, localUpsHdlr).String()

	newPrxCtx := func() (prxCtx *proxy.DNSContext) {
		return &proxy.DNSContext{
			Addr:                 testClientAddrPort,
			Req:                  createTestMessageWithType(reqAddr, dns.TypePTR),
			IsPrivateClient:      true,
			RequestedPrivateRDNS: netip.MustParsePrefix("192.168.1.1/32"),
		}
	}

	t.Run("enabled", func(t *testing.T) {
		s := createTestServer(
			t,
			&filtering.Config{
				BlockingMode: filtering.BlockingModeDefault,
			},
			ServerConfig{
				UDPListenAddrs: []*net.UDPAddr{{}},
				TCPListenAddrs: []*net.TCPAddr{{}},
				TLSConf:        &TLSConfig{},
				Config: Config{
					UpstreamMode:     UpstreamModeLoadBalance,
					EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
					ClientsContainer: EmptyClientsContainer{},
				},
				UsePrivateRDNS:    true,
				LocalPTRResolvers: []string{localUpsAddr},
				ServePlainDNS:     true,
			},
		)
		ctx := testutil.ContextWithTimeout(t, testTimeout)
		pctx := newPrxCtx()
		rc := s.processUpstream(ctx, &dnsContext{proxyCtx: pctx})
		require.Equal(t, resultCodeSuccess, rc)
		require.NotEmpty(t, pctx.Res.Answer)
		ptr := testutil.RequireTypeAssert[*dns.PTR](t, pctx.Res.Answer[0])

		assert.Equal(t, locDomain, ptr.Ptr)
	})

	t.Run("disabled", func(t *testing.T) {
		s := createTestServer(
			t,
			&filtering.Config{
				BlockingMode: filtering.BlockingModeDefault,
			},
			ServerConfig{
				UDPListenAddrs: []*net.UDPAddr{{}},
				TCPListenAddrs: []*net.TCPAddr{{}},
				TLSConf:        &TLSConfig{},
				Config: Config{
					UpstreamMode:     UpstreamModeLoadBalance,
					EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
					ClientsContainer: EmptyClientsContainer{},
				},
				UsePrivateRDNS:    false,
				LocalPTRResolvers: []string{localUpsAddr},
				ServePlainDNS:     true,
			},
		)
		pctx := newPrxCtx()

		ctx := testutil.ContextWithTimeout(t, testTimeout)
		rc := s.processUpstream(ctx, &dnsContext{proxyCtx: pctx})
		require.Equal(t, resultCodeError, rc)
		require.Empty(t, pctx.Res.Answer)
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
