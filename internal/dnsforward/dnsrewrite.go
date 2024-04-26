package dnsforward

import (
	"fmt"
	"net/netip"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

// filterDNSRewriteResponse handles a single DNS rewrite response entry.  It
// returns the properly constructed answer resource record.
func (s *Server) filterDNSRewriteResponse(
	req *dns.Msg,
	rr rules.RRType,
	v rules.RRValue,
) (ans dns.RR, err error) {
	switch rr {
	case dns.TypeA, dns.TypeAAAA:
		return s.ansFromDNSRewriteIP(v, rr, req)
	case dns.TypePTR, dns.TypeTXT:
		return s.ansFromDNSRewriteText(v, rr, req)
	case dns.TypeMX:
		return s.ansFromDNSRewriteMX(v, rr, req)
	case dns.TypeHTTPS, dns.TypeSVCB:
		return s.ansFromDNSRewriteSVCB(v, rr, req)
	case dns.TypeSRV:
		return s.ansFromDNSRewriteSRV(v, rr, req)
	default:
		log.Debug("don't know how to handle dns rr type %d, skipping", rr)

		return nil, nil
	}
}

// ansFromDNSRewriteIP creates a new answer resource record from the A/AAAA
// dnsrewrite rule data.
func (s *Server) ansFromDNSRewriteIP(
	v rules.RRValue,
	rr rules.RRType,
	req *dns.Msg,
) (ans dns.RR, err error) {
	ip, ok := v.(netip.Addr)
	if !ok {
		return nil, fmt.Errorf("value for rr type %s has type %T, not netip.Addr", dns.Type(rr), v)
	}

	if rr == dns.TypeA {
		return s.genAnswerA(req, ip), nil
	}

	return s.genAnswerAAAA(req, ip), nil
}

// ansFromDNSRewriteText creates a new answer resource record from the TXT/PTR
// dnsrewrite rule data.
func (s *Server) ansFromDNSRewriteText(
	v rules.RRValue,
	rr rules.RRType,
	req *dns.Msg,
) (ans dns.RR, err error) {
	str, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("value for rr type %s has type %T, not string", dns.Type(rr), v)
	}

	if rr == dns.TypeTXT {
		return s.genAnswerTXT(req, []string{str}), nil
	}

	return s.genAnswerPTR(req, str), nil
}

// ansFromDNSRewriteMX creates a new answer resource record from the MX
// dnsrewrite rule data.
func (s *Server) ansFromDNSRewriteMX(
	v rules.RRValue,
	rr rules.RRType,
	req *dns.Msg,
) (ans dns.RR, err error) {
	mx, ok := v.(*rules.DNSMX)
	if !ok {
		return nil, fmt.Errorf(
			"value for rr type %s has type %T, not *rules.DNSMX",
			dns.Type(rr),
			v,
		)
	}

	return s.genAnswerMX(req, mx), nil
}

// ansFromDNSRewriteSVCB creates a new answer resource record from the
// SVCB/HTTPS dnsrewrite rule data.
func (s *Server) ansFromDNSRewriteSVCB(
	v rules.RRValue,
	rr rules.RRType,
	req *dns.Msg,
) (ans dns.RR, err error) {
	svcb, ok := v.(*rules.DNSSVCB)
	if !ok {
		return nil, fmt.Errorf(
			"value for rr type %s has type %T, not *rules.DNSSVCB",
			dns.Type(rr),
			v,
		)
	}

	if rr == dns.TypeHTTPS {
		return s.genAnswerHTTPS(req, svcb), nil
	}

	return s.genAnswerSVCB(req, svcb), nil
}

// ansFromDNSRewriteSRV creates a new answer resource record from the SRV
// dnsrewrite rule data.
func (s *Server) ansFromDNSRewriteSRV(
	v rules.RRValue,
	rr rules.RRType,
	req *dns.Msg,
) (dns.RR, error) {
	srv, ok := v.(*rules.DNSSRV)
	if !ok {
		return nil, fmt.Errorf(
			"value for rr type %s has type %T, not *rules.DNSSRV",
			dns.Type(rr),
			v,
		)
	}

	return s.genAnswerSRV(req, srv), nil
}

// filterDNSRewrite handles dnsrewrite filters.  It constructs a DNS response
// and sets it into pctx.Res.  All parameters must not be nil.
func (s *Server) filterDNSRewrite(
	req *dns.Msg,
	res *filtering.Result,
	pctx *proxy.DNSContext,
) (err error) {
	resp := s.replyCompressed(req)
	dnsrr := res.DNSRewriteResult
	if dnsrr == nil {
		return errors.Error("no dns rewrite rule content")
	}

	resp.Rcode = dnsrr.RCode
	if resp.Rcode != dns.RcodeSuccess {
		pctx.Res = resp

		return nil
	}

	if dnsrr.Response == nil {
		return errors.Error("no dns rewrite rule responses")
	}

	qtype := req.Question[0].Qtype
	values := dnsrr.Response[qtype]
	for i, v := range values {
		var ans dns.RR
		ans, err = s.filterDNSRewriteResponse(req, qtype, v)
		if err != nil {
			return fmt.Errorf("dns rewrite response for %s[%d]: %w", dns.Type(qtype), i, err)
		}

		resp.Answer = append(resp.Answer, ans)
	}

	pctx.Res = resp

	return nil
}
