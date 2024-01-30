package dnsforward

import (
	"net"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// Write Stats data and logs
func (s *Server) processQueryLogsAndStats(dctx *dnsContext) (rc resultCode) {
	log.Debug("dnsforward: started processing querylog and stats")
	defer log.Debug("dnsforward: finished processing querylog and stats")

	pctx := dctx.proxyCtx
	q := pctx.Req.Question[0]
	host := aghnet.NormalizeDomain(q.Name)
	processingTime := time.Since(dctx.startTime)

	ip := pctx.Addr.Addr().AsSlice()
	s.anonymizer.Load()(ip)
	ipStr := net.IP(ip).String()

	log.Debug("dnsforward: client ip for stats and querylog: %s", ipStr)

	ids := []string{ipStr, dctx.clientID}
	qt, cl := q.Qtype, q.Qclass

	// Synchronize access to s.queryLog and s.stats so they won't be suddenly
	// uninitialized while in use.  This can happen after proxy server has been
	// stopped, but its workers haven't yet exited.
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	if s.shouldLog(host, qt, cl, ids) {
		s.logQuery(dctx, ip, processingTime)
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
		s.updateStats(dctx, ipStr, processingTime)
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
func (s *Server) logQuery(dctx *dnsContext, ip net.IP, processingTime time.Duration) {
	pctx := dctx.proxyCtx

	p := &querylog.AddParams{
		Question:          pctx.Req,
		ReqECS:            pctx.ReqECS,
		Answer:            pctx.Res,
		OrigAnswer:        dctx.origResp,
		Result:            dctx.result,
		ClientID:          dctx.clientID,
		ClientIP:          ip,
		Elapsed:           processingTime,
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
func (s *Server) updateStats(dctx *dnsContext, clientIP string, processingTime time.Duration) {
	pctx := dctx.proxyCtx

	e := &stats.Entry{
		Domain:         aghnet.NormalizeDomain(pctx.Req.Question[0].Name),
		Result:         stats.RNotFiltered,
		ProcessingTime: processingTime,
		UpstreamTime:   pctx.QueryDuration,
	}

	if pctx.Upstream != nil {
		e.Upstream = pctx.Upstream.Address()
	}

	if clientID := dctx.clientID; clientID != "" {
		e.Client = clientID
	} else {
		e.Client = clientIP
	}

	switch dctx.result.Reason {
	case filtering.FilteredSafeBrowsing:
		e.Result = stats.RSafeBrowsing
	case filtering.FilteredParental:
		e.Result = stats.RParental
	case filtering.FilteredSafeSearch:
		e.Result = stats.RSafeSearch
	case
		filtering.FilteredBlockList,
		filtering.FilteredInvalid,
		filtering.FilteredBlockedService:
		e.Result = stats.RFiltered
	}

	s.stats.Update(e)
}
