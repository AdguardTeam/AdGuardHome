package dnsforward

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/testutil"
	"github.com/AdguardTeam/AdGuardHome/internal/util"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

const (
	tlsServerName     = "testdns.adguard.com"
	testMessagesCount = 10
)

func startDeferStop(t *testing.T, s *Server) {
	t.Helper()

	err := s.Start()
	assert.Nilf(t, err, "failed to start server: %s", err)

	t.Cleanup(func() {
		err := s.Stop()
		assert.Nilf(t, err, "dns server failed to stop: %s", err)
	})
}

func TestServer(t *testing.T) {
	s := createTestServer(t)
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&testUpstream{
			ipv4: map[string][]net.IP{
				"google-public-dns-a.google.com.": {{8, 8, 8, 8}},
			},
		},
	}
	startDeferStop(t, s)

	testCases := []struct {
		name  string
		proto string
	}{{
		name:  "message_over_udp",
		proto: proxy.ProtoUDP,
	}, {
		name:  "message_over_tcp",
		proto: proxy.ProtoTCP,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			addr := s.dnsProxy.Addr(tc.proto)
			client := dns.Client{Net: tc.proto}

			reply, _, err := client.Exchange(createGoogleATestMessage(), addr.String())
			assert.Nilf(t, err, "сouldn't talk to server %s: %s", addr, err)

			assertGoogleAResponse(t, reply)
		})
	}
}

func TestServerWithProtectionDisabled(t *testing.T) {
	s := createTestServer(t)
	s.conf.ProtectionEnabled = false
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&testUpstream{
			ipv4: map[string][]net.IP{
				"google-public-dns-a.google.com.": {{8, 8, 8, 8}},
			},
		},
	}
	startDeferStop(t, s)

	// Message over UDP.
	req := createGoogleATestMessage()
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)
	client := dns.Client{Net: proxy.ProtoUDP}
	reply, _, err := client.Exchange(req, addr.String())
	assert.Nilf(t, err, "сouldn't talk to server %s: %s", addr, err)
	assertGoogleAResponse(t, reply)
}

func createTestTLS(t *testing.T, tlsConf TLSConfig) (s *Server, certPem []byte) {
	t.Helper()

	var keyPem []byte
	_, certPem, keyPem = createServerTLSConfig(t)
	s = createTestServer(t)

	tlsConf.CertificateChainData, tlsConf.PrivateKeyData = certPem, keyPem
	s.conf.TLSConfig = tlsConf

	err := s.Prepare(nil)
	assert.Nilf(t, err, "failed to prepare server: %s", err)

	return s, certPem
}

func TestDoTServer(t *testing.T) {
	s, certPem := createTestTLS(t, TLSConfig{
		TLSListenAddr: &net.TCPAddr{Port: 0},
	})
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&testUpstream{
			ipv4: map[string][]net.IP{
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
	assert.Nilf(t, err, "cannot connect to the proxy: %s", err)

	sendTestMessages(t, conn)
}

func TestDoQServer(t *testing.T) {
	s, _ := createTestTLS(t, TLSConfig{
		QUICListenAddr: &net.UDPAddr{Port: 0},
	})
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&testUpstream{
			ipv4: map[string][]net.IP{
				"google-public-dns-a.google.com.": {{8, 8, 8, 8}},
			},
		},
	}
	startDeferStop(t, s)

	// Create a DNS-over-QUIC upstream.
	addr := s.dnsProxy.Addr(proxy.ProtoQUIC)
	opts := upstream.Options{InsecureSkipVerify: true}
	u, err := upstream.AddressToUpstream(fmt.Sprintf("%s://%s", proxy.ProtoQUIC, addr), opts)
	assert.Nil(t, err)

	// Send the test message.
	req := createGoogleATestMessage()
	res, err := u.Exchange(req)
	assert.Nil(t, err)

	assertGoogleAResponse(t, res)
}

func TestServerRace(t *testing.T) {
	t.Skip("TODO(e.burkov): inspect the golibs/cache package for locks")

	s := createTestServer(t)
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&testUpstream{
			ipv4: map[string][]net.IP{
				"google-public-dns-a.google.com.": {{8, 8, 8, 8}},
			},
		},
	}
	startDeferStop(t, s)

	// Message over UDP.
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)
	conn, err := dns.Dial(proxy.ProtoUDP, addr.String())
	assert.Nilf(t, err, "cannot connect to the proxy: %s", err)

	sendTestMessagesAsync(t, conn)
}

