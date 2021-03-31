package dnsforward

import (
	"fmt"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"

	"github.com/miekg/dns"
)

func (s *Server) beforeRequestHandler(_ *proxy.Proxy, d *proxy.DNSContext) (bool, error) {
	ip := IPFromAddr(d.Addr)
	disallowed, _ := s.access.IsBlockedIP(ip)
	if disallowed {
		log.Tracef("Client IP %s is blocked by settings", ip)
		return false, nil
	}

	if len(d.Req.Question) == 1 {
		host := strings.TrimSuffix(d.Req.Question[0].Name, ".")
		if s.access.IsBlockedDomain(host) {
			log.Tracef("Domain %s is blocked by settings", host)
			return false, nil
		}
	}

	return true, nil
}

// getClientRequestFilteringSettings looks up client filtering settings using
// the client's IP address and ID, if any, from ctx.
func (s *Server) getClientRequestFilteringSettings(ctx *dnsContext) *dnsfilter.FilteringSettings {
	setts := s.dnsFilter.GetConfig()
	setts.FilteringEnabled = true
	if s.conf.FilterHandler != nil {
		s.conf.FilterHandler(IPFromAddr(ctx.proxyCtx.Addr), ctx.clientID, &setts)
	}

	return &setts
}

// filterDNSRequest applies the dnsFilter and sets d.Res if the request
// was filtered.
func (s *Server) filterDNSRequest(ctx *dnsContext) (*dnsfilter.Result, error) {
	d := ctx.proxyCtx
	// TODO(e.burkov): Consistently use req instead of d.Req since it is
	// declared.
	req := d.Req
	host := strings.TrimSuffix(req.Question[0].Name, ".")
	res, err := s.dnsFilter.CheckHost(host, d.Req.Question[0].Qtype, ctx.setts)
	if err != nil {
		// Return immediately if there's an error
		return nil, fmt.Errorf("dnsfilter failed to check host %q: %w", host, err)
	} else if res.IsFiltered {
		log.Tracef("Host %s is filtered, reason - %q, matched rule: %q", host, res.Reason, res.Rules[0].Text)
		d.Res = s.genDNSFilterMessage(d, &res)
	} else if res.Reason.In(dnsfilter.Rewritten, dnsfilter.RewrittenRule) &&
		res.CanonName != "" &&
		len(res.IPList) == 0 {
		// Resolve the new canonical name, not the original host
		// name.  The original question is readded in
		// processFilteringAfterResponse.
		ctx.origQuestion = d.Req.Question[0]
		d.Req.Question[0].Name = dns.Fqdn(res.CanonName)
	} else if res.Reason == dnsfilter.RewrittenAutoHosts && len(res.ReverseHosts) != 0 {
		resp := s.makeResponse(req)
		for _, h := range res.ReverseHosts {
			hdr := dns.RR_Header{
				Name:   req.Question[0].Name,
				Rrtype: dns.TypePTR,
				Ttl:    s.conf.BlockedResponseTTL,
				Class:  dns.ClassINET,
			}

			ptr := &dns.PTR{
				Hdr: hdr,
				Ptr: h,
			}

			resp.Answer = append(resp.Answer, ptr)
		}

		d.Res = resp
	} else if res.Reason == dnsfilter.Rewritten || res.Reason == dnsfilter.RewrittenAutoHosts {
		resp := s.makeResponse(req)

		name := host
		if len(res.CanonName) != 0 {
			resp.Answer = append(resp.Answer, s.genAnswerCNAME(req, res.CanonName))
			name = res.CanonName
		}

		for _, ip := range res.IPList {
			if req.Question[0].Qtype == dns.TypeA {
				a := s.genAnswerA(req, ip.To4())
				a.Hdr.Name = dns.Fqdn(name)
				resp.Answer = append(resp.Answer, a)
			} else if req.Question[0].Qtype == dns.TypeAAAA {
				a := s.genAnswerAAAA(req, ip)
				a.Hdr.Name = dns.Fqdn(name)
				resp.Answer = append(resp.Answer, a)
			}
		}

		d.Res = resp
	} else if res.Reason == dnsfilter.RewrittenRule {
		err = s.filterDNSRewrite(req, res, d)
		if err != nil {
			return nil, err
		}
	}

	return &res, err
}

// If response contains CNAME, A or AAAA records, we apply filtering to each canonical host name or IP address.
// If this is a match, we set a new response in d.Res and return.
func (s *Server) filterDNSResponse(ctx *dnsContext) (*dnsfilter.Result, error) {
	d := ctx.proxyCtx
	for _, a := range d.Res.Answer {
		host := ""

		switch v := a.(type) {
		case *dns.CNAME:
			log.Debug("DNSFwd: Checking CNAME %s for %s", v.Target, v.Hdr.Name)
			host = strings.TrimSuffix(v.Target, ".")

		case *dns.A:
			host = v.A.String()
			log.Debug("DNSFwd: Checking record A (%s) for %s", host, v.Hdr.Name)

		case *dns.AAAA:
			host = v.AAAA.String()
			log.Debug("DNSFwd: Checking record AAAA (%s) for %s", host, v.Hdr.Name)

		default:
			continue
		}

		s.RLock()
		// Synchronize access to s.dnsFilter so it won't be suddenly uninitialized while in use.
		// This could happen after proxy server has been stopped, but its workers are not yet exited.
		if !s.conf.ProtectionEnabled || s.dnsFilter == nil {
			s.RUnlock()
			continue
		}
		res, err := s.dnsFilter.CheckHostRules(host, d.Req.Question[0].Qtype, ctx.setts)
		s.RUnlock()

		if err != nil {
			return nil, err
		} else if res.IsFiltered {
			d.Res = s.genDNSFilterMessage(d, &res)
			log.Debug("DNSFwd: Matched %s by response: %s", d.Req.Question[0].Name, host)
			return &res, nil
		}
	}

	return nil, nil
}
