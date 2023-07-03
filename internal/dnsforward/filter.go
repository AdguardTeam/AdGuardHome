package dnsforward

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/miekg/dns"
)

// beforeRequestHandler is the handler that is called before any other
// processing, including logs.  It performs access checks and puts the client
// ID, if there is one, into the server's cache.
func (s *Server) beforeRequestHandler(
	_ *proxy.Proxy,
	pctx *proxy.DNSContext,
) (reply bool, err error) {
	clientID, err := s.clientIDFromDNSContext(pctx)
	if err != nil {
		return false, fmt.Errorf("getting clientid: %w", err)
	}

	addrPort := netutil.NetAddrToAddrPort(pctx.Addr)
	blocked, _ := s.IsBlockedClient(addrPort.Addr(), clientID)
	if blocked {
		return s.preBlockedResponse(pctx)
	}

	if len(pctx.Req.Question) == 1 {
		q := pctx.Req.Question[0]
		qt := q.Qtype
		host := strings.TrimSuffix(q.Name, ".")
		if s.access.isBlockedHost(host, qt) {
			log.Debug("request %s %s is in access blocklist", dns.Type(qt), host)

			return s.preBlockedResponse(pctx)
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
// the client's IP address and ID, if any, from dctx.
func (s *Server) getClientRequestFilteringSettings(dctx *dnsContext) *filtering.Settings {
	setts := s.dnsFilter.Settings()
	setts.ProtectionEnabled = dctx.protectionEnabled
	if s.conf.FilterHandler != nil {
		ip, _ := netutil.IPAndPortFromAddr(dctx.proxyCtx.Addr)
		s.conf.FilterHandler(ip, dctx.clientID, setts)
	}

	return setts
}

// filterDNSRequest applies the dnsFilter and sets dctx.proxyCtx.Res if the
// request was filtered.
func (s *Server) filterDNSRequest(dctx *dnsContext) (res *filtering.Result, err error) {
	pctx := dctx.proxyCtx
	req := pctx.Req
	q := req.Question[0]
	host := strings.TrimSuffix(q.Name, ".")
	resVal, err := s.dnsFilter.CheckHost(host, q.Qtype, dctx.setts)
	if err != nil {
		return nil, fmt.Errorf("checking host %q: %w", host, err)
	}

	// TODO(a.garipov): Make CheckHost return a pointer.
	res = &resVal
	switch {
	case res.IsFiltered:
		log.Tracef("host %q is filtered, reason %q, rule: %q", host, res.Reason, res.Rules[0].Text)
		pctx.Res = s.genDNSFilterMessage(pctx, res)
	case res.Reason.In(filtering.Rewritten, filtering.RewrittenRule) &&
		res.CanonName != "" &&
		len(res.IPList) == 0:
		// Resolve the new canonical name, not the original host name.  The
		// original question is readded in processFilteringAfterResponse.
		dctx.origQuestion = q
		req.Question[0].Name = dns.Fqdn(res.CanonName)
	case res.Reason == filtering.Rewritten:
		pctx.Res = s.filterRewritten(req, host, res, q.Qtype)
	case res.Reason.In(filtering.RewrittenRule, filtering.RewrittenAutoHosts):
		if err = s.filterDNSRewrite(req, res, pctx); err != nil {
			return nil, err
		}
	}

	return res, err
}

// filterRewritten handles DNS rewrite filters.  It returns a DNS response with
// the data from the filtering result.  All parameters must not be nil.
func (s *Server) filterRewritten(
	req *dns.Msg,
	host string,
	res *filtering.Result,
	qt uint16,
) (resp *dns.Msg) {
	resp = s.makeResponse(req)
	name := host
	if len(res.CanonName) != 0 {
		resp.Answer = append(resp.Answer, s.genAnswerCNAME(req, res.CanonName))
		name = res.CanonName
	}

	for _, ip := range res.IPList {
		switch qt {
		case dns.TypeA:
			a := s.genAnswerA(req, ip.To4())
			a.Hdr.Name = dns.Fqdn(name)
			resp.Answer = append(resp.Answer, a)
		case dns.TypeAAAA:
			a := s.genAnswerAAAA(req, ip)
			a.Hdr.Name = dns.Fqdn(name)
			resp.Answer = append(resp.Answer, a)
		}
	}

	return resp
}

// checkHostRules checks the host against filters.  It is safe for concurrent
// use.
func (s *Server) checkHostRules(host string, rrtype uint16, setts *filtering.Settings) (
	r *filtering.Result,
	err error,
) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	if s.dnsFilter == nil {
		return nil, nil
	}

	var res filtering.Result
	res, err = s.dnsFilter.CheckHostRules(host, rrtype, setts)
	if err != nil {
		return nil, err
	}

	return &res, err
}

// filterDNSResponse checks each resource record of the response's answer
// section from pctx and returns a non-nil res if at least one of canonical
// names or IP addresses in it matches the filtering rules.
func (s *Server) filterDNSResponse(
	pctx *proxy.DNSContext,
	setts *filtering.Settings,
) (res *filtering.Result, err error) {
	if !setts.FilteringEnabled {
		return nil, nil
	}

	for _, a := range pctx.Res.Answer {
		host := ""
		var rrtype uint16
		switch a := a.(type) {
		case *dns.CNAME:
			host = strings.TrimSuffix(a.Target, ".")
			rrtype = dns.TypeCNAME
		case *dns.A:
			host = a.A.String()
			rrtype = dns.TypeA
		case *dns.AAAA:
			host = a.AAAA.String()
			rrtype = dns.TypeAAAA
		default:
			continue
		}

		log.Debug("dnsforward: checking %s %s for %s", dns.Type(rrtype), host, a.Header().Name)

		res, err = s.checkHostRules(host, rrtype, setts)
		if err != nil {
			return nil, err
		} else if res == nil {
			continue
		} else if res.IsFiltered {
			pctx.Res = s.genDNSFilterMessage(pctx, res)
			log.Debug("DNSFwd: Matched %s by response: %s", pctx.Req.Question[0].Name, host)

			return res, nil
		}
	}

	return nil, nil
}
