package dnsforward

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestServer(t *testing.T) {
	s := Server{}
	s.UDPListenAddr = &net.UDPAddr{Port: 0}
	err := s.Start(nil)
	if err != nil {
		t.Fatalf("Failed to start server: %s", err)
	}
	if s.udpListen == nil {
		t.Fatal("Started server has nil udpListen")
	}

	// server is running, send a message
	addr := s.udpListen.LocalAddr()
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
		t.Fatalf("DNS server %s failed to stop: %s", addr, err)
	}
}
