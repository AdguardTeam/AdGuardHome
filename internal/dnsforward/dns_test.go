package dnsforward

import (
	"net"
	"testing"

	"github.com/AdguardTeam/dnsproxy/proxy"
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