// testResolver is a Resolver for tests.
//
//lint:ignore U1000 TODO(e.burkov): move into aghtest package.
type testResolver struct{}

// LookupIPAddr implements Resolver interface for *testResolver.
//
//lint:ignore U1000 TODO(e.burkov): move into aghtest package.
func (r *testResolver) LookupIPAddr(_ context.Context, host string) (ips []net.IPAddr, err error) {
	hash := sha256.Sum256([]byte(host))
	addrs := []net.IPAddr{{
		IP:   net.IP(hash[:4]),
		Zone: "somezone",
	}, {
		IP:   net.IP(hash[4:20]),
		Zone: "somezone",
	}}
	return addrs, nil
}

// LookupHost implements Resolver interface for *testResolver.
//
//lint:ignore U1000 TODO(e.burkov): move into aghtest package.
func (r *testResolver) LookupHost(host string) (addrs []string, err error) {
	hash := sha256.Sum256([]byte(host))
	addrs = []string{
		net.IP(hash[:4]).String(),
		net.IP(hash[4:20]).String(),
	}
	return addrs, nil
}

func TestSafeSearch(t *testing.T) {
	t.Skip("TODO(e.burkov): substitute the dnsfilter by one with custom resolver from aghtest")

	testUpstreamIP := net.IP{213, 180, 193, 56}
	testCases := []string{
		"yandex.com.",
		"yandex.by.",
		"yandex.kz.",
		"yandex.ru.",
		"www.google.com.",
		"www.google.com.af.",
		"www.google.be.",
		"www.google.by.",
	}

	for _, tc := range testCases {
		t.Run("safe_search_"+tc, func(t *testing.T) {
			s := createTestServer(t)
			startDeferStop(t, s)

			addr := s.dnsProxy.Addr(proxy.ProtoUDP)
			client := dns.Client{Net: proxy.ProtoUDP}

			exchangeAndAssertResponse(t, &client, addr, tc, testUpstreamIP)
		})
	}
}

func TestInvalidRequest(t *testing.T) {
	s := createTestServer(t)
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
		Net:     proxy.ProtoUDP,
		Timeout: 500 * time.Millisecond,
	}).Exchange(&req, addr)

	assert.Nil(t, err, "got a response to an invalid query")
}

func TestBlockedRequest(t *testing.T) {
	s := createTestServer(t)
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// Default blocking.
	req := createTestMessage("nxdomain.example.org.")

	reply, err := dns.Exchange(req, addr.String())
	assert.Nilf(t, err, "couldn't talk to server %s: %s", addr, err)
	assert.Equal(t, dns.RcodeSuccess, reply.Rcode)
	assert.True(t, reply.Answer[0].(*dns.A).A.IsUnspecified())
}

func TestServerCustomClientUpstream(t *testing.T) {
	s := createTestServer(t)
	s.conf.GetCustomUpstreamByClient = func(_ string) *proxy.UpstreamConfig {
		return &proxy.UpstreamConfig{
			Upstreams: []upstream.Upstream{
				&testUpstream{
					ipv4: map[string][]net.IP{
						"host.": {{192, 168, 0, 1}},
					},
				},
			},
		}
	}
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// Send test request.
	req := createTestMessage("host.")

	reply, err := dns.Exchange(req, addr.String())

	assert.Nil(t, err)
	assert.Equal(t, dns.RcodeSuccess, reply.Rcode)
	assert.NotEmpty(t, reply.Answer)

	assert.Equal(t, net.IP{192, 168, 0, 1}, reply.Answer[0].(*dns.A).A)
}

