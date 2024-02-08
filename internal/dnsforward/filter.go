package dnsforward

import (
	"encoding/binary"
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
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

	blocked, _ := s.IsBlockedClient(pctx.Addr.Addr(), clientID)
	if blocked {
		return s.preBlockedResponse(pctx)
	}

	if len(pctx.Req.Question) == 1 {
		q := pctx.Req.Question[0]
		qt := q.Qtype
		host := aghnet.NormalizeDomain(q.Name)
		if s.access.isBlockedHost(host, qt) {
			log.Debug("access: request %s %s is in access blocklist", dns.Type(qt), host)

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

// clientRequestFilteringSettings looks up client filtering settings using the
// client's IP address and ID, if any, from dctx.
func (s *Server) clientRequestFilteringSettings(dctx *dnsContext) (setts *filtering.Settings) {
	setts = s.dnsFilter.Settings()
	setts.ProtectionEnabled = dctx.protectionEnabled
	if s.conf.FilterHandler != nil {
		s.conf.FilterHandler(dctx.proxyCtx.Addr.Addr(), dctx.clientID, setts)
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
		log.Debug(
			"dnsforward: host %q is filtered, reason: %q; rule: %q",
			host,
			res.Reason,
			res.Rules[0].Text,
		)
		pctx.Res = s.genDNSFilterMessage(pctx, res)
	case res.Reason.In(filtering.Rewritten, filtering.RewrittenRule) &&
		res.CanonName != "" &&
		len(res.IPList) == 0:
		// Resolve the new canonical name, not the original host name.  The
		// original question is readded in processFilteringAfterResponse.
		dctx.origQuestion = q
		req.Question[0].Name = dns.Fqdn(res.CanonName)
	case res.Reason == filtering.Rewritten:
		pctx.Res = s.getCNAMEWithIPs(req, res.IPList, res.CanonName)
	case res.Reason.In(filtering.RewrittenRule, filtering.RewrittenAutoHosts):
		if err = s.filterDNSRewrite(req, res, pctx); err != nil {
			return nil, err
		}
	}

	return res, err
}

// checkHostRules checks the host against filters.  It is safe for concurrent
// use.
func (s *Server) checkHostRules(
	host string,
	rrtype rules.RRType,
	setts *filtering.Settings,
) (r *filtering.Result, err error) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	res, err := s.dnsFilter.CheckHostRules(host, rrtype, setts)
	if err != nil {
		return nil, err
	}

	return &res, err
}

// filterDNSResponse checks each resource record of answer section of
// dctx.proxyCtx.Res.  It sets dctx.result and dctx.origResp if at least one of
// canonical names, IP addresses, or HTTPS RR hints in it matches the filtering
// rules, as well as sets dctx.proxyCtx.Res to the filtered response.
func (s *Server) filterDNSResponse(dctx *dnsContext) (err error) {
	setts := dctx.setts
	if !setts.FilteringEnabled {
		return nil
	}

	var res *filtering.Result
	pctx := dctx.proxyCtx
	for i, a := range pctx.Res.Answer {
		host := ""
		var rrtype rules.RRType
		switch a := a.(type) {
		case *dns.CNAME:
			host = strings.TrimSuffix(a.Target, ".")
			rrtype = dns.TypeCNAME

			res, err = s.checkHostRules(host, rrtype, setts)
		case *dns.A:
			host = a.A.String()
			rrtype = dns.TypeA

			res, err = s.checkHostRules(host, rrtype, setts)
		case *dns.AAAA:
			host = a.AAAA.String()
			rrtype = dns.TypeAAAA

			res, err = s.checkHostRules(host, rrtype, setts)
		case *dns.HTTPS:
			res, err = s.filterHTTPSRecords(a, setts)
		default:
			continue
		}

		log.Debug("dnsforward: checked %s %s for %s", dns.Type(rrtype), host, a.Header().Name)

		if err != nil {
			return fmt.Errorf("filtering answer at index %d: %w", i, err)
		} else if res != nil && res.IsFiltered {
			dctx.result = res
			dctx.origResp = pctx.Res
			pctx.Res = s.genDNSFilterMessage(pctx, res)

			log.Debug("dnsforward: matched %q by response: %q", pctx.Req.Question[0].Name, host)

			break
		}
	}

	return nil
}

// removeIPv6Hints deletes IPv6 hints from RR values.
func removeIPv6Hints(rr *dns.HTTPS) {
	rr.Value = slices.DeleteFunc(rr.Value, func(kv dns.SVCBKeyValue) (del bool) {
		_, ok := kv.(*dns.SVCBIPv6Hint)

		return ok
	})
}

// filterHTTPSRecords filters HTTPS answers information through all rule list
// filters of the server filters.  Removes IPv6 hints if IPv6 resolving is
// disabled.
func (s *Server) filterHTTPSRecords(rr *dns.HTTPS, setts *filtering.Settings) (r *filtering.Result, err error) {
	if s.conf.AAAADisabled {
		removeIPv6Hints(rr)
	}

	for _, kv := range rr.Value {
		var ips []net.IP
		switch hint := kv.(type) {
		case *dns.SVCBIPv4Hint:
			ips = hint.Hint
		case *dns.SVCBIPv6Hint:
			ips = hint.Hint
		default:
			// Go on.
		}

		if len(ips) == 0 {
			continue
		}

		r, err = s.filterSVCBHint(ips, setts)
		if err != nil {
			return nil, fmt.Errorf("filtering svcb hints: %w", err)
		}

		if r != nil {
			return r, nil
		}
	}

	return nil, nil
}

// filterSVCBHint filters SVCB hint information.
func (s *Server) filterSVCBHint(
	hint []net.IP,
	setts *filtering.Settings,
) (res *filtering.Result, err error) {
	for _, h := range hint {
		res, err = s.checkHostRules(h.String(), dns.TypeHTTPS, setts)
		if err != nil {
			return nil, fmt.Errorf("checking rules for %s: %w", h, err)
		}

		if res != nil && res.IsFiltered {
			return res, nil
		}
	}

	return nil, nil
}
