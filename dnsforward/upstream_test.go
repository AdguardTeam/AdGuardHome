package dnsforward

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestUpstreams(t *testing.T) {
	upstreams := []struct {
		address   string
		bootstrap string
	}{
		{
			address:   "8.8.8.8:53",
			bootstrap: "8.8.8.8:53",
		},
		{
			address:   "1.1.1.1",
			bootstrap: "",
		},
		{
			address:   "tcp://1.1.1.1:53",
			bootstrap: "",
		},
		{
			address:   "176.103.130.130:5353",
			bootstrap: "",
		},
		{
			address:   "tls://1.1.1.1",
			bootstrap: "",
		},
		{
			address:   "tls://9.9.9.9:853",
			bootstrap: "",
		},
		{
			address:   "tls://security-filter-dns.cleanbrowsing.org",
			bootstrap: "8.8.8.8:53",
		},
		{
			address:   "tls://adult-filter-dns.cleanbrowsing.org:853",
			bootstrap: "8.8.8.8:53",
		},
		{
			address:   "https://cloudflare-dns.com/dns-query",
			bootstrap: "8.8.8.8:53",
		},
		{
			address:   "https://dns.google.com/experimental",
			bootstrap: "8.8.8.8:53",
		},
		{
			address:   "https://doh.cleanbrowsing.org/doh/security-filter/",
			bootstrap: "",
		},
	}
	for _, test := range upstreams {
		t.Run(test.address, func(t *testing.T) {
			u, err := AddressToUpstream(test.address, test.bootstrap)
			if err != nil {
				t.Fatalf("Failed to generate upstream from address %s: %s", test.address, err)
			}

			checkUpstream(t, u, test.address)
		})
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