// testUpstream is a mock of real upstream. specify fields with necessary values
// to simulate real upstream behaviour.
//
// TODO(e.burkov): move into aghtest package.
type testUpstream struct {
	// cn is a map of hostname to canonical name.
	cn map[string]string
	// ipv4 is a map of hostname to IPv4.
	ipv4 map[string][]net.IP
	// ipv6 is a map of hostname to IPv6.
	ipv6 map[string][]net.IP
}

// Exchange implements upstream.Upstream interface for *testUpstream.
func (u *testUpstream) Exchange(m *dns.Msg) (*dns.Msg, error) {
	resp := &dns.Msg{}
	resp.SetReply(m)
	hasRec := false

	name := m.Question[0].Name

	if cname, ok := u.cn[name]; ok {
		resp.Answer = append(resp.Answer, &dns.CNAME{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeCNAME,
			},
			Target: cname,
		})
	}

	var rrtype uint16
	var a []net.IP
	switch m.Question[0].Qtype {
	case dns.TypeA:
		rrtype = dns.TypeA
		if ipv4addr, ok := u.ipv4[name]; ok {
			hasRec = true
			a = ipv4addr
		}
	case dns.TypeAAAA:
		rrtype = dns.TypeAAAA
		if ipv6addr, ok := u.ipv6[name]; ok {
			hasRec = true
			a = ipv6addr
		}
	}
	for _, ip := range a {
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: rrtype,
			},
			A: ip,
		})
	}

	if len(resp.Answer) == 0 {
		if hasRec {
			// Set no error RCode if there are some records for
			// given Qname but we didn't apply them.
			resp.SetRcode(m, dns.RcodeSuccess)
			return resp, nil
		}
		// Set NXDomain RCode otherwise.
		resp.SetRcode(m, dns.RcodeNameError)
	}

	return resp, nil
}

// Address implements upstream.Upstream interface for *testUpstream.
func (u *testUpstream) Address() string {
	return "test"
}

func (s *Server) startWithUpstream(u upstream.Upstream) error {
	s.Lock()
	defer s.Unlock()
	err := s.Prepare(nil)
	if err != nil {
		return err
	}
	s.dnsProxy.UpstreamConfig = &proxy.UpstreamConfig{
		Upstreams: []upstream.Upstream{u},
	}
	return s.dnsProxy.Start()
}

// testCNAMEs is a simple map of names and CNAMEs necessary for the testUpstream work
var testCNAMEs = map[string]string{
	"badhost.":               "null.example.org.",
	"whitelist.example.org.": "null.example.org.",
}

// testIPv4 is a simple map of names and IPv4s necessary for the testUpstream work
var testIPv4 = map[string][]net.IP{
	"null.example.org.": {{1, 2, 3, 4}},
	"example.org.":      {{127, 0, 0, 255}},
}

func TestBlockCNAMEProtectionEnabled(t *testing.T) {
	s := createTestServer(t)
	testUpstm := &testUpstream{
		cn:   testCNAMEs,
		ipv4: testIPv4,
		ipv6: nil,
	}
	s.conf.ProtectionEnabled = false
	err := s.startWithUpstream(testUpstm)
	assert.Nil(t, err)
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// 'badhost' has a canonical name 'null.example.org' which is blocked by
	// filters: but protection is disabled so response is _not_ blocked.
	req := createTestMessage("badhost.")
	reply, err := dns.Exchange(req, addr.String())
	assert.Nil(t, err)
	assert.Equal(t, dns.RcodeSuccess, reply.Rcode)
}

func TestBlockCNAME(t *testing.T) {
	s := createTestServer(t)
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&testUpstream{
			cn:   testCNAMEs,
			ipv4: testIPv4,
		},
	}
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP).String()

	testCases := []struct {
		host string
		want bool
	}{{
		host: "badhost.",
		// 'badhost' has a canonical name 'null.example.org' which is
		// blocked by filters: response is blocked.
		want: true,
	}, {
		host: "whitelist.example.org.",
		// 'whitelist.example.org' has a canonical name
		// 'null.example.org' which is blocked by filters
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
			assert.Nil(t, err)
			assert.Equal(t, dns.RcodeSuccess, reply.Rcode)
			if tc.want {
				assert.True(t, reply.Answer[0].(*dns.A).A.IsUnspecified())
			}
		})
	}
}

