package upstream

import (
	"crypto/tls"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
	"time"
)

// TODO: Use persistent connection here

// DnsOverTlsUpstream is the upstream implementation for plain DNS-over-TLS
type DnsOverTlsUpstream struct {
	endpoint      string
	tlsServerName string
	timeout       time.Duration
}

// NewHttpsUpstream creates a new DNS-over-TLS upstream from the endpoint address and TLS server name
func NewDnsOverTlsUpstream(endpoint string, tlsServerName string) (Upstream, error) {
	return &DnsOverTlsUpstream{
		endpoint:      endpoint,
		tlsServerName: tlsServerName,
		timeout:       defaultTimeout,
	}, nil
}

// Exchange provides an implementation for the Upstream interface
func (u *DnsOverTlsUpstream) Exchange(ctx context.Context, query *dns.Msg) (*dns.Msg, error) {

	dnsClient := &dns.Client{
		Net:          "tcp-tls",
		ReadTimeout:  u.timeout,
		WriteTimeout: u.timeout,
		TLSConfig:    new(tls.Config),
	}
	dnsClient.TLSConfig.ServerName = u.tlsServerName

	resp, _, err := dnsClient.Exchange(query, u.endpoint)

	if err != nil {
		resp = &dns.Msg{}
		resp.SetRcode(resp, dns.RcodeServerFailure)
	}

	return resp, err
}
