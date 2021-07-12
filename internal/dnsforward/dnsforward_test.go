package dnsforward

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	aghtest.DiscardLogOutput(m)
}

const (
	tlsServerName     = "testdns.adguard.com"
	testMessagesCount = 10
)

func startDeferStop(t *testing.T, s *Server) {
	t.Helper()

	err := s.Start()
	require.NoErrorf(t, err, "failed to start server: %s", err)

	t.Cleanup(func() {
		serr := s.Stop()
		require.NoErrorf(t, serr, "dns server failed to stop: %s", serr)
	})
}

func createTestServer(
	t *testing.T,
	filterConf *filtering.Config,
	forwardConf ServerConfig,
	localUps upstream.Upstream,
) (s *Server) {
	t.Helper()

	rules := `||nxdomain.example.org
||NULL.example.org^
127.0.0.1	host.example.org
@@||whitelist.example.org^
||127.0.0.255`
	filters := []filtering.Filter{{
		ID: 0, Data: []byte(rules),
	}}

	f := filtering.New(filterConf, filters)
	f.SetEnabled(true)

	snd, err := aghnet.NewSubnetDetector()
	require.NoError(t, err)
	require.NotNil(t, snd)

	s, err = NewServer(DNSCreateParams{
		DHCPServer:     &testDHCP{},
		DNSFilter:      f,
		SubnetDetector: snd,
	})
	require.NoError(t, err)

	s.conf = forwardConf

	err = s.Prepare(nil)
	require.NoError(t, err)

	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	if localUps != nil {
		s.localResolvers.Config.UpstreamConfig.Upstreams = []upstream.Upstream{localUps}
		s.conf.UsePrivateRDNS = true
	}

	return s
}

func createServerTLSConfig(t *testing.T) (*tls.Config, []byte, []byte) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoErrorf(t, err, "cannot generate RSA key: %s", err)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	require.NoErrorf(t, err, "failed to generate serial number: %s", err)

	notBefore := time.Now()
	notAfter := notBefore.Add(5 * 365 * time.Hour * 24)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"AdGuard Tests"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	template.DNSNames = append(template.DNSNames, tlsServerName)

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(privateKey), privateKey)
	require.NoErrorf(t, err, "failed to create certificate: %s", err)

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	cert, err := tls.X509KeyPair(certPem, keyPem)
	require.NoErrorf(t, err, "failed to create certificate: %s", err)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   tlsServerName,
		MinVersion:   tls.VersionTLS12,
	}, certPem, keyPem
}

func createTestTLS(t *testing.T, tlsConf TLSConfig) (s *Server, certPem []byte) {
	t.Helper()

	var keyPem []byte
	_, certPem, keyPem = createServerTLSConfig(t)

	s = createTestServer(t, &filtering.Config{}, ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
	}, nil)

	tlsConf.CertificateChainData, tlsConf.PrivateKeyData = certPem, keyPem
	s.conf.TLSConfig = tlsConf

	err := s.Prepare(nil)
	require.NoErrorf(t, err, "failed to prepare server: %s", err)

	return s, certPem
}

func createGoogleATestMessage() *dns.Msg {
	return createTestMessage("google-public-dns-a.google.com.")
}

func createTestMessage(host string) *dns.Msg {
	return &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: true,
		},
		Question: []dns.Question{{
			Name:   host,
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		}},
	}
}

func createTestMessageWithType(host string, qtype uint16) *dns.Msg {
	req := createTestMessage(host)
	req.Question[0].Qtype = qtype

	return req
}

func assertGoogleAResponse(t *testing.T, reply *dns.Msg) {
	assertResponse(t, reply, net.IP{8, 8, 8, 8})
}

func assertResponse(t *testing.T, reply *dns.Msg, ip net.IP) {
	t.Helper()

	require.Lenf(t, reply.Answer, 1, "dns server returned reply with wrong number of answers - %d", len(reply.Answer))

	a, ok := reply.Answer[0].(*dns.A)
	require.Truef(t, ok, "dns server returned wrong answer type instead of A: %v", reply.Answer[0])
	assert.Truef(t, a.A.Equal(ip), "dns server returned wrong answer instead of %s: %s", ip, a.A)
}