func TestClientRulesForCNAMEMatching(t *testing.T) {
	s := createTestServer(t)
	s.conf.FilterHandler = func(_ net.IP, _ string, settings *dnsfilter.RequestFilteringSettings) {
		settings.FilteringEnabled = false
	}
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&testUpstream{
			cn:   testCNAMEs,
			ipv4: testIPv4,
		},
	}
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// 'badhost' has a canonical name 'null.example.org' which is blocked by
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
	assert.Nil(t, err)
	assert.Equal(t, dns.RcodeSuccess, reply.Rcode)
}

func TestNullBlockedRequest(t *testing.T) {
	s := createTestServer(t)
	s.conf.FilteringConfig.BlockingMode = "null_ip"
	startDeferStop(t, s)
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// Nil filter blocking.
	req := dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: true,
		},
		Question: []dns.Question{{
			Name:   "null.example.org.",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		}},
	}

	reply, err := dns.Exchange(&req, addr.String())
	assert.Nilf(t, err, "couldn't talk to server %s: %s", addr, err)
	assert.Lenf(t, reply.Answer, 1, "dns server %s returned reply with wrong number of answers - %d", addr, len(reply.Answer))
	a, ok := reply.Answer[0].(*dns.A)
	assert.Truef(t, ok, "dns server %s returned wrong answer type instead of A: %v", addr, reply.Answer[0])
	assert.Truef(t, a.A.IsUnspecified(), "dns server %s returned wrong answer instead of 0.0.0.0: %v", addr, a.A)
}

func TestBlockedCustomIP(t *testing.T) {
	rules := "||nxdomain.example.org^\n||null.example.org^\n127.0.0.1	host.example.org\n@@||whitelist.example.org^\n||127.0.0.255\n"
	filters := []dnsfilter.Filter{{
		ID:   0,
		Data: []byte(rules),
	}}

	s := NewServer(DNSCreateParams{
		DNSFilter: dnsfilter.New(&dnsfilter.Config{}, filters),
	})
	conf := ServerConfig{
		UDPListenAddr: &net.UDPAddr{Port: 0},
		TCPListenAddr: &net.TCPAddr{Port: 0},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			BlockingMode:      "custom_ip",
			BlockingIPv4:      nil,
			UpstreamDNS:       []string{"8.8.8.8:53", "8.8.4.4:53"},
		},
	}
	// Invalid BlockingIPv4.
	assert.NotNil(t, s.Prepare(&conf))

	conf.BlockingIPv4 = net.IP{0, 0, 0, 1}
	conf.BlockingIPv6 = net.ParseIP("::1")
	assert.Nil(t, s.Prepare(&conf))

	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	req := createTestMessageWithType("null.example.org.", dns.TypeA)
	reply, err := dns.Exchange(req, addr.String())
	assert.Nil(t, err)
	assert.Len(t, reply.Answer, 1)
	a, ok := reply.Answer[0].(*dns.A)
	assert.True(t, ok)
	assert.True(t, net.IP{0, 0, 0, 1}.Equal(a.A))

	req = createTestMessageWithType("null.example.org.", dns.TypeAAAA)
	reply, err = dns.Exchange(req, addr.String())
	assert.Nil(t, err)
	assert.Len(t, reply.Answer, 1)
	a6, ok := reply.Answer[0].(*dns.AAAA)
	assert.True(t, ok)
	assert.Equal(t, "::1", a6.AAAA.String())
}

