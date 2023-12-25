package dnsforward

import (
	"encoding/binary"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/miekg/dns"
)

// To transfer information between modules
//
// TODO(s.chzhen):  Add lowercased, non-FQDN version of the hostname from the
// question of the request.  Add persistent client.
type dnsContext struct {
	proxyCtx *proxy.DNSContext

	// setts are the filtering settings for the client.
	setts *filtering.Settings

	result *filtering.Result

	// origResp is the response received from upstream.  It is set when the
	// response is modified by filters.
	origResp *dns.Msg

	// unreversedReqIP stores an IP address obtained from a PTR request if it
	// was parsed successfully and belongs to one of the locally served IP
	// ranges.  It is also filled with unmapped version of the address if it's
	// within DNS64 prefixes.
	//
	// TODO(e.burkov):  Use netip.Addr when we switch to netip more fully.
	unreversedReqIP net.IP

	// err is the error returned from a processing function.
	err error

	// clientID is the ClientID from DoH, DoQ, or DoT, if provided.
	clientID string

	// startTime is the time at which the processing of the request has started.
	startTime time.Time

	// origQuestion is the question received from the client.  It is set
	// when the request is modified by rewrites.
	origQuestion dns.Question

	// protectionEnabled shows if the filtering is enabled, and if the
	// server's DNS filter is ready.
	protectionEnabled bool

	// responseFromUpstream shows if the response is received from the
	// upstream servers.
	responseFromUpstream bool

	// responseAD shows if the response had the AD bit set.
	responseAD bool

	// isLocalClient shows if client's IP address is from locally served
	// network.
	isLocalClient bool

	// isDHCPHost is true if the request for a local domain name and the DHCP is
	// available for this request.
	isDHCPHost bool
}

// resultCode is the result of a request processing function.
type resultCode int

const (
	// resultCodeSuccess is returned when a handler performed successfully, and
	// the next handler must be called.
	resultCodeSuccess resultCode = iota

	// resultCodeFinish is returned when a handler performed successfully, and
	// the processing of the request must be stopped.
	resultCodeFinish

	// resultCodeError is returned when a handler failed, and the processing of
	// the request must be stopped.
	resultCodeError
)

// ddrHostFQDN is the FQDN used in Discovery of Designated Resolvers (DDR) requests.
// See https://www.ietf.org/archive/id/draft-ietf-add-ddr-06.html.
const ddrHostFQDN = "_dns.resolver.arpa."

// handleDNSRequest filters the incoming DNS requests and writes them to the query log
func (s *Server) handleDNSRequest(_ *proxy.Proxy, pctx *proxy.DNSContext) error {
	dctx := &dnsContext{
		proxyCtx:  pctx,
		result:    &filtering.Result{},
		startTime: time.Now(),
	}

	type modProcessFunc func(ctx *dnsContext) (rc resultCode)

	// Since (*dnsforward.Server).handleDNSRequest(...) is used as
	// proxy.(Config).RequestHandler, there is no need for additional index
	// out of range checking in any of the following functions, because the
	// (*proxy.Proxy).handleDNSRequest method performs it before calling the
	// appropriate handler.
	mods := []modProcessFunc{
		s.processRecursion,
		s.processInitial,
		s.processDDRQuery,
		s.processDetermineLocal,
		s.processDHCPHosts,
		s.processRestrictLocal,
		s.processDHCPAddrs,
		s.processFilteringBeforeRequest,
		s.processLocalPTR,
		s.processUpstream,
		s.processFilteringAfterResponse,
		s.ipset.process,
		s.processQueryLogsAndStats,
	}
	for _, process := range mods {
		r := process(dctx)
		switch r {
		case resultCodeSuccess:
			// continue: call the next filter

		case resultCodeFinish:
			return nil

		case resultCodeError:
			return dctx.err
		}
	}

	if pctx.Res != nil {
		// Some devices require DNS message compression.
		pctx.Res.Compress = true
	}

	return nil
}

