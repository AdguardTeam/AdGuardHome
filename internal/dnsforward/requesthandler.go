package dnsforward

import (
	"context"
	"log/slog"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// type check
var _ proxy.Handler = (*Server)(nil)

// ServeDNS implements the [proxy.Handler] interface for [*Server].  ctx must
// contain a logger accessible with [slogutil.LoggerFromContext].
func (s *Server) ServeDNS(ctx context.Context, _ *proxy.Proxy, pctx *proxy.DNSContext) (err error) {
	dctx := &dnsContext{
		proxyCtx:  pctx,
		result:    &filtering.Result{},
		startTime: time.Now(),
	}

	l := slogutil.MustLoggerFromContext(ctx)

	type modProcessFunc func(ctx context.Context, l *slog.Logger, dctx *dnsContext) (rc resultCode)

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
		r := process(ctx, l, dctx)
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
