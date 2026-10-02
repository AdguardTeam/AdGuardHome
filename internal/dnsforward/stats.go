package dnsforward

import (
	"context"
	"log/slog"
	"net"
	"net/netip"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/miekg/dns"
)

// processQueryLogsAndStats writes stats data and logs.  l and dctx must not be
// nil.
func (s *Server) processQueryLogsAndStats(
	ctx context.Context,
	l *slog.Logger,
	dctx *dnsContext,
) (rc resultCode) {
	l.DebugContext(ctx, "started processing querylog and stats")
	defer l.DebugContext(ctx, "finished processing querylog and stats")

	pctx := dctx.proxyCtx
	q := pctx.Req.Question[0]
	host := aghnet.NormalizeDomain(q.Name)
	processingTime := time.Since(dctx.startTime)

	ip := pctx.Addr.Addr().AsSlice()
	s.anonymizer.Load()(ip)
	ipStr := net.IP(ip).String()

	l.DebugContext(ctx, "client ip for stats and querylog", "ip", ipStr)

	ids := []string{ipStr}
	if dctx.clientID != "" {
		// Use the ClientID first because it has a higher priority.  Filters
		// have the same priority, see applyAdditionalFiltering.
		ids = []string{dctx.clientID, ipStr}
	}

	qt, cl := q.Qtype, q.Qclass

	// Synchronize access to s.queryLog and s.stats so they won't be suddenly
	// uninitialized while in use.  This can happen after proxy server has been
	// stopped, but its workers haven't yet exited.
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	if s.shouldLog(host, qt, cl, ids) {
		s.logQuery(dctx, ip, processingTime)
	} else {
		l.DebugContext(
			ctx,
			"not adding to querylog",
			"dns_class", dns.Class(cl),
			"ip", ipStr,
		)
	}

	if s.shouldCountStat(host, qt, cl, ids) {
		s.updateStats(dctx, ipStr, processingTime)
	} else {
		l.DebugContext(
			ctx,
			"not counting in stats",
			"dns_class", dns.Class(cl),
			"ip", ipStr,
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
	}

	if qs := pctx.QueryStatistics(); qs != nil {
		ms := qs.Main()
		if len(ms) == 1 && ms[0].IsCached {
			p.Upstream = ms[0].Address
			p.Cached = true
		}
	}

	s.queryLog.Add(p)
}

// retryThreshold returns the duration at or above which a successful exchange
// must have retried after an attempt that timed out, since a single attempt
// cannot outlast the timeout it was made with.  pctx must not be nil.
//
// s.serverLock is expected to be locked.
func (s *Server) retryThreshold(pctx *proxy.DNSContext) (d time.Duration) {
	if pctx.RequestedPrivateRDNS != (netip.Prefix{}) {
		// The private rDNS upstreams are constructed with a timeout of their
		// own, see prepareLocalResolvers.
		return defaultLocalTimeout
	}

	return s.conf.UpstreamTimeout
}

// appendCountedUpstreams appends those of us that should be counted in the
// statistics to stats.
//
// A plain DNS upstream retries once when an attempt times out, for example
// when a UDP datagram is lost, and reports the retried exchange as an ordinary
// success.  Its duration is then at least the whole timeout, ten seconds by
// default, even though the successful attempt itself took a millisecond.
// Averaging such a sample in makes the reported response time an order of
// magnitude higher than the actual one, so leave it out: it describes the retry
// policy and the configured timeout rather than the speed of the upstream.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/8457.
func appendCountedUpstreams(
	stats []*proxy.UpstreamStatistics,
	us []*proxy.UpstreamStatistics,
	threshold time.Duration,
) (appended []*proxy.UpstreamStatistics) {
	for _, u := range us {
		if threshold > 0 && u.Error == nil && u.QueryDuration >= threshold {
			continue
		}

		stats = append(stats, u)
	}

	return stats
}

// handleOptimisticRefresh records the response times of an optimistic cache
// refresh.  It implements [proxy.Config.OnOptimisticRefresh].  dctx must not be
// nil.
//
// Such a refresh is performed in the background once an expired entry has
// already been answered from the cache, so it reaches no request handler and
// belongs to no client.  Without it the response times would only ever be
// sampled from cache misses, and with the optimistic cache enabled the popular
// names, which are exactly the ones kept warm, would never be sampled at all.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/8435.
func (s *Server) handleOptimisticRefresh(dctx *proxy.DNSContext) {
	qs := dctx.QueryStatistics()
	if qs == nil || dctx.Req == nil || len(dctx.Req.Question) == 0 {
		return
	}

	domain := aghnet.NormalizeDomain(dctx.Req.Question[0].Name)

	// Synchronize access to s.stats so it won't be suddenly uninitialized while
	// in use, the same way processQueryLogsAndStats does.
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	if s.stats == nil {
		return
	}

	threshold := s.retryThreshold(dctx)
	for _, u := range appendCountedUpstreams(nil, qs.Main(), threshold) {
		if u.IsCached || u.Error != nil {
			continue
		}

		s.stats.UpdateUpstream(&stats.UpstreamEntry{
			Address:       u.Address,
			Domain:        domain,
			QueryDuration: u.QueryDuration,
		})
	}
}

// updateStats writes the request data into statistics.
func (s *Server) updateStats(dctx *dnsContext, clientIP string, processingTime time.Duration) {
	pctx := dctx.proxyCtx

	var upstreamStats []*proxy.UpstreamStatistics
	if qs := pctx.QueryStatistics(); qs != nil {
		threshold := s.retryThreshold(pctx)
		upstreamStats = appendCountedUpstreams(upstreamStats, qs.Main(), threshold)
		upstreamStats = appendCountedUpstreams(upstreamStats, qs.Fallback(), threshold)
	}

	e := &stats.Entry{
		UpstreamStats:  upstreamStats,
		Domain:         aghnet.NormalizeDomain(pctx.Req.Question[0].Name),
		Result:         stats.RNotFiltered,
		ProcessingTime: processingTime,
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
