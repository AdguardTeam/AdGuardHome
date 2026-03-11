package dnsforward

import (
	"context"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
)

// type check
var _ proxy.Handler = (*Server)(nil)

// ServeDNS implements the [proxy.Handler] interface for [*Server].
func (s *Server) ServeDNS(_ *proxy.Proxy, pctx *proxy.DNSContext) (err error) {
	// TODO(s.chzhen):  Pass context.
	ctx := context.TODO()

	dctx := &dnsContext{
		proxyCtx:  pctx,
		result:    &filtering.Result{},
		startTime: time.Now(),
	}

	type modProcessFunc func(ctx context.Context, dctx *dnsContext) (rc resultCode)

	// Since [*dnsforward.Server] is used as [proxy.Handler], there is no need
	// for additional index out of range checking in any of the following
	// functions, because the (*proxy.Proxy).handleDNSRequest method performs it
	// before calling the appropriate handler.
	mods := []modProcessFunc{
		s.processInitial,
		s.processDDRQuery,
		s.processDHCPHosts,
		s.processDHCPAddrs,
		s.processFilteringBeforeRequest,
		s.processUpstream,
		s.processFilteringAfterResponse,
		s.ipset.process,
		s.processQueryLogsAndStats,
	}
	for _, process := range mods {
		r := process(ctx, dctx)
		switch r {
		case resultCodeSuccess:
			// continue: call the next filter
		case resultCodeFinish:
			return nil
		case resultCodeError:
			return dctx.err
		}
	}

	if pctx.Res != nil {
		// Some devices require DNS message compression.
		pctx.Res.Compress = true
	}

	return nil
}