// processRecursion checks the incoming request and halts its handling by
// answering NXDOMAIN if s has tried to resolve it recently.
func (s *Server) processRecursion(dctx *dnsContext) (rc resultCode) {
	log.Debug("dnsforward: started processing recursion")
	defer log.Debug("dnsforward: finished processing recursion")

	pctx := dctx.proxyCtx

	if msg := pctx.Req; msg != nil && s.recDetector.check(*msg) {
		log.Debug("dnsforward: recursion detected resolving %q", msg.Question[0].Name)
		pctx.Res = s.genNXDomain(pctx.Req)

		return resultCodeFinish
	}

	return resultCodeSuccess
}

// mozillaFQDN is the domain used to signal the Firefox browser to not use its
// own DoH server.
//
// See https://support.mozilla.org/en-US/kb/canary-domain-use-application-dnsnet.
const mozillaFQDN = "use-application-dns.net."

// healthcheckFQDN is a reserved domain-name used for healthchecking.
//
// [Section 6.2 of RFC 6761] states that DNS Registries/Registrars must not
// grant requests to register test names in the normal way to any person or
// entity, making domain names under the .test TLD free to use in internal
// purposes.
//
// [Section 6.2 of RFC 6761]: https://www.rfc-editor.org/rfc/rfc6761.html#section-6.2
const healthcheckFQDN = "healthcheck.adguardhome.test."

// processInitial terminates the following processing for some requests if
// needed and enriches dctx with some client-specific information.
//
// TODO(e.burkov):  Decompose into less general processors.
func (s *Server) processInitial(dctx *dnsContext) (rc resultCode) {
	log.Debug("dnsforward: started processing initial")
	defer log.Debug("dnsforward: finished processing initial")

	pctx := dctx.proxyCtx
	s.processClientIP(pctx.Addr.Addr())

	q := pctx.Req.Question[0]
	qt := q.Qtype
	if s.conf.AAAADisabled && qt == dns.TypeAAAA {
		_ = proxy.CheckDisabledAAAARequest(pctx, true)

		return resultCodeFinish
	}

	if (qt == dns.TypeA || qt == dns.TypeAAAA) && q.Name == mozillaFQDN {
		pctx.Res = s.genNXDomain(pctx.Req)

		return resultCodeFinish
	}

	if q.Name == healthcheckFQDN {
		// Generate a NODATA negative response to make nslookup exit with 0.
		pctx.Res = s.makeResponse(pctx.Req)

		return resultCodeFinish
	}

	// Get the ClientID, if any, before getting client-specific filtering
	// settings.
	var key [8]byte
	binary.BigEndian.PutUint64(key[:], pctx.RequestID)
	dctx.clientID = string(s.clientIDCache.Get(key[:]))

	// Get the client-specific filtering settings.
	dctx.protectionEnabled, _ = s.UpdatedProtectionStatus()
	dctx.setts = s.clientRequestFilteringSettings(dctx)

	return resultCodeSuccess
}

// processClientIP sends the client IP address to s.addrProc, if needed.
func (s *Server) processClientIP(addr netip.Addr) {
	if !addr.IsValid() {
		log.Info("dnsforward: warning: bad client addr %q", addr)

		return
	}

	// Do not assign s.addrProc to a local variable to then use, since this lock
	// also serializes the closure of s.addrProc.
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	s.addrProc.Process(addr)
}

// processDDRQuery responds to Discovery of Designated Resolvers (DDR) SVCB
// queries.  The response contains different types of encryption supported by
// current user configuration.
//
// See https://www.ietf.org/archive/id/draft-ietf-add-ddr-10.html.
func (s *Server) processDDRQuery(dctx *dnsContext) (rc resultCode) {
	log.Debug("dnsforward: started processing ddr")
	defer log.Debug("dnsforward: finished processing ddr")

	if !s.conf.HandleDDR {
		return resultCodeSuccess
	}

	pctx := dctx.proxyCtx
	q := pctx.Req.Question[0]
	if q.Name == ddrHostFQDN {
		pctx.Res = s.makeDDRResponse(pctx.Req)

		return resultCodeFinish
	}

	return resultCodeSuccess
}

