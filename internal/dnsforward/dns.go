package dnsforward

import (
	"crypto/tls"
	"fmt"
	"net"
	"path"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/internal/util"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/lucas-clemente/quic-go"
	"github.com/miekg/dns"
)

// To transfer information between modules
type dnsContext struct {
	srv      *Server
	proxyCtx *proxy.DNSContext
	// setts are the filtering settings for the client.
	setts     *dnsfilter.RequestFilteringSettings
	startTime time.Time
	result    *dnsfilter.Result
	// origResp is the response received from upstream.  It is set when the
	// response is modified by filters.
	origResp *dns.Msg
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
		processInternalHosts,
		processInternalIPAddrs,
		processClientID,
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
func processInternalHosts(ctx *dnsContext) (rc resultCode) {
	s := ctx.srv
	req := ctx.proxyCtx.Req
	if !(req.Question[0].Qtype == dns.TypeA || req.Question[0].Qtype == dns.TypeAAAA) {
		return resultCodeSuccess
	}

	host := req.Question[0].Name
	host = strings.ToLower(host)
	if !strings.HasSuffix(host, ".lan.") {
		return resultCodeSuccess
	}
	host = strings.TrimSuffix(host, ".lan.")

	s.tableHostToIPLock.Lock()
	if s.tableHostToIP == nil {
		s.tableHostToIPLock.Unlock()
		return resultCodeSuccess
	}
	ip, ok := s.tableHostToIP[host]
	s.tableHostToIPLock.Unlock()
	if !ok {
		return resultCodeSuccess
	}

	log.Debug("DNS: internal record: %s -> %s", req.Question[0].Name, ip)

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
	return resultCodeSuccess
}

const maxDomainPartLen = 64

// ValidateClientID returns an error if clientID is not a valid client ID.
func ValidateClientID(clientID string) (err error) {
	if len(clientID) > maxDomainPartLen {
		return fmt.Errorf("client id %q is too long, max: %d", clientID, maxDomainPartLen)
	}

	for i, r := range clientID {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}

		return fmt.Errorf("invalid char %q at index %d in client id %q", r, i, clientID)
	}

	return nil
}

// clientIDFromClientServerName extracts and validates a client ID.  hostSrvName
// is the server name of the host.  cliSrvName is the server name as sent by the
// client.
func clientIDFromClientServerName(hostSrvName, cliSrvName string) (clientID string, err error) {
	if hostSrvName == cliSrvName {
		return "", nil
	}

	if !strings.HasSuffix(cliSrvName, hostSrvName) {
		return "", fmt.Errorf("client server name %q doesn't match host server name %q", cliSrvName, hostSrvName)
	}

	clientID = cliSrvName[:len(cliSrvName)-len(hostSrvName)-1]
	err = ValidateClientID(clientID)
	if err != nil {
		return "", fmt.Errorf("invalid client id: %w", err)
	}

	return clientID, nil
}

// processClientIDHTTPS extracts the client's ID from the path of the
// client's DNS-over-HTTPS request.
func processClientIDHTTPS(ctx *dnsContext) (rc resultCode) {
	pctx := ctx.proxyCtx
	r := pctx.HTTPRequest
	if r == nil {
		ctx.err = fmt.Errorf("proxy ctx http request of proto %s is nil", pctx.Proto)

		return resultCodeError
	}

	origPath := r.URL.Path
	parts := strings.Split(path.Clean(origPath), "/")
	if parts[0] == "" {
		parts = parts[1:]
	}

	if len(parts) == 0 || parts[0] != "dns-query" {
		ctx.err = fmt.Errorf("client id check: invalid path %q", origPath)

		return resultCodeError
	}

	clientID := ""
	switch len(parts) {
	case 1:
		// Just /dns-query, no client ID.
		return resultCodeSuccess
	case 2:
		clientID = parts[1]
	default:
		ctx.err = fmt.Errorf("client id check: invalid path %q: extra parts", origPath)

		return resultCodeError
	}

	err := ValidateClientID(clientID)
	if err != nil {
		ctx.err = fmt.Errorf("client id check: invalid client id: %w", err)

		return resultCodeError
	}

	ctx.clientID = clientID

	return resultCodeSuccess
}

// tlsConn is a narrow interface for *tls.Conn to simplify testing.
type tlsConn interface {
	ConnectionState() (cs tls.ConnectionState)
}

// quicSession is a narrow interface for quic.Session to simplify testing.
type quicSession interface {
	ConnectionState() (cs quic.ConnectionState)
}

// processClientID extracts the client's ID from the server name of the client's
// DOT or DOQ request or the path of the client's DOH.
func processClientID(ctx *dnsContext) (rc resultCode) {
	pctx := ctx.proxyCtx
	proto := pctx.Proto
	if proto == proxy.ProtoHTTPS {
		return processClientIDHTTPS(ctx)
	} else if proto != proxy.ProtoTLS && proto != proxy.ProtoQUIC {
		return resultCodeSuccess
	}

	hostSrvName := ctx.srv.conf.TLSConfig.ServerName
	if hostSrvName == "" {
		return resultCodeSuccess
	}

	cliSrvName := ""
	if proto == proxy.ProtoTLS {
		conn := pctx.Conn
		tc, ok := conn.(tlsConn)
		if !ok {
			ctx.err = fmt.Errorf("proxy ctx conn of proto %s is %T, want *tls.Conn", proto, conn)

			return resultCodeError
		}

		cliSrvName = tc.ConnectionState().ServerName
	} else if proto == proxy.ProtoQUIC {
		qs, ok := pctx.QUICSession.(quicSession)
		if !ok {
			ctx.err = fmt.Errorf("proxy ctx quic session of proto %s is %T, want quic.Session", proto, pctx.QUICSession)

			return resultCodeError
		}

		cliSrvName = qs.ConnectionState().ServerName
	}

	clientID, err := clientIDFromClientServerName(hostSrvName, cliSrvName)
	if err != nil {
		ctx.err = fmt.Errorf("client id check: %w", err)

		return resultCodeError
	}

	ctx.clientID = clientID

	return resultCodeSuccess
}

// Respond to PTR requests if the target IP address is leased by our DHCP server
func processInternalIPAddrs(ctx *dnsContext) (rc resultCode) {
	s := ctx.srv
	req := ctx.proxyCtx.Req
	if req.Question[0].Qtype != dns.TypePTR {
		return resultCodeSuccess
	}

	arpa := req.Question[0].Name
	arpa = strings.TrimSuffix(arpa, ".")
	arpa = strings.ToLower(arpa)
	ip := util.DNSUnreverseAddr(arpa)
	if ip == nil {
		return resultCodeSuccess
	}

	s.tablePTRLock.Lock()
	if s.tablePTR == nil {
		s.tablePTRLock.Unlock()
		return resultCodeSuccess
	}
	host, ok := s.tablePTR[ip.String()]
	s.tablePTRLock.Unlock()
	if !ok {
		return resultCodeSuccess
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
