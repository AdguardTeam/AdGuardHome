package dnsforward

import (
	"strings"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/joomcode/errorx"
	"github.com/miekg/dns"
)

func (s *Server) beforeRequestHandler(_ *proxy.Proxy, d *proxy.DNSContext) (bool, error) {
	ip := ipFromAddr(d.Addr)
	if s.access.IsBlockedIP(ip) {
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

// getClientRequestFilteringSettings lookups client filtering settings
// using the client's IP address from the DNSContext
func (s *Server) getClientRequestFilteringSettings(d *proxy.DNSContext) *dnsfilter.RequestFilteringSettings {
	setts := s.dnsFilter.GetConfig()
	setts.FilteringEnabled = true
	if s.conf.FilterHandler != nil {
		clientAddr := ipFromAddr(d.Addr)
		s.conf.FilterHandler(clientAddr, &setts)
	}
	return &setts
}

// filterDNSRequest applies the dnsFilter and sets d.Res if the request was filtered
func (s *Server) filterDNSRequest(ctx *dnsContext) (*dnsfilter.Result, error) {
	d := ctx.proxyCtx
	req := d.Req
	host := strings.TrimSuffix(req.Question[0].Name, ".")
	res, err := s.dnsFilter.CheckHost(host, d.Req.Question[0].Qtype, ctx.setts)
	if err != nil {
		// Return immediately if there's an error
		return nil, errorx.Decorate(err, "dnsfilter failed to check host '%s'", host)

	} else if res.IsFiltered {
		// log.Tracef("Host %s is filtered, reason - '%s', matched rule: '%s'", host, res.Reason, res.Rule)
		d.Res = s.genDNSFilterMessage(d, &res)

	} else if (res.Reason == dnsfilter.ReasonRewrite || res.Reason == dnsfilter.RewriteEtcHosts) &&
		len(res.IPList) != 0 {
		resp := s.makeResponse(req)

		name := host
		if len(res.CanonName) != 0 {
			resp.Answer = append(resp.Answer, s.genCNAMEAnswer(req, res.CanonName))
			name = res.CanonName
		}

		for _, ip := range res.IPList {
			if req.Question[0].Qtype == dns.TypeA {
				a := s.genAAnswer(req, ip.To4())
				a.Hdr.Name = dns.Fqdn(name)
				resp.Answer = append(resp.Answer, a)
			} else if req.Question[0].Qtype == dns.TypeAAAA {
				a := s.genAAAAAnswer(req, ip)
				a.Hdr.Name = dns.Fqdn(name)
				resp.Answer = append(resp.Answer, a)
			}
		}

		d.Res = resp

	} else if res.Reason == dnsfilter.ReasonRewrite && len(res.CanonName) != 0 {
		ctx.origQuestion = d.Req.Question[0]
		// resolve canonical name, not the original host name
		d.Req.Question[0].Name = dns.Fqdn(res.CanonName)

	} else if res.Reason == dnsfilter.RewriteEtcHosts && len(res.ReverseHost) != 0 {

		resp := s.makeResponse(req)
		ptr := &dns.PTR{}
		ptr.Hdr = dns.RR_Header{
			Name:   req.Question[0].Name,
			Rrtype: dns.TypePTR,
			Ttl:    s.conf.BlockedResponseTTL,
			Class:  dns.ClassINET,
		}
		ptr.Ptr = res.ReverseHost
		resp.Answer = append(resp.Answer, ptr)
		d.Res = resp
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
