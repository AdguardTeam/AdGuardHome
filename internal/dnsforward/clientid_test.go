package dnsforward

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/quic-go/quic-go"
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

// testQUICConnection is a quicConnection for tests.
type testQUICConnection struct {
	// Connection is embedded here simply to make testQUICConnection a
	// quic.Connection without actually implementing all methods.
	quic.Connection

	serverName string
}

// ConnectionState implements the quicConnection interface for
// testQUICConnection.
func (c testQUICConnection) ConnectionState() (cs quic.ConnectionState) {
	cs.TLS.ServerName = c.serverName

	return cs
}

func TestServer_clientIDFromDNSContext(t *testing.T) {
	testCases := []struct {
		name         string
		proto        proxy.Proto
		confSrvName  string
		cliSrvName   string
		wantClientID string
		wantErrMsg   string
		inclHTTPTLS  bool
		strictSNI    bool
	}{{
		name:         "udp",
		proto:        proxy.ProtoUDP,
		confSrvName:  "",
		cliSrvName:   "",
		wantClientID: "",
		wantErrMsg:   "",
		inclHTTPTLS:  false,
		strictSNI:    false,
	}, {
		name:         "tls_no_clientid",
		proto:        proxy.ProtoTLS,
		confSrvName:  "example.com",
		cliSrvName:   "example.com",
		wantClientID: "",
		wantErrMsg:   "",
		inclHTTPTLS:  false,
		strictSNI:    true,
	}, {
		name:         "tls_no_client_server_name",
		proto:        proxy.ProtoTLS,
		confSrvName:  "example.com",
		cliSrvName:   "",
		wantClientID: "",
		wantErrMsg: `clientid check: client server name "" ` +
			`doesn't match host server name "example.com"`,
		inclHTTPTLS: false,
		strictSNI:   true,
	}, {
		name:         "tls_no_client_server_name_no_strict",
		proto:        proxy.ProtoTLS,
		confSrvName:  "example.com",
		cliSrvName:   "",
		wantClientID: "",
		wantErrMsg:   "",
		inclHTTPTLS:  false,
		strictSNI:    false,
	}, {
		name:         "tls_clientid",
		proto:        proxy.ProtoTLS,
		confSrvName:  "example.com",
		cliSrvName:   "cli.example.com",
		wantClientID: "cli",
		wantErrMsg:   "",
		inclHTTPTLS:  false,
		strictSNI:    true,
	}, {
		name:         "tls_clientid_hostname_error",
		proto:        proxy.ProtoTLS,
		confSrvName:  "example.com",
		cliSrvName:   "cli.example.net",
		wantClientID: "",
		wantErrMsg: `clientid check: client server name "cli.example.net" ` +
			`doesn't match host server name "example.com"`,
		inclHTTPTLS: false,
		strictSNI:   true,
	}, {
		name:         "tls_invalid_clientid",
		proto:        proxy.ProtoTLS,
		confSrvName:  "example.com",
		cliSrvName:   "!!!.example.com",
		wantClientID: "",
		wantErrMsg: `clientid check: invalid clientid "!!!": ` +
			`bad hostname label rune '!'`,
		inclHTTPTLS: false,
		strictSNI:   true,
	}, {
		name:        "tls_clientid_too_long",
		proto:       proxy.ProtoTLS,
		confSrvName: "example.com",
		cliSrvName: `abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmno` +
			`pqrstuvwxyz0123456789.example.com`,
		wantClientID: "",
		wantErrMsg: `clientid check: invalid clientid "abcdefghijklmno` +
			`pqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789": ` +
			`hostname label is too long: got 72, max 63`,
		inclHTTPTLS: false,
		strictSNI:   true,
	}, {
		name:         "quic_clientid",
		proto:        proxy.ProtoQUIC,
		confSrvName:  "example.com",
		cliSrvName:   "cli.example.com",
		wantClientID: "cli",
		wantErrMsg:   "",
		inclHTTPTLS:  false,
		strictSNI:    true,
	}, {
		name:         "tls_clientid_issue3437",
		proto:        proxy.ProtoTLS,
		confSrvName:  "example.com",
		cliSrvName:   "cli.myexample.com",
		wantClientID: "",
		wantErrMsg: `clientid check: client server name "cli.myexample.com" ` +
			`doesn't match host server name "example.com"`,
		inclHTTPTLS: false,
		strictSNI:   true,
	}, {
		name:         "tls_case",
		proto:        proxy.ProtoTLS,
		confSrvName:  "example.com",
		cliSrvName:   "InSeNsItIvE.example.com",
		wantClientID: "insensitive",
		wantErrMsg:   ``,
		inclHTTPTLS:  false,
		strictSNI:    true,
	}, {
		name:         "quic_case",
		proto:        proxy.ProtoQUIC,
		confSrvName:  "example.com",
		cliSrvName:   "InSeNsItIvE.example.com",
		wantClientID: "insensitive",
		wantErrMsg:   ``,
		inclHTTPTLS:  false,
		strictSNI:    true,
	}, {
		name:         "https_no_clientid",
		proto:        proxy.ProtoHTTPS,
		confSrvName:  "example.com",
		cliSrvName:   "example.com",
		wantClientID: "",
		wantErrMsg:   "",
		inclHTTPTLS:  true,
		strictSNI:    true,
	}, {
		name:         "https_clientid",
		proto:        proxy.ProtoHTTPS,
		confSrvName:  "example.com",
		cliSrvName:   "cli.example.com",
		wantClientID: "cli",
		wantErrMsg:   "",
		inclHTTPTLS:  true,
		strictSNI:    true,
	}, {
		name:         "https_issue5518",
		proto:        proxy.ProtoHTTPS,
		confSrvName:  "example.com",
		cliSrvName:   "cli.example.com",
		wantClientID: "cli",
		wantErrMsg:   "",
		inclHTTPTLS:  false,
		strictSNI:    true,
	}, {
		name:         "https_no_host",
		proto:        proxy.ProtoHTTPS,
		confSrvName:  "example.com",
		cliSrvName:   "example.com",
		wantClientID: "",
		wantErrMsg:   "",
		inclHTTPTLS:  false,
		strictSNI:    true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tlsConf := TLSConfig{
				ServerName:     tc.confSrvName,
				StrictSNICheck: tc.strictSNI,
			}

			srv := &Server{
				conf:       ServerConfig{TLSConfig: tlsConf},
				baseLogger: slogutil.NewDiscardLogger(),
			}

			var (
				conn    net.Conn
				qconn   quic.Connection
				httpReq *http.Request
			)

			switch tc.proto {
			case proxy.ProtoHTTPS:
				httpReq = newHTTPReq(tc.cliSrvName, tc.inclHTTPTLS)
			case proxy.ProtoQUIC:
				qconn = testQUICConnection{
					serverName: tc.cliSrvName,
				}
			case proxy.ProtoTLS:
				conn = testTLSConn{
					serverName: tc.cliSrvName,
				}
			}

			pctx := &proxy.DNSContext{
				Proto:          tc.proto,
				Conn:           conn,
				HTTPRequest:    httpReq,
				QUICConnection: qconn,
			}

			clientID, err := srv.clientIDFromDNSContext(pctx)
			assert.Equal(t, tc.wantClientID, clientID)

			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

// newHTTPReq is a helper to create HTTP requests for tests.
func newHTTPReq(cliSrvName string, inclTLS bool) (r *http.Request) {
	u := &url.URL{
		Path: "/dns-query",
	}

	r = &http.Request{
		ProtoMajor: 1,
		ProtoMinor: 1,
		URL:        u,
		Host:       cliSrvName,
	}

	if inclTLS {
		r.TLS = &tls.ConnectionState{
			ServerName: cliSrvName,
		}
	}

	return r
}

func TestClientIDFromDNSContextHTTPS(t *testing.T) {
	testCases := []struct {
		name         string
		path         string
		cliSrvName   string
		wantClientID string
		wantErrMsg   string
	}{{
		name:         "no_clientid",
		path:         "/dns-query",
		cliSrvName:   "example.com",
		wantClientID: "",
		wantErrMsg:   "",
	}, {
		name:         "no_clientid_slash",
		path:         "/dns-query/",
		cliSrvName:   "example.com",
		wantClientID: "",
		wantErrMsg:   "",
	}, {
		name:         "clientid",
		path:         "/dns-query/cli",
		cliSrvName:   "example.com",
		wantClientID: "cli",
		wantErrMsg:   "",
	}, {
		name:         "clientid_slash",
		path:         "/dns-query/cli/",
		cliSrvName:   "example.com",
		wantClientID: "cli",
		wantErrMsg:   "",
	}, {
		name:         "clientid_case",
		path:         "/dns-query/InSeNsItIvE",
		cliSrvName:   "example.com",
		wantClientID: "insensitive",
		wantErrMsg:   ``,
	}, {
		name:         "bad_url",
		path:         "/foo",
		cliSrvName:   "example.com",
		wantClientID: "",
		wantErrMsg:   `clientid check: invalid path "/foo"`,
	}, {
		name:         "extra",
		path:         "/dns-query/cli/foo",
		cliSrvName:   "example.com",
		wantClientID: "",
		wantErrMsg:   `clientid check: invalid path "/dns-query/cli/foo": extra parts`,
	}, {
		name:         "invalid_clientid",
		path:         "/dns-query/!!!",
		cliSrvName:   "example.com",
		wantClientID: "",
		wantErrMsg:   `clientid check: invalid clientid "!!!": bad hostname label rune '!'`,
	}, {
		name:         "both_ids",
		path:         "/dns-query/right",
		cliSrvName:   "wrong.example.com",
		wantClientID: "right",
		wantErrMsg:   "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			connState := &tls.ConnectionState{
				ServerName: tc.cliSrvName,
			}

			r := &http.Request{
				URL: &url.URL{
					Path: tc.path,
				},
				TLS: connState,
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