// sendTestMessagesAsync sends messages in parallel to check for race issues.
//
//lint:ignore U1000 it's called from the function which is skipped for now.
func sendTestMessagesAsync(t *testing.T, conn *dns.Conn) {
	t.Helper()

	wg := &sync.WaitGroup{}

	for i := 0; i < testMessagesCount; i++ {
		msg := createGoogleATestMessage()
		wg.Add(1)

		go func() {
			defer wg.Done()

			err := conn.WriteMsg(msg)
			require.NoErrorf(t, err, "cannot write message: %s", err)

			res, err := conn.ReadMsg()
			require.NoErrorf(t, err, "cannot read response to message: %s", err)

			assertGoogleAResponse(t, res)
		}()
	}

	wg.Wait()
}

func sendTestMessages(t *testing.T, conn *dns.Conn) {
	t.Helper()

	for i := 0; i < testMessagesCount; i++ {
		req := createGoogleATestMessage()
		err := conn.WriteMsg(req)
		assert.NoErrorf(t, err, "cannot write message #%d: %s", i, err)

		res, err := conn.ReadMsg()
		assert.NoErrorf(t, err, "cannot read response to message #%d: %s", i, err)
		assertGoogleAResponse(t, res)
	}
}

func TestServer(t *testing.T) {
	s := createTestServer(t, &filtering.Config{}, ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
	}, nil)
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&aghtest.TestUpstream{
			IPv4: map[string][]net.IP{
				"google-public-dns-a.google.com.": {{8, 8, 8, 8}},
			},
		},
	}
	startDeferStop(t, s)

	testCases := []struct {
		name  string
		net   string
		proto proxy.Proto
	}{{
		name:  "message_over_udp",
		net:   "",
		proto: proxy.ProtoUDP,
	}, {
		name:  "message_over_tcp",
		net:   "tcp",
		proto: proxy.ProtoTCP,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			addr := s.dnsProxy.Addr(tc.proto)
			client := dns.Client{Net: tc.net}

			reply, _, err := client.Exchange(createGoogleATestMessage(), addr.String())
			require.NoErrorf(t, err, "сouldn't talk to server %s: %s", addr, err)

			assertGoogleAResponse(t, reply)
		})
	}
}

func TestServer_timeout(t *testing.T) {
	const timeout time.Duration = time.Second

	t.Run("custom", func(t *testing.T) {
		srvConf := &ServerConfig{
			UpstreamTimeout: timeout,
		}

		s, err := NewServer(DNSCreateParams{})
		require.NoError(t, err)

		err = s.Prepare(srvConf)
		require.NoError(t, err)

		assert.Equal(t, timeout, s.conf.UpstreamTimeout)
	})

	t.Run("default", func(t *testing.T) {
		s, err := NewServer(DNSCreateParams{})
		require.NoError(t, err)

		err = s.Prepare(nil)
		require.NoError(t, err)

		assert.Equal(t, DefaultTimeout, s.conf.UpstreamTimeout)
	})
}

func TestServerWithProtectionDisabled(t *testing.T) {
	s := createTestServer(t, &filtering.Config{}, ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
	}, nil)
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&aghtest.TestUpstream{
			IPv4: map[string][]net.IP{
				"google-public-dns-a.google.com.": {{8, 8, 8, 8}},
			},
		},
	}
	startDeferStop(t, s)

	// Message over UDP.
	req := createGoogleATestMessage()
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)
	client := &dns.Client{}

	reply, _, err := client.Exchange(req, addr.String())
	require.NoErrorf(t, err, "сouldn't talk to server %s: %s", addr, err)
	assertGoogleAResponse(t, reply)
}

func TestDoTServer(t *testing.T) {
	s, certPem := createTestTLS(t, TLSConfig{
		TLSListenAddrs: []*net.TCPAddr{{}},
	})
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&aghtest.TestUpstream{
			IPv4: map[string][]net.IP{
				"google-public-dns-a.google.com.": {{8, 8, 8, 8}},
			},
		},
	}
	startDeferStop(t, s)

	// Add our self-signed generated config to roots.
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(certPem)
	tlsConfig := &tls.Config{
		ServerName: tlsServerName,
		RootCAs:    roots,
		MinVersion: tls.VersionTLS12,
	}

	// Create a DNS-over-TLS client connection.
	addr := s.dnsProxy.Addr(proxy.ProtoTLS)
	conn, err := dns.DialWithTLS("tcp-tls", addr.String(), tlsConfig)
	require.NoErrorf(t, err, "cannot connect to the proxy: %s", err)

	sendTestMessages(t, conn)
}