// makeDDRResponse creates a DDR answer based on the server configuration.  The
// constructed SVCB resource records have the priority of 1 for each entry,
// similar to examples provided by the [draft standard].
//
// TODO(a.meshkov):  Consider setting the priority values based on the protocol.
//
// [draft standard]: https://www.ietf.org/archive/id/draft-ietf-add-ddr-10.html.
func (s *Server) makeDDRResponse(req *dns.Msg) (resp *dns.Msg) {
	resp = s.makeResponse(req)
	if req.Question[0].Qtype != dns.TypeSVCB {
		return resp
	}

	// TODO(e.burkov):  Think about storing the FQDN version of the server's
	// name somewhere.
	domainName := dns.Fqdn(s.conf.ServerName)

	for _, addr := range s.conf.HTTPSListenAddrs {
		values := []dns.SVCBKeyValue{
			&dns.SVCBAlpn{Alpn: []string{"h2"}},
			&dns.SVCBPort{Port: uint16(addr.Port)},
			&dns.SVCBDoHPath{Template: "/dns-query{?dns}"},
		}

		ans := &dns.SVCB{
			Hdr:      s.hdr(req, dns.TypeSVCB),
			Priority: 1,
			Target:   domainName,
			Value:    values,
		}

		resp.Answer = append(resp.Answer, ans)
	}

	if s.conf.hasIPAddrs {
		// Only add DNS-over-TLS resolvers in case the certificate contains IP
		// addresses.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/4927.
		for _, addr := range s.dnsProxy.TLSListenAddr {
			values := []dns.SVCBKeyValue{
				&dns.SVCBAlpn{Alpn: []string{"dot"}},
				&dns.SVCBPort{Port: uint16(addr.Port)},
			}

			ans := &dns.SVCB{
				Hdr:      s.hdr(req, dns.TypeSVCB),
				Priority: 1,
				Target:   domainName,
				Value:    values,
			}

			resp.Answer = append(resp.Answer, ans)
		}
	}

	for _, addr := range s.dnsProxy.QUICListenAddr {
		values := []dns.SVCBKeyValue{
			&dns.SVCBAlpn{Alpn: []string{"doq"}},
			&dns.SVCBPort{Port: uint16(addr.Port)},
		}

		ans := &dns.SVCB{
			Hdr:      s.hdr(req, dns.TypeSVCB),
			Priority: 1,
			Target:   domainName,
			Value:    values,
		}

		resp.Answer = append(resp.Answer, ans)
	}

	return resp
}

// processDetermineLocal determines if the client's IP address is from locally
// served network and saves the result into the context.
func (s *Server) processDetermineLocal(dctx *dnsContext) (rc resultCode) {
	log.Debug("dnsforward: started processing local detection")
	defer log.Debug("dnsforward: finished processing local detection")

	rc = resultCodeSuccess

	dctx.isLocalClient = s.privateNets.Contains(dctx.proxyCtx.Addr.Addr().AsSlice())

	return rc
}

// processDHCPHosts respond to A requests if the target hostname is known to
// the server.  It responds with a mapped IP address if the DNS64 is enabled and
// the request is for AAAA.
//
// TODO(a.garipov): Adapt to AAAA as well.
func (s *Server) processDHCPHosts(dctx *dnsContext) (rc resultCode) {
	log.Debug("dnsforward: started processing dhcp hosts")
	defer log.Debug("dnsforward: finished processing dhcp hosts")

	pctx := dctx.proxyCtx
	req := pctx.Req

	q := &req.Question[0]
	dhcpHost := s.dhcpHostFromRequest(q)
	if dctx.isDHCPHost = dhcpHost != ""; !dctx.isDHCPHost {
		return resultCodeSuccess
	}

	if !dctx.isLocalClient {
		log.Debug("dnsforward: %q requests for dhcp host %q", pctx.Addr, dhcpHost)
		pctx.Res = s.genNXDomain(req)

		// Do not even put into query log.
		return resultCodeFinish
	}

	ip := s.dhcpServer.IPByHost(dhcpHost)
	if ip == (netip.Addr{}) {
		// Go on and process them with filters, including dnsrewrite ones, and
		// possibly route them to a domain-specific upstream.
		log.Debug("dnsforward: no dhcp record for %q", dhcpHost)

		return resultCodeSuccess
	}

	log.Debug("dnsforward: dhcp record for %q is %s", dhcpHost, ip)

	resp := s.makeResponse(req)
	switch q.Qtype {
	case dns.TypeA:
		a := &dns.A{
			Hdr: s.hdr(req, dns.TypeA),
			A:   ip.AsSlice(),
		}
		resp.Answer = append(resp.Answer, a)
	case dns.TypeAAAA:
		if s.dns64Pref != (netip.Prefix{}) {
			// Respond with DNS64-mapped address for IPv4 host if DNS64 is
			// enabled.
			aaaa := &dns.AAAA{
				Hdr:  s.hdr(req, dns.TypeAAAA),
				AAAA: s.mapDNS64(ip),
			}
			resp.Answer = append(resp.Answer, aaaa)
		}
	default:
		// Go on.
	}

	dctx.proxyCtx.Res = resp

	return resultCodeSuccess
}

