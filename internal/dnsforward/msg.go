package dnsforward

import (
	"net/netip"
	"slices"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

// TODO(e.burkov):  Name all the methods by a [proxy.MessageConstructor]
// template.  Also extract all the methods to a separate entity.

// reply creates a DNS response for req.
func (*Server) reply(req *dns.Msg, code int) (resp *dns.Msg) {
	resp = (&dns.Msg{}).SetRcode(req, code)
	resp.RecursionAvailable = true

	return resp
}

// replyCompressed creates a DNS response for req and sets the compress flag.
func (s *Server) replyCompressed(req *dns.Msg) (resp *dns.Msg) {
	resp = s.reply(req, dns.RcodeSuccess)
	resp.Compress = true

	return resp
}

// ipsFromRules extracts unique non-IP addresses from the filtering result
// rules.
func ipsFromRules(resRules []*filtering.ResultRule) (ips []netip.Addr) {
	for _, r := range resRules {
		// len(resRules) and len(ips) are actually small enough for O(n^2) to do
		// not raise performance questions.
		if ip := r.IP; ip != (netip.Addr{}) && !slices.Contains(ips, ip) {
			ips = append(ips, ip)
		}
	}

	return ips
}

// genDNSFilterMessage generates a filtered response to req for the filtering
// result res.
func (s *Server) genDNSFilterMessage(
	dctx *proxy.DNSContext,
	res *filtering.Result,
) (resp *dns.Msg) {
	req := dctx.Req
	qt := req.Question[0].Qtype
	if qt != dns.TypeA && qt != dns.TypeAAAA && qt != dns.TypeHTTPS {
		m, _, _ := s.dnsFilter.BlockingMode()
		if m == filtering.BlockingModeNullIP {
			return s.replyCompressed(req)
		}

		return s.NewMsgNODATA(req)
	}

	switch res.Reason {
	case filtering.FilteredSafeBrowsing:
		return s.genBlockedHost(req, s.dnsFilter.SafeBrowsingBlockHost(), dctx)
	case filtering.FilteredParental:
		return s.genBlockedHost(req, s.dnsFilter.ParentalBlockHost(), dctx)
	case filtering.FilteredSafeSearch:
		// If Safe Search generated the necessary IP addresses, use them.
		// Otherwise, if there were no errors, there are no addresses for the
		// requested IP version, so produce a NODATA response.
		return s.getCNAMEWithIPs(req, ipsFromRules(res.Rules), res.CanonName)
	default:
		return s.genForBlockingMode(req, ipsFromRules(res.Rules))
	}
}

// getCNAMEWithIPs generates a filtered response to req for with CNAME record
// and provided ips.
func (s *Server) getCNAMEWithIPs(req *dns.Msg, ips []netip.Addr, cname string) (resp *dns.Msg) {
	resp = s.replyCompressed(req)

	originalName := req.Question[0].Name

	var ans []dns.RR
	if cname != "" {
		ans = append(ans, s.genAnswerCNAME(req, cname))

		// The given IPs actually are resolved for this cname.
		req.Question[0].Name = dns.Fqdn(cname)
		defer func() { req.Question[0].Name = originalName }()
	}

	switch req.Question[0].Qtype {
	case dns.TypeA:
		ans = append(ans, s.genAnswersWithIPv4s(req, ips)...)
	case dns.TypeAAAA:
		for _, ip := range ips {
			if ip.Is6() {
				ans = append(ans, s.genAnswerAAAA(req, ip))
			}
		}
	default:
		// Go on and return an empty response.
	}

	resp.Answer = ans

	return resp
}

// genForBlockingMode generates a filtered response to req based on the server's
// blocking mode.
func (s *Server) genForBlockingMode(req *dns.Msg, ips []netip.Addr) (resp *dns.Msg) {
	switch mode, bIPv4, bIPv6 := s.dnsFilter.BlockingMode(); mode {
	case filtering.BlockingModeCustomIP:
		return s.makeResponseCustomIP(req, bIPv4, bIPv6)
	case filtering.BlockingModeDefault:
		if len(ips) > 0 {
			return s.genResponseWithIPs(req, ips)
		}

		return s.makeResponseNullIP(req)
	case filtering.BlockingModeNullIP:
		return s.makeResponseNullIP(req)
	case filtering.BlockingModeNXDOMAIN:
		return s.NewMsgNXDOMAIN(req)
	case filtering.BlockingModeREFUSED:
		return s.makeResponseREFUSED(req)
	default:
		log.Error("dnsforward: invalid blocking mode %q", mode)

		return s.replyCompressed(req)
	}
}

// makeResponseCustomIP generates a DNS response message for Custom IP blocking
// mode with the provided IP addresses and an appropriate resource record type.
func (s *Server) makeResponseCustomIP(
	req *dns.Msg,
	bIPv4 netip.Addr,
	bIPv6 netip.Addr,
) (resp *dns.Msg) {
	switch qt := req.Question[0].Qtype; qt {
	case dns.TypeA:
		return s.genARecord(req, bIPv4)
	case dns.TypeAAAA:
		return s.genAAAARecord(req, bIPv6)
	default:
		// Generally shouldn't happen, since the types are checked in
		// genDNSFilterMessage.
		log.Error("dnsforward: invalid msg type %s for custom IP blocking mode", dns.Type(qt))

		return s.replyCompressed(req)
	}
}

func (s *Server) genARecord(request *dns.Msg, ip netip.Addr) *dns.Msg {
	resp := s.replyCompressed(request)
	resp.Answer = append(resp.Answer, s.genAnswerA(request, ip))
	return resp
}

func (s *Server) genAAAARecord(request *dns.Msg, ip netip.Addr) *dns.Msg {
	resp := s.replyCompressed(request)
	resp.Answer = append(resp.Answer, s.genAnswerAAAA(request, ip))
	return resp
}

func (s *Server) hdr(req *dns.Msg, rrType rules.RRType) (h dns.RR_Header) {
	return dns.RR_Header{
		Name:   req.Question[0].Name,
		Rrtype: rrType,
		Ttl:    s.dnsFilter.BlockedResponseTTL(),
		Class:  dns.ClassINET,
	}
}

func (s *Server) genAnswerA(req *dns.Msg, ip netip.Addr) (ans *dns.A) {
	return &dns.A{
		Hdr: s.hdr(req, dns.TypeA),
		A:   ip.AsSlice(),
	}
}

func (s *Server) genAnswerAAAA(req *dns.Msg, ip netip.Addr) (ans *dns.AAAA) {
	return &dns.AAAA{
		Hdr:  s.hdr(req, dns.TypeAAAA),
		AAAA: ip.AsSlice(),
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

// genResponseWithIPs generates a DNS response message with the provided IP
// addresses and an appropriate resource record type.  If any of the IPs cannot
// be converted to the correct protocol, genResponseWithIPs returns an empty
// response.
func (s *Server) genResponseWithIPs(req *dns.Msg, ips []netip.Addr) (resp *dns.Msg) {
	var ans []dns.RR
	switch req.Question[0].Qtype {
	case dns.TypeA:
		ans = s.genAnswersWithIPv4s(req, ips)
	case dns.TypeAAAA:
		for _, ip := range ips {
			if ip.Is6() {
				ans = append(ans, s.genAnswerAAAA(req, ip))
			}
		}
	default:
		// Go on and return an empty response.
	}

	resp = s.replyCompressed(req)
	resp.Answer = ans

	return resp
}

// genAnswersWithIPv4s generates DNS A answers provided IPv4 addresses.  If any
// of the IPs isn't an IPv4 address, genAnswersWithIPv4s logs a warning and
// returns nil,
func (s *Server) genAnswersWithIPv4s(req *dns.Msg, ips []netip.Addr) (ans []dns.RR) {
	for _, ip := range ips {
		if !ip.Is4() {
			log.Info("dnsforward: warning: ip %s is not ipv4 address", ip)

			return nil
		}

		ans = append(ans, s.genAnswerA(req, ip))
	}

	return ans
}

// makeResponseNullIP creates a response with 0.0.0.0 for A requests, :: for
// AAAA requests, and an empty response for other types.
func (s *Server) makeResponseNullIP(req *dns.Msg) (resp *dns.Msg) {
	// Respond with the corresponding zero IP type as opposed to simply
	// using one or the other in both cases, because the IPv4 zero IP is
	// converted to a IPV6-mapped IPv4 address, while the IPv6 zero IP is
	// converted into an empty slice instead of the zero IPv4.
	switch req.Question[0].Qtype {
	case dns.TypeA:
		resp = s.genResponseWithIPs(req, []netip.Addr{netip.IPv4Unspecified()})
	case dns.TypeAAAA:
		resp = s.genResponseWithIPs(req, []netip.Addr{netip.IPv6Unspecified()})
	default:
		resp = s.replyCompressed(req)
	}

	return resp
}

func (s *Server) genBlockedHost(request *dns.Msg, newAddr string, d *proxy.DNSContext) *dns.Msg {
	if newAddr == "" {
		log.Info("dnsforward: block host is not specified")

		return s.NewMsgSERVFAIL(request)
	}

	ip, err := netip.ParseAddr(newAddr)
	if err == nil {
		return s.genResponseWithIPs(request, []netip.Addr{ip})
	}

	// look up the hostname, TODO: cache
	replReq := dns.Msg{}
	replReq.SetQuestion(dns.Fqdn(newAddr), request.Question[0].Qtype)
	replReq.RecursionDesired = true

	newContext := &proxy.DNSContext{
		Proto: d.Proto,
		Addr:  d.Addr,
		Req:   &replReq,
	}

	prx := s.proxy()
	if prx == nil {
		log.Debug("dnsforward: %s", srvClosedErr)

		return s.NewMsgSERVFAIL(request)
	}

	err = prx.Resolve(newContext)
	if err != nil {
		log.Info("dnsforward: looking up replacement host %q: %s", newAddr, err)

		return s.NewMsgSERVFAIL(request)
	}

	resp := s.replyCompressed(request)
	if newContext.Res != nil {
		for _, answer := range newContext.Res.Answer {
			answer.Header().Name = request.Question[0].Name
			resp.Answer = append(resp.Answer, answer)
		}
	}

	return resp
}

// Create REFUSED DNS response
func (s *Server) makeResponseREFUSED(req *dns.Msg) *dns.Msg {
	return s.reply(req, dns.RcodeRefused)
}

// type check
var _ proxy.MessageConstructor = (*Server)(nil)

// NewMsgNXDOMAIN implements the [proxy.MessageConstructor] interface for
// *Server.
func (s *Server) NewMsgNXDOMAIN(req *dns.Msg) (resp *dns.Msg) {
	resp = s.reply(req, dns.RcodeNameError)
	resp.Ns = s.genSOA(req)

	return resp
}

// NewMsgSERVFAIL implements the [proxy.MessageConstructor] interface for
// *Server.
func (s *Server) NewMsgSERVFAIL(req *dns.Msg) (resp *dns.Msg) {
	return s.reply(req, dns.RcodeServerFailure)
}

// NewMsgNOTIMPLEMENTED implements the [proxy.MessageConstructor] interface for
// *Server.
func (s *Server) NewMsgNOTIMPLEMENTED(req *dns.Msg) (resp *dns.Msg) {
	resp = s.reply(req, dns.RcodeNotImplemented)

	// Most of the Internet and especially the inner core has an MTU of at least
	// 1500 octets.  Maximum DNS/UDP payload size for IPv6 on MTU 1500 ethernet
	// is 1452 (1500 minus 40 (IPv6 header size) minus 8 (UDP header size)).
	//
	// See appendix A of https://datatracker.ietf.org/doc/draft-ietf-dnsop-avoid-fragmentation/17.
	const maxUDPPayload = 1452

	// NOTIMPLEMENTED without EDNS is treated as 'we don't support EDNS', so
	// explicitly set it.
	resp.SetEdns0(maxUDPPayload, false)

	return resp
}

// NewMsgNODATA implements the [proxy.MessageConstructor] interface for *Server.
func (s *Server) NewMsgNODATA(req *dns.Msg) (resp *dns.Msg) {
	resp = s.reply(req, dns.RcodeSuccess)
	resp.Ns = s.genSOA(req)

	return resp
}

func (s *Server) genSOA(req *dns.Msg) []dns.RR {
	zone := ""
	if len(req.Question) > 0 {
		zone = req.Question[0].Name
	}

	const defaultBlockedResponseTTL = 3600

	soa := dns.SOA{
		// Values copied from verisign's nonexistent.com domain.
		//
		// Their exact values are not important in our use case because they are
		// used for domain transfers between primary/secondary DNS servers.
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
			Ttl:    s.dnsFilter.BlockedResponseTTL(),
			Class:  dns.ClassINET,
		},
		// zone will be appended later if it's not ".".
		Mbox: "hostmaster.",
	}
	if soa.Hdr.Ttl == 0 {
		soa.Hdr.Ttl = defaultBlockedResponseTTL
	}

	if zone != "." {
		soa.Mbox += zone
	}

	return []dns.RR{&soa}
}