func TestDoQServer(t *testing.T) {
	s, _ := createTestTLS(t, TLSConfig{
		QUICListenAddrs: []*net.UDPAddr{{IP: net.IP{127, 0, 0, 1}}},
	})
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&aghtest.TestUpstream{
			IPv4: map[string][]net.IP{
				"google-public-dns-a.google.com.": {{8, 8, 8, 8}},
			},
		},
	}
	startDeferStop(t, s)

	// Create a DNS-over-QUIC upstream.
	addr := s.dnsProxy.Addr(proxy.ProtoQUIC)
	opts := &upstream.Options{InsecureSkipVerify: true}
	u, err := upstream.AddressToUpstream(fmt.Sprintf("%s://%s", proxy.ProtoQUIC, addr), opts)
	require.NoError(t, err)

	// Send the test message.
	req := createGoogleATestMessage()
	res, err := u.Exchange(req)
	require.NoError(t, err)

	assertGoogleAResponse(t, res)
}

func TestServerRace(t *testing.T) {
	t.Skip("TODO(e.burkov): inspect the golibs/cache package for locks")

	filterConf := &filtering.Config{
		SafeBrowsingEnabled:   true,
		SafeBrowsingCacheSize: 1000,
		SafeSearchEnabled:     true,
		SafeSearchCacheSize:   1000,
		ParentalCacheSize:     1000,
		CacheTime:             30,
	}
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			UpstreamDNS:       []string{"8.8.8.8:53", "8.8.4.4:53"},
		},
		ConfigModified: func() {},
	}
	s := createTestServer(t, filterConf, forwardConf, nil)
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&aghtest.TestUpstream{
			IPv4: map[string][]net.IP{
				"google-public-dns-a.google.com.": {{8, 8, 8, 8}},
			},
		},
	}
	startDeferStop(t, s)

	// Message over UDP.
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)
	conn, err := dns.Dial("udp", addr.String())
	require.NoErrorf(t, err, "cannot connect to the proxy: %s", err)

	sendTestMessagesAsync(t, conn)
}

func TestSafeSearch(t *testing.T) {
	resolver := &aghtest.TestResolver{}
	filterConf := &filtering.Config{
		SafeSearchEnabled:   true,
		SafeSearchCacheSize: 1000,
		CacheTime:           30,
		CustomResolver:      resolver,
	}
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
		},
	}
	s := createTestServer(t, filterConf, forwardConf, nil)
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP).String()
	client := &dns.Client{}

	yandexIP := net.IP{213, 180, 193, 56}
	googleIP, _ := resolver.HostToIPs("forcesafesearch.google.com")

	testCases := []struct {
		host string
		want net.IP
	}{{
		host: "yandex.com.",
		want: yandexIP,
	}, {
		host: "yandex.by.",
		want: yandexIP,
	}, {
		host: "yandex.kz.",
		want: yandexIP,
	}, {
		host: "yandex.ru.",
		want: yandexIP,
	}, {
		host: "www.google.com.",
		want: googleIP,
	}, {
		host: "www.google.com.af.",
		want: googleIP,
	}, {
		host: "www.google.be.",
		want: googleIP,
	}, {
		host: "www.google.by.",
		want: googleIP,
	}}

	for _, tc := range testCases {
		t.Run(tc.host, func(t *testing.T) {
			req := createTestMessage(tc.host)

			reply, _, err := client.Exchange(req, addr)
			require.NoErrorf(t, err, "couldn't talk to server %s: %s", addr, err)
			assertResponse(t, reply, tc.want)
		})
	}
}

func TestInvalidRequest(t *testing.T) {
	s := createTestServer(t, &filtering.Config{}, ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
	}, nil)
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP).String()
	req := dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: true,
		},
	}

	// Send a DNS request without question.
	_, _, err := (&dns.Client{
		Timeout: 500 * time.Millisecond,
	}).Exchange(&req, addr)

	assert.NoErrorf(t, err, "got a response to an invalid query")
}

