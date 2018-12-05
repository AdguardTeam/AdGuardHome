package dnsforward

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestUpstreamDNS(t *testing.T) {
	upstreams := []string{
		"8.8.8.8:53",
		"1.1.1.1",
		"tcp://1.1.1.1:53",
		"176.103.130.130:5353",
	}
	for _, input := range upstreams {
		u, err := GetUpstream(input)
		if err != nil {
			t.Fatalf("Failed to choose upstream for %s: %s", input, err)
		}

		checkUpstream(t, u, input)
	}
}

func TestUpstreamTLS(t *testing.T) {
	upstreams := []string{
		"tls://1.1.1.1",
		"tls://9.9.9.9:853",
		"tls://security-filter-dns.cleanbrowsing.org",
		"tls://adult-filter-dns.cleanbrowsing.org:853",
	}
	for _, input := range upstreams {
		u, err := GetUpstream(input)
		if err != nil {
			t.Fatalf("Failed to choose upstream for %s: %s", input, err)
		}

		checkUpstream(t, u, input)
	}
}

func TestUpstreamHTTPS(t *testing.T) {
	upstreams := []string{
		"https://cloudflare-dns.com/dns-query",
		"https://dns.google.com/experimental",
		"https://doh.cleanbrowsing.org/doh/security-filter/",
	}
	for _, input := range upstreams {
		u, err := GetUpstream(input)
		if err != nil {
			t.Fatalf("Failed to choose upstream for %s: %s", input, err)
		}

		checkUpstream(t, u, input)
	}
}

func checkUpstream(t *testing.T, u Upstream, addr string) {
	t.Helper()

	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{Name: "google-public-dns-a.google.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}

	reply, err := u.Exchange(&req)
	if err != nil {
		t.Fatalf("Couldn't talk to upstream %s: %s", addr, err)
	}
	if len(reply.Answer) != 1 {
		t.Fatalf("DNS upstream %s returned reply with wrong number of answers - %d", addr, len(reply.Answer))
	}
	if a, ok := reply.Answer[0].(*dns.A); ok {
		if !net.IPv4(8, 8, 8, 8).Equal(a.A) {
			t.Fatalf("DNS upstream %s returned wrong answer instead of 8.8.8.8: %v", addr, a.A)
		}
	} else {
		t.Fatalf("DNS upstream %s returned wrong answer type instead of A: %v", addr, reply.Answer[0])
	}
}
