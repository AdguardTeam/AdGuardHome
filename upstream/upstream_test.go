package upstream

import (
	"github.com/miekg/dns"
	"log"
	"net"
	"testing"
)

func TestDnsUpstream(t *testing.T) {

	u, err := NewDnsUpstream("8.8.8.8:53")

	if err != nil {
		t.Errorf("cannot create a DNS upstream")
	}

	testUpstream(t, u)
}

func TestHttpsUpstream(t *testing.T) {

	testCases := []string{
		"https://cloudflare-dns.com/dns-query",
		"https://dns.google.com/experimental",
		"https://doh.cleanbrowsing.org/doh/security-filter/",
	}

	for _, url := range testCases {
		u, err := NewHttpsUpstream(url)

		if err != nil {
			t.Errorf("cannot create a DNS-over-HTTPS upstream")
		}

		testUpstream(t, u)
	}
}

func TestDnsOverTlsUpstream(t *testing.T) {

	var tests = []struct {
		endpoint      string
		tlsServerName string
	}{
		{"1.1.1.1:853", ""},
		{"8.8.8.8:853", ""},
		{"185.228.168.10:853", "security-filter-dns.cleanbrowsing.org"},
	}

	for _, test := range tests {
		u, err := NewDnsOverTlsUpstream(test.endpoint, test.tlsServerName)

		if err != nil {
			t.Errorf("cannot create a DNS-over-TLS upstream")
		}

		testUpstream(t, u)
	}
}

func testUpstream(t *testing.T, u Upstream) {
	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{Name: "google-public-dns-a.google.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}

	resp, err := u.Exchange(nil, &req)

	if err != nil {
		t.Errorf("error while making an upstream request: %s", err)
	}

	if len(resp.Answer) != 1 {
		t.Errorf("no answer section in the response")
	}
	if answer, ok := resp.Answer[0].(*dns.A); ok {
		if !net.IPv4(8, 8, 8, 8).Equal(answer.A) {
			t.Errorf("wrong IP in the response: %v", answer.A)
		}
	}

	log.Printf("response: %v", resp)
}
