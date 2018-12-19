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
		{
			// AdGuard DNS (DNSCrypt)
			address:   "sdns://AQIAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
			bootstrap: "",
		},
		{
			// Cisco OpenDNS (DNSCrypt)
			address:   "sdns://AQAAAAAAAAAADjIwOC42Ny4yMjAuMjIwILc1EUAgbyJdPivYItf9aR6hwzzI1maNDL4Ev6vKQ_t5GzIuZG5zY3J5cHQtY2VydC5vcGVuZG5zLmNvbQ",
			bootstrap: "8.8.8.8:53",
		},
		{
			// Cloudflare DNS (DoH)
			address:   "sdns://AgcAAAAAAAAABzEuMC4wLjGgENk8mGSlIfMGXMOlIlCcKvq7AVgcrZxtjon911-ep0cg63Ul-I8NlFj4GplQGb_TTLiczclX57DvMV8Q-JdjgRgSZG5zLmNsb3VkZmxhcmUuY29tCi9kbnMtcXVlcnk",
			bootstrap: "8.8.8.8:53",
		},
		{
			// doh-cleanbrowsing-security (https://doh.cleanbrowsing.org/doh/security-filter/)
			address:   "sdns://AgMAAAAAAAAAAAAVZG9oLmNsZWFuYnJvd3Npbmcub3JnFS9kb2gvc2VjdXJpdHktZmlsdGVyLw",
			bootstrap: "8.8.8.8:53",
		},
		{
			// Google (DNS-over-HTTPS)
			address:   "sdns://AgUAAAAAAAAAACAe9iTP_15r07rd8_3b_epWVGfjdymdx-5mdRZvMAzBuQ5kbnMuZ29vZ2xlLmNvbQ0vZXhwZXJpbWVudGFs",
			bootstrap: "8.8.8.8:53",
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
