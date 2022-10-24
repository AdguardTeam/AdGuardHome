package dnsforward

import (
	"encoding/binary"
	"net"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/miekg/dns"
	"golang.org/x/exp/slices"
)

//lint:file-ignore SA1019 TODO(a.garipov):  Replace [*netutil.IPMap].

// To transfer information between modules
type dnsContext struct {
	proxyCtx *proxy.DNSContext

	// setts are the filtering settings for the client.
	setts *filtering.Settings

	result *filtering.Result
	// origResp is the response received from upstream.  It is set when the
	// response is modified by filters.
	origResp *dns.Msg

	// unreversedReqIP stores an IP address obtained from PTR request if it
	// parsed successfully and belongs to one of locally-served IP ranges as per
	// RFC 6303.
	unreversedReqIP net.IP

	// err is the error returned from a processing function.
	err error

	// clientID is the ClientID from DoH, DoQ, or DoT, if provided.
	clientID string

	// origQuestion is the question received from the client.  It is set
	// when the request is modified by rewrites.
	origQuestion dns.Question

	// startTime is the time at which the processing of the request has started.
	startTime time.Time

	// protectionEnabled shows if the filtering is enabled, and if the
	// server's DNS filter is ready.
	protectionEnabled bool

	// responseFromUpstream shows if the response is received from the
	// upstream servers.
	responseFromUpstream bool

	// responseAD shows if the response had the AD bit set.
	responseAD bool

	// isLocalClient shows if client's IP address is from locally-served
	// network.
	isLocalClient bool
}

// resultCode is the result of a request processing function.
type resultCode int

