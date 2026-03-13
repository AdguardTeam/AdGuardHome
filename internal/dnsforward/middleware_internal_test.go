package dnsforward

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Common constants for tests.
const (
	blockedHost = "blockedhost.org"
	testFQDN    = "example.org."

	dnsClientTimeout = 200 * time.Millisecond
)

func TestServer_middlewareTLS(t *testing.T) {
	t.Parallel()

	const clientID = "client-1"

	testCases := []struct {
		clientSrvName     string
		name              string
		host              string
		allowedClients    []string
		disallowedClients []string
		blockedHosts      []string
		wantRCode         int
	}{{
		clientSrvName:     tlsServerName,
		name:              "allow_all",
		host:              testFQDN,
		allowedClients:    []string{},
		disallowedClients: []string{},
		blockedHosts:      []string{},
		wantRCode:         dns.RcodeSuccess,
	}, {
		clientSrvName:     "%" + "." + tlsServerName,
		name:              "invalid_client_id",
		host:              testFQDN,
		allowedClients:    []string{},
		disallowedClients: []string{},
		blockedHosts:      []string{},
		wantRCode:         dns.RcodeServerFailure,
	}, {
		clientSrvName:     clientID + "." + tlsServerName,
		name:              "allowed_client_allowed",
		host:              testFQDN,
		allowedClients:    []string{clientID},
		disallowedClients: []string{},
		blockedHosts:      []string{},
		wantRCode:         dns.RcodeSuccess,
	}, {
		clientSrvName:     "client-2." + tlsServerName,
		name:              "allowed_client_rejected",
		host:              testFQDN,
		allowedClients:    []string{clientID},
		disallowedClients: []string{},
		blockedHosts:      []string{},
		wantRCode:         dns.RcodeRefused,
	}, {
		clientSrvName:     tlsServerName,
		name:              "disallowed_client_allowed",
		host:              testFQDN,
		allowedClients:    []string{},
		disallowedClients: []string{clientID},
		blockedHosts:      []string{},
		wantRCode:         dns.RcodeSuccess,
	}, {
		clientSrvName:     clientID + "." + tlsServerName,
		name:              "disallowed_client_rejected",
		host:              testFQDN,
		allowedClients:    []string{},
		disallowedClients: []string{clientID},
		blockedHosts:      []string{},
		wantRCode:         dns.RcodeRefused,
	}, {
		clientSrvName:     tlsServerName,
		name:              "blocked_hosts_allowed",
		host:              testFQDN,
		allowedClients:    []string{},
		disallowedClients: []string{},
		blockedHosts:      []string{blockedHost},
		wantRCode:         dns.RcodeSuccess,
	}, {
		clientSrvName:     tlsServerName,
		name:              "blocked_hosts_rejected",
		host:              dns.Fqdn(blockedHost),
		allowedClients:    []string{},
		disallowedClients: []string{},
		blockedHosts:      []string{blockedHost},
		wantRCode:         dns.RcodeRefused,
	}}

	localAns := newTestDNSAnswer(testFQDN, net.IP{1, 2, 3, 4})
	localUpsAddr := newTestUpstream(t, localAns)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, _ := createTestTLS(t, &TLSConfig{
				TLSListenAddrs: []*net.TCPAddr{{}},
				ServerName:     tlsServerName,
			})

			s.conf.UpstreamDNS = []string{localUpsAddr}
			s.conf.AllowedClients = tc.allowedClients
			s.conf.DisallowedClients = tc.disallowedClients
			s.conf.BlockedHosts = tc.blockedHosts

			err := s.Prepare(testutil.ContextWithTimeout(t, testTimeout), &s.conf)
			require.NoError(t, err)

			startDeferStop(t, s)

			client := newTestTCPClient(tc.clientSrvName)

			req := createTestMessage(tc.host)
			addr := s.dnsProxy.Addr(proxy.ProtoTLS).String()

			reply, _, err := client.Exchange(req, addr)
			require.NoError(t, err)

			if tc.wantRCode == dns.RcodeSuccess {
				assertSuccessResponse(t, reply, localAns)
			} else {
				assertRejectedResponse(t, reply, tc.wantRCode)
			}
		})
	}
}

