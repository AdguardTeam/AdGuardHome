package dnsforward

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

const (
	tlsServerName     = "testdns.adguard.com"
	testMessagesCount = 10
)

func TestServer(t *testing.T) {
	s := createTestServer(t)
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

	// message over UDP
	req := createGoogleATestMessage()
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)
	client := dns.Client{Net: "udp"}
	reply, _, err := client.Exchange(req, addr.String())
	if err != nil {
		t.Fatalf("Couldn't talk to server %s: %s", addr, err)
	}
	assertGoogleAResponse(t, reply)

	// message over TCP
	req = createGoogleATestMessage()
	addr = s.dnsProxy.Addr("tcp")
	client = dns.Client{Net: "tcp"}
	reply, _, err = client.Exchange(req, addr.String())
	if err != nil {
		t.Fatalf("Couldn't talk to server %s: %s", addr, err)
	}
	assertGoogleAResponse(t, reply)

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestServerWithProtectionDisabled(t *testing.T) {
	s := createTestServer(t)
	s.conf.ProtectionEnabled = false
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

	// message over UDP
	req := createGoogleATestMessage()
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)
	client := dns.Client{Net: "udp"}
	reply, _, err := client.Exchange(req, addr.String())
	if err != nil {
		t.Fatalf("Couldn't talk to server %s: %s", addr, err)
	}
	assertGoogleAResponse(t, reply)

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestDotServer(t *testing.T) {
	// Prepare the proxy server
	_, certPem, keyPem := createServerTLSConfig(t)
	s := createTestServer(t)

	s.conf.TLSConfig = TLSConfig{
		TLSListenAddr:        &net.TCPAddr{Port: 0},
		CertificateChainData: certPem,
		PrivateKeyData:       keyPem,
	}

	_ = s.Prepare(nil)
	// Starting the server
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

	// Add our self-signed generated config to roots
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(certPem)
	tlsConfig := &tls.Config{
		ServerName: tlsServerName,
		RootCAs:    roots,
		MinVersion: tls.VersionTLS12,
	}

	// Create a DNS-over-TLS client connection
	addr := s.dnsProxy.Addr(proxy.ProtoTLS)
	conn, err := dns.DialWithTLS("tcp-tls", addr.String(), tlsConfig)
	if err != nil {
		t.Fatalf("cannot connect to the proxy: %s", err)
	}

	sendTestMessages(t, conn)

	// Stop the proxy
	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestServerRace(t *testing.T) {
	s := createTestServer(t)
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

	// message over UDP
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)
	conn, err := dns.Dial("udp", addr.String())
	if err != nil {
		t.Fatalf("cannot connect to the proxy: %s", err)
	}

	sendTestMessagesAsync(t, conn)

	// Stop the proxy
	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestSafeSearch(t *testing.T) {
	s := createTestServer(t)
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

	// Test safe search for yandex. We already know safe search ip
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)
	client := dns.Client{Net: "udp"}
	yandexDomains := []string{"yandex.com.", "yandex.by.", "yandex.kz.", "yandex.ru.", "yandex.com."}
	for _, host := range yandexDomains {
		exchangeAndAssertResponse(t, &client, addr, host, "213.180.193.56")
	}

	// Let's lookup for google safesearch ip
	ips, err := net.LookupIP("forcesafesearch.google.com")
	if err != nil {
		t.Fatalf("Failed to lookup for forcesafesearch.google.com: %s", err)
	}

	ip := ips[0]
	for _, i := range ips {
		if i.To4() != nil {
			ip = i
			break
		}
	}

	// Test safe search for google.
	googleDomains := []string{"www.google.com.", "www.google.com.af.", "www.google.be.", "www.google.by."}
	for _, host := range googleDomains {
		exchangeAndAssertResponse(t, &client, addr, host, ip.String())
	}

	err = s.Stop()
	if err != nil {
		t.Fatalf("Can not stopd server cause: %s", err)
	}
}

