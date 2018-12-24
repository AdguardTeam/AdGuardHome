package dnsforward

import (
	"net"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"

	"github.com/miekg/dns"
)

const (
	listenPort = 48122
)

func TestServer(t *testing.T) {
	s := Server{}
	s.UDPListenAddr = &net.UDPAddr{Port: listenPort}
	err := s.Start(nil)
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

	// server is running, send a message
	addr := s.UDPListenAddr
	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{Name: "google-public-dns-a.google.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}

	reply, err := dns.Exchange(&req, addr.String())
	if err != nil {
		t.Fatalf("Couldn't talk to server %s: %s", addr, err)
	}
	if len(reply.Answer) != 1 {
		t.Fatalf("DNS server %s returned reply with wrong number of answers - %d", addr, len(reply.Answer))
	}
	if a, ok := reply.Answer[0].(*dns.A); ok {
		if !net.IPv4(8, 8, 8, 8).Equal(a.A) {
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

func TestInvalidRequest(t *testing.T) {
	s := Server{}
	s.UDPListenAddr = &net.UDPAddr{Port: listenPort}
	err := s.Start(nil)
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

	// server is running, send a message
	addr := s.UDPListenAddr
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
	s, addr := createTestServer()

	err := s.Start(nil)
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

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

func TestBlockedByHosts(t *testing.T) {
	s, addr := createTestServer()

	err := s.Start(nil)
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

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
	s, addr := createTestServer()

	err := s.Start(nil)
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}

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

func createTestServer() (*Server, net.Addr) {
	s := Server{}
	addr := &net.UDPAddr{Port: listenPort}
	s.UDPListenAddr = addr
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
	return &s, addr
}