// indexFirstV4Label returns the index at which the reversed IPv4 address
// starts, assuming the domain is pre-validated ARPA domain having in-addr and
// arpa labels removed.
func indexFirstV4Label(domain string) (idx int) {
	idx = len(domain)
	for labelsNum := 0; labelsNum < net.IPv4len && idx > 0; labelsNum++ {
		curIdx := strings.LastIndexByte(domain[:idx-1], '.') + 1
		_, parseErr := strconv.ParseUint(domain[curIdx:idx-1], 10, 8)
		if parseErr != nil {
			return idx
		}

		idx = curIdx
	}

	return idx
}

// indexFirstV6Label returns the index at which the reversed IPv6 address
// starts, assuming the domain is pre-validated ARPA domain having ip6 and arpa
// labels removed.
func indexFirstV6Label(domain string) (idx int) {
	idx = len(domain)
	for labelsNum := 0; labelsNum < net.IPv6len*2 && idx > 0; labelsNum++ {
		curIdx := idx - len("a.")
		if curIdx > 1 && domain[curIdx-1] != '.' {
			return idx
		}

		nibble := domain[curIdx]
		if (nibble < '0' || nibble > '9') && (nibble < 'a' || nibble > 'f') {
			return idx
		}

		idx = curIdx
	}

	return idx
}

// extractARPASubnet tries to convert a reversed ARPA address being a part of
// domain to an IP network.  domain must be an FQDN.
//
// TODO(e.burkov):  Move to golibs.
func extractARPASubnet(domain string) (pref netip.Prefix, err error) {
	err = netutil.ValidateDomainName(strings.TrimSuffix(domain, "."))
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return netip.Prefix{}, err
	}

	const (
		v4Suffix = "in-addr.arpa."
		v6Suffix = "ip6.arpa."
	)

	domain = strings.ToLower(domain)

	var idx int
	switch {
	case strings.HasSuffix(domain, v4Suffix):
		idx = indexFirstV4Label(domain[:len(domain)-len(v4Suffix)])
	case strings.HasSuffix(domain, v6Suffix):
		idx = indexFirstV6Label(domain[:len(domain)-len(v6Suffix)])
	default:
		return netip.Prefix{}, &netutil.AddrError{
			Err:  netutil.ErrNotAReversedSubnet,
			Kind: netutil.AddrKindARPA,
			Addr: domain,
		}
	}

	var subnet *net.IPNet
	subnet, err = netutil.SubnetFromReversedAddr(domain[idx:])
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return netip.Prefix{}, err
	}

	return netutil.IPNetToPrefixNoMapped(subnet)
}