func TestServer_middlewareUDP(t *testing.T) {
	t.Parallel()

	const (
		clientIPv4 = "127.0.0.1"
		clientIPv6 = "::1"
	)

	clientIPs := []string{clientIPv4, clientIPv6}

	testCases := []struct {
		name              string
		host              string
		allowedClients    []string
		disallowedClients []string
		blockedHosts      []string
		wantTimeout       bool
	}{{
		name:              "allow_all",
		host:              testFQDN,
		allowedClients:    []string{},
		disallowedClients: []string{},
		blockedHosts:      []string{},
		wantTimeout:       false,
	}, {
		name:              "allowed_client_allowed",
		host:              testFQDN,
		allowedClients:    clientIPs,
		disallowedClients: []string{},
		blockedHosts:      []string{},
		wantTimeout:       false,
	}, {
		name:              "allowed_client_rejected",
		host:              testFQDN,
		allowedClients:    []string{"1:2:3::4"},
		disallowedClients: []string{},
		blockedHosts:      []string{},
		wantTimeout:       true,
	}, {
		name:              "disallowed_client_allowed",
		host:              testFQDN,
		allowedClients:    []string{},
		disallowedClients: []string{"1:2:3::4"},
		blockedHosts:      []string{},
		wantTimeout:       false,
	}, {
		name:              "disallowed_client_rejected",
		host:              testFQDN,
		allowedClients:    []string{},
		disallowedClients: clientIPs,
		blockedHosts:      []string{},
		wantTimeout:       true,
	}, {
		name:              "blocked_hosts_allowed",
		host:              testFQDN,
		allowedClients:    []string{},
		disallowedClients: []string{},
		blockedHosts:      []string{blockedHost},
		wantTimeout:       false,
	}, {
		name:              "blocked_hosts_rejected",
		host:              dns.Fqdn(blockedHost),
		allowedClients:    []string{},
		disallowedClients: []string{},
		blockedHosts:      []string{blockedHost},
		wantTimeout:       true,
	}}

	localAns := newTestDNSAnswer(testFQDN, net.IP{1, 2, 3, 4})
	localUpsAddr := newTestUpstream(t, localAns)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := createTestServer(t, &filtering.Config{
				BlockingMode: filtering.BlockingModeDefault,
			}, ServerConfig{
				UDPListenAddrs: []*net.UDPAddr{{}},
				TCPListenAddrs: []*net.TCPAddr{{}},
				TLSConf:        &TLSConfig{},
				Config: Config{
					AllowedClients:    tc.allowedClients,
					DisallowedClients: tc.disallowedClients,
					BlockedHosts:      tc.blockedHosts,
					UpstreamDNS:       []string{localUpsAddr},
					UpstreamMode:      UpstreamModeLoadBalance,
					EDNSClientSubnet:  &EDNSClientSubnet{Enabled: false},
					ClientsContainer:  EmptyClientsContainer{},
				},
				ServePlainDNS: true,
			})

			startDeferStop(t, s)

			client := &dns.Client{
				Net:     "udp",
				Timeout: dnsClientTimeout,
			}

			req := createTestMessage(tc.host)
			addr := s.dnsProxy.Addr(proxy.ProtoUDP).String()

			reply, _, err := client.Exchange(req, addr)
			if tc.wantTimeout {
				assertTimeoutError(t, err, reply)
			} else {
				require.NoError(t, err)

				assertSuccessResponse(t, reply, localAns)
			}
		})
	}
}

// newTestDNSAnswer creates a standard A record answer for testing.
func newTestDNSAnswer(fqdn string, ip net.IP) (ans []dns.RR) {
	return []dns.RR{&dns.A{
		Hdr: dns.RR_Header{
			Name:     fqdn,
			Rrtype:   dns.TypeA,
			Class:    dns.ClassINET,
			Ttl:      3600,
			Rdlength: 4,
		},
		A: ip,
	}}
}

// newTestUpstream creates a test upstream handler and returns its address.
func newTestUpstream(tb testing.TB, answer []dns.RR) (addr string) {
	tb.Helper()

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		resp := (&dns.Msg{}).SetReply(req)
		resp.Answer = answer

		require.NoError(testutil.PanicT{}, w.WriteMsg(resp))
	})

	return aghtest.StartLocalhostUpstream(tb, handler).String()
}

// newTestTCPClient creates a new TCP client for testing.
func newTestTCPClient(clientSrvName string) (c *dns.Client) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         clientSrvName,
	}

	return &dns.Client{
		Net:       "tcp-tls",
		TLSConfig: tlsConfig,
		Timeout:   dnsClientTimeout,
	}
}

// assertSuccessResponse checks that the response is successful with expected
// answer.
func assertSuccessResponse(tb testing.TB, reply *dns.Msg, expectedAns []dns.RR) {
	tb.Helper()

	require.NotNil(tb, reply)

	assert.Equal(tb, dns.RcodeSuccess, reply.Rcode)
	assert.Equal(tb, expectedAns, reply.Answer)
}

// assertRejectedResponse checks that the response has the expected error code
// and no answer.
func assertRejectedResponse(tb testing.TB, reply *dns.Msg, wantRCode int) {
	tb.Helper()

	require.NotNil(tb, reply)

	assert.Equal(tb, wantRCode, reply.Rcode)
	assert.Empty(tb, reply.Answer)
}

// assertTimeoutError checks that the error is a timeout error and reply is nil.
func assertTimeoutError(tb testing.TB, err error, reply *dns.Msg) {
	tb.Helper()

	wantErr := &net.OpError{}
	require.ErrorAs(tb, err, &wantErr)

	assert.True(tb, wantErr.Timeout())
	assert.Nil(tb, reply)
}
