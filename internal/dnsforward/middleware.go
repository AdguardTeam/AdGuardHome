package dnsforward

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/syncutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/miekg/dns"
)

// type check
var _ proxy.Middleware = (*Server)(nil)

// Wrap implements the [proxy.Middleware] interface for *Server.  ctx must
// contain a logger accessible with [slogutil.LoggerFromContext].
//
// TODO(d.kolyshev):  Move to a dedicated package.
func (s *Server) Wrap(h proxy.Handler) (wrapped proxy.Handler) {
	f := func(ctx context.Context, p *proxy.Proxy, pctx *proxy.DNSContext) (err error) {
		l := slogutil.MustLoggerFromContext(ctx)

		clientID, err := s.clientIDFromDNSContext(ctx, l, pctx)
		if err != nil {
			s.logger.WarnContext(ctx, "resolving client id", slogutil.KeyError, err)

			pctx.Res = s.NewMsgSERVFAIL(pctx.Req)

			return nil
		}

		blocked, _ := s.IsBlockedClient(pctx.Addr.Addr(), clientID)
		if blocked {
			return s.serveBlockedResponse(pctx)
		}

		blocked = s.isBlockedHost(ctx, l, pctx.Req.Question)
		if blocked {
			return s.serveBlockedResponse(pctx)
		}

		if clientID != "" {
			ctx = contextWithClientID(ctx, clientID)
		}

		return h.ServeDNS(ctx, p, pctx)
	}

	return proxy.HandlerFunc(f)
}

// serveBlockedResponse sets a protocol-appropriate response for a request that
// was blocked by access settings.  pctx must be filled with the request.
func (s *Server) serveBlockedResponse(pctx *proxy.DNSContext) (err error) {
	if pctx.Proto == proxy.ProtoUDP || pctx.Proto == proxy.ProtoDNSCrypt {
		// Return nil so that dnsproxy drops the connection and thus prevent DNS
		// amplification attacks.
		return proxy.ErrDrop
	}

	pctx.Res = s.makeResponseREFUSED(pctx.Req)

	return nil
}

// isBlockedHost checks if the request is in the access blocklist.  l must not
// be nil.
func (s *Server) isBlockedHost(
	ctx context.Context,
	l *slog.Logger,
	question []dns.Question,
) (blocked bool) {
	if len(question) != 1 {
		return false
	}

	q := question[0]
	qt := q.Qtype
	host := aghnet.NormalizeDomain(q.Name)

	if s.access.isBlockedHost(host, qt) {
		l.DebugContext(ctx, "request is in access blocklist")

		return true
	}

	return false
}

// clientIDFromDNSContext extracts the client's ID from the server name of the
// client's DoT or DoQ request or the path of the client's DoH.  If the protocol
// is not one of these, clientID is an empty string and err is nil.  l and pctx
// must not be nil.
func (s *Server) clientIDFromDNSContext(
	ctx context.Context,
	l *slog.Logger,
	pctx *proxy.DNSContext,
) (clientID string, err error) {
	proto := pctx.Proto
	if proto == proxy.ProtoHTTPS {
		clientID, err = clientIDFromDNSContextHTTPS(pctx)
		if err != nil {
			return "", fmt.Errorf("checking url: %w", err)
		} else if clientID != "" {
			return clientID, nil
		}

		// Go on and check the domain name as well.
	} else if proto != proxy.ProtoTLS && proto != proxy.ProtoQUIC {
		return "", nil
	}

	hostSrvName := s.conf.TLSConf.ServerName
	if hostSrvName == "" {
		return "", nil
	}

	cliSrvName, err := clientServerName(ctx, l, pctx, proto)
	if err != nil {
		return "", fmt.Errorf("getting client server-name: %w", err)
	}

	clientID, err = clientIDFromClientServerName(
		hostSrvName,
		cliSrvName,
		s.conf.TLSConf.StrictSNICheck,
	)
	if err != nil {
		return "", fmt.Errorf("clientid check: %w", err)
	}

	return clientID, nil
}

// logMiddleware adds a logger using [slogutil.ContextWithLogger] and logs the
// starts and ends of queries at a given level.
//
// TODO(d.kolyshev):  Consider moving to dnsproxy.
type logMiddleware struct {
	attrPool *syncutil.Pool[[]slog.Attr]
	logger   *slog.Logger
	lvl      slog.Level
}

// logMwAttrNum is the number of attributes used by the logger set by
// [logMiddleware].
const logMwAttrNum = 3

// newLogMiddleware returns a new *logMiddleware with l as the base logger.
func newLogMiddleware(l *slog.Logger, lvl slog.Level) (mw *logMiddleware) {
	return &logMiddleware{
		attrPool: syncutil.NewSlicePool[slog.Attr](logMwAttrNum),
		logger:   l,
		lvl:      lvl,
	}
}

// type check
var _ proxy.Middleware = (*logMiddleware)(nil)

// Wrap implements the [proxy.Middleware] interface for *logMiddleware.  It adds
// a logger to the context and logs the starts and ends of queries at a given
// level.
func (m *logMiddleware) Wrap(h proxy.Handler) (wrapped proxy.Handler) {
	f := func(ctx context.Context, p *proxy.Proxy, dctx *proxy.DNSContext) (err error) {
		startTime := time.Now()

		attrsPtr := m.attrsSlicePtr(dctx.Req)
		defer m.attrPool.Put(attrsPtr)

		logHdlr := m.logger.Handler().WithAttrs(*attrsPtr)
		l := slog.New(logHdlr)
		ctx = slogutil.ContextWithLogger(ctx, l)

		l.Log(ctx, m.lvl, "started")
		defer m.logFinished(ctx, l, startTime)

		return h.ServeDNS(ctx, p, dctx)
	}

	return proxy.HandlerFunc(f)
}

// attrsSlicePtr returns a pointer to a slice with the attributes from the
// request set.  Callers should defer returning attrsPtr back to the pool.
func (m *logMiddleware) attrsSlicePtr(r *dns.Msg) (attrsPtr *[]slog.Attr) {
	attrsPtr = m.attrPool.Get()

	attrs := *attrsPtr

	// Optimize bounds checking.
	_ = attrs[logMwAttrNum-1]

	attrs[0] = slog.Uint64("id", uint64(r.Id))

	if len(r.Question) > 0 {
		q := r.Question[0]
		attrs[1] = slog.String("qtype", dns.Type(q.Qtype).String())
		attrs[2] = slog.String("target", q.Name)
	} else {
		attrs[1] = slog.Attr{}
		attrs[2] = slog.Attr{}
	}

	return attrsPtr
}

// logFinished is called at the end of handling of a query.
func (m *logMiddleware) logFinished(ctx context.Context, l *slog.Logger, startTime time.Time) {
	if !l.Enabled(ctx, m.lvl) {
		return
	}

	l.Log(ctx, m.lvl, "finished", "elapsed", timeutil.Duration(time.Since(startTime)))
}
