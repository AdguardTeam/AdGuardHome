package dnsforward

import (
	"net"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/util"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// To transfer information between modules
type dnsContext struct {
	srv                  *Server
	proxyCtx             *proxy.DNSContext
	setts                *dnsfilter.RequestFilteringSettings // filtering settings for this client
	startTime            time.Time
	result               *dnsfilter.Result
	origResp             *dns.Msg     // response received from upstream servers.  Set when response is modified by filtering
	origQuestion         dns.Question // question received from client.  Set when Rewrites are used.
	err                  error        // error returned from the module
	protectionEnabled    bool         // filtering is enabled, dnsfilter object is ready
	responseFromUpstream bool         // response is received from upstream servers
	origReqDNSSEC        bool         // DNSSEC flag in the original request from user
}

const (
	resultDone   = iota // module has completed its job, continue
	resultFinish        // module has completed its job, exit normally
	resultError         // an error occurred, exit with an error
)

// handleDNSRequest filters the incoming DNS requests and writes them to the query log
func (s *Server) handleDNSRequest(_ *proxy.Proxy, d *proxy.DNSContext) error {
	ctx := &dnsContext{srv: s, proxyCtx: d}
	ctx.result = &dnsfilter.Result{}
	ctx.startTime = time.Now()

	type modProcessFunc func(ctx *dnsContext) int
	mods := []modProcessFunc{
		processInitial,
		processInternalHosts,
		processInternalIPAddrs,
		processFilteringBeforeRequest,
		processUpstream,
		processDNSSECAfterResponse,
		processFilteringAfterResponse,
		s.ipset.process,
		processQueryLogsAndStats,
	}
	for _, process := range mods {
		r := process(ctx)
		switch r {
		case resultDone:
			// continue: call the next filter

		case resultFinish:
			return nil

		case resultError:
			return ctx.err
		}
	}

	if d.Res != nil {
		d.Res.Compress = true // some devices require DNS message compression
	}
	return nil
}

// Perform initial checks;  process WHOIS & rDNS
func processInitial(ctx *dnsContext) int {
	s := ctx.srv
	d := ctx.proxyCtx
	if s.conf.AAAADisabled && d.Req.Question[0].Qtype == dns.TypeAAAA {
		_ = proxy.CheckDisabledAAAARequest(d, true)
		return resultFinish
	}

	if s.conf.OnDNSRequest != nil {
		s.conf.OnDNSRequest(d)
	}

	// disable Mozilla DoH
	// https://support.mozilla.org/en-US/kb/canary-domain-use-application-dnsnet
	if (d.Req.Question[0].Qtype == dns.TypeA || d.Req.Question[0].Qtype == dns.TypeAAAA) &&
		d.Req.Question[0].Name == "use-application-dns.net." {
		d.Res = s.genNXDomain(d.Req)
		return resultFinish
	}

	return resultDone
}

// Return TRUE if host names doesn't contain disallowed characters
func isHostnameOK(hostname string) bool {
	for _, c := range hostname {
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '.' || c == '-') {
			log.Debug("DNS: skipping invalid hostname %s from DHCP", hostname)
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

	log.Debug("DNS: added %d A/PTR entries from DHCP", len(m))

	s.tableHostToIPLock.Lock()
	s.tableHostToIP = hostToIP
	s.tableHostToIPLock.Unlock()

	s.tablePTRLock.Lock()
	s.tablePTR = m
	s.tablePTRLock.Unlock()
}

// Respond to A requests if the target host name is associated with a lease from our DHCP server
func processInternalHosts(ctx *dnsContext) int {
	s := ctx.srv
	req := ctx.proxyCtx.Req
	if !(req.Question[0].Qtype == dns.TypeA || req.Question[0].Qtype == dns.TypeAAAA) {
		return resultDone
	}

	host := req.Question[0].Name
	host = strings.ToLower(host)
	if !strings.HasSuffix(host, ".lan.") {
		return resultDone
	}
	host = strings.TrimSuffix(host, ".lan.")

	s.tableHostToIPLock.Lock()
	if s.tableHostToIP == nil {
		s.tableHostToIPLock.Unlock()
		return resultDone
	}
	ip, ok := s.tableHostToIP[host]
	s.tableHostToIPLock.Unlock()
	if !ok {
		return resultDone
	}

	log.Debug("DNS: internal record: %s -> %s", req.Question[0].Name, ip.String())

	resp := s.makeResponse(req)

	if req.Question[0].Qtype == dns.TypeA {
		a := &dns.A{}
		a.Hdr = dns.RR_Header{
			Name:   req.Question[0].Name,
			Rrtype: dns.TypeA,
			Ttl:    s.conf.BlockedResponseTTL,
			Class:  dns.ClassINET,
		}
		a.A = make([]byte, 4)
		copy(a.A, ip)
		resp.Answer = append(resp.Answer, a)
	}

	ctx.proxyCtx.Res = resp
	return resultDone
}

// Respond to PTR requests if the target IP address is leased by our DHCP server
func processInternalIPAddrs(ctx *dnsContext) int {
	s := ctx.srv
	req := ctx.proxyCtx.Req
	if req.Question[0].Qtype != dns.TypePTR {
		return resultDone
	}

	arpa := req.Question[0].Name
	arpa = strings.TrimSuffix(arpa, ".")
	arpa = strings.ToLower(arpa)
	ip := util.DNSUnreverseAddr(arpa)
	if ip == nil {
		return resultDone
	}

	s.tablePTRLock.Lock()
	if s.tablePTR == nil {
		s.tablePTRLock.Unlock()
		return resultDone
	}
	host, ok := s.tablePTR[ip.String()]
	s.tablePTRLock.Unlock()
	if !ok {
		return resultDone
	}

	log.Debug("DNS: reverse-lookup: %s -> %s", arpa, host)

	resp := s.makeResponse(req)
	ptr := &dns.PTR{}
	ptr.Hdr = dns.RR_Header{
		Name:   req.Question[0].Name,
		Rrtype: dns.TypePTR,
		Ttl:    s.conf.BlockedResponseTTL,
		Class:  dns.ClassINET,
	}
	ptr.Ptr = host + "."
	resp.Answer = append(resp.Answer, ptr)
	ctx.proxyCtx.Res = resp
	return resultDone
}

// Apply filtering logic
func processFilteringBeforeRequest(ctx *dnsContext) int {
	s := ctx.srv
	d := ctx.proxyCtx

	if d.Res != nil {
		return resultDone // response is already set - nothing to do
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
		ctx.setts = s.getClientRequestFilteringSettings(d)
		ctx.result, err = s.filterDNSRequest(ctx)
	}
	s.RUnlock()

	if err != nil {
		ctx.err = err
		return resultError
	}
	return resultDone
}

// Pass request to upstream servers;  process the response
func processUpstream(ctx *dnsContext) int {
	s := ctx.srv
	d := ctx.proxyCtx
	if d.Res != nil {
		return resultDone // response is already set - nothing to do
	}

	if d.Addr != nil && s.conf.GetCustomUpstreamByClient != nil {
		clientIP := ipFromAddr(d.Addr)
		upstreamsConf := s.conf.GetCustomUpstreamByClient(clientIP)
		if upstreamsConf != nil {
			log.Debug("Using custom upstreams for %s", clientIP)
			d.CustomUpstreamConfig = upstreamsConf
		}
	}

	if s.conf.EnableDNSSEC {
		opt := d.Req.IsEdns0()
		if opt == nil {
			log.Debug("DNS: Adding OPT record with DNSSEC flag")
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
		return resultError
	}

	ctx.responseFromUpstream = true
	return resultDone
}

// Process DNSSEC after response from upstream server
func processDNSSECAfterResponse(ctx *dnsContext) int {
	d := ctx.proxyCtx

	if !ctx.responseFromUpstream || // don't process response if it's not from upstream servers
		!ctx.srv.conf.EnableDNSSEC {
		return resultDone
	}

	if !ctx.origReqDNSSEC {
		optResp := d.Res.IsEdns0()
		if optResp != nil && !optResp.Do() {
			return resultDone
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

	return resultDone
}

// Apply filtering logic after we have received response from upstream servers
func processFilteringAfterResponse(ctx *dnsContext) int {
	s := ctx.srv
	d := ctx.proxyCtx
	res := ctx.result
	var err error

	switch res.Reason {
	case dnsfilter.ReasonRewrite:
		if len(ctx.origQuestion.Name) == 0 {
			// origQuestion is set in case we get only CNAME without IP from rewrites table
			break
		}

		d.Req.Question[0] = ctx.origQuestion
		d.Res.Question[0] = ctx.origQuestion

		if len(d.Res.Answer) != 0 {
			answer := []dns.RR{}
			answer = append(answer, s.genCNAMEAnswer(d.Req, res.CanonName))
			answer = append(answer, d.Res.Answer...) // host -> IP
			d.Res.Answer = answer
		}

	case dnsfilter.NotFilteredWhiteList:
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
			return resultError
		}
		if ctx.result != nil {
			ctx.origResp = origResp2 // matched by response
		} else {
			ctx.result = &dnsfilter.Result{}
		}
	}

	return resultDone
}
