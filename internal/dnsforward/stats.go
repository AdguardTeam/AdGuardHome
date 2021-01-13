package dnsforward

import (
	"net"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/miekg/dns"
)

// Write Stats data and logs
func processQueryLogsAndStats(ctx *dnsContext) int {
	elapsed := time.Since(ctx.startTime)
	s := ctx.srv
	d := ctx.proxyCtx

	shouldLog := true
	msg := d.Req

	// don't log ANY request if refuseAny is enabled
	if len(msg.Question) >= 1 && msg.Question[0].Qtype == dns.TypeANY && s.conf.RefuseAny {
		shouldLog = false
	}

	s.RLock()
	// Synchronize access to s.queryLog and s.stats so they won't be suddenly uninitialized while in use.
	// This can happen after proxy server has been stopped, but its workers haven't yet exited.
	if shouldLog && s.queryLog != nil {
		p := querylog.AddParams{
			Question:   msg,
			Answer:     d.Res,
			OrigAnswer: ctx.origResp,
			Result:     ctx.result,
			Elapsed:    elapsed,
			ClientIP:   getIP(d.Addr),
		}

		switch d.Proto {
		case proxy.ProtoHTTPS:
			p.ClientProto = querylog.ClientProtoDOH
		case proxy.ProtoQUIC:
			p.ClientProto = querylog.ClientProtoDOQ
		case proxy.ProtoTLS:
			p.ClientProto = querylog.ClientProtoDOT
		default:
			// Consider this a plain DNS-over-UDP or DNS-over-TCL
			// request.
		}

		if d.Upstream != nil {
			p.Upstream = d.Upstream.Address()
		}
		s.queryLog.Add(p)
	}

	s.updateStats(d, elapsed, *ctx.result)
	s.RUnlock()

	return resultDone
}

func (s *Server) updateStats(d *proxy.DNSContext, elapsed time.Duration, res dnsfilter.Result) {
	if s.stats == nil {
		return
	}

	e := stats.Entry{}
	e.Domain = strings.ToLower(d.Req.Question[0].Name)
	e.Domain = e.Domain[:len(e.Domain)-1] // remove last "."
	switch addr := d.Addr.(type) {
	case *net.UDPAddr:
		e.Client = addr.IP
	case *net.TCPAddr:
		e.Client = addr.IP
	}
	e.Time = uint32(elapsed / 1000)
	e.Result = stats.RNotFiltered

	switch res.Reason {

	case dnsfilter.FilteredSafeBrowsing:
		e.Result = stats.RSafeBrowsing

	case dnsfilter.FilteredParental:
		e.Result = stats.RParental

	case dnsfilter.FilteredSafeSearch:
		e.Result = stats.RSafeSearch

	case dnsfilter.FilteredRebind:
		// Rebinding is considered as filtered, not processed
		fallthrough
	case dnsfilter.FilteredBlockList:
		fallthrough
	case dnsfilter.FilteredInvalid:
		fallthrough
	case dnsfilter.FilteredBlockedService:
		e.Result = stats.RFiltered
	}

	s.stats.Update(e)
}
