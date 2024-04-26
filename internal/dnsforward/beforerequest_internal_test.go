package dnsforward

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	blockedHost      = "blockedhost.org"
	testFQDN         = "example.org."
	dnsClientTimeout = 200 * time.Millisecond
)

func TestServer_HandleBefore_tls(t *testing.T) {
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

	localAns := []dns.RR{&dns.A{
		Hdr: dns.RR_Header{
			Name:     testFQDN,
			Rrtype:   dns.TypeA,
			Class:    dns.ClassINET,
			Ttl:      3600,
			Rdlength: 4,
		},
		A: net.IP{1, 2, 3, 4},
	}}
	localUpsHdlr := dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		resp := (&dns.Msg{}).SetReply(req)
		resp.Answer = localAns

		require.NoError(t, w.WriteMsg(resp))
	})
	localUpsAddr := aghtest.StartLocalhostUpstream(t, localUpsHdlr).String()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, _ := createTestTLS(t, TLSConfig{
				TLSListenAddrs: []*net.TCPAddr{{}},
				ServerName:     tlsServerName,
			})

			s.conf.UpstreamDNS = []string{localUpsAddr}

			s.conf.AllowedClients = tc.allowedClients
			s.conf.DisallowedClients = tc.disallowedClients
			s.conf.BlockedHosts = tc.blockedHosts

			err := s.Prepare(&s.conf)
			require.NoError(t, err)

			startDeferStop(t, s)

			tlsConfig := &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         tc.clientSrvName,
			}

			client := &dns.Client{
				Net:       "tcp-tls",
				TLSConfig: tlsConfig,
				Timeout:   dnsClientTimeout,
			}

			req := createTestMessage(tc.host)
			addr := s.dnsProxy.Addr(proxy.ProtoTLS).String()

			reply, _, err := client.Exchange(req, addr)
			require.NoError(t, err)

			assert.Equal(t, tc.wantRCode, reply.Rcode)
			if tc.wantRCode == dns.RcodeSuccess {
				assert.Equal(t, localAns, reply.Answer)
			} else {
				assert.Empty(t, reply.Answer)
			}
		})
	}
}

func TestServer_HandleBefore_udp(t *testing.T) {
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

	localAns := []dns.RR{&dns.A{
		Hdr: dns.RR_Header{
			Name:     testFQDN,
			Rrtype:   dns.TypeA,
			Class:    dns.ClassINET,
			Ttl:      3600,
			Rdlength: 4,
		},
		A: net.IP{1, 2, 3, 4},
	}}
	localUpsHdlr := dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		resp := (&dns.Msg{}).SetReply(req)
		resp.Answer = localAns

		require.NoError(t, w.WriteMsg(resp))
	})
	localUpsAddr := aghtest.StartLocalhostUpstream(t, localUpsHdlr).String()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := createTestServer(t, &filtering.Config{
				BlockingMode: filtering.BlockingModeDefault,
			}, ServerConfig{
				UDPListenAddrs: []*net.UDPAddr{{}},
				TCPListenAddrs: []*net.TCPAddr{{}},
				Config: Config{
					AllowedClients:    tc.allowedClients,
					DisallowedClients: tc.disallowedClients,
					BlockedHosts:      tc.blockedHosts,
					UpstreamDNS:       []string{localUpsAddr},
					UpstreamMode:      UpstreamModeLoadBalance,
					EDNSClientSubnet:  &EDNSClientSubnet{Enabled: false},
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
				wantErr := &net.OpError{}
				require.ErrorAs(t, err, &wantErr)
				assert.True(t, wantErr.Timeout())

				assert.Nil(t, reply)
			} else {
				require.NoError(t, err)
				require.NotNil(t, reply)

				assert.Equal(t, dns.RcodeSuccess, reply.Rcode)
				assert.Equal(t, localAns, reply.Answer)
			}
		})
	}
}
