package upstream

import (
	"crypto/tls"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
	"time"
)

// DnsUpstream is a very simple upstream implementation for plain DNS
type DnsUpstream struct {
	endpoint  string        // IP:port
	timeout   time.Duration // Max read and write timeout
	proto     string        // Protocol (tcp, tcp-tls, or udp)
	transport *Transport    // Persistent connections cache
}

// NewDnsUpstream creates a new DNS upstream
func NewDnsUpstream(endpoint string, proto string, tlsServerName string) (Upstream, error) {

	u := &DnsUpstream{
		endpoint: endpoint,
		timeout:  defaultTimeout,
		proto:    proto,
	}

	var tlsConfig *tls.Config

	if tlsServerName != "" {
		tlsConfig = new(tls.Config)
		tlsConfig.ServerName = tlsServerName
	}

	// Initialize the connections cache
	u.transport = NewTransport(endpoint)
	u.transport.tlsConfig = tlsConfig
	u.transport.Start()

	return u, nil
}

// Exchange provides an implementation for the Upstream interface
func (u *DnsUpstream) Exchange(ctx context.Context, query *dns.Msg) (*dns.Msg, error) {

	resp, err := u.exchange(u.proto, query)

	// Retry over TCP if response is truncated
	if err == dns.ErrTruncated && u.proto == "udp" {
		resp, err = u.exchange("tcp", query)
	} else if err == dns.ErrTruncated && resp != nil {
		// Reassemble something to be sent to client
		m := new(dns.Msg)
		m.SetReply(query)
		m.Truncated = true
		m.Authoritative = true
		m.Rcode = dns.RcodeSuccess
		return m, nil
	}

	if err != nil {
		resp = &dns.Msg{}
		resp.SetRcode(resp, dns.RcodeServerFailure)
	}

	return resp, err
}

// Clear resources
func (u *DnsUpstream) Close() error {

	// Close active connections
	u.transport.Stop()
	return nil
}

// Performs a synchronous query. It sends the message m via the conn
// c and waits for a reply. The conn c is not closed.
func (u *DnsUpstream) exchange(proto string, query *dns.Msg) (r *dns.Msg, err error) {

	// Establish a connection if needed (or reuse cached)
	conn, err := u.transport.Dial(proto)
	if err != nil {
		return nil, err
	}

	// Write the request with a timeout
	conn.SetWriteDeadline(time.Now().Add(u.timeout))
	if err = conn.WriteMsg(query); err != nil {
		conn.Close() // Not giving it back
		return nil, err
	}

	// Write response with a timeout
	conn.SetReadDeadline(time.Now().Add(u.timeout))
	r, err = conn.ReadMsg()
	if err != nil {
		conn.Close() // Not giving it back
	} else if err == nil && r.Id != query.Id {
		err = dns.ErrId
		conn.Close() // Not giving it back
	}

	u.transport.Yield(conn)
	return r, err
}
