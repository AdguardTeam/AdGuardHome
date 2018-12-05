package dnsforward

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"

	"github.com/joomcode/errorx"
)

type bootstrapper struct {
	address        string        // in form of "tls://one.one.one.one:853"
	resolver       *net.Resolver // resolver to use to resolve hostname, if neccessary
	resolved       string        // in form "IP:port"
	resolvedConfig *tls.Config
	sync.Mutex
}

func toBoot(address, bootstrapAddr string) bootstrapper {
	var resolver *net.Resolver
	if bootstrapAddr != "" {
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, network, bootstrapAddr)
			},
		}
	}
	return bootstrapper{
		address:  address,
		resolver: resolver,
	}
}

// will get usable IP address from Address field, and caches the result
func (n *bootstrapper) get() (string, *tls.Config, error) {
	// TODO: RLock() here but atomically upgrade to Lock() if fast path doesn't work
	n.Lock()
	if n.resolved != "" { // fast path
		retval, tlsconfig := n.resolved, n.resolvedConfig
		n.Unlock()
		return retval, tlsconfig, nil
	}

	//
	// slow path
	//

	defer n.Unlock()

	justHostPort := n.address
	if strings.Contains(n.address, "://") {
		url, err := url.Parse(n.address)
		if err != nil {
			return "", nil, errorx.Decorate(err, "Failed to parse %s", n.address)
		}

		justHostPort = url.Host
	}

	// convert host to IP if neccessary, we know that it's scheme://hostname:port/

	// get a host without port
	host, port, err := net.SplitHostPort(justHostPort)
	if err != nil {
		return "", nil, fmt.Errorf("bootstrapper requires port in address %s", n.address)
	}

	// if it's an IP
	ip := net.ParseIP(host)
	if ip != nil {
		n.resolved = justHostPort
		return n.resolved, nil, nil
	}

	//
	// if it's a hostname
	//

	resolver := n.resolver // no need to check for nil resolver -- documented that nil is default resolver
	addrs, err := resolver.LookupIPAddr(context.TODO(), host)
	if err != nil {
		return "", nil, errorx.Decorate(err, "Failed to lookup %s", host)
	}
	for _, addr := range addrs {
		// TODO: support ipv6, support multiple ipv4
		if addr.IP.To4() == nil {
			continue
		}
		ip = addr.IP
		break
	}

	if ip == nil {
		// couldn't find any suitable IP address
		return "", nil, fmt.Errorf("Couldn't find any suitable IP address for host %s", host)
	}

	n.resolved = net.JoinHostPort(ip.String(), port)
	n.resolvedConfig = &tls.Config{ServerName: host}
	return n.resolved, n.resolvedConfig, nil
}
