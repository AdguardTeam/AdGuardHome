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
)

// testTLSConn is a tlsConn for tests.
type testTLSConn struct {
	// Conn is embedded here simply to make testTLSConn a net.Conn without
	// acctually implementing all methods.
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
	cs.ServerName = c.serverName

	return cs
}

func TestProcessClientID(t *testing.T) {
	testCases := []struct {
		name         string
		proto        string
		hostSrvName  string
		cliSrvName   string
		wantClientID string
		wantErrMsg   string
		wantRes      resultCode
	}{{
		name:         "udp",
		proto:        proxy.ProtoUDP,
		hostSrvName:  "",
		cliSrvName:   "",
		wantClientID: "",
		wantErrMsg:   "",
		wantRes:      resultCodeSuccess,
	}, {
		name:         "tls_no_client_id",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "example.com",
		wantClientID: "",
		wantErrMsg:   "",
		wantRes:      resultCodeSuccess,
	}, {
		name:         "tls_client_id",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "cli.example.com",
		wantClientID: "cli",
		wantErrMsg:   "",
		wantRes:      resultCodeSuccess,
	}, {
		name:         "tls_client_id_hostname_error",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "cli.example.net",
		wantClientID: "",
		wantErrMsg:   `client id check: client server name "cli.example.net" doesn't match host server name "example.com"`,
		wantRes:      resultCodeError,
	}, {
		name:         "tls_invalid_client_id",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "!!!.example.com",
		wantClientID: "",
		wantErrMsg:   `client id check: invalid client id: invalid char '!' at index 0 in client id "!!!"`,
		wantRes:      resultCodeError,
	}, {
		name:         "tls_client_id_too_long",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789.example.com",
		wantClientID: "",
		wantErrMsg:   `client id check: invalid client id: client id "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789" is too long, max: 64`,
		wantRes:      resultCodeError,
	}, {
		name:         "quic_client_id",
		proto:        proxy.ProtoQUIC,
		hostSrvName:  "example.com",
		cliSrvName:   "cli.example.com",
		wantClientID: "cli",
		wantErrMsg:   "",
		wantRes:      resultCodeSuccess,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := &Server{
				conf: ServerConfig{
					TLSConfig: TLSConfig{ServerName: tc.hostSrvName},
				},
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

			dctx := &dnsContext{
				srv: srv,
				proxyCtx: &proxy.DNSContext{
					Proto:       tc.proto,
					Conn:        conn,
					QUICSession: qs,
				},
			}

			res := processClientID(dctx)
			assert.Equal(t, tc.wantRes, res)
			assert.Equal(t, tc.wantClientID, dctx.clientID)

			if tc.wantErrMsg != "" && assert.NotNil(t, dctx.err) {
				assert.Equal(t, tc.wantErrMsg, dctx.err.Error())
			} else {
				assert.Nil(t, dctx.err)
			}
		})
	}
}

func TestProcessClientID_https(t *testing.T) {
	testCases := []struct {
		name         string
		path         string
		wantClientID string
		wantErrMsg   string
		wantRes      resultCode
	}{{
		name:         "no_client_id",
		path:         "/dns-query",
		wantClientID: "",
		wantErrMsg:   "",
		wantRes:      resultCodeSuccess,
	}, {
		name:         "no_client_id_slash",
		path:         "/dns-query/",
		wantClientID: "",
		wantErrMsg:   "",
		wantRes:      resultCodeSuccess,
	}, {
		name:         "client_id",
		path:         "/dns-query/cli",
		wantClientID: "cli",
		wantErrMsg:   "",
		wantRes:      resultCodeSuccess,
	}, {
		name:         "client_id_slash",
		path:         "/dns-query/cli/",
		wantClientID: "cli",
		wantErrMsg:   "",
		wantRes:      resultCodeSuccess,
	}, {
		name:         "bad_url",
		path:         "/foo",
		wantClientID: "",
		wantErrMsg:   `client id check: invalid path "/foo"`,
		wantRes:      resultCodeError,
	}, {
		name:         "extra",
		path:         "/dns-query/cli/foo",
		wantClientID: "",
		wantErrMsg:   `client id check: invalid path "/dns-query/cli/foo": extra parts`,
		wantRes:      resultCodeError,
	}, {
		name:         "invalid_client_id",
		path:         "/dns-query/!!!",
		wantClientID: "",
		wantErrMsg:   `client id check: invalid client id: invalid char '!' at index 0 in client id "!!!"`,
		wantRes:      resultCodeError,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{
				URL: &url.URL{
					Path: tc.path,
				},
			}

			dctx := &dnsContext{
				proxyCtx: &proxy.DNSContext{
					Proto:       proxy.ProtoHTTPS,
					HTTPRequest: r,
				},
			}

			res := processClientID(dctx)
			assert.Equal(t, tc.wantRes, res)
			assert.Equal(t, tc.wantClientID, dctx.clientID)

			if tc.wantErrMsg != "" && assert.NotNil(t, dctx.err) {
				assert.Equal(t, tc.wantErrMsg, dctx.err.Error())
			} else {
				assert.Nil(t, dctx.err)
			}
		})
	}
}
