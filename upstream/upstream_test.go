package upstream

import (
	"github.com/miekg/dns"
	"net"
	"testing"
)

func TestDnsUpstreamIsAlive(t *testing.T) {

	var tests = []struct {
		endpoint string
		proto    string
	}{
		{"8.8.8.8:53", "udp"},
		{"8.8.8.8:53", "tcp"},
		{"1.1.1.1:53", "udp"},
	}

	for _, test := range tests {
		u, err := NewDnsUpstream(test.endpoint, test.proto, "")

		if err != nil {
			t.Errorf("cannot create a DNS upstream")
		}

		testUpstreamIsAlive(t, u)
	}
}

func TestHttpsUpstreamIsAlive(t *testing.T) {

	var tests = []struct {
		url       string
		bootstrap string
	}{
		{"https://cloudflare-dns.com/dns-query", "8.8.8.8:53"},
		{"https://dns.google.com/experimental", "8.8.8.8:53"},
		{"https://doh.cleanbrowsing.org/doh/security-filter/", ""}, // TODO: status 201??
	}

	for _, test := range tests {
		u, err := NewHttpsUpstream(test.url, test.bootstrap)

		if err != nil {
			t.Errorf("cannot create a DNS-over-HTTPS upstream")
		}

		testUpstreamIsAlive(t, u)
	}
}

func TestDnsOverTlsIsAlive(t *testing.T) {

	var tests = []struct {
		endpoint      string
		tlsServerName string
	}{
		{"1.1.1.1:853", ""},
		{"9.9.9.9:853", ""},
		{"185.228.168.10:853", "security-filter-dns.cleanbrowsing.org"},
	}

	for _, test := range tests {
		u, err := NewDnsUpstream(test.endpoint, "tcp-tls", test.tlsServerName)

		if err != nil {
			t.Errorf("cannot create a DNS-over-TLS upstream")
		}

		testUpstreamIsAlive(t, u)
	}
}

func TestDnsUpstream(t *testing.T) {

	var tests = []struct {
		endpoint string
		proto    string
	}{
		{"8.8.8.8:53", "udp"},
		{"8.8.8.8:53", "tcp"},
		{"1.1.1.1:53", "udp"},
	}

	for _, test := range tests {
		u, err := NewDnsUpstream(test.endpoint, test.proto, "")

		if err != nil {
			t.Errorf("cannot create a DNS upstream")
		}

		testUpstream(t, u)
	}
}

func TestHttpsUpstream(t *testing.T) {

	var tests = []struct {
		url       string
		bootstrap string
	}{
		{"https://cloudflare-dns.com/dns-query", "8.8.8.8:53"},
		{"https://dns.google.com/experimental", "8.8.8.8:53"},
		{"https://doh.cleanbrowsing.org/doh/security-filter/", ""},
	}

	for _, test := range tests {
		u, err := NewHttpsUpstream(test.url, test.bootstrap)

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
		{"9.9.9.9:853", ""},
		{"185.228.168.10:853", "security-filter-dns.cleanbrowsing.org"},
	}

	for _, test := range tests {
		u, err := NewDnsUpstream(test.endpoint, "tcp-tls", test.tlsServerName)

		if err != nil {
			t.Errorf("cannot create a DNS-over-TLS upstream")
		}

		testUpstream(t, u)
	}
}

func testUpstreamIsAlive(t *testing.T, u Upstream) {
	alive, err := IsAlive(u)
	if !alive || err != nil {
		t.Errorf("Upstream is not alive")
	}

	u.Close()
}

func testUpstream(t *testing.T, u Upstream) {

	var tests = []struct {
		name     string
		expected net.IP
	}{
		{"google-public-dns-a.google.com.", net.IPv4(8, 8, 8, 8)},
		{"google-public-dns-b.google.com.", net.IPv4(8, 8, 4, 4)},
	}

	for _, test := range tests {
		req := dns.Msg{}
		req.Id = dns.Id()
		req.RecursionDesired = true
		req.Question = []dns.Question{
			{Name: test.name, Qtype: dns.TypeA, Qclass: dns.ClassINET},
		}

		resp, err := u.Exchange(nil, &req)

		if err != nil {
			t.Errorf("error while making an upstream request: %s", err)
		}

		if len(resp.Answer) != 1 {
			t.Errorf("no answer section in the response")
		}
		if answer, ok := resp.Answer[0].(*dns.A); ok {
			if !test.expected.Equal(answer.A) {
				t.Errorf("wrong IP in the response: %v", answer.A)
			}
		}
	}

	err := u.Close()
	if err != nil {
		t.Errorf("Error while closing the upstream: %s", err)
	}
}