func TestBlockedRequest(t *testing.T) {
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			BlockingMode:      BlockingModeDefault,
		},
	}
	s := createTestServer(t, &filtering.Config{}, forwardConf, nil)
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// Default blocking.
	req := createTestMessage("nxdomain.example.org.")

	reply, err := dns.Exchange(req, addr.String())
	require.NoErrorf(t, err, "couldn't talk to server %s: %s", addr, err)

	assert.Equal(t, dns.RcodeSuccess, reply.Rcode)

	require.Len(t, reply.Answer, 1)
	assert.True(t, reply.Answer[0].(*dns.A).A.IsUnspecified())
}

func TestServerCustomClientUpstream(t *testing.T) {
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
		},
	}
	s := createTestServer(t, &filtering.Config{}, forwardConf, nil)
	s.conf.GetCustomUpstreamByClient = func(_ string) (conf *proxy.UpstreamConfig, err error) {
		ups := &aghtest.TestUpstream{
			IPv4: map[string][]net.IP{
				"host.": {{192, 168, 0, 1}},
			},
		}

		return &proxy.UpstreamConfig{
			Upstreams: []upstream.Upstream{ups},
		}, nil
	}
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// Send test request.
	req := createTestMessage("host.")

	reply, err := dns.Exchange(req, addr.String())
	require.NoError(t, err)

	assert.Equal(t, dns.RcodeSuccess, reply.Rcode)
	require.NotEmpty(t, reply.Answer)

	require.Len(t, reply.Answer, 1)
	assert.Equal(t, net.IP{192, 168, 0, 1}, reply.Answer[0].(*dns.A).A)
}

// testCNAMEs is a map of names and CNAMEs necessary for the TestUpstream work.
var testCNAMEs = map[string]string{
	"badhost.":               "NULL.example.org.",
	"whitelist.example.org.": "NULL.example.org.",
}

// testIPv4 is a map of names and IPv4s necessary for the TestUpstream work.
var testIPv4 = map[string][]net.IP{
	"NULL.example.org.": {{1, 2, 3, 4}},
	"example.org.":      {{127, 0, 0, 255}},
}

func TestBlockCNAMEProtectionEnabled(t *testing.T) {
	s := createTestServer(t, &filtering.Config{}, ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
	}, nil)
	testUpstm := &aghtest.TestUpstream{
		CName: testCNAMEs,
		IPv4:  testIPv4,
		IPv6:  nil,
	}
	s.conf.ProtectionEnabled = false
	s.dnsProxy.UpstreamConfig = &proxy.UpstreamConfig{
		Upstreams: []upstream.Upstream{testUpstm},
	}
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// 'badhost' has a canonical name 'NULL.example.org' which should be
	// blocked by filters, but protection is disabled so it is not.
	req := createTestMessage("badhost.")

	reply, err := dns.Exchange(req, addr.String())
	require.NoError(t, err)

	assert.Equal(t, dns.RcodeSuccess, reply.Rcode)
}

func TestBlockCNAME(t *testing.T) {
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			BlockingMode:      BlockingModeDefault,
		},
	}
	s := createTestServer(t, &filtering.Config{}, forwardConf, nil)
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&aghtest.TestUpstream{
			CName: testCNAMEs,
			IPv4:  testIPv4,
		},
	}
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP).String()

	testCases := []struct {
		host string
		want bool
	}{{
		host: "badhost.",
		// 'badhost' has a canonical name 'NULL.example.org' which is
		// blocked by filters: response is blocked.
		want: true,
	}, {
		host: "whitelist.example.org.",
		// 'whitelist.example.org' has a canonical name
		// 'NULL.example.org' which is blocked by filters
		// but 'whitelist.example.org' is in a whitelist:
		// response isn't blocked.
		want: false,
	}, {
		host: "example.org.",
		// 'example.org' has a canonical name 'cname1' with IP
		// 127.0.0.255 which is blocked by filters: response is blocked.
		want: true,
	}}

	for _, tc := range testCases {
		t.Run("block_cname_"+tc.host, func(t *testing.T) {
			req := createTestMessage(tc.host)

			reply, err := dns.Exchange(req, addr)
			require.NoError(t, err)

			assert.Equal(t, dns.RcodeSuccess, reply.Rcode)
			if tc.want {
				require.Len(t, reply.Answer, 1)

				ans := reply.Answer[0]
				a, ok := ans.(*dns.A)
				require.Truef(t, ok, "got %T", ans)

				assert.True(t, a.A.IsUnspecified())
			}
		})
	}
}

