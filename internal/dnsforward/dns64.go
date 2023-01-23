package dnsforward

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/miekg/dns"
)

const (
	// maxNAT64PrefixBitLen is the maximum length of a NAT64 prefix in bits.
	// See https://datatracker.ietf.org/doc/html/rfc6147#section-5.2.
	maxNAT64PrefixBitLen = 96

	// nat64PrefixLen is the length of a NAT64 prefix in bytes.
	nat64PrefixLen = net.IPv6len - net.IPv4len

	// maxDNS64SynTTL is the maximum TTL for synthesized DNS64 responses with no
	// SOA records in seconds.
	//
	// If the SOA RR was not delivered with the negative response to the AAAA
	// query, then the DNS64 SHOULD use the TTL of the original A RR or 600
	// seconds, whichever is shorter.
	//
	// See https://datatracker.ietf.org/doc/html/rfc6147#section-5.1.7.
	maxDNS64SynTTL uint32 = 600
)

// setupDNS64 initializes DNS64 settings, the NAT64 prefixes in particular.  If
// the DNS64 feature is enabled and no prefixes are configured, the default
// Well-Known Prefix is used, just like Section 5.2 of RFC 6147 prescribes.  Any
// configured set of prefixes discards the default Well-Known prefix unless it
// is specified explicitly.  Each prefix also validated to be a valid IPv6
// CIDR with a maximum length of 96 bits.  The first specified prefix is then
// used to synthesize AAAA records.
func (s *Server) setupDNS64() (err error) {
	if !s.conf.UseDNS64 {
		return nil
	}

	l := len(s.conf.DNS64Prefixes)
	if l == 0 {
		s.dns64Prefs = []netip.Prefix{dns64WellKnownPref}

		return nil
	}

	prefs := make([]netip.Prefix, 0, l)
	for i, pref := range s.conf.DNS64Prefixes {
		var p netip.Prefix
		p, err = netip.ParsePrefix(pref)
		if err != nil {
			return fmt.Errorf("prefix at index %d: %w", i, err)
		}

		addr := p.Addr()
		if !addr.Is6() {
			return fmt.Errorf("prefix at index %d: %q is not an IPv6 prefix", i, pref)
		}

		if p.Bits() > maxNAT64PrefixBitLen {
			return fmt.Errorf("prefix at index %d: %q is too long for DNS64", i, pref)
		}

		prefs = append(prefs, p.Masked())
	}

	s.dns64Prefs = prefs

	return nil
}

// checkDNS64 checks if DNS64 should be performed.  It returns a DNS64 request
// to resolve or nil if DNS64 is not desired.  It also filters resp to not
// contain any NAT64 excluded addresses in the answer section, if needed.  Both
// req and resp must not be nil.
//
// See https://datatracker.ietf.org/doc/html/rfc6147.
func (s *Server) checkDNS64(req, resp *dns.Msg) (dns64Req *dns.Msg) {
	if len(s.dns64Prefs) == 0 {
		return nil
	}

	q := req.Question[0]
	if q.Qtype != dns.TypeAAAA || q.Qclass != dns.ClassINET {
		// DNS64 operation for classes other than IN is undefined, and a DNS64
		// MUST behave as though no DNS64 function is configured.
		return nil
	}

	rcode := resp.Rcode
	if rcode == dns.RcodeNameError {
		// A result with RCODE=3 (Name Error) is handled according to normal DNS
		// operation (which is normally to return the error to the client).
		return nil
	}

	if rcode == dns.RcodeSuccess {
		// If resolver receives an answer with at least one AAAA record
		// containing an address outside any of the excluded range(s), then it
		// by default SHOULD build an answer section for a response including
		// only the AAAA record(s) that do not contain any of the addresses
		// inside the excluded ranges.
		var hasAnswers bool
		if resp.Answer, hasAnswers = s.filterNAT64Answers(resp.Answer); hasAnswers {
			return nil
		}

		// Any other RCODE is treated as though the RCODE were 0 and the answer
		// section were empty.
	}

	return &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:                dns.Id(),
			RecursionDesired:  req.RecursionDesired,
			AuthenticatedData: req.AuthenticatedData,
			CheckingDisabled:  req.CheckingDisabled,
		},
		Question: []dns.Question{{
			Name:   req.Question[0].Name,
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		}},
	}
}

