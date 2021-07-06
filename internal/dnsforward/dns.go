package dnsforward

import (
	"net"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghstrings"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
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
	setts     *filtering.Settings
	startTime time.Time
	result    *filtering.Result
	// origResp is the response received from upstream.  It is set when the
	// response is modified by filters.
	origResp *dns.Msg
	// unreversedReqIP stores an IP address obtained from PTR request if it
	// was successfully parsed.
	unreversedReqIP net.IP
	// err is the error returned from a processing function.
	err error
	// clientID is the clientID from DoH, DoQ, or DoT, if provided.
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

// handleDNSRequest filters the incoming DNS requests and writes them to the query log
func (s *Server) handleDNSRequest(_ *proxy.Proxy, d *proxy.DNSContext) error {
	ctx := &dnsContext{
		srv:       s,
		proxyCtx:  d,
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
		processInitial,
		s.processDetermineLocal,
		s.processInternalHosts,
		s.processRestrictLocal,
		s.processInternalIPAddrs,
		s.processClientID,
		processFilteringBeforeRequest,
		s.processLocalPTR,
		s.processUpstream,
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

func (s *Server) setTableHostToIP(t hostToIPTable) {
	s.tableHostToIPLock.Lock()
	defer s.tableHostToIPLock.Unlock()

	s.tableHostToIP = t
}

func (s *Server) setTableIPToHost(t *aghnet.IPMap) {
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
	var ipToHost *aghnet.IPMap
	if add {
		ll := s.dhcpServer.Leases(dhcpd.LeasesAll)

		hostToIP = make(hostToIPTable, len(ll))
		ipToHost = aghnet.NewIPMap(len(ll))

		for _, l := range ll {
			// TODO(a.garipov): Remove this after we're finished
			// with the client hostname validations in the DHCP
			// server code.
			err = aghnet.ValidateDomainName(l.Hostname)
			if err != nil {
				log.Debug(
					"dns: skipping invalid hostname %q from dhcp: %s",
					l.Hostname,
					err,
				)
			}

			lowhost := strings.ToLower(l.Hostname)

			ipToHost.Set(l.IP, lowhost)

			ip := make(net.IP, 4)
			copy(ip, l.IP.To4())
			hostToIP[lowhost] = ip
		}

		log.Debug("dns: added %d A/PTR entries from DHCP", ipToHost.Len())
	}

	s.setTableHostToIP(hostToIP)
	s.setTableIPToHost(ipToHost)
}

// processDetermineLocal determines if the client's IP address is from
// locally-served network and saves the result into the context.
func (s *Server) processDetermineLocal(dctx *dnsContext) (rc resultCode) {
	rc = resultCodeSuccess

	var ip net.IP
	if ip = aghnet.IPFromAddr(dctx.proxyCtx.Addr); ip == nil {
		return rc
	}

	dctx.isLocalClient = s.subnetDetector.IsLocallyServedNetwork(ip)

	return rc
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
	if !s.dhcpServer.Enabled() {
		return resultCodeSuccess
	}

	req := dctx.proxyCtx.Req
	q := req.Question[0]

	// Go on processing the AAAA request despite the fact that we don't
	// support it yet.  The expected behavior here is to respond with an
	// empty asnwer and not NXDOMAIN.
	if q.Qtype != dns.TypeA && q.Qtype != dns.TypeAAAA {
		return resultCodeSuccess
	}

	reqHost := strings.ToLower(q.Name)
	host := strings.TrimSuffix(reqHost, s.localDomainSuffix)
	if host == reqHost {
		return resultCodeSuccess
	}

	d := dctx.proxyCtx
	if !dctx.isLocalClient {
		log.Debug("dns: %q requests for internal host", d.Addr)
		d.Res = s.genNXDomain(req)

		// Do not even put into query log.
		return resultCodeFinish
	}

	ip, ok := s.hostToIP(host)
	if !ok {
		// TODO(e.burkov): Inspect special cases when user want to apply
		// some rules handled by other processors to the hosts with TLD.
		d.Res = s.genNXDomain(req)

		return resultCodeFinish
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

// processRestrictLocal responds with NXDOMAIN to PTR requests for IP addresses
// in locally-served network from external clients.
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
		if !ctx.isLocalClient {
			log.Debug("dns: %q requests for internal ip", d.Addr)
			d.Res = s.genNXDomain(req)

			// Do not even put into query log.
			return resultCodeFinish
		}
	}

	// Do not perform unreversing ever again.
	ctx.unreversedReqIP = ip

	// Disable redundant filtering.
	filterSetts := s.getClientRequestFilteringSettings(ctx)
	filterSetts.ParentalEnabled = false
	filterSetts.SafeBrowsingEnabled = false
	filterSetts.SafeSearchEnabled = false
	filterSetts.ServicesRules = nil
	ctx.setts = filterSetts

	// Nothing to restrict.
	return resultCodeSuccess
}

// ipToHost tries to get a hostname leased by DHCP.  It's safe for concurrent
// use.
func (s *Server) ipToHost(ip net.IP) (host string, ok bool) {
	s.tableIPToHostLock.Lock()
	defer s.tableIPToHostLock.Unlock()

	if s.tableIPToHost == nil {
		return "", false
	}

	var v interface{}
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

	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	if !s.subnetDetector.IsLocallyServedNetwork(ip) {
		return resultCodeSuccess
	}

	if s.conf.UsePrivateRDNS {
		s.recDetector.add(*d.Req)
		if err := s.localResolvers.Resolve(d); err != nil {
			ctx.err = err

			return resultCodeError
		}
	}

	if d.Res == nil {
		d.Res = s.genNXDomain(d.Req)

		// Do not even put into query log.
		return resultCodeFinish
	}

	return resultCodeSuccess
}

// Apply filtering logic
func processFilteringBeforeRequest(ctx *dnsContext) (rc resultCode) {
	s := ctx.srv
	d := ctx.proxyCtx

	if d.Res != nil {
		return resultCodeSuccess // response is already set - nothing to do
	}

	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	ctx.protectionEnabled = s.conf.ProtectionEnabled && s.dnsFilter != nil
	if !ctx.protectionEnabled {
		return resultCodeSuccess
	}

	if ctx.setts == nil {
		ctx.setts = s.getClientRequestFilteringSettings(ctx)
	}

	var err error
	ctx.result, err = s.filterDNSRequest(ctx)
	if err != nil {
		ctx.err = err

		return resultCodeError
	}

	return resultCodeSuccess
}

// ipStringFromAddr extracts an IP address string from net.Addr.
func ipStringFromAddr(addr net.Addr) (ipStr string) {
	if ip := aghnet.IPFromAddr(addr); ip != nil {
		return ip.String()
	}

	return ""
}

// processUpstream passes request to upstream servers and handles the response.
func (s *Server) processUpstream(ctx *dnsContext) (rc resultCode) {
	d := ctx.proxyCtx
	if d.Res != nil {
		return resultCodeSuccess // response is already set - nothing to do
	}

	if d.Addr != nil && s.conf.GetCustomUpstreamByClient != nil {
		// Use the clientID first, since it has a higher priority.
		id := aghstrings.Coalesce(ctx.clientID, ipStringFromAddr(d.Addr))
		upsConf, err := s.conf.GetCustomUpstreamByClient(id)
		if err != nil {
			log.Error("dns: getting custom upstreams for client %s: %s", id, err)
		} else if upsConf != nil {
			log.Debug("dns: using custom upstreams for client %s", id)
			d.CustomUpstreamConfig = upsConf
		}
	}

	req := d.Req
	if s.conf.EnableDNSSEC {
		opt := req.IsEdns0()
		if opt == nil {
			log.Debug("dns: adding OPT record with DNSSEC flag")
			req.SetEdns0(4096, true)
		} else if !opt.Do() {
			opt.SetDo(true)
		} else {
			ctx.origReqDNSSEC = true
		}
	}

	// request was not filtered so let it be processed further
	if ctx.err = s.dnsProxy.Resolve(d); ctx.err != nil {
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
	case filtering.Rewritten,
		filtering.RewrittenRule:

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

	case filtering.NotFilteredAllowList:
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
			ctx.result = &filtering.Result{}
		}
	}

	return resultCodeSuccess
}
