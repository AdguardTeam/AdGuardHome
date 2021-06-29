package dnsforward

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// beforeRequestHandler is the handler that is called before any other
// processing, including logs.  It performs access checks and puts the client
// ID, if there is one, into the server's cache.
func (s *Server) beforeRequestHandler(
	_ *proxy.Proxy,
	pctx *proxy.DNSContext,
) (reply bool, err error) {
	ip := aghnet.IPFromAddr(pctx.Addr)
	clientID, err := s.clientIDFromDNSContext(pctx)
	if err != nil {
		return false, fmt.Errorf("getting clientid: %w", err)
	}

	blocked, _ := s.IsBlockedClient(ip, clientID)
	if blocked {
		return false, nil
	}

	if len(pctx.Req.Question) == 1 {
		host := strings.TrimSuffix(pctx.Req.Question[0].Name, ".")
		if s.access.isBlockedHost(host) {
			log.Debug("host %s is in access blocklist", host)

			return false, nil
		}
	}

	if clientID != "" {
		key := [8]byte{}
		binary.BigEndian.PutUint64(key[:], pctx.RequestID)
		s.clientIDCache.Set(key[:], []byte(clientID))
	}

	return true, nil
}

// getClientRequestFilteringSettings looks up client filtering settings using
// the client's IP address and ID, if any, from ctx.
func (s *Server) getClientRequestFilteringSettings(ctx *dnsContext) *filtering.Settings {
	setts := s.dnsFilter.GetConfig()
	if s.conf.FilterHandler != nil {
		s.conf.FilterHandler(aghnet.IPFromAddr(ctx.proxyCtx.Addr), ctx.clientID, &setts)
	}

	return &setts
}

// filterDNSRequest applies the dnsFilter and sets d.Res if the request was
// filtered.
func (s *Server) filterDNSRequest(ctx *dnsContext) (*filtering.Result, error) {
	d := ctx.proxyCtx
	req := d.Req
	host := strings.TrimSuffix(req.Question[0].Name, ".")
	res, err := s.dnsFilter.CheckHost(host, req.Question[0].Qtype, ctx.setts)
	if err != nil {
		// Return immediately if there's an error
		return nil, fmt.Errorf("filtering failed to check host %q: %w", host, err)
	} else if res.IsFiltered {
		log.Tracef("Host %s is filtered, reason - %q, matched rule: %q", host, res.Reason, res.Rules[0].Text)
		d.Res = s.genDNSFilterMessage(d, &res)
	} else if res.Reason.In(filtering.Rewritten, filtering.RewrittenRule) &&
		res.CanonName != "" &&
		len(res.IPList) == 0 {
		// Resolve the new canonical name, not the original host
		// name.  The original question is readded in
		// processFilteringAfterResponse.
		ctx.origQuestion = req.Question[0]
		req.Question[0].Name = dns.Fqdn(res.CanonName)
	} else if res.Reason == filtering.RewrittenAutoHosts && len(res.ReverseHosts) != 0 {
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
	} else if res.Reason.In(filtering.Rewritten, filtering.RewrittenAutoHosts) {
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
	} else if res.Reason == filtering.RewrittenRule {
		err = s.filterDNSRewrite(req, res, d)
		if err != nil {
			return nil, err
		}
	}

	return &res, err
}

// checkHostRules checks the host against filters.  It is safe for concurrent
// use.
func (s *Server) checkHostRules(host string, qtype uint16, setts *filtering.Settings) (
	r *filtering.Result,
	err error,
) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	if s.dnsFilter == nil {
		return nil, nil
	}

	var res filtering.Result
	res, err = s.dnsFilter.CheckHostRules(host, qtype, setts)
	if err != nil {
		return nil, err
	}

	return &res, err
}

// If response contains CNAME, A or AAAA records, we apply filtering to each
// canonical host name or IP address.  If this is a match, we set a new response
// in d.Res and return.
func (s *Server) filterDNSResponse(ctx *dnsContext) (*filtering.Result, error) {
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

		res, err := s.checkHostRules(host, d.Req.Question[0].Qtype, ctx.setts)
		if err != nil {
			return nil, err
		} else if res == nil {
			continue
		} else if res.IsFiltered {
			d.Res = s.genDNSFilterMessage(d, res)
			log.Debug("DNSFwd: Matched %s by response: %s", d.Req.Question[0].Name, host)

			return res, nil
		}
	}

	return nil, nil
}
