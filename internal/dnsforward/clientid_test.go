package dnsforward

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/lucas-clemente/quic-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTLSConn is a tlsConn for tests.
type testTLSConn struct {
	// Conn is embedded here simply to make testTLSConn a net.Conn without
	// actually implementing all methods.
	net.Conn

	serverName string
}

// ConnectionState implements the tlsConn interface for testTLSConn.
func (c testTLSConn) ConnectionState() (cs tls.ConnectionState) {
	cs.ServerName = c.serverName

	return cs
}

// testQUICSession is a quicSession for tests.
type testQUICSession struct {
	// Session is embedded here simply to make testQUICSession
	// a quic.Session without acctually implementing all methods.
	quic.Session

	serverName string
}

// ConnectionState implements the quicSession interface for testQUICSession.
func (c testQUICSession) ConnectionState() (cs quic.ConnectionState) {
	cs.TLS.ServerName = c.serverName

	return cs
}

func TestServer_clientIDFromDNSContext(t *testing.T) {
	testCases := []struct {
		name         string
		proto        proxy.Proto
		hostSrvName  string
		cliSrvName   string
		wantClientID string
		wantErrMsg   string
		strictSNI    bool
	}{{
		name:         "udp",
		proto:        proxy.ProtoUDP,
		hostSrvName:  "",
		cliSrvName:   "",
		wantClientID: "",
		wantErrMsg:   "",
		strictSNI:    false,
	}, {
		name:         "tls_no_client_id",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "example.com",
		wantClientID: "",
		wantErrMsg:   "",
		strictSNI:    true,
	}, {
		name:         "tls_no_client_server_name",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "",
		wantClientID: "",
		wantErrMsg: `client id check: client server name "" ` +
			`doesn't match host server name "example.com"`,
		strictSNI: true,
	}, {
		name:         "tls_no_client_server_name_no_strict",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "",
		wantClientID: "",
		wantErrMsg:   "",
		strictSNI:    false,
	}, {
		name:         "tls_client_id",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "cli.example.com",
		wantClientID: "cli",
		wantErrMsg:   "",
		strictSNI:    true,
	}, {
		name:         "tls_client_id_hostname_error",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "cli.example.net",
		wantClientID: "",
		wantErrMsg: `client id check: client server name "cli.example.net" ` +
			`doesn't match host server name "example.com"`,
		strictSNI: true,
	}, {
		name:         "tls_invalid_client_id",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "!!!.example.com",
		wantClientID: "",
		wantErrMsg: `client id check: invalid client id "!!!": ` +
			`invalid char '!' at index 0`,
		strictSNI: true,
	}, {
		name:        "tls_client_id_too_long",
		proto:       proxy.ProtoTLS,
		hostSrvName: "example.com",
		cliSrvName: `abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmno` +
			`pqrstuvwxyz0123456789.example.com`,
		wantClientID: "",
		wantErrMsg: `client id check: invalid client id "abcdefghijklmno` +
			`pqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789": ` +
			`label is too long, max: 63`,
		strictSNI: true,
	}, {
		name:         "quic_client_id",
		proto:        proxy.ProtoQUIC,
		hostSrvName:  "example.com",
		cliSrvName:   "cli.example.com",
		wantClientID: "cli",
		wantErrMsg:   "",
		strictSNI:    true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tlsConf := TLSConfig{
				ServerName:     tc.hostSrvName,
				StrictSNICheck: tc.strictSNI,
			}

			srv := &Server{
				conf: ServerConfig{TLSConfig: tlsConf},
			}

			var conn net.Conn
			if tc.proto == proxy.ProtoTLS {
				conn = testTLSConn{
					serverName: tc.cliSrvName,
				}
			}

			var qs quic.Session
			if tc.proto == proxy.ProtoQUIC {
				qs = testQUICSession{
					serverName: tc.cliSrvName,
				}
			}

			pctx := &proxy.DNSContext{
				Proto:       tc.proto,
				Conn:        conn,
				QUICSession: qs,
			}

			clientID, err := srv.clientIDFromDNSContext(pctx)
			assert.Equal(t, tc.wantClientID, clientID)

			if tc.wantErrMsg == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)

				assert.Equal(t, tc.wantErrMsg, err.Error())
			}
		})
	}
}

func TestClientIDFromDNSContextHTTPS(t *testing.T) {
	testCases := []struct {
		name         string
		path         string
		wantClientID string
		wantErrMsg   string
	}{{
		name:         "no_client_id",
		path:         "/dns-query",
		wantClientID: "",
		wantErrMsg:   "",
	}, {
		name:         "no_client_id_slash",
		path:         "/dns-query/",
		wantClientID: "",
		wantErrMsg:   "",
	}, {
		name:         "client_id",
		path:         "/dns-query/cli",
		wantClientID: "cli",
		wantErrMsg:   "",
	}, {
		name:         "client_id_slash",
		path:         "/dns-query/cli/",
		wantClientID: "cli",
		wantErrMsg:   "",
	}, {
		name:         "bad_url",
		path:         "/foo",
		wantClientID: "",
		wantErrMsg:   `client id check: invalid path "/foo"`,
	}, {
		name:         "extra",
		path:         "/dns-query/cli/foo",
		wantClientID: "",
		wantErrMsg:   `client id check: invalid path "/dns-query/cli/foo": extra parts`,
	}, {
		name:         "invalid_client_id",
		path:         "/dns-query/!!!",
		wantClientID: "",
		wantErrMsg: `client id check: invalid client id "!!!": ` +
			`invalid char '!' at index 0`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{
				URL: &url.URL{
					Path: tc.path,
				},
			}

			pctx := &proxy.DNSContext{
				Proto:       proxy.ProtoHTTPS,
				HTTPRequest: r,
			}

			clientID, err := clientIDFromDNSContextHTTPS(pctx)
			assert.Equal(t, tc.wantClientID, clientID)

			if tc.wantErrMsg == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)

				assert.Equal(t, tc.wantErrMsg, err.Error())
			}
		})
	}
}
