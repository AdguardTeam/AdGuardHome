package dnsforward

import (
	"net"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/querylog"
	"github.com/miekg/dns"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
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

		if d.Proto == "https" {
			p.ClientProto = "doh"
		} else if d.Proto == "tls" {
			p.ClientProto = "dot"
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

	case dnsfilter.FilteredBlackList:
		fallthrough
	case dnsfilter.FilteredInvalid:
		fallthrough
	case dnsfilter.FilteredBlockedService:
		e.Result = stats.RFiltered
	}

	s.stats.Update(e)
}