func TestClientRulesForCNAMEMatching(t *testing.T) {
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			FilterHandler: func(_ net.IP, _ string, settings *filtering.Settings) {
				settings.FilteringEnabled = false
			},
		},
	}
	s := createTestServer(t, &filtering.Config{}, forwardConf, nil)
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&aghtest.TestUpstream{
			CName: testCNAMEs,
			IPv4:  testIPv4,
		},
	}
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// 'badhost' has a canonical name 'NULL.example.org' which is blocked by
	// filters: response is blocked.
	req := dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id: dns.Id(),
		},
		Question: []dns.Question{{
			Name:   "badhost.",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		}},
	}

	// However, in our case it should not be blocked as filtering is
	// disabled on the client level.
	reply, err := dns.Exchange(&req, addr.String())
	require.NoError(t, err)

	assert.Equal(t, dns.RcodeSuccess, reply.Rcode)
}

func TestNullBlockedRequest(t *testing.T) {
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			BlockingMode:      BlockingModeNullIP,
		},
	}
	s := createTestServer(t, &filtering.Config{}, forwardConf, nil)
	startDeferStop(t, s)
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// Nil filter blocking.
	req := dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: true,
		},
		Question: []dns.Question{{
			Name:   "NULL.example.org.",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		}},
	}

	reply, err := dns.Exchange(&req, addr.String())
	require.NoErrorf(t, err, "couldn't talk to server %s: %s", addr, err)
	require.Lenf(t, reply.Answer, 1, "dns server %s returned reply with wrong number of answers - %d", addr, len(reply.Answer))
	a, ok := reply.Answer[0].(*dns.A)
	require.Truef(t, ok, "dns server %s returned wrong answer type instead of A: %v", addr, reply.Answer[0])
	assert.Truef(t, a.A.IsUnspecified(), "dns server %s returned wrong answer instead of 0.0.0.0: %v", addr, a.A)
}

func TestBlockedCustomIP(t *testing.T) {
	rules := "||nxdomain.example.org^\n||NULL.example.org^\n127.0.0.1	host.example.org\n@@||whitelist.example.org^\n||127.0.0.255\n"
	filters := []filtering.Filter{{
		ID:   0,
		Data: []byte(rules),
	}}

	snd, err := aghnet.NewSubnetDetector()
	require.NoError(t, err)
	require.NotNil(t, snd)

	f := filtering.New(&filtering.Config{}, filters)
	var s *Server
	s, err = NewServer(DNSCreateParams{
		DHCPServer:     &testDHCP{},
		DNSFilter:      f,
		SubnetDetector: snd,
	})
	require.NoError(t, err)

	conf := &ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			BlockingMode:      BlockingModeCustomIP,
			BlockingIPv4:      nil,
			UpstreamDNS:       []string{"8.8.8.8:53", "8.8.4.4:53"},
		},
	}

	// Invalid BlockingIPv4.
	err = s.Prepare(conf)
	assert.Error(t, err)

	conf.BlockingIPv4 = net.IP{0, 0, 0, 1}
	conf.BlockingIPv6 = net.ParseIP("::1")

	err = s.Prepare(conf)
	require.NoError(t, err)

	f.SetEnabled(true)
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	req := createTestMessageWithType("NULL.example.org.", dns.TypeA)
	reply, err := dns.Exchange(req, addr.String())
	require.NoError(t, err)

	require.Len(t, reply.Answer, 1)

	a, ok := reply.Answer[0].(*dns.A)
	require.True(t, ok)

	assert.True(t, net.IP{0, 0, 0, 1}.Equal(a.A))

	req = createTestMessageWithType("NULL.example.org.", dns.TypeAAAA)
	reply, err = dns.Exchange(req, addr.String())
	require.NoError(t, err)

	require.Len(t, reply.Answer, 1)

	a6, ok := reply.Answer[0].(*dns.AAAA)
	require.True(t, ok)

	assert.Equal(t, "::1", a6.AAAA.String())
}

