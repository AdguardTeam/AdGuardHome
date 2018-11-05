package upstream

import (
	"github.com/miekg/dns"
	"golang.org/x/net/context"
	"net"
	"strings"
)

// Detects the upstream type from the specified url and creates a proper Upstream object
func NewUpstream(url string, bootstrap string) (Upstream, error) {

	proto := "udp"
	prefix := ""

	switch {
	case strings.HasPrefix(url, "tcp://"):
		proto = "tcp"
		prefix = "tcp://"
	case strings.HasPrefix(url, "tls://"):
		proto = "tcp-tls"
		prefix = "tls://"
	case strings.HasPrefix(url, "https://"):
		return NewHttpsUpstream(url, bootstrap)
	}

	hostname := strings.TrimPrefix(url, prefix)

	host, port, err := net.SplitHostPort(hostname)
	if err != nil {
		// Set port depending on the protocol
		switch proto {
		case "udp":
			port = "53"
		case "tcp":
			port = "53"
		case "tcp-tls":
			port = "853"
		}

		// Set host = hostname
		host = hostname
	}

	// Try to resolve the host address (or check if it's an IP address)
	bootstrapResolver := CreateResolver(bootstrap)
	ips, err := bootstrapResolver.LookupIPAddr(context.Background(), host)

	if err != nil || len(ips) == 0 {
		return nil, err
	}

	addr := ips[0].String()
	endpoint := net.JoinHostPort(addr, port)
	tlsServerName := ""

	if proto == "tcp-tls" && host != addr {
		// Check if we need to specify TLS server name
		tlsServerName = host
	}

	return NewDnsUpstream(endpoint, proto, tlsServerName)
}

func CreateResolver(bootstrap string) *net.Resolver {

	bootstrapResolver := net.DefaultResolver

	if bootstrap != "" {
		bootstrapResolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				var d net.Dialer
				conn, err := d.DialContext(ctx, network, bootstrap)
				return conn, err
			},
		}
	}

	return bootstrapResolver
}

// Performs a simple health-check of the specified upstream
func IsAlive(u Upstream) (bool, error) {

	// Using ipv4only.arpa. domain as it is a part of DNS64 RFC and it should exist everywhere
	ping := new(dns.Msg)
	ping.SetQuestion("ipv4only.arpa.", dns.TypeA)

	resp, err := u.Exchange(context.Background(), ping)

	// If we got a header, we're alright, basically only care about I/O errors 'n stuff.
	if err != nil && resp != nil {
		// Silly check, something sane came back.
		if resp.Response || resp.Opcode == dns.OpcodeQuery {
			err = nil
		}
	}

	return err == nil, err
}