const (
	// resultCodeSuccess is returned when a handler performed successfully,
	// and the next handler must be called.
	resultCodeSuccess resultCode = iota
	// resultCodeFinish is returned when a handler performed successfully,
	// and the processing of the request must be stopped.
	resultCodeFinish
	// resultCodeError is returned when a handler failed, and the processing
	// of the request must be stopped.
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

// processRecursion checks the incoming request and halts it's handling if s
// have tried to resolve it recently.
func (s *Server) processRecursion(dctx *dnsContext) (rc resultCode) {
	pctx := dctx.proxyCtx

	if msg := pctx.Req; msg != nil && s.recDetector.check(*msg) {
		log.Debug("recursion detected resolving %q", msg.Question[0].Name)
		pctx.Res = s.genNXDomain(pctx.Req)

		return resultCodeFinish
	}

	return resultCodeSuccess
}

// processInitial terminates the following processing for some requests if
// needed and enriches the ctx with some client-specific information.
//
// TODO(e.burkov):  Decompose into less general processors.
func (s *Server) processInitial(dctx *dnsContext) (rc resultCode) {
	pctx := dctx.proxyCtx
	q := pctx.Req.Question[0]
	qt := q.Qtype
	if s.conf.AAAADisabled && qt == dns.TypeAAAA {
		_ = proxy.CheckDisabledAAAARequest(pctx, true)

		return resultCodeFinish
	}

	if s.conf.OnDNSRequest != nil {
		s.conf.OnDNSRequest(pctx)
	}

	// Disable Mozilla DoH.
	//
	// See https://support.mozilla.org/en-US/kb/canary-domain-use-application-dnsnet.
	if (qt == dns.TypeA || qt == dns.TypeAAAA) && q.Name == "use-application-dns.net." {
		pctx.Res = s.genNXDomain(pctx.Req)

		return resultCodeFinish
	}

	// Get the ClientID, if any, before getting client-specific filtering
	// settings.
	var key [8]byte
	binary.BigEndian.PutUint64(key[:], pctx.RequestID)
	dctx.clientID = string(s.clientIDCache.Get(key[:]))

	// Get the client-specific filtering settings.
	dctx.protectionEnabled = s.conf.ProtectionEnabled
	dctx.setts = s.getClientRequestFilteringSettings(dctx)

	return resultCodeSuccess
}

func (s *Server) setTableHostToIP(t hostToIPTable) {
	s.tableHostToIPLock.Lock()
	defer s.tableHostToIPLock.Unlock()

	s.tableHostToIP = t
}

func (s *Server) setTableIPToHost(t *netutil.IPMap) {
	s.tableIPToHostLock.Lock()
	defer s.tableIPToHostLock.Unlock()

	s.tableIPToHost = t
}

func (s *Server) onDHCPLeaseChanged(flags int) {
	var err error

	add := true
	switch flags {
	case dhcpd.LeaseChangedAdded,
		dhcpd.LeaseChangedAddedStatic,
		dhcpd.LeaseChangedRemovedStatic:
		// Go on.
	case dhcpd.LeaseChangedRemovedAll:
		add = false
	default:
		return
	}

	var hostToIP hostToIPTable
	var ipToHost *netutil.IPMap
	if add {
		ll := s.dhcpServer.Leases(dhcpd.LeasesAll)

		hostToIP = make(hostToIPTable, len(ll))
		ipToHost = netutil.NewIPMap(len(ll))

		for _, l := range ll {
			// TODO(a.garipov): Remove this after we're finished with the client
			// hostname validations in the DHCP server code.
			err = netutil.ValidateDomainName(l.Hostname)
			if err != nil {
				log.Debug(
					"dns: skipping invalid hostname %q from dhcp: %s",
					l.Hostname,
					err,
				)
			}

			lowhost := strings.ToLower(l.Hostname + "." + s.localDomainSuffix)
			ip := slices.Clone(l.IP)

			ipToHost.Set(ip, lowhost)
			hostToIP[lowhost] = ip
		}

		log.Debug("dns: added %d A/PTR entries from DHCP", ipToHost.Len())
	}

	s.setTableHostToIP(hostToIP)
	s.setTableIPToHost(ipToHost)
}

// processDDRQuery responds to Discovery of Designated Resolvers (DDR) SVCB
// queries.  The response contains different types of encryption supported by
// current user configuration.
//
// See https://www.ietf.org/archive/id/draft-ietf-add-ddr-10.html.
func (s *Server) processDDRQuery(dctx *dnsContext) (rc resultCode) {
	pctx := dctx.proxyCtx
	q := pctx.Req.Question[0]

	if !s.conf.HandleDDR {
		return resultCodeSuccess
	}

	if q.Name == ddrHostFQDN {
		if s.dnsProxy.TLSListenAddr == nil && s.conf.HTTPSListenAddrs == nil &&
			s.dnsProxy.QUICListenAddr == nil || q.Qtype != dns.TypeSVCB {
			pctx.Res = s.makeResponse(pctx.Req)

			return resultCodeFinish
		}

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

// processDetermineLocal determines if the client's IP address is from
// locally-served network and saves the result into the context.
func (s *Server) processDetermineLocal(dctx *dnsContext) (rc resultCode) {
	rc = resultCodeSuccess

	var ip net.IP
	if ip, _ = netutil.IPAndPortFromAddr(dctx.proxyCtx.Addr); ip == nil {
		return rc
	}

	dctx.isLocalClient = s.privateNets.Contains(ip)

	return rc
}

// dhcpHostToIP tries to get an IP leased by DHCP and returns the copy of
// address since the data inside the internal table may be changed while request
// processing.  It's safe for concurrent use.
func (s *Server) dhcpHostToIP(host string) (ip net.IP, ok bool) {
	s.tableHostToIPLock.Lock()
	defer s.tableHostToIPLock.Unlock()

	if s.tableHostToIP == nil {
		return nil, false
	}

	var ipFromTable net.IP
	ipFromTable, ok = s.tableHostToIP[host]
	if !ok {
		return nil, false
	}

	ip = make(net.IP, len(ipFromTable))
	copy(ip, ipFromTable)

	return ip, true
}

// processDHCPHosts respond to A requests if the target hostname is known to
// the server.
//
// TODO(a.garipov): Adapt to AAAA as well.
func (s *Server) processDHCPHosts(dctx *dnsContext) (rc resultCode) {
	pctx := dctx.proxyCtx
	req := pctx.Req
	q := req.Question[0]
	reqHost, ok := s.isDHCPClientHostQ(q)
	if !ok {
		return resultCodeSuccess
	}

	if !dctx.isLocalClient {
		log.Debug("dns: %q requests for dhcp host %q", pctx.Addr, reqHost)
		pctx.Res = s.genNXDomain(req)

		// Do not even put into query log.
		return resultCodeFinish
	}

	ip, ok := s.dhcpHostToIP(reqHost)
	if !ok {
		// Go on and process them with filters, including dnsrewrite ones, and
		// possibly route them to a domain-specific upstream.
		log.Debug("dns: no dhcp record for %q", reqHost)

		return resultCodeSuccess
	}

	log.Debug("dns: dhcp record for %q is %s", reqHost, ip)

	resp := s.makeResponse(req)
	if q.Qtype == dns.TypeA {
		a := &dns.A{
			Hdr: s.hdr(req, dns.TypeA),
			A:   ip,
		}
		resp.Answer = append(resp.Answer, a)
	}
	dctx.proxyCtx.Res = resp

	return resultCodeSuccess
}

// processRestrictLocal responds with NXDOMAIN to PTR requests for IP addresses
// in locally-served network from external clients.
func (s *Server) processRestrictLocal(dctx *dnsContext) (rc resultCode) {
	pctx := dctx.proxyCtx
	req := pctx.Req
	q := req.Question[0]
	if q.Qtype != dns.TypePTR {
		// No need for restriction.
		return resultCodeSuccess
	}

	ip, err := netutil.IPFromReversedAddr(q.Name)
	if err != nil {
		log.Debug("dns: parsing reversed addr: %s", err)

		// DNS-Based Service Discovery uses PTR records having not an ARPA
		// format of the domain name in question.  Those shouldn't be
		// invalidated.  See http://www.dns-sd.org/ServerStaticSetup.html and
		// RFC 2782.
		name := strings.TrimSuffix(q.Name, ".")
		if err = netutil.ValidateSRVDomainName(name); err != nil {
			log.Debug("dns: validating service domain: %s", err)

			return resultCodeError
		}

		log.Debug("dns: request is for a service domain")

		return resultCodeSuccess
	}

	// Restrict an access to local addresses for external clients.  We also
	// assume that all the DHCP leases we give are locally-served or at least
	// don't need to be accessible externally.
	if !s.privateNets.Contains(ip) {
		log.Debug("dns: addr %s is not from locally-served network", ip)

		return resultCodeSuccess
	}

	if !dctx.isLocalClient {
		log.Debug("dns: %q requests an internal ip", pctx.Addr)
		pctx.Res = s.genNXDomain(req)

		// Do not even put into query log.
		return resultCodeFinish
	}

	// Do not perform unreversing ever again.
	dctx.unreversedReqIP = ip

	// There is no need to filter request from external addresses since this
	// code is only executed when the request is for locally-served ARPA
	// hostname so disable redundant filters.
	dctx.setts.ParentalEnabled = false
	dctx.setts.SafeBrowsingEnabled = false
	dctx.setts.SafeSearchEnabled = false
	dctx.setts.ServicesRules = nil

	// Nothing to restrict.
	return resultCodeSuccess
}

// ipToDHCPHost tries to get a hostname leased by DHCP.  It's safe for
// concurrent use.
func (s *Server) ipToDHCPHost(ip net.IP) (host string, ok bool) {
	s.tableIPToHostLock.Lock()
	defer s.tableIPToHostLock.Unlock()

	if s.tableIPToHost == nil {
		return "", false
	}

	var v any
	v, ok = s.tableIPToHost.Get(ip)
	if !ok {
		return "", false
	}

	if host, ok = v.(string); !ok {
		log.Error("dns: bad type %T in tableIPToHost for %s", v, ip)

		return "", false
	}

	return host, true
}

// processDHCPAddrs responds to PTR requests if the target IP is leased by the
// DHCP server.
func (s *Server) processDHCPAddrs(dctx *dnsContext) (rc resultCode) {
	pctx := dctx.proxyCtx
	if pctx.Res != nil {
		return resultCodeSuccess
	}

	ip := dctx.unreversedReqIP
	if ip == nil {
		return resultCodeSuccess
	}

	host, ok := s.ipToDHCPHost(ip)
	if !ok {
		return resultCodeSuccess
	}

	log.Debug("dns: dhcp reverse record for %s is %q", ip, host)

	req := pctx.Req
	resp := s.makeResponse(req)
	ptr := &dns.PTR{
		Hdr: dns.RR_Header{
			Name:   req.Question[0].Name,
			Rrtype: dns.TypePTR,
			Ttl:    s.conf.BlockedResponseTTL,
			Class:  dns.ClassINET,
		},
		Ptr: dns.Fqdn(host),
	}
	resp.Answer = append(resp.Answer, ptr)
	pctx.Res = resp

	return resultCodeSuccess
}

// processLocalPTR responds to PTR requests if the target IP is detected to be
// inside the local network and the query was not answered from DHCP.
func (s *Server) processLocalPTR(dctx *dnsContext) (rc resultCode) {
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

	if !s.privateNets.Contains(ip) {
		return resultCodeSuccess
	}

	if s.conf.UsePrivateRDNS {
		s.recDetector.add(*pctx.Req)
		if err := s.localResolvers.Resolve(pctx); err != nil {
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
func (s *Server) processFilteringBeforeRequest(ctx *dnsContext) (rc resultCode) {
	if ctx.proxyCtx.Res != nil {
		// Go on since the response is already set.
		return resultCodeSuccess
	}

	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	if s.dnsFilter == nil {
		return resultCodeSuccess
	}

	var err error
	if ctx.result, err = s.filterDNSRequest(ctx); err != nil {
		ctx.err = err

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
	pctx := dctx.proxyCtx
	req := pctx.Req
	q := req.Question[0]
	if pctx.Res != nil {
		// The response has already been set.
		return resultCodeSuccess
	} else if reqHost, ok := s.isDHCPClientHostQ(q); ok {
		// A DHCP client hostname query that hasn't been handled or filtered.
		// Respond with an NXDOMAIN.
		//
		// TODO(a.garipov): Route such queries to a custom upstream for the
		// local domain name if there is one.
		log.Debug("dns: dhcp client hostname %q was not filtered", reqHost)
		pctx.Res = s.genNXDomain(req)

		return resultCodeFinish
	}

	s.setCustomUpstream(pctx, dctx.clientID)

	origReqAD := false
	if s.conf.EnableDNSSEC {
		if req.AuthenticatedData {
			origReqAD = true
		} else {
			req.AuthenticatedData = true
		}
	}

	// Process the request further since it wasn't filtered.
	prx := s.proxy()
	if prx == nil {
		dctx.err = srvClosedErr

		return resultCodeError
	}

	if dctx.err = prx.Resolve(pctx); dctx.err != nil {
		return resultCodeError
	}

	dctx.responseFromUpstream = true
	dctx.responseAD = pctx.Res.AuthenticatedData

	if s.conf.EnableDNSSEC && !origReqAD {
		pctx.Req.AuthenticatedData = false
		pctx.Res.AuthenticatedData = false
	}

	return resultCodeSuccess
}

// isDHCPClientHostQ returns true if q is from a request for a DHCP client
// hostname.  If ok is true, reqHost contains the requested hostname.
func (s *Server) isDHCPClientHostQ(q dns.Question) (reqHost string, ok bool) {
	if !s.dhcpServer.Enabled() {
		return "", false
	}

	// Include AAAA here, because despite the fact that we don't support it yet,
	// the expected behavior here is to respond with an empty answer and not
	// NXDOMAIN.
	if qt := q.Qtype; qt != dns.TypeA && qt != dns.TypeAAAA {
		return "", false
	}

	reqHost = strings.ToLower(q.Name[:len(q.Name)-1])
	if strings.HasSuffix(reqHost, s.localDomainSuffix) {
		return reqHost, true
	}

	return "", false
}

// setCustomUpstream sets custom upstream settings in pctx, if necessary.
func (s *Server) setCustomUpstream(pctx *proxy.DNSContext, clientID string) {
	customUpsByClient := s.conf.GetCustomUpstreamByClient
	if pctx.Addr == nil || customUpsByClient == nil {
		return
	}

	// Use the ClientID first, since it has a higher priority.
	id := stringutil.Coalesce(clientID, ipStringFromAddr(pctx.Addr))
	upsConf, err := customUpsByClient(id)
	if err != nil {
		log.Error("dns: getting custom upstreams for client %s: %s", id, err)

		return
	}

	if upsConf != nil {
		log.Debug("dns: using custom upstreams for client %s", id)
	}

	pctx.CustomUpstreamConfig = upsConf
}

// Apply filtering logic after we have received response from upstream servers
func (s *Server) processFilteringAfterResponse(dctx *dnsContext) (rc resultCode) {
	pctx := dctx.proxyCtx
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

		pctx.Req.Question[0], pctx.Res.Question[0] = dctx.origQuestion, dctx.origQuestion
		if len(pctx.Res.Answer) > 0 {
			rr := s.genAnswerCNAME(pctx.Req, res.CanonName)
			answer := append([]dns.RR{rr}, pctx.Res.Answer...)
			pctx.Res.Answer = answer
		}

		return resultCodeSuccess
	default:
		return s.filterAfterResponse(dctx, pctx)
	}
}

// filterAfterResponse returns the result of filtering the response that wasn't
// explicitly allowed or rewritten.
func (s *Server) filterAfterResponse(dctx *dnsContext, pctx *proxy.DNSContext) (res resultCode) {
	// Check the response only if it's from an upstream.  Don't check the
	// response if the protection is disabled since dnsrewrite rules aren't
	// applied to it anyway.
	if !dctx.protectionEnabled || !dctx.responseFromUpstream || s.dnsFilter == nil {
		return resultCodeSuccess
	}

	result, err := s.filterDNSResponse(pctx, dctx.setts)
	if err != nil {
		dctx.err = err

		return resultCodeError
	}

	if result != nil {
		dctx.result = result
		dctx.origResp = pctx.Res
	}

	return resultCodeSuccess
}