func TestBlockedByHosts(t *testing.T) {
	s := createTestServer(t)
	startDeferStop(t, s)
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// Hosts blocking.
	req := createTestMessage("host.example.org.")

	reply, err := dns.Exchange(req, addr.String())
	assert.Nilf(t, err, "couldn't talk to server %s: %s", addr, err)
	assert.Lenf(t, reply.Answer, 1, "dns server %s returned reply with wrong number of answers - %d", addr, len(reply.Answer))

	a, ok := reply.Answer[0].(*dns.A)
	assert.Truef(t, ok, "dns server %s returned wrong answer type instead of A: %v", addr, reply.Answer[0])
	assert.Equalf(t, net.IP{127, 0, 0, 1}, a.A, "dns server %s returned wrong answer instead of 8.8.8.8: %v", addr, a.A)
}

func TestBlockedBySafeBrowsing(t *testing.T) {
	t.Skip("TODO(e.burkov): substitute the dnsfilter by one with custom safeBrowsingUpstream")
	resolver := &testResolver{}
	ips, _ := resolver.LookupIPAddr(context.Background(), safeBrowsingBlockHost)
	addrs, _ := resolver.LookupHost(safeBrowsingBlockHost)

	s := createTestServer(t)
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&testUpstream{
			ipv4: map[string][]net.IP{
				"wmconvirus.narod.ru.": {ips[0].IP},
			},
		},
	}
	startDeferStop(t, s)
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// SafeBrowsing blocking.
	req := createTestMessage("wmconvirus.narod.ru.")

	reply, err := dns.Exchange(req, addr.String())
	assert.Nilf(t, err, "couldn't talk to server %s: %s", addr, err)
	assert.Lenf(t, reply.Answer, 1, "dns server %s returned reply with wrong number of answers - %d", addr, len(reply.Answer))

	a, ok := reply.Answer[0].(*dns.A)
	if assert.Truef(t, ok, "dns server %s returned wrong answer type instead of A: %v", addr, reply.Answer[0]) {
		found := false
		for _, blockAddr := range addrs {
			if blockAddr == a.A.String() {
				found = true
				break
			}
		}
		assert.Truef(t, found, "dns server %s returned wrong answer: %v", addr, a.A)
	}
}