func TestBlockedByHosts(t *testing.T) {
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			BlockingMode:      BlockingModeDefault,
		},
	}

	s := createTestServer(t, &filtering.Config{}, forwardConf, nil)
	startDeferStop(t, s)
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// Hosts blocking.
	req := createTestMessage("host.example.org.")

	reply, err := dns.Exchange(req, addr.String())
	require.NoErrorf(t, err, "couldn't talk to server %s: %s", addr, err)
	require.Lenf(t, reply.Answer, 1, "dns server %s returned reply with wrong number of answers - %d", addr, len(reply.Answer))
	a, ok := reply.Answer[0].(*dns.A)
	require.Truef(t, ok, "dns server %s returned wrong answer type instead of A: %v", addr, reply.Answer[0])
	assert.Equalf(t, net.IP{127, 0, 0, 1}, a.A, "dns server %s returned wrong answer instead of 8.8.8.8: %v", addr, a.A)
}

func TestBlockedBySafeBrowsing(t *testing.T) {
	const hostname = "wmconvirus.narod.ru"

	sbUps := &aghtest.TestBlockUpstream{
		Hostname: hostname,
		Block:    true,
	}
	ans4, _ := (&aghtest.TestResolver{}).HostToIPs(hostname)

	filterConf := &filtering.Config{
		SafeBrowsingEnabled: true,
	}
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		FilteringConfig: FilteringConfig{
			SafeBrowsingBlockHost: ans4.String(),
			ProtectionEnabled:     true,
		},
	}
	s := createTestServer(t, filterConf, forwardConf, nil)
	s.dnsFilter.SetSafeBrowsingUpstream(sbUps)
	startDeferStop(t, s)
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// SafeBrowsing blocking.
	req := createTestMessage(hostname + ".")

	reply, err := dns.Exchange(req, addr.String())
	require.NoErrorf(t, err, "couldn't talk to server %s: %s", addr, err)
	require.Lenf(t, reply.Answer, 1, "dns server %s returned reply with wrong number of answers - %d", addr, len(reply.Answer))

	a, ok := reply.Answer[0].(*dns.A)
	require.Truef(t, ok, "dns server %s returned wrong answer type instead of A: %v", addr, reply.Answer[0])
	assert.Equal(t, ans4, a.A, "dns server %s returned wrong answer: %v", addr, a.A)
}

func TestRewrite(t *testing.T) {
	c := &filtering.Config{
		Rewrites: []filtering.RewriteEntry{{
			Domain: "test.com",
			Answer: "1.2.3.4",
			Type:   dns.TypeA,
		}, {
			Domain: "alias.test.com",
			Answer: "test.com",
			Type:   dns.TypeCNAME,
		}, {
			Domain: "my.alias.example.org",
			Answer: "example.org",
			Type:   dns.TypeCNAME,
		}},
	}
	f := filtering.New(c, nil)

	snd, err := aghnet.NewSubnetDetector()
	require.NoError(t, err)
	require.NotNil(t, snd)

	var s *Server
	s, err = NewServer(DNSCreateParams{
		DHCPServer:     &testDHCP{},
		DNSFilter:      f,
		SubnetDetector: snd,
	})
	require.NoError(t, err)

	assert.NoError(t, s.Prepare(&ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			UpstreamDNS:       []string{"8.8.8.8:53"},
		},
	}))

	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&aghtest.TestUpstream{
			CName: map[string]string{
				"example.org": "somename",
			},
			IPv4: map[string][]net.IP{
				"example.org.": {{4, 3, 2, 1}},
			},
		},
	}
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	req := createTestMessageWithType("test.com.", dns.TypeA)
	reply, err := dns.Exchange(req, addr.String())
	require.NoError(t, err)

	require.Len(t, reply.Answer, 1)

	a, ok := reply.Answer[0].(*dns.A)
	require.True(t, ok)

	assert.True(t, net.IP{1, 2, 3, 4}.Equal(a.A))

	req = createTestMessageWithType("test.com.", dns.TypeAAAA)
	reply, err = dns.Exchange(req, addr.String())
	require.NoError(t, err)

	assert.Empty(t, reply.Answer)

	req = createTestMessageWithType("alias.test.com.", dns.TypeA)
	reply, err = dns.Exchange(req, addr.String())
	require.NoError(t, err)

	require.Len(t, reply.Answer, 2)

	assert.Equal(t, "test.com.", reply.Answer[0].(*dns.CNAME).Target)
	assert.True(t, net.IP{1, 2, 3, 4}.Equal(reply.Answer[1].(*dns.A).A))

	req = createTestMessageWithType("my.alias.example.org.", dns.TypeA)
	reply, err = dns.Exchange(req, addr.String())
	require.NoError(t, err)

	// The original question is restored.
	require.Len(t, reply.Question, 1)

	assert.Equal(t, "my.alias.example.org.", reply.Question[0].Name)

	require.Len(t, reply.Answer, 2)

	assert.Equal(t, "example.org.", reply.Answer[0].(*dns.CNAME).Target)
	assert.Equal(t, dns.TypeA, reply.Answer[1].Header().Rrtype)
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey

	case *ecdsa.PrivateKey:
		return &k.PublicKey

	default:
		return nil
	}
}

