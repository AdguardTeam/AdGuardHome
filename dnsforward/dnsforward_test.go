package dnsforward

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/miekg/dns"
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
	addr := s.dnsProxy.Addr("udp")
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

func TestSafeSearch(t *testing.T) {
	s := createTestServer(t)
	s.SafeSearchEnabled = true
	defer removeDataDir(t)
	err := s.Start(nil)
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

	// Test safesearch for yandex. We already know safesearch ip
	addr := s.dnsProxy.Addr("udp")
	client := dns.Client{Net: "udp"}
	yandexDomains := []string{"yandex.com.", "yandex.by.", "yandex.kz.", "yandex.ru.", "yandex.com."}
	for _, host := range yandexDomains {
		exchangeAndAssertResponse(t, client, addr, host, "213.180.193.56")
	}

	// Let's lookup for google safesearch ip
	ips, err := net.LookupIP("forcesafesearch.google.com")
	if err != nil {
		t.Fatalf("Failed to lookup for forcesafesearch.google.com: %s", err)
	}

	ip := ips[0]
	for _, i := range ips {
		if len(i) == net.IPv6len && i.To4() != nil {
			ip = i
			break
		}
	}

	// Test safeseacrh for google.
	googleDomains := []string{"www.google.com.", "www.google.com.af.", "www.google.be.", "www.google.by."}
	for _, host := range googleDomains {
		exchangeAndAssertResponse(t, client, addr, host, ip.String())
	}

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
	addr := s.dnsProxy.Addr("udp")
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
	addr := s.dnsProxy.Addr("udp")

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

func TestBlockedByHosts(t *testing.T) {
	s := createTestServer(t)
	defer removeDataDir(t)
	err := s.Start(nil)
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}
	addr := s.dnsProxy.Addr("udp")

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
	addr := s.dnsProxy.Addr("udp")

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
	s.UDPListenAddr = &net.UDPAddr{Port: 0}
	s.TCPListenAddr = &net.TCPAddr{Port: 0}
	s.QueryLogEnabled = true
	s.FilteringConfig.FilteringEnabled = true
	s.FilteringConfig.ProtectionEnabled = true
	s.FilteringConfig.SafeBrowsingEnabled = true
	s.Filters = make([]dnsfilter.Filter, 0)

	rules := []string{
		"||nxdomain.example.org^",
		"127.0.0.1	host.example.org",
	}
	filter := dnsfilter.Filter{ID: 1, Rules: rules}
	s.Filters = append(s.Filters, filter)
	return s
}

func createDataDir(t *testing.T) string {
	dir := "testData"
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		t.Fatalf("Cannot create %s: %s", dir, err)
	}
	return dir
}

func removeDataDir(t *testing.T) {
	dir := "testData"
	err := os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("Cannot remove %s: %s", dir, err)
	}
}

func exchangeAndAssertResponse(t *testing.T, client dns.Client, addr net.Addr, host, ip string) {
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
