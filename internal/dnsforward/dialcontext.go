package dnsforward

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// DialContext is a [whois.DialContextFunc] that uses s to resolve hostnames.
func (s *Server) DialContext(ctx context.Context, network, addr string) (conn net.Conn, err error) {
	log.Debug("dnsforward: dialing %q for network %q", addr, network)

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		// TODO(a.garipov): Consider making configurable.
		Timeout: time.Minute * 5,
	}

	if net.ParseIP(host) != nil {
		return dialer.DialContext(ctx, network, addr)
	}

	addrs, err := s.Resolve(host)
	if err != nil {
		return nil, fmt.Errorf("resolving %q: %w", host, err)
	}

	log.Debug("dnsforward: resolving %q: %v", host, addrs)

	if len(addrs) == 0 {
		return nil, fmt.Errorf("no addresses for host %q", host)
	}

	var dialErrs []error
	for _, a := range addrs {
		addr = net.JoinHostPort(a.String(), port)
		conn, err = dialer.DialContext(ctx, network, addr)
		if err != nil {
			dialErrs = append(dialErrs, err)

			continue
		}

		return conn, err
	}

	// TODO(a.garipov): Use errors.Join in Go 1.20.
	return nil, errors.List(fmt.Sprintf("dialing %q", addr), dialErrs...)
}
