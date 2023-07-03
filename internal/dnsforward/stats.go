package dnsforward

import (
	"net"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/miekg/dns"
	"golang.org/x/exp/slices"
)

// Write Stats data and logs
func (s *Server) processQueryLogsAndStats(dctx *dnsContext) (rc resultCode) {
	log.Debug("dnsforward: started processing querylog and stats")
	defer log.Debug("dnsforward: finished processing querylog and stats")

	elapsed := time.Since(dctx.startTime)
	pctx := dctx.proxyCtx

	q := pctx.Req.Question[0]
	host := strings.ToLower(strings.TrimSuffix(q.Name, "."))

	ip, _ := netutil.IPAndPortFromAddr(pctx.Addr)
	ip = slices.Clone(ip)
	s.anonymizer.Load()(ip)

	log.Debug("dnsforward: client ip for stats and querylog: %s", ip)

	ipStr := ip.String()
	ids := []string{ipStr, dctx.clientID}
	qt, cl := q.Qtype, q.Qclass

	// Synchronize access to s.queryLog and s.stats so they won't be suddenly
	// uninitialized while in use.  This can happen after proxy server has been
	// stopped, but its workers haven't yet exited.
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	if s.shouldLog(host, qt, cl, ids) {
		s.logQuery(dctx, pctx, elapsed, ip)
	} else {
		log.Debug(
			"dnsforward: request %s %s %q from %s ignored; not adding to querylog",
			dns.Class(cl),
			dns.Type(qt),
			host,
			ip,
		)
	}

	if s.shouldCountStat(host, qt, cl, ids) {
		s.updateStats(dctx, elapsed, *dctx.result, ipStr)
	} else {
		log.Debug(
			"dnsforward: request %s %s %q from %s ignored; not counting in stats",
			dns.Class(cl),
			dns.Type(qt),
			host,
			ip,
		)
	}

	return resultCodeSuccess
}

// shouldLog returns true if the query with the given data should be logged in
// the query log.  s.serverLock is expected to be locked.
func (s *Server) shouldLog(host string, qt, cl uint16, ids []string) (ok bool) {
	if qt == dns.TypeANY && s.conf.RefuseAny {
		return false
	}

	// TODO(s.chzhen):  Use dnsforward.dnsContext when it will start containing
	// persistent client.
	return s.queryLog != nil && s.queryLog.ShouldLog(host, qt, cl, ids)
}

// shouldCountStat returns true if the query with the given data should be
// counted in the statistics.  s.serverLock is expected to be locked.
func (s *Server) shouldCountStat(host string, qt, cl uint16, ids []string) (ok bool) {
	// TODO(s.chzhen):  Use dnsforward.dnsContext when it will start containing
	// persistent client.
	return s.stats != nil && s.stats.ShouldCount(host, qt, cl, ids)
}

// logQuery pushes the request details into the query log.
func (s *Server) logQuery(
	dctx *dnsContext,
	pctx *proxy.DNSContext,
	elapsed time.Duration,
	ip net.IP,
) {
	p := &querylog.AddParams{
		Question:          pctx.Req,
		ReqECS:            pctx.ReqECS,
		Answer:            pctx.Res,
		OrigAnswer:        dctx.origResp,
		Result:            dctx.result,
		ClientID:          dctx.clientID,
		ClientIP:          ip,
		Elapsed:           elapsed,
		AuthenticatedData: dctx.responseAD,
	}

	switch pctx.Proto {
	case proxy.ProtoHTTPS:
		p.ClientProto = querylog.ClientProtoDoH
	case proxy.ProtoQUIC:
		p.ClientProto = querylog.ClientProtoDoQ
	case proxy.ProtoTLS:
		p.ClientProto = querylog.ClientProtoDoT
	case proxy.ProtoDNSCrypt:
		p.ClientProto = querylog.ClientProtoDNSCrypt
	default:
		// Consider this a plain DNS-over-UDP or DNS-over-TCP request.
	}

	if pctx.Upstream != nil {
		p.Upstream = pctx.Upstream.Address()
	} else if cachedUps := pctx.CachedUpstreamAddr; cachedUps != "" {
		p.Upstream = pctx.CachedUpstreamAddr
		p.Cached = true
	}

	s.queryLog.Add(p)
}

// updatesStats writes the request into statistics.
func (s *Server) updateStats(
	ctx *dnsContext,
	elapsed time.Duration,
	res filtering.Result,
	clientIP string,
) {
	pctx := ctx.proxyCtx
	e := stats.Entry{}
	e.Domain = strings.ToLower(pctx.Req.Question[0].Name)
	if e.Domain != "." {
		// Remove last ".", but save the domain as is for "." queries.
		e.Domain = e.Domain[:len(e.Domain)-1]
	}

	if clientID := ctx.clientID; clientID != "" {
		e.Client = clientID
	} else {
		e.Client = clientIP
	}

	e.Time = uint32(elapsed / 1000)
	e.Result = stats.RNotFiltered

	switch res.Reason {
	case filtering.FilteredSafeBrowsing:
		e.Result = stats.RSafeBrowsing
	case filtering.FilteredParental:
		e.Result = stats.RParental
	case filtering.FilteredSafeSearch:
		e.Result = stats.RSafeSearch
	case filtering.FilteredBlockList,
		filtering.FilteredInvalid,
		filtering.FilteredBlockedService:
		e.Result = stats.RFiltered
	}

	s.stats.Update(e)
}