func TestInvalidRequest(t *testing.T) {
	s := createTestServer(t)
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

	// server is running, send a message
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)
	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true

	// send a DNS request without question
	client := dns.Client{Net: "udp", Timeout: 500 * time.Millisecond}
	_, _, err = client.Exchange(&req, addr.String())
	if err != nil {
		t.Fatalf("got a response to an invalid query")
	}

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestBlockedRequest(t *testing.T) {
	s := createTestServer(t)
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	//
	// NXDomain blocking
	//
	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{Name: "nxdomain.example.org.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}

	reply, err := dns.Exchange(&req, addr.String())
	if err != nil {
		t.Fatalf("Couldn't talk to server %s: %s", addr, err)
	}
	if reply.Rcode != dns.RcodeNameError {
		t.Fatalf("Wrong response: %s", reply.String())
	}

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

// testUpstream is a mock of real upstream.
// specify fields with necessary values to simulate real upstream behaviour
type testUpstream struct {
	cn   map[string]string   // Map of [name]canonical_name
	ipv4 map[string][]net.IP // Map of [name]IPv4
	ipv6 map[string][]net.IP // Map of [name]IPv6
}

func (u *testUpstream) Exchange(m *dns.Msg) (*dns.Msg, error) {
	resp := dns.Msg{}
	resp.SetReply(m)
	hasARecord := false
	hasAAAARecord := false

	reqType := m.Question[0].Qtype
	name := m.Question[0].Name

	// Let's check if we have any CNAME for given name
	if cname, ok := u.cn[name]; ok {
		cn := dns.CNAME{}
		cn.Hdr.Name = name
		cn.Hdr.Rrtype = dns.TypeCNAME
		cn.Target = cname
		resp.Answer = append(resp.Answer, &cn)
	}

	// Let's check if we can add some A records to the answer
	if ipv4addr, ok := u.ipv4[name]; ok && reqType == dns.TypeA {
		hasARecord = true
		for _, ipv4 := range ipv4addr {
			respA := dns.A{}
			respA.Hdr.Rrtype = dns.TypeA
			respA.Hdr.Name = name
			respA.A = ipv4
			resp.Answer = append(resp.Answer, &respA)
		}
	}

	// Let's check if we can add some AAAA records to the answer
	if u.ipv6 != nil {
		if ipv6addr, ok := u.ipv6[name]; ok && reqType == dns.TypeAAAA {
			hasAAAARecord = true
			for _, ipv6 := range ipv6addr {
				respAAAA := dns.A{}
				respAAAA.Hdr.Rrtype = dns.TypeAAAA
				respAAAA.Hdr.Name = name
				respAAAA.A = ipv6
				resp.Answer = append(resp.Answer, &respAAAA)
			}
		}
	}

	if len(resp.Answer) == 0 {
		if hasARecord || hasAAAARecord {
			// Set No Error RCode if there are some records for given Qname but we didn't apply them
			resp.SetRcode(m, dns.RcodeSuccess)
		} else {
			// Set NXDomain RCode otherwise
			resp.SetRcode(m, dns.RcodeNameError)
		}
	}

	return &resp, nil
}

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
	s.dnsProxy.Upstreams = []upstream.Upstream{u}
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
	testUpstm := &testUpstream{testCNAMEs, testIPv4, nil}
	s.conf.ProtectionEnabled = false
	err := s.startWithUpstream(testUpstm)
	assert.True(t, err == nil)
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// 'badhost' has a canonical name 'null.example.org' which is blocked by filters:
	// but protection is disabled - response is NOT blocked
	req := createTestMessage("badhost.")
	reply, err := dns.Exchange(req, addr.String())
	assert.True(t, err == nil)
	assert.True(t, reply.Rcode == dns.RcodeSuccess)
}

func TestBlockCNAME(t *testing.T) {
	s := createTestServer(t)
	testUpstm := &testUpstream{testCNAMEs, testIPv4, nil}
	err := s.startWithUpstream(testUpstm)
	assert.True(t, err == nil)
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// 'badhost' has a canonical name 'null.example.org' which is blocked by filters:
	// response is blocked
	req := createTestMessage("badhost.")
	reply, err := dns.Exchange(req, addr.String())
	assert.True(t, err == nil)
	assert.True(t, reply.Rcode == dns.RcodeNameError)

	// 'whitelist.example.org' has a canonical name 'null.example.org' which is blocked by filters
	//   but 'whitelist.example.org' is in a whitelist:
	// response isn't blocked
	req = createTestMessage("whitelist.example.org.")
	reply, err = dns.Exchange(req, addr.String())
	assert.True(t, err == nil)
	assert.True(t, reply.Rcode == dns.RcodeSuccess)

	// 'example.org' has a canonical name 'cname1' with IP 127.0.0.255 which is blocked by filters:
	// response is blocked
	req = createTestMessage("example.org.")
	reply, err = dns.Exchange(req, addr.String())
	assert.True(t, err == nil)
	assert.True(t, reply.Rcode == dns.RcodeNameError)

	_ = s.Stop()
}

func TestClientRulesForCNAMEMatching(t *testing.T) {
	s := createTestServer(t)
	testUpstm := &testUpstream{testCNAMEs, testIPv4, nil}
	s.conf.FilterHandler = func(clientAddr string, settings *dnsfilter.RequestFilteringSettings) {
		settings.FilteringEnabled = false
	}
	err := s.startWithUpstream(testUpstm)
	assert.Nil(t, err)
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	// 'badhost' has a canonical name 'null.example.org' which is blocked by filters:
	// response is blocked
	req := dns.Msg{}
	req.Id = dns.Id()
	req.Question = []dns.Question{
		{Name: "badhost.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}
	// However, in our case it should not be blocked
	// as filtering is disabled on the client level
	reply, err := dns.Exchange(&req, addr.String())
	assert.Nil(t, err)
	assert.Equal(t, dns.RcodeSuccess, reply.Rcode)
}

func TestNullBlockedRequest(t *testing.T) {
	s := createTestServer(t)
	s.conf.FilteringConfig.BlockingMode = "null_ip"
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	//
	// Null filter blocking
	//
	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{Name: "null.example.org.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}

	reply, err := dns.Exchange(&req, addr.String())
	if err != nil {
		t.Fatalf("Couldn't talk to server %s: %s", addr, err)
	}
	if len(reply.Answer) != 1 {
		t.Fatalf("DNS server %s returned reply with wrong number of answers - %d", addr, len(reply.Answer))
	}
	if a, ok := reply.Answer[0].(*dns.A); ok {
		if !net.IPv4zero.Equal(a.A) {
			t.Fatalf("DNS server %s returned wrong answer instead of 0.0.0.0: %v", addr, a.A)
		}
	} else {
		t.Fatalf("DNS server %s returned wrong answer type instead of A: %v", addr, reply.Answer[0])
	}

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestBlockedCustomIP(t *testing.T) {
	rules := "||nxdomain.example.org^\n||null.example.org^\n127.0.0.1	host.example.org\n@@||whitelist.example.org^\n||127.0.0.255\n"
	filters := []dnsfilter.Filter{dnsfilter.Filter{
		ID: 0, Data: []byte(rules),
	}}
	c := dnsfilter.Config{}

	f := dnsfilter.New(&c, filters)
	s := NewServer(f, nil, nil)
	conf := ServerConfig{}
	conf.UDPListenAddr = &net.UDPAddr{Port: 0}
	conf.TCPListenAddr = &net.TCPAddr{Port: 0}
	conf.ProtectionEnabled = true
	conf.BlockingMode = "custom_ip"
	conf.BlockingIPv4 = "bad IP"
	conf.UpstreamDNS = []string{"8.8.8.8:53", "8.8.4.4:53"}
	err := s.Prepare(&conf)
	assert.True(t, err != nil) // invalid BlockingIPv4

	conf.BlockingIPv4 = "0.0.0.1"
	conf.BlockingIPv6 = "::1"
	err = s.Prepare(&conf)
	assert.True(t, err == nil)
	err = s.Start()
	assert.True(t, err == nil, "%s", err)

	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	req := createTestMessageWithType("null.example.org.", dns.TypeA)
	reply, err := dns.Exchange(req, addr.String())
	assert.True(t, err == nil)
	assert.True(t, len(reply.Answer) == 1)
	a, ok := reply.Answer[0].(*dns.A)
	assert.True(t, ok)
	assert.True(t, a.A.String() == "0.0.0.1")

	req = createTestMessageWithType("null.example.org.", dns.TypeAAAA)
	reply, err = dns.Exchange(req, addr.String())
	assert.True(t, err == nil)
	assert.True(t, len(reply.Answer) == 1)
	a6, ok := reply.Answer[0].(*dns.AAAA)
	assert.True(t, ok)
	assert.True(t, a6.AAAA.String() == "::1")

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestBlockedByHosts(t *testing.T) {
	s := createTestServer(t)
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	//
	// Hosts blocking
	//
	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{Name: "host.example.org.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}

	reply, err := dns.Exchange(&req, addr.String())
	if err != nil {
		t.Fatalf("Couldn't talk to server %s: %s", addr, err)
	}
	if len(reply.Answer) != 1 {
		t.Fatalf("DNS server %s returned reply with wrong number of answers - %d", addr, len(reply.Answer))
	}
	if a, ok := reply.Answer[0].(*dns.A); ok {
		if !net.IPv4(127, 0, 0, 1).Equal(a.A) {
			t.Fatalf("DNS server %s returned wrong answer instead of 8.8.8.8: %v", addr, a.A)
		}
	} else {
		t.Fatalf("DNS server %s returned wrong answer type instead of A: %v", addr, reply.Answer[0])
	}

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestBlockedBySafeBrowsing(t *testing.T) {
	s := createTestServer(t)
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}
	addr := s.dnsProxy.Addr(proxy.ProtoUDP)

	//
	// Safebrowsing blocking
	//
	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{Name: "wmconvirus.narod.ru.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}
	reply, err := dns.Exchange(&req, addr.String())
	if err != nil {
		t.Fatalf("Couldn't talk to server %s: %s", addr, err)
	}
	if len(reply.Answer) != 1 {
		t.Fatalf("DNS server %s returned reply with wrong number of answers - %d", addr, len(reply.Answer))
	}
	if a, ok := reply.Answer[0].(*dns.A); ok {
		addrs, lookupErr := net.LookupHost(safeBrowsingBlockHost)
		if lookupErr != nil {
			t.Fatalf("cannot resolve %s due to %s", safeBrowsingBlockHost, lookupErr)
		}

		found := false
		for _, blockAddr := range addrs {
			if blockAddr == a.A.String() {
				found = true
			}
		}

		if !found {
			t.Fatalf("DNS server %s returned wrong answer: %v", addr, a.A)
		}
	} else {
		t.Fatalf("DNS server %s returned wrong answer type instead of A: %v", addr, reply.Answer[0])
	}

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func createTestServer(t *testing.T) *Server {
	rules := `||nxdomain.example.org
||null.example.org^
127.0.0.1	host.example.org
@@||whitelist.example.org^
||127.0.0.255`
	filters := []dnsfilter.Filter{dnsfilter.Filter{
		ID: 0, Data: []byte(rules),
	}}
	c := dnsfilter.Config{}
	c.SafeBrowsingEnabled = true
	c.SafeBrowsingCacheSize = 1000
	c.SafeSearchEnabled = true
	c.SafeSearchCacheSize = 1000
	c.ParentalCacheSize = 1000
	c.CacheTime = 30

	f := dnsfilter.New(&c, filters)
	s := NewServer(f, nil, nil)
	s.conf.UDPListenAddr = &net.UDPAddr{Port: 0}
	s.conf.TCPListenAddr = &net.TCPAddr{Port: 0}
	s.conf.UpstreamDNS = []string{"8.8.8.8:53", "8.8.4.4:53"}
	s.conf.FilteringConfig.ProtectionEnabled = true
	err := s.Prepare(nil)
	assert.True(t, err == nil)
	return s
}

func createServerTLSConfig(t *testing.T) (*tls.Config, []byte, []byte) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("cannot generate RSA key: %s", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		t.Fatalf("failed to generate serial number: %s", err)
	}

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
	if err != nil {
		t.Fatalf("failed to create certificate: %s", err)
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		t.Fatalf("failed to create certificate: %s", err)
	}

	return &tls.Config{Certificates: []tls.Certificate{cert}, ServerName: tlsServerName, MinVersion: tls.VersionTLS12}, certPem, keyPem
}

func sendTestMessageAsync(t *testing.T, conn *dns.Conn, g *sync.WaitGroup) {
	defer func() {
		g.Done()
	}()

	req := createGoogleATestMessage()
	err := conn.WriteMsg(req)
	if err != nil {
		t.Fatalf("cannot write message: %s", err)
	}

	res, err := conn.ReadMsg()
	if err != nil {
		t.Fatalf("cannot read response to message: %s", err)
	}
	assertGoogleAResponse(t, res)
}

// sendTestMessagesAsync sends messages in parallel
// so that we could find race issues
func sendTestMessagesAsync(t *testing.T, conn *dns.Conn) {
	g := &sync.WaitGroup{}
	g.Add(testMessagesCount)

	for i := 0; i < testMessagesCount; i++ {
		go sendTestMessageAsync(t, conn, g)
	}

	g.Wait()
}

func sendTestMessages(t *testing.T, conn *dns.Conn) {
	for i := 0; i < 10; i++ {
		req := createGoogleATestMessage()
		err := conn.WriteMsg(req)
		if err != nil {
			t.Fatalf("cannot write message #%d: %s", i, err)
		}

		res, err := conn.ReadMsg()
		if err != nil {
			t.Fatalf("cannot read response to message #%d: %s", i, err)
		}
		assertGoogleAResponse(t, res)
	}
}

func exchangeAndAssertResponse(t *testing.T, client *dns.Client, addr net.Addr, host, ip string) {
	req := createTestMessage(host)
	reply, _, err := client.Exchange(req, addr.String())
	if err != nil {
		t.Fatalf("Couldn't talk to server %s: %s", addr, err)
	}
	assertResponse(t, reply, ip)
}

func createGoogleATestMessage() *dns.Msg {
	return createTestMessage("google-public-dns-a.google.com.")
}

func createTestMessage(host string) *dns.Msg {
	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{Name: host, Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}
	return &req
}

func createTestMessageWithType(host string, qtype uint16) *dns.Msg {
	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{Name: host, Qtype: qtype, Qclass: dns.ClassINET},
	}
	return &req
}

func assertGoogleAResponse(t *testing.T, reply *dns.Msg) {
	assertResponse(t, reply, "8.8.8.8")
}

func assertResponse(t *testing.T, reply *dns.Msg, ip string) {
	if len(reply.Answer) != 1 {
		t.Fatalf("DNS server returned reply with wrong number of answers - %d", len(reply.Answer))
	}
	if a, ok := reply.Answer[0].(*dns.A); ok {
		if !net.ParseIP(ip).Equal(a.A) {
			t.Fatalf("DNS server returned wrong answer instead of %s: %v", ip, a.A)
		}
	} else {
		t.Fatalf("DNS server returned wrong answer type instead of A: %v", reply.Answer[0])
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

func TestIsBlockedIPAllowed(t *testing.T) {
	a := &accessCtx{}
	assert.True(t, a.Init([]string{"1.1.1.1", "2.2.0.0/16"}, nil, nil) == nil)

	assert.True(t, !a.IsBlockedIP("1.1.1.1"))
	assert.True(t, a.IsBlockedIP("1.1.1.2"))
	assert.True(t, !a.IsBlockedIP("2.2.1.1"))
	assert.True(t, a.IsBlockedIP("2.3.1.1"))
}

func TestIsBlockedIPDisallowed(t *testing.T) {
	a := &accessCtx{}
	assert.True(t, a.Init(nil, []string{"1.1.1.1", "2.2.0.0/16"}, nil) == nil)

	assert.True(t, a.IsBlockedIP("1.1.1.1"))
	assert.True(t, !a.IsBlockedIP("1.1.1.2"))
	assert.True(t, a.IsBlockedIP("2.2.1.1"))
	assert.True(t, !a.IsBlockedIP("2.3.1.1"))
}

func TestIsBlockedIPBlockedDomain(t *testing.T) {
	a := &accessCtx{}
	assert.True(t, a.Init(nil, nil, []string{"host1", "host2"}) == nil)

	assert.True(t, a.IsBlockedDomain("host1"))
	assert.True(t, a.IsBlockedDomain("host2"))
	assert.True(t, !a.IsBlockedDomain("host3"))
}

func TestValidateUpstream(t *testing.T) {
	invalidUpstreams := []string{"1.2.3.4.5",
		"123.3.7m",
		"htttps://google.com/dns-query",
		"[/host.com]tls://dns.adguard.com",
		"[host.ru]#",
	}

	validDefaultUpstreams := []string{"1.1.1.1",
		"tls://1.1.1.1",
		"https://dns.adguard.com/dns-query",
		"sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
	}

	validUpstreams := []string{"[/host.com/]1.1.1.1",
		"[//]tls://1.1.1.1",
		"[/www.host.com/]#",
		"[/host.com/google.com/]8.8.8.8",
		"[/host/]sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
	}
	for _, u := range invalidUpstreams {
		_, err := validateUpstream(u)
		if err == nil {
			t.Fatalf("upstream %s is invalid but it pass through validation", u)
		}
	}

	for _, u := range validDefaultUpstreams {
		defaultUpstream, err := validateUpstream(u)
		if err != nil {
			t.Fatalf("upstream %s is valid but it doen't pass through validation cause: %s", u, err)
		}
		if !defaultUpstream {
			t.Fatalf("upstream %s is default one!", u)
		}
	}

	for _, u := range validUpstreams {
		defaultUpstream, err := validateUpstream(u)
		if err != nil {
			t.Fatalf("upstream %s is valid but it doen't pass through validation cause: %s", u, err)
		}
		if defaultUpstream {
			t.Fatalf("upstream %s is default one!", u)
		}
	}
}

func TestValidateUpstreamsSet(t *testing.T) {
	// Set of valid upstreams. There is no default upstream specified
	upstreamsSet := []string{"[/host.com/]1.1.1.1",
		"[//]tls://1.1.1.1",
		"[/www.host.com/]#",
		"[/host.com/google.com/]8.8.8.8",
		"[/host/]sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
	}
	err := ValidateUpstreams(upstreamsSet)
	if err == nil {
		t.Fatalf("there is no default upstream")
	}

	// Let's add default upstream
	upstreamsSet = append(upstreamsSet, "8.8.8.8")
	err = ValidateUpstreams(upstreamsSet)
	if err != nil {
		t.Fatalf("upstreams set is valid, but doesn't pass through validation cause: %s", err)
	}

	// Let's add invalid upstream
	upstreamsSet = append(upstreamsSet, "dhcp://fake.dns")
	err = ValidateUpstreams(upstreamsSet)
	if err == nil {
		t.Fatalf("there is an invalid upstream in set, but it pass through validation")
	}
}

func TestIpFromAddr(t *testing.T) {
	addr := net.UDPAddr{}
	addr.IP = net.ParseIP("1:2:3::4")
	addr.Port = 12345
	addr.Zone = "eth0"
	a := ipFromAddr(&addr)
	assert.True(t, a == "1:2:3::4")

	a = ipFromAddr(nil)
	assert.True(t, a == "")
}

func TestMatchDNSName(t *testing.T) {
	dnsNames := []string{"host1", "*.host2", "1.2.3.4"}
	sort.Strings(dnsNames)
	assert.True(t, matchDNSName(dnsNames, "host1"))
	assert.True(t, matchDNSName(dnsNames, "a.host2"))
	assert.True(t, matchDNSName(dnsNames, "b.a.host2"))
	assert.True(t, matchDNSName(dnsNames, "1.2.3.4"))
	assert.True(t, !matchDNSName(dnsNames, "host2"))
	assert.True(t, !matchDNSName(dnsNames, ""))
	assert.True(t, !matchDNSName(dnsNames, "*.host2"))
}
