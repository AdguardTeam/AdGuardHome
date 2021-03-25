package dnsforward

import (
	"crypto/tls"
	"fmt"
	"path"
	"strings"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/lucas-clemente/quic-go"
)

// maxDomainLabelLen is the maximum allowed length of a domain name label
// according to RFC 1035.
const maxDomainLabelLen = 63

// validateDomainNameLabel returns an error if label is not a valid label of
// a domain name.
func validateDomainNameLabel(label string) (err error) {
	if len(label) > maxDomainLabelLen {
		return fmt.Errorf("%q is too long, max: %d", label, maxDomainLabelLen)
	}

	for i, r := range label {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return fmt.Errorf("invalid char %q at index %d in %q", r, i, label)
		}
	}

	return nil
}

// ValidateClientID returns an error if clientID is not a valid client ID.
func ValidateClientID(clientID string) (err error) {
	err = validateDomainNameLabel(clientID)
	if err != nil {
		return fmt.Errorf("invalid client id: %w", err)
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

// processClientIDHTTPS extracts the client's ID from the path of the
// client's DNS-over-HTTPS request.
func processClientIDHTTPS(ctx *dnsContext) (rc resultCode) {
	pctx := ctx.proxyCtx
	r := pctx.HTTPRequest
	if r == nil {
		ctx.err = fmt.Errorf("proxy ctx http request of proto %s is nil", pctx.Proto)

		return resultCodeError
	}

	origPath := r.URL.Path
	parts := strings.Split(path.Clean(origPath), "/")
	if parts[0] == "" {
		parts = parts[1:]
	}

	if len(parts) == 0 || parts[0] != "dns-query" {
		ctx.err = fmt.Errorf("client id check: invalid path %q", origPath)

		return resultCodeError
	}

	clientID := ""
	switch len(parts) {
	case 1:
		// Just /dns-query, no client ID.
		return resultCodeSuccess
	case 2:
		clientID = parts[1]
	default:
		ctx.err = fmt.Errorf("client id check: invalid path %q: extra parts", origPath)

		return resultCodeError
	}

	err := ValidateClientID(clientID)
	if err != nil {
		ctx.err = fmt.Errorf("client id check: %w", err)

		return resultCodeError
	}

	ctx.clientID = clientID

	return resultCodeSuccess
}

// tlsConn is a narrow interface for *tls.Conn to simplify testing.
type tlsConn interface {
	ConnectionState() (cs tls.ConnectionState)
}

// quicSession is a narrow interface for quic.Session to simplify testing.
type quicSession interface {
	ConnectionState() (cs quic.ConnectionState)
}

// processClientID extracts the client's ID from the server name of the client's
// DOT or DOQ request or the path of the client's DOH.
func processClientID(dctx *dnsContext) (rc resultCode) {
	pctx := dctx.proxyCtx
	proto := pctx.Proto
	if proto == proxy.ProtoHTTPS {
		return processClientIDHTTPS(dctx)
	} else if proto != proxy.ProtoTLS && proto != proxy.ProtoQUIC {
		return resultCodeSuccess
	}

	srvConf := dctx.srv.conf
	hostSrvName := srvConf.TLSConfig.ServerName
	if hostSrvName == "" {
		return resultCodeSuccess
	}

	cliSrvName := ""
	if proto == proxy.ProtoTLS {
		conn := pctx.Conn
		tc, ok := conn.(tlsConn)
		if !ok {
			dctx.err = fmt.Errorf("proxy ctx conn of proto %s is %T, want *tls.Conn", proto, conn)

			return resultCodeError
		}

		cliSrvName = tc.ConnectionState().ServerName
	} else if proto == proxy.ProtoQUIC {
		qs, ok := pctx.QUICSession.(quicSession)
		if !ok {
			dctx.err = fmt.Errorf("proxy ctx quic session of proto %s is %T, want quic.Session", proto, pctx.QUICSession)

			return resultCodeError
		}

		cliSrvName = qs.ConnectionState().ServerName
	}

	clientID, err := clientIDFromClientServerName(hostSrvName, cliSrvName, srvConf.StrictSNICheck)
	if err != nil {
		dctx.err = fmt.Errorf("client id check: %w", err)

		return resultCodeError
	}

	dctx.clientID = clientID

	return resultCodeSuccess
}