type testDHCP struct{}

func (d *testDHCP) Enabled() (ok bool) { return true }

func (d *testDHCP) Leases(flags dhcpd.GetLeasesFlags) (leases []*dhcpd.Lease) {
	return []*dhcpd.Lease{{
		IP:       net.IP{192, 168, 12, 34},
		HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		Hostname: "myhost",
	}}
}

func (d *testDHCP) SetOnLeaseChanged(onLeaseChanged dhcpd.OnLeaseChangedT) {}

func TestPTRResponseFromDHCPLeases(t *testing.T) {
	snd, err := aghnet.NewSubnetDetector()
	require.NoError(t, err)
	require.NotNil(t, snd)

	var s *Server
	s, err = NewServer(DNSCreateParams{
		DNSFilter:      filtering.New(&filtering.Config{}, nil),
		DHCPServer:     &testDHCP{},
		SubnetDetector: snd,
	})
	require.NoError(t, err)

	s.conf.UDPListenAddrs = []*net.UDPAddr{{}}
	s.conf.TCPListenAddrs = []*net.TCPAddr{{}}
	s.conf.UpstreamDNS = []string{"127.0.0.1:53"}
	s.conf.FilteringConfig.ProtectionEnabled = true

	err = s.Prepare(nil)
	require.NoError(t, err)

	err = s.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		s.Close()
	})

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)
	req := createTestMessageWithType("34.12.168.192.in-addr.arpa.", dns.TypePTR)

	resp, err := dns.Exchange(req, addr.String())
	require.NoError(t, err)

	require.Len(t, resp.Answer, 1)

	assert.Equal(t, dns.TypePTR, resp.Answer[0].Header().Rrtype)
	assert.Equal(t, "34.12.168.192.in-addr.arpa.", resp.Answer[0].Header().Name)

	ptr, ok := resp.Answer[0].(*dns.PTR)
	require.True(t, ok)
	assert.Equal(t, "myhost.", ptr.Ptr)
}

func TestPTRResponseFromHosts(t *testing.T) {
	c := filtering.Config{
		EtcHosts: &aghnet.EtcHostsContainer{},
	}

	// Prepare test hosts file.
	hf, err := os.CreateTemp("", "")
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, hf.Close())
		assert.NoError(t, os.Remove(hf.Name()))
	})

	_, _ = hf.WriteString("  127.0.0.1   host # comment \n")
	_, _ = hf.WriteString("  ::1   localhost#comment  \n")

	c.EtcHosts.Init(hf.Name())
	t.Cleanup(c.EtcHosts.Close)

	var snd *aghnet.SubnetDetector
	snd, err = aghnet.NewSubnetDetector()
	require.NoError(t, err)
	require.NotNil(t, snd)

	var s *Server
	s, err = NewServer(DNSCreateParams{
		DHCPServer:     &testDHCP{},
		DNSFilter:      filtering.New(&c, nil),
		SubnetDetector: snd,
	})
	require.NoError(t, err)

	s.conf.UDPListenAddrs = []*net.UDPAddr{{}}
	s.conf.TCPListenAddrs = []*net.TCPAddr{{}}
	s.conf.UpstreamDNS = []string{"127.0.0.1:53"}
	s.conf.FilteringConfig.ProtectionEnabled = true

	err = s.Prepare(nil)
	require.NoError(t, err)

	err = s.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		s.Close()
	})

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)
	req := createTestMessageWithType("1.0.0.127.in-addr.arpa.", dns.TypePTR)

	resp, err := dns.Exchange(req, addr.String())
	require.NoError(t, err)

	require.Len(t, resp.Answer, 1)

	assert.Equal(t, dns.TypePTR, resp.Answer[0].Header().Rrtype)
	assert.Equal(t, "1.0.0.127.in-addr.arpa.", resp.Answer[0].Header().Name)

	ptr, ok := resp.Answer[0].(*dns.PTR)
	require.True(t, ok)
	assert.Equal(t, "host.", ptr.Ptr)
}

