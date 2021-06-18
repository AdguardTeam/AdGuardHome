package dnsforward

import (
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/miekg/dns"
)

// Write Stats data and logs
func processQueryLogsAndStats(ctx *dnsContext) (rc resultCode) {
	elapsed := time.Since(ctx.startTime)
	s := ctx.srv
	pctx := ctx.proxyCtx

	shouldLog := true
	msg := pctx.Req

	// don't log ANY request if refuseAny is enabled
	if len(msg.Question) >= 1 && msg.Question[0].Qtype == dns.TypeANY && s.conf.RefuseAny {
		shouldLog = false
	}

	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	// Synchronize access to s.queryLog and s.stats so they won't be suddenly uninitialized while in use.
	// This can happen after proxy server has been stopped, but its workers haven't yet exited.
	if shouldLog && s.queryLog != nil {
		p := querylog.AddParams{
			Question:   msg,
			Answer:     pctx.Res,
			OrigAnswer: ctx.origResp,
			Result:     ctx.result,
			Elapsed:    elapsed,
			ClientIP:   aghnet.IPFromAddr(pctx.Addr),
			ClientID:   ctx.clientID,
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
			// Consider this a plain DNS-over-UDP or DNS-over-TCP
			// request.
		}

		if pctx.Upstream != nil {
			p.Upstream = pctx.Upstream.Address()
		}

		s.queryLog.Add(p)
	}

	s.updateStats(ctx, elapsed, *ctx.result)

	return resultCodeSuccess
}

func (s *Server) updateStats(ctx *dnsContext, elapsed time.Duration, res filtering.Result) {
	if s.stats == nil {
		return
	}

	pctx := ctx.proxyCtx
	e := stats.Entry{}
	e.Domain = strings.ToLower(pctx.Req.Question[0].Name)
	e.Domain = e.Domain[:len(e.Domain)-1] // remove last "."

	if clientID := ctx.clientID; clientID != "" {
		e.Client = clientID
	} else if ip := aghnet.IPFromAddr(pctx.Addr); ip != nil {
		e.Client = ip.String()
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
