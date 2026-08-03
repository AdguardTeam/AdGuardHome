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
func (s *Server) WrapUpstreamConfig(uc *proxy.UpstreamConfig) (wrapped *proxy.UpstreamConfig) {
	if uc == nil {
		return nil
	}

	return &proxy.UpstreamConfig{
		DomainReservedUpstreams:  s.wrapUpstreamsMap(uc.DomainReservedUpstreams),
		SpecifiedDomainUpstreams: s.wrapUpstreamsMap(uc.SpecifiedDomainUpstreams),
		SubdomainExclusions:      uc.SubdomainExclusions,
		Upstreams:                s.wrapUpstreams(uc.Upstreams),
	}
}

// wrapUpstreamsMap returns a copy of ups with each upstream wrapped into
// [*statsUpstream], or nil if ups is nil.
func (s *Server) wrapUpstreamsMap(
	ups map[string][]upstream.Upstream,
) (wrapped map[string][]upstream.Upstream) {
	if ups == nil {
		return nil
	}

	wrapped = make(map[string][]upstream.Upstream, len(ups))
	for domain, u := range ups {
		wrapped[domain] = s.wrapUpstreams(u)
	}

	return wrapped
}

// wrapUpstreams returns a copy of ups with each upstream wrapped into
// [*statsUpstream], or nil if ups is nil.  A single upstream may be referenced
// by several fields of a [proxy.UpstreamConfig], in which case it gets a
// wrapper of its own for each reference, just like it gets closed once for each
// of them, see [proxy.UpstreamConfig.Close].
func (s *Server) wrapUpstreams(ups []upstream.Upstream) (wrapped []upstream.Upstream) {
	if ups == nil {
		return nil
	}

	timeout := s.conf.UpstreamTimeout

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