// filterNAT64Answers filters out AAAA records that are within one of NAT64
// exclusion prefixes.  hasAnswers is true if the filtered slice contains at
// least a single AAAA answer not within the prefixes or a CNAME.
func (s *Server) filterNAT64Answers(rrs []dns.RR) (filtered []dns.RR, hasAnswers bool) {
	filtered = make([]dns.RR, 0, len(rrs))
	for _, ans := range rrs {
		switch ans := ans.(type) {
		case *dns.AAAA:
			addr, err := netutil.IPToAddrNoMapped(ans.AAAA)
			if err != nil {
				log.Error("dnsforward: bad AAAA record: %s", err)

				continue
			}

			if s.withinDNS64(addr) {
				// Filter the record.
				continue
			}

			filtered, hasAnswers = append(filtered, ans), true
		case *dns.CNAME, *dns.DNAME:
			// If the response contains a CNAME or a DNAME, then the CNAME or
			// DNAME chain is followed until the first terminating A or AAAA
			// record is reached.
			//
			// Just treat CNAME and DNAME responses as passable answers since
			// AdGuard Home doesn't follow any of these chains except the
			// dnsrewrite-defined ones.
			filtered, hasAnswers = append(filtered, ans), true
		default:
			filtered = append(filtered, ans)
		}
	}

	return filtered, hasAnswers
}

// synthDNS64 synthesizes a DNS64 response using the original response as a
// basis and modifying it with data from resp.  It returns true if the response
// was actually modified.
func (s *Server) synthDNS64(origReq, origResp, resp *dns.Msg) (ok bool) {
	if len(resp.Answer) == 0 {
		// If there is an empty answer, then the DNS64 responds to the original
		// querying client with the answer the DNS64 received to the original
		// (initiator's) query.
		return false
	}

	// The Time to Live (TTL) field is set to the minimum of the TTL of the
	// original A RR and the SOA RR for the queried domain.  If the original
	// response contains no SOA records, the minimum of the TTL of the original
	// A RR and [maxDNS64SynTTL] should be used.  See [maxDNS64SynTTL].
	soaTTL := maxDNS64SynTTL
	for _, rr := range origResp.Ns {
		if hdr := rr.Header(); hdr.Rrtype == dns.TypeSOA && hdr.Name == origReq.Question[0].Name {
			soaTTL = hdr.Ttl

			break
		}
	}

	newAns := make([]dns.RR, 0, len(resp.Answer))
	for _, ans := range resp.Answer {
		rr := s.synthRR(ans, soaTTL)
		if rr == nil {
			// The error should have already been logged.
			return false
		}

		newAns = append(newAns, rr)
	}

	origResp.Answer = newAns
	origResp.Ns = resp.Ns
	origResp.Extra = resp.Extra

	return true
}

// dns64WellKnownPref is the default prefix to use in an algorithmic mapping for
// DNS64.  See https://datatracker.ietf.org/doc/html/rfc6052#section-2.1.
var dns64WellKnownPref = netip.MustParsePrefix("64:ff9b::/96")

// withinDNS64 checks if ip is within one of the configured DNS64 prefixes.
//
// TODO(e.burkov):  We actually using bytes of only the first prefix from the
// set to construct the answer, so consider using some implementation of a
// prefix set for the rest.
func (s *Server) withinDNS64(ip netip.Addr) (ok bool) {
	for _, n := range s.dns64Prefs {
		if n.Contains(ip) {
			return true
		}
	}

	return false
}

