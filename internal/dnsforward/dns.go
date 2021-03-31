package dnsforward

import (
	"errors"
	"net"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// To transfer information between modules
type dnsContext struct {
	// TODO(a.garipov): Remove this and rewrite processors to be methods of
	// *Server instead.
	srv      *Server
	proxyCtx *proxy.DNSContext
	// setts are the filtering settings for the client.
	setts     *dnsfilter.FilteringSettings
	startTime time.Time
	result    *dnsfilter.Result
	// origResp is the response received from upstream.  It is set when the
	// response is modified by filters.
	origResp *dns.Msg
	// unreversedReqIP stores an IP address obtained from PTR request if it
	// was successfully parsed.
	unreversedReqIP net.IP
	// err is the error returned from a processing function.
	err error
	// clientID is the clientID from DOH, DOQ, or DOT, if provided.
	clientID string
	// origQuestion is the question received from the client.  It is set
	// when the request is modified by rewrites.
	origQuestion dns.Question
	// protectionEnabled shows if the filtering is enabled, and if the
	// server's DNS filter is ready.
	protectionEnabled bool
	// responseFromUpstream shows if the response is received from the
	// upstream servers.
	responseFromUpstream bool
	// origReqDNSSEC shows if the DNSSEC flag in the original request from
	// the client is set.
	origReqDNSSEC bool
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

// handleDNSRequest filters the incoming DNS requests and writes them to the query log
func (s *Server) handleDNSRequest(_ *proxy.Proxy, d *proxy.DNSContext) error {
	ctx := &dnsContext{
		srv:       s,
		proxyCtx:  d,
		result:    &dnsfilter.Result{},
		startTime: time.Now(),
	}

	type modProcessFunc func(ctx *dnsContext) (rc resultCode)

	// Since (*dnsforward.Server).handleDNSRequest(...) is used as
	// proxy.(Config).RequestHandler, there is no need for additional index
	// out of range checking in any of the following functions, because the
	// (*proxy.Proxy).handleDNSRequest method performs it before calling the
	// appropriate handler.
	mods := []modProcessFunc{
		processInitial,
		s.processInternalHosts,
		s.processRestrictLocal,
		s.processInternalIPAddrs,
		processClientID,
		processFilteringBeforeRequest,
		s.processLocalPTR,
		processUpstream,
		processDNSSECAfterResponse,
		processFilteringAfterResponse,
		s.ipset.process,
		processQueryLogsAndStats,
	}
	for _, process := range mods {
		r := process(ctx)
		switch r {
		case resultCodeSuccess:
			// continue: call the next filter

		case resultCodeFinish:
			return nil

		case resultCodeError:
			return ctx.err
		}
	}

	if d.Res != nil {
		d.Res.Compress = true // some devices require DNS message compression
	}
	return nil
}

// Perform initial checks;  process WHOIS & rDNS
func processInitial(ctx *dnsContext) (rc resultCode) {
	s := ctx.srv
	d := ctx.proxyCtx
	if s.conf.AAAADisabled && d.Req.Question[0].Qtype == dns.TypeAAAA {
		_ = proxy.CheckDisabledAAAARequest(d, true)
		return resultCodeFinish
	}

	if s.conf.OnDNSRequest != nil {
		s.conf.OnDNSRequest(d)
	}

	// disable Mozilla DoH
	// https://support.mozilla.org/en-US/kb/canary-domain-use-application-dnsnet
	if (d.Req.Question[0].Qtype == dns.TypeA || d.Req.Question[0].Qtype == dns.TypeAAAA) &&
		d.Req.Question[0].Name == "use-application-dns.net." {
		d.Res = s.genNXDomain(d.Req)
		return resultCodeFinish
	}

	return resultCodeSuccess
}

// Return TRUE if host names doesn't contain disallowed characters
func isHostnameOK(hostname string) bool {
	for _, c := range hostname {
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '.' || c == '-') {
			log.Debug("dns: skipping invalid hostname %s from DHCP", hostname)
			return false
		}
	}
	return true
}

func (s *Server) onDHCPLeaseChanged(flags int) {
	switch flags {
	case dhcpd.LeaseChangedAdded,
		dhcpd.LeaseChangedAddedStatic,
		dhcpd.LeaseChangedRemovedStatic:
		//
	default:
		return
	}

	hostToIP := make(map[string]net.IP)
	m := make(map[string]string)

	ll := s.dhcpServer.Leases(dhcpd.LeasesAll)

	for _, l := range ll {
		if len(l.Hostname) == 0 || !isHostnameOK(l.Hostname) {
			continue
		}

		lowhost := strings.ToLower(l.Hostname)

		m[l.IP.String()] = lowhost

		ip := make(net.IP, 4)
		copy(ip, l.IP.To4())
		hostToIP[lowhost] = ip
	}

	log.Debug("dns: added %d A/PTR entries from DHCP", len(m))

	s.tableHostToIPLock.Lock()
	s.tableHostToIP = hostToIP
	s.tableHostToIPLock.Unlock()

	s.tablePTRLock.Lock()
	s.tablePTR = m
	s.tablePTRLock.Unlock()
}

// hostToIP tries to get an IP leased by DHCP and returns the copy of address
// since the data inside the internal table may be changed while request
// processing.  It's safe for concurrent use.
func (s *Server) hostToIP(host string) (ip net.IP, ok bool) {
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

// processInternalHosts respond to A requests if the target hostname is known to
// the server.
//
// TODO(a.garipov): Adapt to AAAA as well.
func (s *Server) processInternalHosts(dctx *dnsContext) (rc resultCode) {
	req := dctx.proxyCtx.Req
	q := req.Question[0]

	// Go on processing the AAAA request despite the fact that we don't
	// support it yet.  The expected behavior here is to respond with an
	// empty asnwer and not NXDOMAIN.
	if q.Qtype != dns.TypeA && q.Qtype != dns.TypeAAAA {
		return resultCodeSuccess
	}

	reqHost := strings.ToLower(q.Name)
	host := strings.TrimSuffix(reqHost, s.autohostSuffix)
	if host == reqHost {
		return resultCodeSuccess
	}

	// TODO(e.burkov): Restrict the access for external clients.

	ip, ok := s.hostToIP(host)
	if !ok {
		return resultCodeSuccess
	}

	log.Debug("dns: internal record: %s -> %s", q.Name, ip)

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

// processRestrictLocal responds with empty answers to PTR requests for IP
// addresses in locally-served network from external clients.
func (s *Server) processRestrictLocal(ctx *dnsContext) (rc resultCode) {
	d := ctx.proxyCtx
	req := d.Req
	q := req.Question[0]
	if q.Qtype != dns.TypePTR {
		// No need for restriction.
		return resultCodeSuccess
	}

	ip := aghnet.UnreverseAddr(q.Name)
	if ip == nil {
		// That's weird.
		//
		// TODO(e.burkov): Research the cases when it could happen.
		return resultCodeSuccess
	}

	// Restrict an access to local addresses for external clients.  We also
	// assume that all the DHCP leases we give are locally-served or at
	// least don't need to be unaccessable externally.
	if s.subnetDetector.IsLocallyServedNetwork(ip) {
		clientIP := IPFromAddr(d.Addr)
		if !s.subnetDetector.IsLocallyServedNetwork(clientIP) {
			log.Debug("dns: %q requests for internal ip", clientIP)
			d.Res = s.makeResponse(req)

			// Do not even put into query log.
			return resultCodeFinish
		}
	}

	// Do not perform unreversing ever again.
	ctx.unreversedReqIP = ip

	// Nothing to restrict.
	return resultCodeSuccess
}

// ipToHost tries to get a hostname leased by DHCP.  It's safe for concurrent
// use.
func (s *Server) ipToHost(ip net.IP) (host string, ok bool) {
	s.tablePTRLock.Lock()
	defer s.tablePTRLock.Unlock()

	if s.tablePTR == nil {
		return "", false
	}

	host, ok = s.tablePTR[ip.String()]

	return host, ok
}

// Respond to PTR requests if the target IP is leased by our DHCP server and the
// requestor is inside the local network.
func (s *Server) processInternalIPAddrs(ctx *dnsContext) (rc resultCode) {
	d := ctx.proxyCtx
	if d.Res != nil {
		return resultCodeSuccess
	}

	ip := ctx.unreversedReqIP
	if ip == nil {
		return resultCodeSuccess
	}

	host, ok := s.ipToHost(ip)
	if !ok {
		return resultCodeSuccess
	}

	log.Debug("dns: reverse-lookup: %s -> %s", ip, host)

	req := d.Req
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
	d.Res = resp

	return resultCodeSuccess
}

// processLocalPTR responds to PTR requests if the target IP is detected to be
// inside the local network and the query was not answered from DHCP.
func (s *Server) processLocalPTR(ctx *dnsContext) (rc resultCode) {
	d := ctx.proxyCtx
	if d.Res != nil {
		return resultCodeSuccess
	}

	ip := ctx.unreversedReqIP
	if ip == nil {
		return resultCodeSuccess
	}

	if !s.subnetDetector.IsLocallyServedNetwork(ip) {
		return resultCodeSuccess
	}

	req := d.Req
	resp, err := s.localResolvers.Exchange(req)
	if err != nil {
		if errors.Is(err, aghnet.NoUpstreamsErr) {
			d.Res = s.genNXDomain(req)

			return resultCodeFinish
		}

		ctx.err = err

		return resultCodeError
	}

	d.Res = resp

	return resultCodeSuccess
}

// Apply filtering logic
func processFilteringBeforeRequest(ctx *dnsContext) (rc resultCode) {
	s := ctx.srv
	d := ctx.proxyCtx

	if d.Res != nil {
		return resultCodeSuccess // response is already set - nothing to do
	}

	s.RLock()
	// Synchronize access to s.dnsFilter so it won't be suddenly uninitialized while in use.
	// This could happen after proxy server has been stopped, but its workers are not yet exited.
	//
	// A better approach is for proxy.Stop() to wait until all its workers exit,
	//  but this would require the Upstream interface to have Close() function
	//  (to prevent from hanging while waiting for unresponsive DNS server to respond).

	var err error
	ctx.protectionEnabled = s.conf.ProtectionEnabled && s.dnsFilter != nil
	if ctx.protectionEnabled {
		ctx.setts = s.getClientRequestFilteringSettings(ctx)
		ctx.result, err = s.filterDNSRequest(ctx)
	}
	s.RUnlock()

	if err != nil {
		ctx.err = err
		return resultCodeError
	}
	return resultCodeSuccess
}

// processUpstream passes request to upstream servers and handles the response.
func processUpstream(ctx *dnsContext) (rc resultCode) {
	s := ctx.srv
	d := ctx.proxyCtx
	if d.Res != nil {
		return resultCodeSuccess // response is already set - nothing to do
	}

	if d.Addr != nil && s.conf.GetCustomUpstreamByClient != nil {
		clientIP := IPStringFromAddr(d.Addr)
		upstreamsConf := s.conf.GetCustomUpstreamByClient(clientIP)
		if upstreamsConf != nil {
			log.Debug("Using custom upstreams for %s", clientIP)
			d.CustomUpstreamConfig = upstreamsConf
		}
	}

	if s.conf.EnableDNSSEC {
		opt := d.Req.IsEdns0()
		if opt == nil {
			log.Debug("dns: Adding OPT record with DNSSEC flag")
			d.Req.SetEdns0(4096, true)
		} else if !opt.Do() {
			opt.SetDo(true)
		} else {
			ctx.origReqDNSSEC = true
		}
	}

	// request was not filtered so let it be processed further
	err := s.dnsProxy.Resolve(d)
	if err != nil {
		ctx.err = err
		return resultCodeError
	}

	ctx.responseFromUpstream = true
	return resultCodeSuccess
}

// Process DNSSEC after response from upstream server
func processDNSSECAfterResponse(ctx *dnsContext) (rc resultCode) {
	d := ctx.proxyCtx

	if !ctx.responseFromUpstream || // don't process response if it's not from upstream servers
		!ctx.srv.conf.EnableDNSSEC {
		return resultCodeSuccess
	}

	if !ctx.origReqDNSSEC {
		optResp := d.Res.IsEdns0()
		if optResp != nil && !optResp.Do() {
			return resultCodeSuccess
		}

		// Remove RRSIG records from response
		// because there is no DO flag in the original request from client,
		// but we have EnableDNSSEC set, so we have set DO flag ourselves,
		// and now we have to clean up the DNS records our client didn't ask for.

		answers := []dns.RR{}
		for _, a := range d.Res.Answer {
			switch a.(type) {
			case *dns.RRSIG:
				log.Debug("Removing RRSIG record from response: %v", a)
			default:
				answers = append(answers, a)
			}
		}
		d.Res.Answer = answers

		answers = []dns.RR{}
		for _, a := range d.Res.Ns {
			switch a.(type) {
			case *dns.RRSIG:
				log.Debug("Removing RRSIG record from response: %v", a)
			default:
				answers = append(answers, a)
			}
		}
		d.Res.Ns = answers
	}

	return resultCodeSuccess
}

// Apply filtering logic after we have received response from upstream servers
func processFilteringAfterResponse(ctx *dnsContext) (rc resultCode) {
	s := ctx.srv
	d := ctx.proxyCtx
	res := ctx.result
	var err error

	switch res.Reason {
	case dnsfilter.Rewritten,
		dnsfilter.RewrittenRule:

		if len(ctx.origQuestion.Name) == 0 {
			// origQuestion is set in case we get only CNAME without IP from rewrites table
			break
		}

		d.Req.Question[0] = ctx.origQuestion
		d.Res.Question[0] = ctx.origQuestion

		if len(d.Res.Answer) != 0 {
			answer := []dns.RR{}
			answer = append(answer, s.genAnswerCNAME(d.Req, res.CanonName))
			answer = append(answer, d.Res.Answer...)
			d.Res.Answer = answer
		}

	case dnsfilter.NotFilteredAllowList:
		// nothing

	default:
		if !ctx.protectionEnabled || // filters are disabled: there's nothing to check for
			!ctx.responseFromUpstream { // only check response if it's from an upstream server
			break
		}
		origResp2 := d.Res
		ctx.result, err = s.filterDNSResponse(ctx)
		if err != nil {
			ctx.err = err
			return resultCodeError
		}
		if ctx.result != nil {
			ctx.origResp = origResp2 // matched by response
		} else {
			ctx.result = &dnsfilter.Result{}
		}
	}

	return resultCodeSuccess
}
