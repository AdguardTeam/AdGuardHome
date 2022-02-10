package dnsforward

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/lucas-clemente/quic-go"
	"github.com/stretchr/testify/assert"
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
	// Session is embedded here simply to make testQUICSession a quic.Session
	// without actually implementing all methods.
	quic.Session

	serverName string
}

// ConnectionState implements the quicSession interface for testQUICSession.
func (c testQUICSession) ConnectionState() (cs quic.ConnectionState) {
	cs.TLS.ServerName = c.serverName

	return cs
}

func TestServer_clientIDFromDNSContext(t *testing.T) {
	// TODO(a.garipov): Consider moving away from the text-based error
	// checks and onto a more structured approach.
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
		name:         "tls_no_clientid",
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
		wantErrMsg: `clientid check: client server name "" ` +
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
		name:         "tls_clientid",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "cli.example.com",
		wantClientID: "cli",
		wantErrMsg:   "",
		strictSNI:    true,
	}, {
		name:         "tls_clientid_hostname_error",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "cli.example.net",
		wantClientID: "",
		wantErrMsg: `clientid check: client server name "cli.example.net" ` +
			`doesn't match host server name "example.com"`,
		strictSNI: true,
	}, {
		name:         "tls_invalid_clientid",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "!!!.example.com",
		wantClientID: "",
		wantErrMsg: `clientid check: invalid clientid "!!!": ` +
			`bad domain name label rune '!'`,
		strictSNI: true,
	}, {
		name:        "tls_clientid_too_long",
		proto:       proxy.ProtoTLS,
		hostSrvName: "example.com",
		cliSrvName: `abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmno` +
			`pqrstuvwxyz0123456789.example.com`,
		wantClientID: "",
		wantErrMsg: `clientid check: invalid clientid "abcdefghijklmno` +
			`pqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789": ` +
			`domain name label is too long: got 72, max 63`,
		strictSNI: true,
	}, {
		name:         "quic_clientid",
		proto:        proxy.ProtoQUIC,
		hostSrvName:  "example.com",
		cliSrvName:   "cli.example.com",
		wantClientID: "cli",
		wantErrMsg:   "",
		strictSNI:    true,
	}, {
		name:         "tls_clientid_issue3437",
		proto:        proxy.ProtoTLS,
		hostSrvName:  "example.com",
		cliSrvName:   "cli.myexample.com",
		wantClientID: "",
		wantErrMsg: `clientid check: client server name "cli.myexample.com" ` +
			`doesn't match host server name "example.com"`,
		strictSNI: true,
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

			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
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
		name:         "no_clientid",
		path:         "/dns-query",
		wantClientID: "",
		wantErrMsg:   "",
	}, {
		name:         "no_clientid_slash",
		path:         "/dns-query/",
		wantClientID: "",
		wantErrMsg:   "",
	}, {
		name:         "clientid",
		path:         "/dns-query/cli",
		wantClientID: "cli",
		wantErrMsg:   "",
	}, {
		name:         "clientid_slash",
		path:         "/dns-query/cli/",
		wantClientID: "cli",
		wantErrMsg:   "",
	}, {
		name:         "bad_url",
		path:         "/foo",
		wantClientID: "",
		wantErrMsg:   `clientid check: invalid path "/foo"`,
	}, {
		name:         "extra",
		path:         "/dns-query/cli/foo",
		wantClientID: "",
		wantErrMsg:   `clientid check: invalid path "/dns-query/cli/foo": extra parts`,
	}, {
		name:         "invalid_clientid",
		path:         "/dns-query/!!!",
		wantClientID: "",
		wantErrMsg:   `clientid check: invalid clientid "!!!": bad domain name label rune '!'`,
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

			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}
