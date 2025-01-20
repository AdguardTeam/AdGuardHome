package dnsforward

import (
	"encoding/binary"
	"fmt"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// type check
var _ proxy.BeforeRequestHandler = (*Server)(nil)

// HandleBefore is the handler that is called before any other processing,
// including logs.  It performs access checks and puts the client ID, if there
// is one, into the server's cache.
//
// TODO(d.kolyshev): Extract to separate package.
func (s *Server) HandleBefore(
	_ *proxy.Proxy,
	pctx *proxy.DNSContext,
) (err error) {
	clientID, err := s.clientIDFromDNSContext(pctx)
	if err != nil {
		return &proxy.BeforeRequestError{
			Err:      fmt.Errorf("getting clientid: %w", err),
			Response: s.NewMsgSERVFAIL(pctx.Req),
		}
	}

	blocked, _ := s.IsBlockedClient(pctx.Addr.Addr(), clientID)
	if blocked {
		return s.preBlockedResponse(pctx)
	}

	if len(pctx.Req.Question) == 1 {
		q := pctx.Req.Question[0]
		qt := q.Qtype
		host := aghnet.NormalizeDomain(q.Name)
		if s.access.isBlockedHost(host, qt) {
			log.Debug("access: request %s %s is in access blocklist", dns.Type(qt), host)

			return s.preBlockedResponse(pctx)
		}
	}

	if clientID != "" {
		key := [8]byte{}
		binary.BigEndian.PutUint64(key[:], pctx.RequestID)
		s.clientIDCache.Set(key[:], []byte(clientID))
	}

	return nil
}

// clientIDFromDNSContext extracts the client's ID from the server name of the
// client's DoT or DoQ request or the path of the client's DoH.  If the protocol
// is not one of these, clientID is an empty string and err is nil.
func (s *Server) clientIDFromDNSContext(pctx *proxy.DNSContext) (clientID string, err error) {
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

	hostSrvName := s.conf.ServerName
	if hostSrvName == "" {
		return "", nil
	}

	cliSrvName, err := clientServerName(pctx, proto)
	if err != nil {
		return "", fmt.Errorf("getting client server-name: %w", err)
	}

	clientID, err = clientIDFromClientServerName(
		hostSrvName,
		cliSrvName,
		s.conf.StrictSNICheck,
	)
	if err != nil {
		return "", fmt.Errorf("clientid check: %w", err)
	}

	return clientID, nil
}

// errAccessBlocked is a sentinel error returned when a request is blocked by
// access settings.
var errAccessBlocked errors.Error = "blocked by access settings"

// preBlockedResponse returns a protocol-appropriate response for a request that
// was blocked by access settings.
func (s *Server) preBlockedResponse(pctx *proxy.DNSContext) (err error) {
	if pctx.Proto == proxy.ProtoUDP || pctx.Proto == proxy.ProtoDNSCrypt {
		// Return nil so that dnsproxy drops the connection and thus
		// prevent DNS amplification attacks.
		return errAccessBlocked
	}

	return &proxy.BeforeRequestError{
		Err:      errAccessBlocked,
		Response: s.makeResponseREFUSED(pctx.Req),
	}
}