func TestRewrite(t *testing.T) {
	c := &dnsfilter.Config{
		Rewrites: []dnsfilter.RewriteEntry{{
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
	f := dnsfilter.New(c, nil)

	s := NewServer(DNSCreateParams{DNSFilter: f})
	err := s.Prepare(&ServerConfig{
		UDPListenAddr: &net.UDPAddr{Port: 0},
		TCPListenAddr: &net.TCPAddr{Port: 0},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			UpstreamDNS:       []string{"8.8.8.8:53"},
		},
	})
	s.conf.UpstreamConfig.Upstreams = []upstream.Upstream{
		&testUpstream{
			cn: map[string]string{
				"example.org": "somename",
			},
			ipv4: map[string][]net.IP{
				"example.org.": {{4, 3, 2, 1}},
			},
		},
	}
	assert.Nil(t, err)
	startDeferStop(t, s)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	req := createTestMessageWithType("test.com.", dns.TypeA)
	reply, err := dns.Exchange(req, addr.String())
	assert.Nil(t, err)
	assert.Len(t, reply.Answer, 1)
	a, ok := reply.Answer[0].(*dns.A)
	assert.True(t, ok)
	assert.True(t, net.IP{1, 2, 3, 4}.Equal(a.A))

	req = createTestMessageWithType("test.com.", dns.TypeAAAA)
	reply, err = dns.Exchange(req, addr.String())
	assert.Nil(t, err)
	assert.Empty(t, reply.Answer)

	req = createTestMessageWithType("alias.test.com.", dns.TypeA)
	reply, err = dns.Exchange(req, addr.String())
	assert.Nil(t, err)
	assert.Len(t, reply.Answer, 2)
	assert.Equal(t, "test.com.", reply.Answer[0].(*dns.CNAME).Target)
	assert.True(t, net.IP{1, 2, 3, 4}.Equal(reply.Answer[1].(*dns.A).A))

	req = createTestMessageWithType("my.alias.example.org.", dns.TypeA)
	reply, err = dns.Exchange(req, addr.String())
	assert.Nil(t, err)
	assert.Equal(t, "my.alias.example.org.", reply.Question[0].Name) // the original question is restored
	assert.Len(t, reply.Answer, 2)
	assert.Equal(t, "example.org.", reply.Answer[0].(*dns.CNAME).Target)
	assert.Equal(t, dns.TypeA, reply.Answer[1].Header().Rrtype)
}

func createTestServer(t *testing.T) *Server {
	rules := `||nxdomain.example.org
||null.example.org^
127.0.0.1	host.example.org
@@||whitelist.example.org^
||127.0.0.255`
	filters := []dnsfilter.Filter{{
		ID: 0, Data: []byte(rules),
	}}
	c := dnsfilter.Config{
		SafeBrowsingEnabled:   true,
		SafeBrowsingCacheSize: 1000,
		SafeSearchEnabled:     true,
		SafeSearchCacheSize:   1000,
		ParentalCacheSize:     1000,
		CacheTime:             30,
	}

	f := dnsfilter.New(&c, filters)

	s := NewServer(DNSCreateParams{DNSFilter: f})
	s.conf = ServerConfig{
		UDPListenAddr: &net.UDPAddr{Port: 0},
		TCPListenAddr: &net.TCPAddr{Port: 0},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			UpstreamDNS:       []string{"8.8.8.8:53", "8.8.4.4:53"},
		},
		ConfigModified: func() {},
	}

	err := s.Prepare(nil)
	assert.Nil(t, err)
	return s
}

func createServerTLSConfig(t *testing.T) (*tls.Config, []byte, []byte) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Nilf(t, err, "cannot generate RSA key: %s", err)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	assert.Nilf(t, err, "failed to generate serial number: %s", err)

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
	assert.Nilf(t, err, "failed to create certificate: %s", err)

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	cert, err := tls.X509KeyPair(certPem, keyPem)
	assert.Nilf(t, err, "failed to create certificate: %s", err)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   tlsServerName,
		MinVersion:   tls.VersionTLS12,
	}, certPem, keyPem
}

