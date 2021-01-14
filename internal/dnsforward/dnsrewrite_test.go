package dnsforward

import (
	"net"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestServer_FilterDNSRewrite(t *testing.T) {
	// Helper data.
	ip4 := net.IP{127, 0, 0, 1}
	ip6 := net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	mx := &rules.DNSMX{
		Exchange:   "mail.example.com",
		Preference: 32,
	}
	svcb := &rules.DNSSVCB{
		Params:   map[string]string{"alpn": "h3"},
		Target:   "example.com",
		Priority: 32,
	}
	const domain = "example.com"

	// Helper functions and entities.
	srv := &Server{}
	makeQ := func(qtype rules.RRType) (req *dns.Msg) {
		return &dns.Msg{
			Question: []dns.Question{{
				Qtype: qtype,
			}},
		}
	}
	makeRes := func(rcode rules.RCode, rr rules.RRType, v rules.RRValue) (res dnsfilter.Result) {
		resp := dnsfilter.DNSRewriteResultResponse{
			rr: []rules.RRValue{v},
		}
		return dnsfilter.Result{
			DNSRewriteResult: &dnsfilter.DNSRewriteResult{
				RCode:    rcode,
				Response: resp,
			},
		}
	}

	// Tests.
	t.Run("nxdomain", func(t *testing.T) {
		req := makeQ(dns.TypeA)
		res := makeRes(dns.RcodeNameError, 0, nil)
		d := &proxy.DNSContext{}

		err := srv.filterDNSRewrite(req, res, d)
		assert.Nil(t, err)
		assert.Equal(t, dns.RcodeNameError, d.Res.Rcode)
	})

	t.Run("noerror_empty", func(t *testing.T) {
		req := makeQ(dns.TypeA)
		res := makeRes(dns.RcodeSuccess, 0, nil)
		d := &proxy.DNSContext{}

		err := srv.filterDNSRewrite(req, res, d)
		assert.Nil(t, err)
		assert.Equal(t, dns.RcodeSuccess, d.Res.Rcode)
		assert.Empty(t, d.Res.Answer)
	})

	t.Run("noerror_a", func(t *testing.T) {
		req := makeQ(dns.TypeA)
		res := makeRes(dns.RcodeSuccess, dns.TypeA, ip4)
		d := &proxy.DNSContext{}

		err := srv.filterDNSRewrite(req, res, d)
		assert.Nil(t, err)
		assert.Equal(t, dns.RcodeSuccess, d.Res.Rcode)
		if assert.Len(t, d.Res.Answer, 1) {
			assert.Equal(t, ip4, d.Res.Answer[0].(*dns.A).A)
		}
	})

	t.Run("noerror_aaaa", func(t *testing.T) {
		req := makeQ(dns.TypeAAAA)
		res := makeRes(dns.RcodeSuccess, dns.TypeAAAA, ip6)
		d := &proxy.DNSContext{}

		err := srv.filterDNSRewrite(req, res, d)
		assert.Nil(t, err)
		assert.Equal(t, dns.RcodeSuccess, d.Res.Rcode)
		if assert.Len(t, d.Res.Answer, 1) {
			assert.Equal(t, ip6, d.Res.Answer[0].(*dns.AAAA).AAAA)
		}
	})

	t.Run("noerror_ptr", func(t *testing.T) {
		req := makeQ(dns.TypePTR)
		res := makeRes(dns.RcodeSuccess, dns.TypePTR, domain)
		d := &proxy.DNSContext{}

		err := srv.filterDNSRewrite(req, res, d)
		assert.Nil(t, err)
		assert.Equal(t, dns.RcodeSuccess, d.Res.Rcode)
		if assert.Len(t, d.Res.Answer, 1) {
			assert.Equal(t, domain, d.Res.Answer[0].(*dns.PTR).Ptr)
		}
	})

	t.Run("noerror_txt", func(t *testing.T) {
		req := makeQ(dns.TypeTXT)
		res := makeRes(dns.RcodeSuccess, dns.TypeTXT, domain)
		d := &proxy.DNSContext{}

		err := srv.filterDNSRewrite(req, res, d)
		assert.Nil(t, err)
		assert.Equal(t, dns.RcodeSuccess, d.Res.Rcode)
		if assert.Len(t, d.Res.Answer, 1) {
			assert.Equal(t, []string{domain}, d.Res.Answer[0].(*dns.TXT).Txt)
		}
	})

	t.Run("noerror_mx", func(t *testing.T) {
		req := makeQ(dns.TypeMX)
		res := makeRes(dns.RcodeSuccess, dns.TypeMX, mx)
		d := &proxy.DNSContext{}

		err := srv.filterDNSRewrite(req, res, d)
		assert.Nil(t, err)
		assert.Equal(t, dns.RcodeSuccess, d.Res.Rcode)
		if assert.Len(t, d.Res.Answer, 1) {
			ans := d.Res.Answer[0].(*dns.MX)
			assert.Equal(t, mx.Exchange, ans.Mx)
			assert.Equal(t, mx.Preference, ans.Preference)
		}
	})

	t.Run("noerror_svcb", func(t *testing.T) {
		req := makeQ(dns.TypeSVCB)
		res := makeRes(dns.RcodeSuccess, dns.TypeSVCB, svcb)
		d := &proxy.DNSContext{}

		err := srv.filterDNSRewrite(req, res, d)
		assert.Nil(t, err)
		assert.Equal(t, dns.RcodeSuccess, d.Res.Rcode)
		if assert.Len(t, d.Res.Answer, 1) {
			ans := d.Res.Answer[0].(*dns.SVCB)
			assert.Equal(t, dns.SVCB_ALPN, ans.Value[0].Key())
			assert.Equal(t, svcb.Params["alpn"], ans.Value[0].String())
			assert.Equal(t, svcb.Target, ans.Target)
			assert.Equal(t, svcb.Priority, ans.Priority)
		}
	})

	t.Run("noerror_https", func(t *testing.T) {
		req := makeQ(dns.TypeHTTPS)
		res := makeRes(dns.RcodeSuccess, dns.TypeHTTPS, svcb)
		d := &proxy.DNSContext{}

		err := srv.filterDNSRewrite(req, res, d)
		assert.Nil(t, err)
		assert.Equal(t, dns.RcodeSuccess, d.Res.Rcode)
		if assert.Len(t, d.Res.Answer, 1) {
			ans := d.Res.Answer[0].(*dns.HTTPS)
			assert.Equal(t, dns.SVCB_ALPN, ans.Value[0].Key())
			assert.Equal(t, svcb.Params["alpn"], ans.Value[0].String())
			assert.Equal(t, svcb.Target, ans.Target)
			assert.Equal(t, svcb.Priority, ans.Priority)
		}
	})
}