// shouldStripDNS64 returns true if DNS64 is enabled and ip has either one of
// custom DNS64 prefixes or the Well-Known one.  This is intended to be used
// with PTR requests.
//
// The requirement is to match any Pref64::/n used at the site, and not merely
// the locally configured Pref64::/n.  This is because end clients could ask for
// a PTR record matching an address received through a different (site-provided)
// DNS64.
//
// See https://datatracker.ietf.org/doc/html/rfc6147#section-5.3.1.
func (s *Server) shouldStripDNS64(ip net.IP) (ok bool) {
	if len(s.dns64Prefs) == 0 {
		return false
	}

	addr, err := netutil.IPToAddr(ip, netutil.AddrFamilyIPv6)
	if err != nil {
		return false
	}

	switch {
	case s.withinDNS64(addr):
		log.Debug("dnsforward: %s is within DNS64 custom prefix set", ip)
	case dns64WellKnownPref.Contains(addr):
		log.Debug("dnsforward: %s is within DNS64 well-known prefix", ip)
	default:
		return false
	}

	return true
}

// mapDNS64 maps ip to IPv6 address using configured DNS64 prefix.  ip must be a
// valid IPv4.  It panics, if there are no configured DNS64 prefixes, because
// synthesis should not be performed unless DNS64 function enabled.
func (s *Server) mapDNS64(ip netip.Addr) (mapped net.IP) {
	// Don't mask the address here since it should have already been masked on
	// initialization stage.
	pref := s.dns64Prefs[0].Addr().As16()
	ipData := ip.As4()

	mapped = make(net.IP, net.IPv6len)
	copy(mapped[:nat64PrefixLen], pref[:])
	copy(mapped[nat64PrefixLen:], ipData[:])

	return mapped
}

// performDNS64 processes the current state of dctx assuming that it has already
// been tried to resolve, checks if it contains any acceptable response, and if
// it doesn't, performs DNS64 request and the following synthesis.  It returns
// the [resultCodeError] if there was an error set to dctx.
func (s *Server) performDNS64(prx *proxy.Proxy, dctx *dnsContext) (rc resultCode) {
	pctx := dctx.proxyCtx
	req := pctx.Req

	dns64Req := s.checkDNS64(req, pctx.Res)
	if dns64Req == nil {
		return resultCodeSuccess
	}

	log.Debug("dnsforward: received an empty AAAA response, checking DNS64")

	origReq := pctx.Req
	origResp := pctx.Res
	origUps := pctx.Upstream

	pctx.Req = dns64Req
	defer func() { pctx.Req = origReq }()

	if dctx.err = prx.Resolve(pctx); dctx.err != nil {
		return resultCodeError
	}

	dns64Resp := pctx.Res
	pctx.Res = origResp
	if dns64Resp != nil && s.synthDNS64(origReq, pctx.Res, dns64Resp) {
		log.Debug("dnsforward: synthesized AAAA response for %q", origReq.Question[0].Name)
	} else {
		pctx.Upstream = origUps
	}

	return resultCodeSuccess
}

// synthRR synthesizes a DNS64 resource record in compliance with RFC 6147.  If
// rr is not an A record, it's returned as is.  A records are modified to become
// a DNS64-synthesized AAAA records, and the TTL is set according to the
// original TTL of a record and soaTTL.  It returns nil on invalid A records.
func (s *Server) synthRR(rr dns.RR, soaTTL uint32) (result dns.RR) {
	aResp, ok := rr.(*dns.A)
	if !ok {
		return rr
	}

	addr, err := netutil.IPToAddr(aResp.A, netutil.AddrFamilyIPv4)
	if err != nil {
		log.Error("dnsforward: bad A record: %s", err)

		return nil
	}

	aaaa := &dns.AAAA{
		Hdr: dns.RR_Header{
			Name:   aResp.Hdr.Name,
			Rrtype: dns.TypeAAAA,
			Class:  aResp.Hdr.Class,
		},
		AAAA: s.mapDNS64(addr),
	}

	if rrTTL := aResp.Hdr.Ttl; rrTTL < soaTTL {
		aaaa.Hdr.Ttl = rrTTL
	} else {
		aaaa.Hdr.Ttl = soaTTL
	}

	return aaaa
}