// processRestrictLocal responds with NXDOMAIN to PTR requests for IP addresses
// in locally served network from external clients.
func (s *Server) processRestrictLocal(dctx *dnsContext) (rc resultCode) {
	log.Debug("dnsforward: started processing local restriction")
	defer log.Debug("dnsforward: finished processing local restriction")

	pctx := dctx.proxyCtx
	req := pctx.Req
	q := req.Question[0]
	if q.Qtype != dns.TypePTR {
		// No need for restriction.
		return resultCodeSuccess
	}

	subnet, err := extractARPASubnet(q.Name)
	if err != nil {
		if errors.Is(err, netutil.ErrNotAReversedSubnet) {
			log.Debug("dnsforward: request is not for arpa domain")

			return resultCodeSuccess
		}

		log.Debug("dnsforward: parsing reversed addr: %s", err)

		return resultCodeError
	}

	// Restrict an access to local addresses for external clients.  We also
	// assume that all the DHCP leases we give are locally served or at least
	// shouldn't be accessible externally.
	subnetAddr := subnet.Addr()
	addrData := subnetAddr.AsSlice()
	if !s.privateNets.Contains(addrData) {
		return resultCodeSuccess
	}

	log.Debug("dnsforward: addr %s is from locally served network", subnetAddr)

	if !dctx.isLocalClient {
		log.Debug("dnsforward: %q requests an internal ip", pctx.Addr)
		pctx.Res = s.genNXDomain(req)

		// Do not even put into query log.
		return resultCodeFinish
	}

	// Do not perform unreversing ever again.
	dctx.unreversedReqIP = addrData

	// There is no need to filter request from external addresses since this
	// code is only executed when the request is for locally served ARPA
	// hostname so disable redundant filters.
	dctx.setts.ParentalEnabled = false
	dctx.setts.SafeBrowsingEnabled = false
	dctx.setts.SafeSearchEnabled = false
	dctx.setts.ServicesRules = nil

	// Nothing to restrict.
	return resultCodeSuccess
}

// processDHCPAddrs responds to PTR requests if the target IP is leased by the
// DHCP server.
func (s *Server) processDHCPAddrs(dctx *dnsContext) (rc resultCode) {
	log.Debug("dnsforward: started processing dhcp addrs")
	defer log.Debug("dnsforward: finished processing dhcp addrs")

	pctx := dctx.proxyCtx
	if pctx.Res != nil {
		return resultCodeSuccess
	}

	ip := dctx.unreversedReqIP
	if ip == nil {
		return resultCodeSuccess
	}

	// TODO(a.garipov):  Remove once we switch to [netip.Addr] more fully.
	ipAddr, err := netutil.IPToAddrNoMapped(ip)
	if err != nil {
		log.Debug("dnsforward: bad reverse ip %v from dhcp: %s", ip, err)

		return resultCodeSuccess
	}

	host := s.dhcpServer.HostByIP(ipAddr)
	if host == "" {
		return resultCodeSuccess
	}

	log.Debug("dnsforward: dhcp client %s is %q", ip, host)

	req := pctx.Req
	resp := s.makeResponse(req)
	ptr := &dns.PTR{
		Hdr: dns.RR_Header{
			Name:   req.Question[0].Name,
			Rrtype: dns.TypePTR,
			// TODO(e.burkov):  Use [dhcpsvc.Lease.Expiry].  See
			// https://github.com/AdguardTeam/AdGuardHome/issues/3932.
			Ttl:   s.dnsFilter.BlockedResponseTTL(),
			Class: dns.ClassINET,
		},
		Ptr: dns.Fqdn(strings.Join([]string{host, s.localDomainSuffix}, ".")),
	}
	resp.Answer = append(resp.Answer, ptr)
	pctx.Res = resp

	return resultCodeSuccess
}

// processLocalPTR responds to PTR requests if the target IP is detected to be
// inside the local network and the query was not answered from DHCP.
func (s *Server) processLocalPTR(dctx *dnsContext) (rc resultCode) {
	log.Debug("dnsforward: started processing local ptr")
	defer log.Debug("dnsforward: finished processing local ptr")

	pctx := dctx.proxyCtx
	if pctx.Res != nil {
		return resultCodeSuccess
	}

	ip := dctx.unreversedReqIP
	if ip == nil {
		return resultCodeSuccess
	}

	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	if s.conf.UsePrivateRDNS {
		s.recDetector.add(*pctx.Req)
		if err := s.localResolvers.Resolve(pctx); err != nil {
			log.Debug("dnsforward: resolving private address: %s", err)

			// Generate the server failure if the private upstream configuration
			// is empty.
			//
			// This is a crutch, see TODO at [Server.localResolvers].
			if errors.Is(err, upstream.ErrNoUpstreams) {
				pctx.Res = s.genServerFailure(pctx.Req)

				// Do not even put into query log.
				return resultCodeFinish
			}

			dctx.err = err

			return resultCodeError
		}
	}

	if pctx.Res == nil {
		pctx.Res = s.genNXDomain(pctx.Req)

		// Do not even put into query log.
		return resultCodeFinish
	}

	return resultCodeSuccess
}

