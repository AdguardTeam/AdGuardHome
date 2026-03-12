package dnsforward

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/miekg/dns"
)

// type check
var _ proxy.Middleware = (*Server)(nil)

// Wrap implements the [proxy.Middleware] interface for *Server.
//
// TODO(d.kolyshev): Move to a dedicated package.
func (s *Server) Wrap(h proxy.Handler) (wrapped proxy.Handler) {
	f := func(p *proxy.Proxy, pctx *proxy.DNSContext) (err error) {
		// TODO(f.setrakov): Obtain context from arguments.
		ctx := context.TODO()

		clientID, err := s.clientIDFromDNSContext(ctx, pctx)
		if err != nil {
			s.logger.WarnContext(ctx, "resolving client id", slogutil.KeyError, err)

			pctx.Res = s.NewMsgSERVFAIL(pctx.Req)
			pctx.Res.Compress = true

			return nil
		}

		blocked, _ := s.IsBlockedClient(pctx.Addr.Addr(), clientID)
		if blocked {
			return s.serveBlockedResponse(pctx)
		}

		blocked = s.isBlockedHost(ctx, pctx.Req.Question)
		if blocked {
			return s.serveBlockedResponse(pctx)
		}

		if clientID != "" {
			key := [8]byte{}
			binary.BigEndian.PutUint64(key[:], pctx.RequestID)
			s.clientIDCache.Set(key[:], []byte(clientID))
		}

		return h.ServeDNS(p, pctx)
	}

	return proxy.HandlerFunc(f)
}

// serveBlockedResponse sets a protocol-appropriate response for a request that
// was blocked by access settings.
func (s *Server) serveBlockedResponse(pctx *proxy.DNSContext) (err error) {
	if pctx.Proto == proxy.ProtoUDP || pctx.Proto == proxy.ProtoDNSCrypt {
		// Return nil so that dnsproxy drops the connection and thus prevent DNS
		// amplification attacks.
		return proxy.ErrDrop
	}

	pctx.Res = s.makeResponseREFUSED(pctx.Req)
	pctx.Res.Compress = true

	return nil
}

// isBlockedHost checks if the request is in the access blocklist.
func (s *Server) isBlockedHost(ctx context.Context, question []dns.Question) (blocked bool) {
	if len(question) != 1 {
		return false
	}

	q := question[0]
	qt := q.Qtype
	host := aghnet.NormalizeDomain(q.Name)

	if s.access.isBlockedHost(host, qt) {
		s.logger.DebugContext(
			ctx,
			"request is in access blocklist",
			"dns_type", dns.Type(qt),
			"host", host,
		)

		return true
	}

	return false
}

// clientIDFromDNSContext extracts the client's ID from the server name of the
// client's DoT or DoQ request or the path of the client's DoH.  If the protocol
// is not one of these, clientID is an empty string and err is nil.
func (s *Server) clientIDFromDNSContext(
	ctx context.Context,
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

	cliSrvName, err := clientServerName(ctx, s.logger, pctx, proto)
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
