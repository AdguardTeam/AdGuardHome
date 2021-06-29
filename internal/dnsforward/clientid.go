package dnsforward

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"path"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/lucas-clemente/quic-go"
)

// ValidateClientID returns an error if clientID is not a valid client ID.
func ValidateClientID(clientID string) (err error) {
	err = aghnet.ValidateDomainNameLabel(clientID)
	if err != nil {
		// Replace the domain name label wrapper with our own.
		return fmt.Errorf("invalid client id %q: %w", clientID, errors.Unwrap(err))
	}

	return nil
}

// clientIDFromClientServerName extracts and validates a client ID.  hostSrvName
// is the server name of the host.  cliSrvName is the server name as sent by the
// client.  When strict is true, and client and host server name don't match,
// clientIDFromClientServerName will return an error.
func clientIDFromClientServerName(hostSrvName, cliSrvName string, strict bool) (clientID string, err error) {
	if hostSrvName == cliSrvName {
		return "", nil
	}

	if !strings.HasSuffix(cliSrvName, hostSrvName) {
		if !strict {
			return "", nil
		}

		return "", fmt.Errorf("client server name %q doesn't match host server name %q", cliSrvName, hostSrvName)
	}

	clientID = cliSrvName[:len(cliSrvName)-len(hostSrvName)-1]
	err = ValidateClientID(clientID)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return "", err
	}

	return clientID, nil
}

// clientIDFromDNSContextHTTPS extracts the client's ID from the path of the
// client's DNS-over-HTTPS request.
func clientIDFromDNSContextHTTPS(pctx *proxy.DNSContext) (clientID string, err error) {
	r := pctx.HTTPRequest
	if r == nil {
		return "", fmt.Errorf(
			"proxy ctx http request of proto %s is nil",
			pctx.Proto,
		)
	}

	origPath := r.URL.Path
	parts := strings.Split(path.Clean(origPath), "/")
	if parts[0] == "" {
		parts = parts[1:]
	}

	if len(parts) == 0 || parts[0] != "dns-query" {
		return "", fmt.Errorf("client id check: invalid path %q", origPath)
	}

	switch len(parts) {
	case 1:
		// Just /dns-query, no client ID.
		return "", nil
	case 2:
		clientID = parts[1]
	default:
		return "", fmt.Errorf("client id check: invalid path %q: extra parts", origPath)
	}

	err = ValidateClientID(clientID)
	if err != nil {
		return "", fmt.Errorf("client id check: %w", err)
	}

	return clientID, nil
}

// tlsConn is a narrow interface for *tls.Conn to simplify testing.
type tlsConn interface {
	ConnectionState() (cs tls.ConnectionState)
}

// quicSession is a narrow interface for quic.Session to simplify testing.
type quicSession interface {
	ConnectionState() (cs quic.ConnectionState)
}

// clientIDFromDNSContext extracts the client's ID from the server name of the
// client's DoT or DoQ request or the path of the client's DoH.  If the protocol
// is not one of these, clientID is an empty string and err is nil.
func (s *Server) clientIDFromDNSContext(pctx *proxy.DNSContext) (clientID string, err error) {
	proto := pctx.Proto
	if proto == proxy.ProtoHTTPS {
		return clientIDFromDNSContextHTTPS(pctx)
	} else if proto != proxy.ProtoTLS && proto != proxy.ProtoQUIC {
		return "", nil
	}

	hostSrvName := s.conf.ServerName
	if hostSrvName == "" {
		return "", nil
	}

	cliSrvName := ""
	switch proto {
	case proxy.ProtoTLS:
		conn := pctx.Conn
		tc, ok := conn.(tlsConn)
		if !ok {
			return "", fmt.Errorf(
				"proxy ctx conn of proto %s is %T, want *tls.Conn",
				proto,
				conn,
			)
		}

		cliSrvName = tc.ConnectionState().ServerName
	case proxy.ProtoQUIC:
		qs, ok := pctx.QUICSession.(quicSession)
		if !ok {
			return "", fmt.Errorf(
				"proxy ctx quic session of proto %s is %T, want quic.Session",
				proto,
				pctx.QUICSession,
			)
		}

		cliSrvName = qs.ConnectionState().TLS.ServerName
	}

	clientID, err = clientIDFromClientServerName(
		hostSrvName,
		cliSrvName,
		s.conf.StrictSNICheck,
	)
	if err != nil {
		return "", fmt.Errorf("client id check: %w", err)
	}

	return clientID, nil
}

// processClientID puts the clientID into the DNS context, if there is one.
func (s *Server) processClientID(dctx *dnsContext) (rc resultCode) {
	pctx := dctx.proxyCtx

	var key [8]byte
	binary.BigEndian.PutUint64(key[:], pctx.RequestID)
	clientIDData := s.clientIDCache.Get(key[:])
	if clientIDData == nil {
		return resultCodeSuccess
	}

	dctx.clientID = string(clientIDData)

	return resultCodeSuccess
}