// Apply filtering logic
func (s *Server) processFilteringBeforeRequest(dctx *dnsContext) (rc resultCode) {
	log.Debug("dnsforward: started processing filtering before req")
	defer log.Debug("dnsforward: finished processing filtering before req")

	if dctx.proxyCtx.Res != nil {
		// Go on since the response is already set.
		return resultCodeSuccess
	}

	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	var err error
	if dctx.result, err = s.filterDNSRequest(dctx); err != nil {
		dctx.err = err

		return resultCodeError
	}

	return resultCodeSuccess
}

// ipStringFromAddr extracts an IP address string from net.Addr.
func ipStringFromAddr(addr net.Addr) (ipStr string) {
	if ip, _ := netutil.IPAndPortFromAddr(addr); ip != nil {
		return ip.String()
	}

	return ""
}

// processUpstream passes request to upstream servers and handles the response.
func (s *Server) processUpstream(dctx *dnsContext) (rc resultCode) {
	log.Debug("dnsforward: started processing upstream")
	defer log.Debug("dnsforward: finished processing upstream")

	pctx := dctx.proxyCtx
	req := pctx.Req

	if pctx.Res != nil {
		// The response has already been set.
		return resultCodeSuccess
	} else if dctx.isDHCPHost {
		// A DHCP client hostname query that hasn't been handled or filtered.
		// Respond with an NXDOMAIN.
		//
		// TODO(a.garipov): Route such queries to a custom upstream for the
		// local domain name if there is one.
		name := req.Question[0].Name
		log.Debug("dnsforward: dhcp client hostname %q was not filtered", name[:len(name)-1])
		pctx.Res = s.genNXDomain(req)

		return resultCodeFinish
	}

	s.setCustomUpstream(pctx, dctx.clientID)

	reqWantsDNSSEC := s.setReqAD(req)

	// Process the request further since it wasn't filtered.
	prx := s.proxy()
	if prx == nil {
		dctx.err = srvClosedErr

		return resultCodeError
	}

	if err := prx.Resolve(pctx); err != nil {
		if errors.Is(err, upstream.ErrNoUpstreams) {
			// Do not even put into querylog.  Currently this happens either
			// when the private resolvers enabled and the request is DNS64 PTR,
			// or when the client isn't considered local by prx.
			//
			// TODO(e.burkov):  Make proxy detect local client the same way as
			// AGH does.
			pctx.Res = s.genNXDomain(req)

			return resultCodeFinish
		}

		dctx.err = err

		return resultCodeError
	}

	dctx.responseFromUpstream = true
	dctx.responseAD = pctx.Res.AuthenticatedData

	s.setRespAD(pctx, reqWantsDNSSEC)

	return resultCodeSuccess
}

// setReqAD changes the request based on the server settings.  wantsDNSSEC is
// false if the response should be cleared of the AD bit.
//
// TODO(a.garipov, e.burkov): This should probably be done in module dnsproxy.
func (s *Server) setReqAD(req *dns.Msg) (wantsDNSSEC bool) {
	if !s.conf.EnableDNSSEC {
		return false
	}

	origReqAD := req.AuthenticatedData
	req.AuthenticatedData = true

	// Per [RFC 6840] says, validating resolvers should only set the AD bit when
	// the response has the AD bit set and the request contained either a set DO
	// bit or a set AD bit.  So, if neither of these is true, clear the AD bits
	// in [Server.setRespAD].
	//
	// [RFC 6840]: https://datatracker.ietf.org/doc/html/rfc6840#section-5.8
	return origReqAD || hasDO(req)
}

