package dnsforward

import (
	"crypto/tls"
	"fmt"
	"path"
	"strings"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/lucas-clemente/quic-go"
)

// ValidateClientID returns an error if id is not a valid ClientID.
func ValidateClientID(id string) (err error) {
	err = netutil.ValidateDomainNameLabel(id)
	if err != nil {
		// Replace the domain name label wrapper with our own.
		return fmt.Errorf("invalid clientid %q: %w", id, errors.Unwrap(err))
	}

	return nil
}

// hasLabelSuffix returns true if s ends with suffix preceded by a dot.  It's
// a helper function to prevent unnecessary allocations in code like:
//
// if strings.HasSuffix(s, "." + suffix) { /* … */ }
//
// s must be longer than suffix.
func hasLabelSuffix(s, suffix string) (ok bool) {
	return strings.HasSuffix(s, suffix) && s[len(s)-len(suffix)-1] == '.'
}

// clientIDFromClientServerName extracts and validates a ClientID.  hostSrvName
// is the server name of the host.  cliSrvName is the server name as sent by the
// client.  When strict is true, and client and host server name don't match,
// clientIDFromClientServerName will return an error.
func clientIDFromClientServerName(
	hostSrvName string,
	cliSrvName string,
	strict bool,
) (clientID string, err error) {
	if hostSrvName == cliSrvName {
		return "", nil
	}

	if !hasLabelSuffix(cliSrvName, hostSrvName) {
		if !strict {
			return "", nil
		}

		return "", fmt.Errorf(
			"client server name %q doesn't match host server name %q",
			cliSrvName,
			hostSrvName,
		)
	}

	clientID = cliSrvName[:len(cliSrvName)-len(hostSrvName)-1]
	err = ValidateClientID(clientID)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return "", err
	}

	return strings.ToLower(clientID), nil
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
		return "", fmt.Errorf("clientid check: invalid path %q", origPath)
	}

	switch len(parts) {
	case 1:
		// Just /dns-query, no ClientID.
		return "", nil
	case 2:
		clientID = parts[1]
	default:
		return "", fmt.Errorf("clientid check: invalid path %q: extra parts", origPath)
	}

	err = ValidateClientID(clientID)
	if err != nil {
		return "", fmt.Errorf("clientid check: %w", err)
	}

	return strings.ToLower(clientID), nil
}

// tlsConn is a narrow interface for *tls.Conn to simplify testing.
type tlsConn interface {
	ConnectionState() (cs tls.ConnectionState)
}

// quicConnection is a narrow interface for quic.Connection to simplify testing.
type quicConnection interface {
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
		conn, ok := pctx.QUICConnection.(quicConnection)
		if !ok {
			return "", fmt.Errorf(
				"proxy ctx quic conn of proto %s is %T, want quic.Connection",
				proto,
				pctx.QUICConnection,
			)
		}

		cliSrvName = conn.ConnectionState().TLS.ServerName
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
