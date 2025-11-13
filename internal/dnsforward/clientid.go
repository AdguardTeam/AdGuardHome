package dnsforward

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/netutil"
)

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

	if !netutil.IsImmediateSubdomain(cliSrvName, hostSrvName) {
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
	err = client.ValidateClientID(clientID)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return "", err
	}

	return strings.ToLower(clientID), nil
}

// clientIDFromDNSContextHTTPS extracts the ClientID from the path of the
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

	err = client.ValidateClientID(clientID)
	if err != nil {
		return "", fmt.Errorf("clientid check: %w", err)
	}

	return strings.ToLower(clientID), nil
}

// tlsConn is a narrow interface for *tls.Conn to simplify testing.
type tlsConn interface {
	ConnectionState() (cs tls.ConnectionState)
}

// clientServerName returns the TLS server name based on the protocol.  For
// DNS-over-HTTPS requests, it will return the hostname part of the Host header
// if there is one.  l and pctx must not be nil.
func clientServerName(
	ctx context.Context,
	l *slog.Logger,
	pctx *proxy.DNSContext,
	proto proxy.Proto,
) (srvName string, err error) {
	from := "tls conn"

	switch proto {
	case proxy.ProtoHTTPS:
		var fromHost bool
		srvName, fromHost, err = clientServerNameFromHTTP(pctx.HTTPRequest)
		if err != nil {
			return "", fmt.Errorf("from http: %w", err)
		}

		if fromHost {
			from = "host header"
		}
	case proxy.ProtoQUIC:
		srvName = pctx.QUICConnection.ConnectionState().TLS.ServerName
	case proxy.ProtoTLS:
		conn := pctx.Conn
		tc, ok := conn.(tlsConn)
		if !ok {
			return "", fmt.Errorf("pctx conn of proto %s is %T, want *tls.Conn", proto, conn)
		}

		srvName = tc.ConnectionState().ServerName
	}

	l.DebugContext(ctx, "got client server name", "name", srvName, "from", from)

	return srvName, nil
}

// clientServerNameFromHTTP returns the TLS server name or the value of the host
// header depending on the protocol.  fromHost is true if srvName comes from the
// "Host" HTTP header.
func clientServerNameFromHTTP(r *http.Request) (srvName string, fromHost bool, err error) {
	if connState := r.TLS; connState != nil {
		return connState.ServerName, false, nil
	}

	if r.Host == "" {
		return "", false, nil
	}

	srvName, err = netutil.SplitHost(r.Host)
	if err != nil {
		return "", false, fmt.Errorf("parsing host: %w", err)
	}

	return srvName, true, nil
}
