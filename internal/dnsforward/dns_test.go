package dnsforward

import (
	"net"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_ProcessInternalHosts(t *testing.T) {
	knownIP := net.IP{1, 2, 3, 4}
	testCases := []struct {
		name       string
		host       string
		suffix     string
		wantErrMsg string
		wantIP     net.IP
		qtyp       uint16
		wantRes    resultCode
	}{{
		name:       "success_external",
		host:       "example.com",
		suffix:     defaultAutohostSuffix,
		wantErrMsg: "",
		wantIP:     nil,
		qtyp:       dns.TypeA,
		wantRes:    resultCodeSuccess,
	}, {
		name:       "success_external_non_a",
		host:       "example.com",
		suffix:     defaultAutohostSuffix,
		wantErrMsg: "",
		wantIP:     nil,
		qtyp:       dns.TypeCNAME,
		wantRes:    resultCodeSuccess,
	}, {
		name:       "success_internal",
		host:       "example.lan",
		suffix:     defaultAutohostSuffix,
		wantErrMsg: "",
		wantIP:     knownIP,
		qtyp:       dns.TypeA,
		wantRes:    resultCodeSuccess,
	}, {
		name:       "success_internal_unknown",
		host:       "example-new.lan",
		suffix:     defaultAutohostSuffix,
		wantErrMsg: "",
		wantIP:     nil,
		qtyp:       dns.TypeA,
		wantRes:    resultCodeSuccess,
	}, {
		name:       "success_internal_aaaa",
		host:       "example.lan",
		suffix:     defaultAutohostSuffix,
		wantErrMsg: "",
		wantIP:     nil,
		qtyp:       dns.TypeAAAA,
		wantRes:    resultCodeSuccess,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{
				autohostSuffix: tc.suffix,
				tableHostToIP: map[string]net.IP{
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
			}

			res := s.processInternalHosts(dctx)
			assert.Equal(t, tc.wantRes, res)

			if tc.wantErrMsg == "" {
				assert.NoError(t, dctx.err)
			} else {
				require.Error(t, dctx.err)

				assert.Equal(t, tc.wantErrMsg, dctx.err.Error())
			}

			pctx := dctx.proxyCtx
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

func TestLocalRestriction(t *testing.T) {
	s := createTestServer(t, &dnsfilter.Config{}, ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
	})
	ups := &aghtest.TestUpstream{
		Reverse: map[string][]string{
			"251.252.253.254.in-addr.arpa.": {"host1.example.net."},
			"1.1.168.192.in-addr.arpa.":     {"some.local-client."},
		},
	}
	s.localResolvers = &aghtest.Exchanger{Ups: ups}
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
			require.Nil(t, err)
			require.NotNil(t, pctx.Res)
			require.Len(t, pctx.Res.Answer, tc.wantLen)
			if tc.wantLen > 0 {
				assert.Equal(t, tc.want, pctx.Res.Answer[0].Header().Name)
			}
		})
	}
}
