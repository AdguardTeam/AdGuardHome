package upstream

import (
	"github.com/miekg/dns"
	"golang.org/x/net/context"
	"time"
)

// DnsUpstream is a very simple upstream implementation for plain DNS
type DnsUpstream struct {
	nameServer string        // IP:port
	timeout    time.Duration // Max read and write timeout
}

// NewDnsUpstream creates a new plain-DNS upstream
func NewDnsUpstream(nameServer string) (Upstream, error) {
	return &DnsUpstream{nameServer: nameServer, timeout: defaultTimeout}, nil
}

// Exchange provides an implementation for the Upstream interface
func (u *DnsUpstream) Exchange(ctx context.Context, query *dns.Msg) (*dns.Msg, error) {

	dnsClient := &dns.Client{
		ReadTimeout:  u.timeout,
		WriteTimeout: u.timeout,
	}

	resp, _, err := dnsClient.Exchange(query, u.nameServer)

	if err != nil {
		resp = &dns.Msg{}
		resp.SetRcode(resp, dns.RcodeServerFailure)
	}

	return resp, err
}
