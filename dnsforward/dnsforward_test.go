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
	"os"
	"sync"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

const (
	tlsServerName     = "testdns.adguard.com"
	dataDir           = "testData"
	testMessagesCount = 10
)

func TestServer(t *testing.T) {
	s := createTestServer(t)
	defer removeDataDir(t)
	err := s.Start(nil)
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

	// check query log and stats
	log := s.GetQueryLog()
	assert.Equal(t, 1, len(log), "Log size")
	stats := s.GetStatsTop()
	assert.Equal(t, 1, len(stats.Domains), "Top domains length")
	assert.Equal(t, 0, len(stats.Blocked), "Top blocked length")
	assert.Equal(t, 1, len(stats.Clients), "Top clients length")

	// message over TCP
	req = createGoogleATestMessage()
	addr = s.dnsProxy.Addr("tcp")
	client = dns.Client{Net: "tcp"}
	reply, _, err = client.Exchange(req, addr.String())
	if err != nil {
		t.Fatalf("Couldn't talk to server %s: %s", addr, err)
	}
	assertGoogleAResponse(t, reply)

	// check query log and stats again
	log = s.GetQueryLog()
	assert.Equal(t, 2, len(log), "Log size")
	stats = s.GetStatsTop()
	// Length did not change as we queried the same domain
	assert.Equal(t, 1, len(stats.Domains), "Top domains length")
	assert.Equal(t, 0, len(stats.Blocked), "Top blocked length")
	assert.Equal(t, 1, len(stats.Clients), "Top clients length")

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestServerWithProtectionDisabled(t *testing.T) {
	s := createTestServer(t)
	s.conf.ProtectionEnabled = false
	defer removeDataDir(t)
	err := s.Start(nil)
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

	// check query log and stats
	log := s.GetQueryLog()
	assert.Equal(t, 1, len(log), "Log size")
	stats := s.GetStatsTop()
	assert.Equal(t, 1, len(stats.Domains), "Top domains length")
	assert.Equal(t, 0, len(stats.Blocked), "Top blocked length")
	assert.Equal(t, 1, len(stats.Clients), "Top clients length")

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestDotServer(t *testing.T) {
	// Prepare the proxy server
	_, certPem, keyPem := createServerTLSConfig(t)
	s := createTestServer(t)
	defer removeDataDir(t)

	s.conf.TLSConfig = TLSConfig{
		TLSListenAddr:        &net.TCPAddr{Port: 0},
		CertificateChainData: certPem,
		PrivateKeyData:       keyPem,
	}

	// Starting the server
	err := s.Start(nil)
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
	defer removeDataDir(t)
	err := s.Start(nil)
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
	s.conf.SafeSearchEnabled = true
	defer removeDataDir(t)
	err := s.Start(nil)
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

	// Check aggregated stats
	assert.Equal(t, s.GetAggregatedStats()["replaced_safesearch"], float64(len(yandexDomains)))
	assert.Equal(t, s.GetAggregatedStats()["blocked_filtering"], float64(len(yandexDomains)))
	assert.Equal(t, s.GetAggregatedStats()["dns_queries"], float64(len(yandexDomains)))

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

	// Check aggregated stats
	assert.Equal(t, s.GetAggregatedStats()["replaced_safesearch"], float64(len(yandexDomains)+len(googleDomains)))
	assert.Equal(t, s.GetAggregatedStats()["blocked_filtering"], float64(len(yandexDomains)+len(googleDomains)))
	assert.Equal(t, s.GetAggregatedStats()["dns_queries"], float64(len(yandexDomains)+len(googleDomains)))

	// Do one more exchange
	exchangeAndAssertResponse(t, &client, addr, "google-public-dns-a.google.com.", "8.8.8.8")

	// Check aggregated stats
	assert.Equal(t, s.GetAggregatedStats()["replaced_safesearch"], float64(len(yandexDomains)+len(googleDomains)))
	assert.Equal(t, s.GetAggregatedStats()["blocked_filtering"], float64(len(yandexDomains)+len(googleDomains)))
	assert.Equal(t, s.GetAggregatedStats()["dns_queries"], float64(len(yandexDomains)+len(googleDomains)+1))

	// Count of blocked domains	(there is `yandex.com` duplicate in yandexDomains array)
	blockedCount := len(yandexDomains) - 1 + len(googleDomains)
	assert.Equal(t, len(s.GetStatsTop().Blocked), blockedCount)

	// Count of domains (blocked domains + `google-public-dns-a.google.com`)
	domainsCount := blockedCount + 1
	assert.Equal(t, len(s.GetStatsTop().Domains), domainsCount)

	err = s.Stop()
	if err != nil {
		t.Fatalf("Can not stopd server cause: %s", err)
	}
}

func TestInvalidRequest(t *testing.T) {
	s := createTestServer(t)
	defer removeDataDir(t)
	err := s.Start(nil)
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

	// check query log and stats
	// invalid requests aren't written to the query log
	log := s.GetQueryLog()
	assert.Equal(t, 0, len(log), "Log size")
	stats := s.GetStatsTop()
	assert.Equal(t, 0, len(stats.Domains), "Top domains length")
	assert.Equal(t, 0, len(stats.Blocked), "Top blocked length")
	assert.Equal(t, 0, len(stats.Clients), "Top clients length")

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestBlockedRequest(t *testing.T) {
	s := createTestServer(t)
	defer removeDataDir(t)
	err := s.Start(nil)
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

	// check query log and stats
	log := s.GetQueryLog()
	assert.Equal(t, 1, len(log), "Log size")
	stats := s.GetStatsTop()
	assert.Equal(t, 1, len(stats.Domains), "Top domains length")
	assert.Equal(t, 1, len(stats.Blocked), "Top blocked length")
	assert.Equal(t, 1, len(stats.Clients), "Top clients length")

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestNullBlockedRequest(t *testing.T) {
	s := createTestServer(t)
	s.conf.FilteringConfig.BlockingMode = "null_ip"
	defer removeDataDir(t)
	err := s.Start(nil)
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

	// check query log and stats
	log := s.GetQueryLog()
	assert.Equal(t, 1, len(log), "Log size")
	stats := s.GetStatsTop()
	assert.Equal(t, 1, len(stats.Domains), "Top domains length")
	assert.Equal(t, 1, len(stats.Blocked), "Top blocked length")
	assert.Equal(t, 1, len(stats.Clients), "Top clients length")

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestBlockedByHosts(t *testing.T) {
	s := createTestServer(t)
	defer removeDataDir(t)
	err := s.Start(nil)
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

	// check query log and stats
	log := s.GetQueryLog()
	assert.Equal(t, 1, len(log), "Log size")
	stats := s.GetStatsTop()
	assert.Equal(t, 1, len(stats.Domains), "Top domains length")
	assert.Equal(t, 1, len(stats.Blocked), "Top blocked length")
	assert.Equal(t, 1, len(stats.Clients), "Top clients length")

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func TestBlockedBySafeBrowsing(t *testing.T) {
	s := createTestServer(t)
	defer removeDataDir(t)
	err := s.Start(nil)
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

	// check query log and stats
	log := s.GetQueryLog()
	assert.Equal(t, 1, len(log), "Log size")
	stats := s.GetStatsTop()
	assert.Equal(t, 1, len(stats.Domains), "Top domains length")
	assert.Equal(t, 1, len(stats.Blocked), "Top blocked length")
	assert.Equal(t, 1, len(stats.Clients), "Top clients length")

	err = s.Stop()
	if err != nil {
		t.Fatalf("DNS server failed to stop: %s", err)
	}
}

func createTestServer(t *testing.T) *Server {
	s := NewServer(createDataDir(t))
	s.conf.UDPListenAddr = &net.UDPAddr{Port: 0}
	s.conf.TCPListenAddr = &net.TCPAddr{Port: 0}

	s.conf.QueryLogEnabled = true
	s.conf.FilteringConfig.FilteringEnabled = true
	s.conf.FilteringConfig.ProtectionEnabled = true
	s.conf.FilteringConfig.SafeBrowsingEnabled = true
	s.conf.Filters = make([]dnsfilter.Filter, 0)

	s.conf.SafeBrowsingCacheSize = 1000
	s.conf.SafeSearchCacheSize = 1000
	s.conf.ParentalCacheSize = 1000
	s.conf.CacheTime = 30

	rules := "||nxdomain.example.org^\n||null.example.org^\n127.0.0.1	host.example.org\n"
	filter := dnsfilter.Filter{ID: 0, Data: []byte(rules)}
	s.conf.Filters = append(s.conf.Filters, filter)
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

func createDataDir(t *testing.T) string {
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		t.Fatalf("Cannot create %s: %s", dataDir, err)
	}
	return dataDir
}

func removeDataDir(t *testing.T) {
	err := os.RemoveAll(dataDir)
	if err != nil {
		t.Fatalf("Cannot remove %s: %s", dataDir, err)
	}
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
	s := createTestServer(t)
	s.conf.AllowedClients = []string{"1.1.1.1", "2.2.0.0/16"}

	err := s.Start(nil)
	defer removeDataDir(t)
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

	if s.isBlockedIP("1.1.1.1") {
		t.Fatalf("isBlockedIP")
	}
	if !s.isBlockedIP("1.1.1.2") {
		t.Fatalf("isBlockedIP")
	}
	if s.isBlockedIP("2.2.1.1") {
		t.Fatalf("isBlockedIP")
	}
	if !s.isBlockedIP("2.3.1.1") {
		t.Fatalf("isBlockedIP")
	}
}

func TestIsBlockedIPDisallowed(t *testing.T) {
	s := createTestServer(t)
	s.conf.DisallowedClients = []string{"1.1.1.1", "2.2.0.0/16"}

	err := s.Start(nil)
	defer removeDataDir(t)
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

	if !s.isBlockedIP("1.1.1.1") {
		t.Fatalf("isBlockedIP")
	}
	if s.isBlockedIP("1.1.1.2") {
		t.Fatalf("isBlockedIP")
	}
	if !s.isBlockedIP("2.2.1.1") {
		t.Fatalf("isBlockedIP")
	}
	if s.isBlockedIP("2.3.1.1") {
		t.Fatalf("isBlockedIP")
	}
}

func TestIsBlockedIPBlockedDomain(t *testing.T) {
	s := createTestServer(t)
	s.conf.BlockedHosts = []string{"host1", "host2"}

	err := s.Start(nil)
	defer removeDataDir(t)
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

	if !s.isBlockedDomain("host1") {
		t.Fatalf("isBlockedDomain")
	}
	if !s.isBlockedDomain("host2") {
		t.Fatalf("isBlockedDomain")
	}
	if s.isBlockedDomain("host3") {
		t.Fatalf("isBlockedDomain")
	}
}