// sendTestMessagesAsync sends messages in parallel to check for race issues.
//lint:ignore U1000 it's called from the function which is skipped for now.
func sendTestMessagesAsync(t *testing.T, conn *dns.Conn) {
	wg := &sync.WaitGroup{}

	for i := 0; i < testMessagesCount; i++ {
		msg := createGoogleATestMessage()
		wg.Add(1)

		go func() {
			defer wg.Done()

			err := conn.WriteMsg(msg)
			assert.Nilf(t, err, "cannot write message: %s", err)

			res, err := conn.ReadMsg()
			assert.Nilf(t, err, "cannot read response to message: %s", err)

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
		assert.Nilf(t, err, "cannot write message #%d: %s", i, err)

		res, err := conn.ReadMsg()
		assert.Nilf(t, err, "cannot read response to message #%d: %s", i, err)
		assertGoogleAResponse(t, res)
	}
}

func exchangeAndAssertResponse(t *testing.T, client *dns.Client, addr net.Addr, host string, ip net.IP) {
	t.Helper()

	req := createTestMessage(host)
	reply, _, err := client.Exchange(req, addr.String())
	assert.Nilf(t, err, "couldn't talk to server %s: %s", addr, err)
	assertResponse(t, reply, ip)
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

	assert.Lenf(t, reply.Answer, 1, "dns server returned reply with wrong number of answers - %d", len(reply.Answer))
	a, ok := reply.Answer[0].(*dns.A)
	if assert.Truef(t, ok, "dns server returned wrong answer type instead of A: %v", reply.Answer[0]) {
		assert.Truef(t, a.A.Equal(ip), "dns server returned wrong answer instead of %s: %s", ip, a.A)
	}
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

func TestValidateUpstream(t *testing.T) {
	testCases := []struct {
		name     string
		upstream string
		valid    bool
		wantDef  bool
	}{{
		name:     "invalid",
		upstream: "1.2.3.4.5",
		valid:    false,
	}, {
		name:     "invalid",
		upstream: "123.3.7m",
		valid:    false,
	}, {
		name:     "invalid",
		upstream: "htttps://google.com/dns-query",
		valid:    false,
	}, {
		name:     "invalid",
		upstream: "[/host.com]tls://dns.adguard.com",
		valid:    false,
	}, {
		name:     "invalid",
		upstream: "[host.ru]#",
		valid:    false,
	}, {
		name:     "valid_default",
		upstream: "1.1.1.1",
		valid:    true,
		wantDef:  true,
	}, {
		name:     "valid_default",
		upstream: "tls://1.1.1.1",
		valid:    true,
		wantDef:  true,
	}, {
		name:     "valid_default",
		upstream: "https://dns.adguard.com/dns-query",
		valid:    true,
		wantDef:  true,
	}, {
		name:     "valid_default",
		upstream: "sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
		valid:    true,
		wantDef:  true,
	}, {
		name:     "valid",
		upstream: "[/host.com/]1.1.1.1",
		valid:    true,
		wantDef:  false,
	}, {
		name:     "valid",
		upstream: "[//]tls://1.1.1.1",
		valid:    true,
		wantDef:  false,
	}, {
		name:     "valid",
		upstream: "[/www.host.com/]#",
		valid:    true,
		wantDef:  false,
	}, {
		name:     "valid",
		upstream: "[/host.com/google.com/]8.8.8.8",
		valid:    true,
		wantDef:  false,
	}, {
		name:     "valid",
		upstream: "[/host/]sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
		valid:    true,
		wantDef:  false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defaultUpstream, err := validateUpstream(tc.upstream)
			assert.Equal(t, tc.valid, err == nil)
			if err == nil {
				assert.Equal(t, tc.wantDef, defaultUpstream)
			}
		})
	}
}

func TestValidateUpstreamsSet(t *testing.T) {
	// Empty upstreams array.
	var upstreamsSet []string
	assert.Nil(t, ValidateUpstreams(upstreamsSet), "empty upstreams array should be valid")

	// Comment in upstreams array.
	upstreamsSet = []string{"# comment"}
	assert.Nil(t, ValidateUpstreams(upstreamsSet), "comments should not be validated")

	// Set of valid upstreams. There is no default upstream specified.
	upstreamsSet = []string{
		"[/host.com/]1.1.1.1",
		"[//]tls://1.1.1.1",
		"[/www.host.com/]#",
		"[/host.com/google.com/]8.8.8.8",
		"[/host/]sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
	}
	assert.NotNil(t, ValidateUpstreams(upstreamsSet), "there is no default upstream")

	// Let's add default upstream.
	upstreamsSet = append(upstreamsSet, "8.8.8.8")
	err := ValidateUpstreams(upstreamsSet)
	assert.Nilf(t, err, "upstreams set is valid, but doesn't pass through validation cause: %s", err)

	// Let's add invalid upstream.
	upstreamsSet = append(upstreamsSet, "dhcp://fake.dns")
	assert.NotNil(t, ValidateUpstreams(upstreamsSet), "there is an invalid upstream in set, but it pass through validation")
}

func TestIPStringFromAddr(t *testing.T) {
	addr := net.UDPAddr{
		IP:   net.ParseIP("1:2:3::4"),
		Port: 12345,
		Zone: "eth0",
	}
	assert.Equal(t, IPStringFromAddr(&addr), addr.IP.String())
	assert.Empty(t, IPStringFromAddr(nil))
}

func TestMatchDNSName(t *testing.T) {
	dnsNames := []string{"host1", "*.host2", "1.2.3.4"}
	sort.Strings(dnsNames)

	testCases := []struct {
		name    string
		dnsName string
		want    bool
	}{{
		name:    "match",
		dnsName: "host1",
		want:    true,
	}, {
		name:    "match",
		dnsName: "a.host2",
		want:    true,
	}, {
		name:    "match",
		dnsName: "b.a.host2",
		want:    true,
	}, {
		name:    "match",
		dnsName: "1.2.3.4",
		want:    true,
	}, {
		name:    "mismatch",
		dnsName: "host2",
		want:    false,
	}, {
		name:    "mismatch",
		dnsName: "",
		want:    false,
	}, {
		name:    "mismatch",
		dnsName: "*.host2",
		want:    false,
	}}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, matchDNSName(dnsNames, tc.dnsName))
		})
	}
}

