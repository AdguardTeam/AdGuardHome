package dnsforward

import (
	"fmt"
	"net"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

// filterDNSRewriteResponse handles a single DNS rewrite response entry.
// It returns the constructed answer resource record.
func (s *Server) filterDNSRewriteResponse(req *dns.Msg, rr rules.RRType, v rules.RRValue) (ans dns.RR, err error) {
	switch rr {
	case dns.TypeA, dns.TypeAAAA:
		ip, ok := v.(net.IP)
		if !ok {
			return nil, fmt.Errorf("value has type %T, not net.IP", v)
		}

		if rr == dns.TypeA {
			return s.genAAnswer(req, ip.To4()), nil
		}

		return s.genAAAAAnswer(req, ip), nil
	case dns.TypeTXT:
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("value has type %T, not string", v)
		}

		return s.genTXTAnswer(req, []string{str}), nil
	default:
		log.Debug("don't know how to handle dns rr type %d, skipping", rr)

		return nil, nil
	}
}

// filterDNSRewrite handles dnsrewrite filters.  It constructs a DNS
// response and sets it into d.Res.
func (s *Server) filterDNSRewrite(req *dns.Msg, res dnsfilter.Result, d *proxy.DNSContext) (err error) {
	resp := s.makeResponse(req)
	dnsrr := res.DNSRewriteResult
	if dnsrr == nil {
		return agherr.Error("no dns rewrite rule content")
	}

	resp.Rcode = dnsrr.RCode
	if resp.Rcode != dns.RcodeSuccess {
		d.Res = resp

		return nil
	}

	if dnsrr.Response == nil {
		return agherr.Error("no dns rewrite rule responses")
	}

	rr := req.Question[0].Qtype
	values := dnsrr.Response[rr]
	for i, v := range values {
		var ans dns.RR
		ans, err = s.filterDNSRewriteResponse(req, rr, v)
		if err != nil {
			return fmt.Errorf("dns rewrite response for %d[%d]: %w", rr, i, err)
		}

		resp.Answer = append(resp.Answer, ans)
	}

	d.Res = resp

	return nil
}