func TestNewServer(t *testing.T) {
	testCases := []struct {
		name       string
		in         DNSCreateParams
		wantErrMsg string
	}{{
		name:       "success",
		in:         DNSCreateParams{},
		wantErrMsg: "",
	}, {
		name: "success_local_tld",
		in: DNSCreateParams{
			LocalDomain: "mynet",
		},
		wantErrMsg: "",
	}, {
		name: "success_local_domain",
		in: DNSCreateParams{
			LocalDomain: "my.local.net",
		},
		wantErrMsg: "",
	}, {
		name: "bad_local_domain",
		in: DNSCreateParams{
			LocalDomain: "!!!",
		},
		wantErrMsg: `local domain: validating domain name "!!!": ` +
			`invalid domain name label at index 0: ` +
			`validating label "!!!": invalid char '!' at index 0`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewServer(tc.in)
			if tc.wantErrMsg == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.wantErrMsg, err.Error())
			}
		})
	}
}

func TestServer_Exchange(t *testing.T) {
	extUpstream := &aghtest.TestUpstream{
		Reverse: map[string][]string{
			"1.1.1.1.in-addr.arpa.": {"one.one.one.one"},
		},
	}
	locUpstream := &aghtest.TestUpstream{
		Reverse: map[string][]string{
			"1.1.168.192.in-addr.arpa.": {"local.domain"},
			"2.1.168.192.in-addr.arpa.": {},
		},
	}
	upstreamErr := errors.Error("upstream error")
	errUpstream := &aghtest.TestErrUpstream{
		Err: upstreamErr,
	}
	nonPtrUpstream := &aghtest.TestBlockUpstream{
		Hostname: "some-host",
		Block:    true,
	}

	srv := NewCustomServer(&proxy.Proxy{
		Config: proxy.Config{
			UpstreamConfig: &proxy.UpstreamConfig{
				Upstreams: []upstream.Upstream{extUpstream},
			},
		},
	})
	srv.conf.ResolveClients = true
	srv.conf.UsePrivateRDNS = true

	var err error
	srv.subnetDetector, err = aghnet.NewSubnetDetector()
	require.NoError(t, err)

	localIP := net.IP{192, 168, 1, 1}
	testCases := []struct {
		name        string
		want        string
		wantErr     error
		locUpstream upstream.Upstream
		req         net.IP
	}{{
		name:        "external_good",
		want:        "one.one.one.one",
		wantErr:     nil,
		locUpstream: nil,
		req:         net.IP{1, 1, 1, 1},
	}, {
		name:        "local_good",
		want:        "local.domain",
		wantErr:     nil,
		locUpstream: locUpstream,
		req:         localIP,
	}, {
		name:        "upstream_error",
		want:        "",
		wantErr:     upstreamErr,
		locUpstream: errUpstream,
		req:         localIP,
	}, {
		name:        "empty_answer_error",
		want:        "",
		wantErr:     rDNSEmptyAnswerErr,
		locUpstream: locUpstream,
		req:         net.IP{192, 168, 1, 2},
	}, {
		name:        "not_ptr_error",
		want:        "",
		wantErr:     rDNSNotPTRErr,
		locUpstream: nonPtrUpstream,
		req:         localIP,
	}}

	for _, tc := range testCases {
		pcfg := proxy.Config{
			UpstreamConfig: &proxy.UpstreamConfig{
				Upstreams: []upstream.Upstream{tc.locUpstream},
			},
		}
		srv.localResolvers = &proxy.Proxy{
			Config: pcfg,
		}

		t.Run(tc.name, func(t *testing.T) {
			host, eerr := srv.Exchange(tc.req)

			require.ErrorIs(t, eerr, tc.wantErr)
			assert.Equal(t, tc.want, host)
		})
	}

	t.Run("resolving_disabled", func(t *testing.T) {
		srv.conf.UsePrivateRDNS = false

		host, eerr := srv.Exchange(localIP)

		require.NoError(t, eerr)
		assert.Empty(t, host)
	})
}
