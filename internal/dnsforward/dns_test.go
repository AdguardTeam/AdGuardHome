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
			&dns.SVCBDoHPath{Template: "/dns-query?dns"},
		},
	}

	dotSVCB := &dns.SVCB{
		Priority: 2,
		Target:   ddrTestFQDN,
		Value: []dns.SVCBKeyValue{
			&dns.SVCBAlpn{Alpn: []string{"dot"}},
			&dns.SVCBPort{Port: 8043},
		},
	}

	doqSVCB := &dns.SVCB{
		Priority: 3,
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

	proxyConf := proxy.Config{}

	if portDoT > 0 {
		proxyConf.TLSListenAddr = []*net.TCPAddr{{Port: portDoT}}
	}

	if portDoQ > 0 {
		proxyConf.QUICListenAddr = []*net.UDPAddr{{Port: portDoQ}}
	}

	s = &Server{
		dnsProxy: &proxy.Proxy{
			Config: proxyConf,
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

	if portDoH > 0 {
		s.conf.TLSConfig.HTTPSListenAddrs = []*net.TCPAddr{{Port: portDoH}}
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

func TestServer_ProcessInternalHosts_localRestriction(t *testing.T) {
	knownIP := net.IP{1, 2, 3, 4}

	testCases := []struct {
		name       string
		host       string
		wantIP     net.IP
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
		wantIP:     nil,
		wantRes:    resultCodeFinish,
		isLocalCli: true,
	}, {
		name:       "external_client_known_host",
		host:       "example.lan",
		wantIP:     nil,
		wantRes:    resultCodeFinish,
		isLocalCli: false,
	}, {
		name:       "external_client_unknown_host",
		host:       "wronghost.lan",
		wantIP:     nil,
		wantRes:    resultCodeFinish,
		isLocalCli: false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{
				dhcpServer:        &testDHCP{},
				localDomainSuffix: defaultLocalDomainSuffix,
				tableHostToIP: hostToIPTable{
					"example": knownIP,
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

			res := s.processInternalHosts(dctx)
			require.Equal(t, tc.wantRes, res)
			pctx := dctx.proxyCtx
			if tc.wantRes == resultCodeFinish {
				require.NotNil(t, pctx.Res)

				assert.Equal(t, dns.RcodeNameError, pctx.Res.Rcode)
				assert.Len(t, pctx.Res.Answer, 0)

				return
			}

			if tc.wantIP == nil {
				assert.Nil(t, pctx.Res)
			} else {
				require.NotNil(t, pctx.Res)

				ans := pctx.Res.Answer
				require.Len(t, ans, 1)

				assert.Equal(t, tc.wantIP, ans[0].(*dns.A).A)
			}
		})
	}
}

func TestServer_ProcessInternalHosts(t *testing.T) {
	const (
		examplecom = "example.com"
		examplelan = "example.lan"
	)

	knownIP := net.IP{1, 2, 3, 4}
	testCases := []struct {
		name    string
		host    string
		suffix  string
		wantIP  net.IP
		wantRes resultCode
		qtyp    uint16
	}{{
		name:    "success_external",
		host:    examplecom,
		suffix:  defaultLocalDomainSuffix,
		wantIP:  nil,
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeA,
	}, {
		name:    "success_external_non_a",
		host:    examplecom,
		suffix:  defaultLocalDomainSuffix,
		wantIP:  nil,
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
		wantIP:  nil,
		wantRes: resultCodeFinish,
		qtyp:    dns.TypeA,
	}, {
		name:    "success_internal_aaaa",
		host:    examplelan,
		suffix:  defaultLocalDomainSuffix,
		wantIP:  nil,
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeAAAA,
	}, {
		name:    "success_custom_suffix",
		host:    "example.custom",
		suffix:  ".custom.",
		wantIP:  knownIP,
		wantRes: resultCodeSuccess,
		qtyp:    dns.TypeA,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{
				dhcpServer:        &testDHCP{},
				localDomainSuffix: tc.suffix,
				tableHostToIP: hostToIPTable{
					"example": knownIP,
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

			res := s.processInternalHosts(dctx)
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
			} else if tc.wantIP == nil {
				assert.Nil(t, pctx.Res)
			} else {
				require.NotNil(t, pctx.Res)

				ans := pctx.Res.Answer
				require.Len(t, ans, 1)

				assert.Equal(t, tc.wantIP, ans[0].(*dns.A).A)
			}
		})
	}
}

func TestServer_ProcessRestrictLocal(t *testing.T) {
	ups := &aghtest.Upstream{
		Reverse: map[string][]string{
			"251.252.253.254.in-addr.arpa.": {"host1.example.net."},
			"1.1.168.192.in-addr.arpa.":     {"some.local-client."},
		},
	}
	s := createTestServer(t, &filtering.Config{}, ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
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

	s := createTestServer(t, &filtering.Config{}, ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
	}, &aghtest.Upstream{
		Reverse: map[string][]string{
			reqAddr: {locDomain},
		},
	})

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