// hasDO returns true if msg has EDNS(0) options and the DNSSEC OK flag is set
// in there.
//
// TODO(a.garipov): Move to golibs/dnsmsg when it's there.
func hasDO(msg *dns.Msg) (do bool) {
	o := msg.IsEdns0()
	if o == nil {
		return false
	}

	return o.Do()
}

// setRespAD changes the request and response based on the server settings and
// the original request data.
func (s *Server) setRespAD(pctx *proxy.DNSContext, reqWantsDNSSEC bool) {
	if s.conf.EnableDNSSEC && !reqWantsDNSSEC {
		pctx.Req.AuthenticatedData = false
		pctx.Res.AuthenticatedData = false
	}
}

// dhcpHostFromRequest returns a hostname from question, if the request is for a
// DHCP client's hostname when DHCP is enabled, and an empty string otherwise.
func (s *Server) dhcpHostFromRequest(q *dns.Question) (reqHost string) {
	if !s.dhcpServer.Enabled() {
		return ""
	}

	// Include AAAA here, because despite the fact that we don't support it yet,
	// the expected behavior here is to respond with an empty answer and not
	// NXDOMAIN.
	if qt := q.Qtype; qt != dns.TypeA && qt != dns.TypeAAAA {
		return ""
	}

	reqHost = strings.ToLower(q.Name[:len(q.Name)-1])
	if !netutil.IsImmediateSubdomain(reqHost, s.localDomainSuffix) {
		return ""
	}

	return reqHost[:len(reqHost)-len(s.localDomainSuffix)-1]
}

// setCustomUpstream sets custom upstream settings in pctx, if necessary.
func (s *Server) setCustomUpstream(pctx *proxy.DNSContext, clientID string) {
	if !pctx.Addr.IsValid() || s.conf.ClientsContainer == nil {
		return
	}

	// Use the ClientID first, since it has a higher priority.
	id := stringutil.Coalesce(clientID, pctx.Addr.Addr().String())
	upsConf, err := s.conf.ClientsContainer.UpstreamConfigByID(id, s.bootstrap)
	if err != nil {
		log.Error("dnsforward: getting custom upstreams for client %s: %s", id, err)

		return
	}

	if upsConf != nil {
		log.Debug("dnsforward: using custom upstreams for client %s", id)

		pctx.CustomUpstreamConfig = upsConf
	}
}

// Apply filtering logic after we have received response from upstream servers
func (s *Server) processFilteringAfterResponse(dctx *dnsContext) (rc resultCode) {
	log.Debug("dnsforward: started processing filtering after resp")
	defer log.Debug("dnsforward: finished processing filtering after resp")

	switch res := dctx.result; res.Reason {
	case filtering.NotFilteredAllowList:
		return resultCodeSuccess
	case
		filtering.Rewritten,
		filtering.RewrittenRule:

		if dctx.origQuestion.Name == "" {
			// origQuestion is set in case we get only CNAME without IP from
			// rewrites table.
			return resultCodeSuccess
		}

		pctx := dctx.proxyCtx
		pctx.Req.Question[0], pctx.Res.Question[0] = dctx.origQuestion, dctx.origQuestion
		if len(pctx.Res.Answer) > 0 {
			rr := s.genAnswerCNAME(pctx.Req, res.CanonName)
			answer := append([]dns.RR{rr}, pctx.Res.Answer...)
			pctx.Res.Answer = answer
		}

		return resultCodeSuccess
	default:
		return s.filterAfterResponse(dctx)
	}
}

// filterAfterResponse returns the result of filtering the response that wasn't
// explicitly allowed or rewritten.
func (s *Server) filterAfterResponse(dctx *dnsContext) (res resultCode) {
	// Check the response only if it's from an upstream.  Don't check the
	// response if the protection is disabled since dnsrewrite rules aren't
	// applied to it anyway.
	if !dctx.protectionEnabled || !dctx.responseFromUpstream {
		return resultCodeSuccess
	}

	err := s.filterDNSResponse(dctx)
	if err != nil {
		dctx.err = err

		return resultCodeError
	}

	return resultCodeSuccess
}
