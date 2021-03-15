package dnsforward

import (
	"net"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

// makeResponse creates a DNS response by req and sets necessary flags.  It also
// guarantees that req.Question will be not empty.
func (s *Server) makeResponse(req *dns.Msg) (resp *dns.Msg) {
	resp = &dns.Msg{
		MsgHdr: dns.MsgHdr{
			RecursionAvailable: true,
		},
		Compress: true,
	}

	resp.SetReply(req)

	return resp
}

// genDNSFilterMessage generates a DNS message corresponding to the filtering result
func (s *Server) genDNSFilterMessage(d *proxy.DNSContext, result *dnsfilter.Result) *dns.Msg {
	m := d.Req

	if m.Question[0].Qtype != dns.TypeA && m.Question[0].Qtype != dns.TypeAAAA {
		if s.conf.BlockingMode == "null_ip" {
			return s.makeResponse(m)
		}
		return s.genNXDomain(m)
	}

	switch result.Reason {
	case dnsfilter.FilteredSafeBrowsing:
		return s.genBlockedHost(m, s.conf.SafeBrowsingBlockHost, d)
	case dnsfilter.FilteredParental:
		return s.genBlockedHost(m, s.conf.ParentalBlockHost, d)
	default:
		// If the query was filtered by "Safe search", dnsfilter also must return
		// the IP address that must be used in response.
		// In this case regardless of the filtering method, we should return it
		if result.Reason == dnsfilter.FilteredSafeSearch &&
			len(result.Rules) > 0 &&
			result.Rules[0].IP != nil {
			return s.genResponseWithIP(m, result.Rules[0].IP)
		}

		if s.conf.BlockingMode == "null_ip" {
			// it means that we should return 0.0.0.0 or :: for any blocked request
			return s.makeResponseNullIP(m)
		} else if s.conf.BlockingMode == "custom_ip" {
			// means that we should return custom IP for any blocked request

			switch m.Question[0].Qtype {
			case dns.TypeA:
				return s.genARecord(m, s.conf.BlockingIPv4)
			case dns.TypeAAAA:
				return s.genAAAARecord(m, s.conf.BlockingIPv6)
			}
		} else if s.conf.BlockingMode == "nxdomain" {
			// means that we should return NXDOMAIN for any blocked request

			return s.genNXDomain(m)
		} else if s.conf.BlockingMode == "refused" {
			// means that we should return NXDOMAIN for any blocked request

			return s.makeResponseREFUSED(m)
		}

		// Default blocking mode
		// If there's an IP specified in the rule, return it
		// For host-type rules, return null IP
		if len(result.Rules) > 0 && result.Rules[0].IP != nil {
			return s.genResponseWithIP(m, result.Rules[0].IP)
		}

		return s.makeResponseNullIP(m)
	}
}

func (s *Server) genServerFailure(request *dns.Msg) *dns.Msg {
	resp := dns.Msg{}
	resp.SetRcode(request, dns.RcodeServerFailure)
	resp.RecursionAvailable = true
	return &resp
}

func (s *Server) genARecord(request *dns.Msg, ip net.IP) *dns.Msg {
	resp := s.makeResponse(request)
	resp.Answer = append(resp.Answer, s.genAnswerA(request, ip))
	return resp
}

func (s *Server) genAAAARecord(request *dns.Msg, ip net.IP) *dns.Msg {
	resp := s.makeResponse(request)
	resp.Answer = append(resp.Answer, s.genAnswerAAAA(request, ip))
	return resp
}

func (s *Server) hdr(req *dns.Msg, rrType rules.RRType) (h dns.RR_Header) {
	return dns.RR_Header{
		Name:   req.Question[0].Name,
		Rrtype: rrType,
		Ttl:    s.conf.BlockedResponseTTL,
		Class:  dns.ClassINET,
	}
}

func (s *Server) genAnswerA(req *dns.Msg, ip net.IP) (ans *dns.A) {
	return &dns.A{
		Hdr: s.hdr(req, dns.TypeA),
		A:   ip,
	}
}

func (s *Server) genAnswerAAAA(req *dns.Msg, ip net.IP) (ans *dns.AAAA) {
	return &dns.AAAA{
		Hdr:  s.hdr(req, dns.TypeAAAA),
		AAAA: ip,
	}
}

func (s *Server) genAnswerCNAME(req *dns.Msg, cname string) (ans *dns.CNAME) {
	return &dns.CNAME{
		Hdr:    s.hdr(req, dns.TypeCNAME),
		Target: dns.Fqdn(cname),
	}
}

func (s *Server) genAnswerMX(req *dns.Msg, mx *rules.DNSMX) (ans *dns.MX) {
	return &dns.MX{
		Hdr:        s.hdr(req, dns.TypeMX),
		Preference: mx.Preference,
		Mx:         dns.Fqdn(mx.Exchange),
	}
}

func (s *Server) genAnswerPTR(req *dns.Msg, ptr string) (ans *dns.PTR) {
	return &dns.PTR{
		Hdr: s.hdr(req, dns.TypePTR),
		Ptr: dns.Fqdn(ptr),
	}
}

func (s *Server) genAnswerSRV(req *dns.Msg, srv *rules.DNSSRV) (ans *dns.SRV) {
	return &dns.SRV{
		Hdr:      s.hdr(req, dns.TypeSRV),
		Priority: srv.Priority,
		Weight:   srv.Weight,
		Port:     srv.Port,
		Target:   dns.Fqdn(srv.Target),
	}
}

func (s *Server) genAnswerTXT(req *dns.Msg, strs []string) (ans *dns.TXT) {
	return &dns.TXT{
		Hdr: s.hdr(req, dns.TypeTXT),
		Txt: strs,
	}
}

// generate DNS response message with an IP address
func (s *Server) genResponseWithIP(req *dns.Msg, ip net.IP) *dns.Msg {
	if req.Question[0].Qtype == dns.TypeA && ip.To4() != nil {
		return s.genARecord(req, ip.To4())
	} else if req.Question[0].Qtype == dns.TypeAAAA &&
		len(ip) == net.IPv6len && ip.To4() == nil {
		return s.genAAAARecord(req, ip)
	}

	// empty response
	resp := s.makeResponse(req)
	return resp
}

// Respond with 0.0.0.0 for A, :: for AAAA, empty response for other types
func (s *Server) makeResponseNullIP(req *dns.Msg) *dns.Msg {
	if req.Question[0].Qtype == dns.TypeA {
		return s.genARecord(req, []byte{0, 0, 0, 0})
	} else if req.Question[0].Qtype == dns.TypeAAAA {
		return s.genAAAARecord(req, net.IPv6zero)
	}

	return s.makeResponse(req)
}

func (s *Server) genBlockedHost(request *dns.Msg, newAddr string, d *proxy.DNSContext) *dns.Msg {
	ip := net.ParseIP(newAddr)
	if ip != nil {
		return s.genResponseWithIP(request, ip)
	}

	// look up the hostname, TODO: cache
	replReq := dns.Msg{}
	replReq.SetQuestion(dns.Fqdn(newAddr), request.Question[0].Qtype)
	replReq.RecursionDesired = true

	newContext := &proxy.DNSContext{
		Proto:     d.Proto,
		Addr:      d.Addr,
		StartTime: time.Now(),
		Req:       &replReq,
	}

	err := s.dnsProxy.Resolve(newContext)
	if err != nil {
		log.Printf("Couldn't look up replacement host %q: %s", newAddr, err)
		return s.genServerFailure(request)
	}

	resp := s.makeResponse(request)
	if newContext.Res != nil {
		for _, answer := range newContext.Res.Answer {
			answer.Header().Name = request.Question[0].Name
			resp.Answer = append(resp.Answer, answer)
		}
	}

	return resp
}

// Create REFUSED DNS response
func (s *Server) makeResponseREFUSED(request *dns.Msg) *dns.Msg {
	resp := dns.Msg{}
	resp.SetRcode(request, dns.RcodeRefused)
	resp.RecursionAvailable = true
	return &resp
}

func (s *Server) genNXDomain(request *dns.Msg) *dns.Msg {
	resp := dns.Msg{}
	resp.SetRcode(request, dns.RcodeNameError)
	resp.RecursionAvailable = true
	resp.Ns = s.genSOA(request)
	return &resp
}

func (s *Server) genSOA(request *dns.Msg) []dns.RR {
	zone := ""
	if len(request.Question) > 0 {
		zone = request.Question[0].Name
	}

	soa := dns.SOA{
		// values copied from verisign's nonexistent .com domain
		// their exact values are not important in our use case because they are used for domain transfers between primary/secondary DNS servers
		Refresh: 1800,
		Retry:   900,
		Expire:  604800,
		Minttl:  86400,
		// copied from AdGuard DNS
		Ns:     "fake-for-negative-caching.adguard.com.",
		Serial: 100500,
		// rest is request-specific
		Hdr: dns.RR_Header{
			Name:   zone,
			Rrtype: dns.TypeSOA,
			Ttl:    s.conf.BlockedResponseTTL,
			Class:  dns.ClassINET,
		},
		Mbox: "hostmaster.", // zone will be appended later if it's not empty or "."
	}
	if soa.Hdr.Ttl == 0 {
		soa.Hdr.Ttl = defaultValues.BlockedResponseTTL
	}
	if len(zone) > 0 && zone[0] != '.' {
		soa.Mbox += zone
	}
	return []dns.RR{&soa}
}
