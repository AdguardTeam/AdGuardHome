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
	elapsed := time.Since(dctx.startTime)
	pctx := dctx.proxyCtx

	shouldLog := true
	msg := pctx.Req
	q := msg.Question[0]
	host := strings.ToLower(strings.TrimSuffix(q.Name, "."))

	// don't log ANY request if refuseAny is enabled
	if q.Qtype == dns.TypeANY && s.conf.RefuseAny {
		shouldLog = false
	}

	ip, _ := netutil.IPAndPortFromAddr(pctx.Addr)
	ip = slices.Clone(ip)

	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	s.anonymizer.Load()(ip)

	log.Debug("client ip: %s", ip)

	ipStr := ip.String()
	ids := []string{ipStr, dctx.clientID}

	// Synchronize access to s.queryLog and s.stats so they won't be suddenly
	// uninitialized while in use.  This can happen after proxy server has been
	// stopped, but its workers haven't yet exited.
	if shouldLog &&
		s.queryLog != nil &&
		// TODO(s.chzhen):  Use dnsforward.dnsContext when it will start
		// containing persistent client.
		s.queryLog.ShouldLog(host, q.Qtype, q.Qclass, ids) {
		s.logQuery(dctx, pctx, elapsed, ip)
	} else {
		log.Debug(
			"dnsforward: request %s %s from %s ignored; not logging",
			dns.Type(q.Qtype),
			host,
			ip,
		)
	}

	if s.stats != nil &&
		// TODO(s.chzhen):  Use dnsforward.dnsContext when it will start
		// containing persistent client.
		s.stats.ShouldCount(host, q.Qtype, q.Qclass, ids) {
		s.updateStats(dctx, elapsed, *dctx.result, ipStr)
	}

	return resultCodeSuccess
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
	e.Domain = e.Domain[:len(e.Domain)-1] // remove last "."

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
