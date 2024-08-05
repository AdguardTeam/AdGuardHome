package dnsforward

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
)

// DialContext is an [aghnet.DialContextFunc] that uses s to resolve hostnames.
// addr should be a valid host:port address, where host could be a domain name
// or an IP address.
func (s *Server) DialContext(ctx context.Context, network, addr string) (conn net.Conn, err error) {
	log.Debug("dnsforward: dialing %q for network %q", addr, network)

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		// TODO(a.garipov): Consider making configurable.
		Timeout: time.Minute * 5,
	}

	if netutil.IsValidIPString(host) {
		return dialer.DialContext(ctx, network, addr)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port %s: %w", portStr, err)
	}

	ips, err := s.Resolve(ctx, network, host)
	if err != nil {
		return nil, fmt.Errorf("resolving %q: %w", host, err)
	} else if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses for host %q", host)
	}

	log.Debug("dnsforward: resolved %q: %v", host, ips)

	var dialErrs []error
	for _, ip := range ips {
		addrPort := netip.AddrPortFrom(ip, uint16(port))
		conn, err = dialer.DialContext(ctx, network, addrPort.String())
		if err != nil {
			dialErrs = append(dialErrs, err)

			continue
		}

		return conn, nil
	}

	return nil, errors.Join(dialErrs...)
}
