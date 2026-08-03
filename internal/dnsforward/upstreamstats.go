package dnsforward

import (
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/miekg/dns"
)

// statsUpstream is an [upstream.Upstream] that records the duration of each
// successful exchange in the statistics.
//
// The response times are collected here, and not from
// [proxy.DNSContext.QueryStatistics], because the latter only describes the
// exchanges performed for a client's request.  The optimistic cache replies
// from the cache right away and refreshes the expired entry in a background
// goroutine, so those exchanges would never be counted, and the average
// upstream response time would only be based on cache misses, which are biased
// towards rare and slow domain names.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/8435.
type statsUpstream struct {
	// upstream is the wrapped upstream DNS server.  It must not be nil.
	upstream upstream.Upstream

	// srv is used to access the statistics.  It must not be nil.
	srv *Server

	// timeout is the timeout of a single exchange attempt with the wrapped
	// upstream, see [ServerConfig.UpstreamTimeout].  It is zero if there is no
	// timeout.
	timeout time.Duration
}

// type check
var _ upstream.Upstream = (*statsUpstream)(nil)

// Exchange implements the [upstream.Upstream] interface for *statsUpstream.
func (u *statsUpstream) Exchange(req *dns.Msg) (resp *dns.Msg, err error) {
	startTime := time.Now()
	resp, err = u.upstream.Exchange(req)
	if err != nil {
		// Don't count the failed exchanges, since their duration is mostly
		// defined by the timeout.
		return resp, err
	}

	u.srv.updateUpstreamStats(u.upstream.Address(), req, time.Since(startTime), u.timeout)

	return resp, nil
}

// markIgnoredReq records the request of dctx as one whose upstream exchanges
// must not be counted, when the statistics ignore its client or its domain.  It
// returns the function that undoes it, which must be called once the request
// has been resolved.
//
// An upstream exchange carries no client identity, so [statsUpstream] cannot
// consult [stats.Interface.ShouldCount] itself.  The decision is therefore made
// here, where the client is still known, and left for the wrappers to find.
// Exchanges that belong to no request at all, such as the background refreshes
// of the optimistic cache, are never marked and so are always counted.
//
// dctx and its proxy context must not be nil.
func (s *Server) markIgnoredReq(dctx *dnsContext) (unmark func()) {
	pctx := dctx.proxyCtx
	q := pctx.Req.Question[0]
	_, _, ids := s.clientIdentity(dctx)

	s.serverLock.RLock()
	count := s.shouldCountStat(aghnet.NormalizeDomain(q.Name), q.Qtype, q.Qclass, ids)
	s.serverLock.RUnlock()

	if count {
		return func() {}
	}

	req := pctx.Req
	s.ignoredReqs.Store(req, struct{}{})

	return func() { s.ignoredReqs.Delete(req) }
}

// isIgnoredReq returns true if req belongs to a query whose statistics are
// ignored, see [Server.markIgnoredReq].
func (s *Server) isIgnoredReq(req *dns.Msg) (ok bool) {
	_, ok = s.ignoredReqs.Load(req)

	return ok
}

// Address implements the [upstream.Upstream] interface for *statsUpstream.
func (u *statsUpstream) Address() (addr string) {
	return u.upstream.Address()
}

// Close implements the [upstream.Upstream] interface for *statsUpstream.
func (u *statsUpstream) Close() (err error) {
	return u.upstream.Close()
}

// updateUpstreamStats writes the response time of a single successful exchange
// with the upstream server at addr into the statistics.  timeout is the timeout
// of a single exchange attempt, if any.  req must not be nil.
func (s *Server) updateUpstreamStats(addr string, req *dns.Msg, dur, timeout time.Duration) {
	if len(req.Question) == 0 {
		return
	}

	if s.isIgnoredReq(req) {
		// The statistics ignore the client or the domain of the query this
		// exchange was performed for, see [Server.markIgnoredReq].
		return
	}

	if timeout > 0 && dur >= timeout {
		// A single attempt cannot take longer than the timeout, so this
		// exchange has retried at least once after an attempt that timed out.
		//
		// Such a duration describes the retry policy and the configured
		// timeout, which defaults to ten seconds, rather than the speed of the
		// upstream.  Averaging it in makes the reported response time an order
		// of magnitude higher than the actual one: with a ten-second timeout,
		// one retried query out of a hundred outweighs the other ninety-nine
		// several times over.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/8457.
		s.logger.Debug(
			"upstream retried after timeout; not counting in stats",
			"addr", addr,
			"duration", dur,
			"timeout", timeout,
		)

		return
	}

	// NOTE:  s.serverLock must not be acquired here, see [Server.upstreamStats].
	if s.upstreamStats == nil {
		return
	}

	s.upstreamStats.UpdateUpstream(&stats.UpstreamEntry{
		Address:       addr,
		Domain:        aghnet.NormalizeDomain(req.Question[0].Name),
		QueryDuration: dur,
	})
}

// WrapUpstreamConfig implements the [client.UpstreamConfigWrapper] interface
// for *Server.  It returns a copy of uc with each upstream wrapped into
// [*statsUpstream], or nil if uc is nil.  uc itself is not modified, but the
// returned configuration shares the underlying upstreams with it, so only one
// of the two should ever be closed.
//
// timeout must be the timeout that the upstreams of uc have actually been
// constructed with, since it is what tells a retried exchange from a slow one,
// see [Server.updateUpstreamStats].  It differs between configurations: the
// private rDNS upstreams use [defaultLocalTimeout] rather than the configured
// one.  It is taken as an argument rather than read from [Server.conf], which
// is mutable and is written under serverLock while this method may be called
// without it, from the client upstream manager.
func (s *Server) WrapUpstreamConfig(
	uc *proxy.UpstreamConfig,
	timeout time.Duration,
) (wrapped *proxy.UpstreamConfig) {
	if uc == nil {
		return nil
	}

	return &proxy.UpstreamConfig{
		DomainReservedUpstreams:  s.wrapUpstreamsMap(uc.DomainReservedUpstreams, timeout),
		SpecifiedDomainUpstreams: s.wrapUpstreamsMap(uc.SpecifiedDomainUpstreams, timeout),
		SubdomainExclusions:      uc.SubdomainExclusions,
		Upstreams:                s.wrapUpstreams(uc.Upstreams, timeout),
	}
}

// wrapUpstreamsMap returns a copy of ups with each upstream wrapped into
// [*statsUpstream], or nil if ups is nil.
func (s *Server) wrapUpstreamsMap(
	ups map[string][]upstream.Upstream,
	timeout time.Duration,
) (wrapped map[string][]upstream.Upstream) {
	if ups == nil {
		return nil
	}

	wrapped = make(map[string][]upstream.Upstream, len(ups))
	for domain, u := range ups {
		wrapped[domain] = s.wrapUpstreams(u, timeout)
	}

	return wrapped
}

// wrapUpstreams returns a copy of ups with each upstream wrapped into
// [*statsUpstream], or nil if ups is nil.  A single upstream may be referenced
// by several fields of a [proxy.UpstreamConfig], in which case it gets a
// wrapper of its own for each reference, just like it gets closed once for each
// of them, see [proxy.UpstreamConfig.Close].
func (s *Server) wrapUpstreams(
	ups []upstream.Upstream,
	timeout time.Duration,
) (wrapped []upstream.Upstream) {
	if ups == nil {
		return nil
	}

	wrapped = make([]upstream.Upstream, 0, len(ups))
	for _, u := range ups {
		wrapped = append(wrapped, &statsUpstream{
			upstream: u,
			srv:      s,
			timeout:  timeout,
		})
	}

	return wrapped
}