type testDHCP struct{}

func (d *testDHCP) Leases(flags int) []dhcpd.Lease {
	l := dhcpd.Lease{
		IP:       net.IP{127, 0, 0, 1},
		HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		Hostname: "localhost",
	}
	return []dhcpd.Lease{l}
}
func (d *testDHCP) SetOnLeaseChanged(onLeaseChanged dhcpd.OnLeaseChangedT) {}

func TestPTRResponseFromDHCPLeases(t *testing.T) {
	dhcp := &testDHCP{}

	s := NewServer(DNSCreateParams{
		DNSFilter:  dnsfilter.New(&dnsfilter.Config{}, nil),
		DHCPServer: dhcp,
	})

	s.conf.UDPListenAddr = &net.UDPAddr{Port: 0}
	s.conf.TCPListenAddr = &net.TCPAddr{Port: 0}
	s.conf.UpstreamDNS = []string{"127.0.0.1:53"}
	s.conf.FilteringConfig.ProtectionEnabled = true
	err := s.Prepare(nil)
	assert.Nil(t, err)

	assert.Nil(t, s.Start())
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	req := createTestMessageWithType("1.0.0.127.in-addr.arpa.", dns.TypePTR)

	resp, err := dns.Exchange(req, addr.String())

	assert.Nil(t, err)
	assert.Len(t, resp.Answer, 1)
	assert.Equal(t, dns.TypePTR, resp.Answer[0].Header().Rrtype)
	assert.Equal(t, "1.0.0.127.in-addr.arpa.", resp.Answer[0].Header().Name)

	ptr := resp.Answer[0].(*dns.PTR)
	assert.Equal(t, "localhost.", ptr.Ptr)

	s.Close()
}

func TestPTRResponseFromHosts(t *testing.T) {
	c := dnsfilter.Config{
		AutoHosts: &util.AutoHosts{},
	}

	// Prepare test hosts file.
	hf, err := ioutil.TempFile("", "")
	if assert.Nil(t, err) {
		t.Cleanup(func() {
			assert.Nil(t, hf.Close())
			assert.Nil(t, os.Remove(hf.Name()))
		})
	}

	_, _ = hf.WriteString("  127.0.0.1   host # comment \n")
	_, _ = hf.WriteString("  ::1   localhost#comment  \n")

	// Init auto hosts.
	c.AutoHosts.Init(hf.Name())
	t.Cleanup(c.AutoHosts.Close)

	s := NewServer(DNSCreateParams{DNSFilter: dnsfilter.New(&c, nil)})
	s.conf.UDPListenAddr = &net.UDPAddr{Port: 0}
	s.conf.TCPListenAddr = &net.TCPAddr{Port: 0}
	s.conf.UpstreamDNS = []string{"127.0.0.1:53"}
	s.conf.FilteringConfig.ProtectionEnabled = true
	assert.Nil(t, s.Prepare(nil))

	assert.Nil(t, s.Start())

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)
	req := createTestMessageWithType("1.0.0.127.in-addr.arpa.", dns.TypePTR)

	resp, err := dns.Exchange(req, addr.String())
	assert.Nil(t, err)
	assert.Len(t, resp.Answer, 1)
	assert.Equal(t, dns.TypePTR, resp.Answer[0].Header().Rrtype)
	assert.Equal(t, "1.0.0.127.in-addr.arpa.", resp.Answer[0].Header().Name)

	ptr := resp.Answer[0].(*dns.PTR)
	assert.Equal(t, "host.", ptr.Ptr)

	s.Close()
}
