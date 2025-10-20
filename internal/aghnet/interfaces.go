package aghnet

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"
)

// IPVersion is a alias for int for documentation purposes.  Use it when the
// integer means IP version.
type IPVersion = int

// IP version constants.
const (
	IPVersion4 IPVersion = 4
	IPVersion6 IPVersion = 6
)

// NetIface is the interface for network interface methods.
type NetIface interface {
	Addrs() ([]net.Addr, error)
}

// IfaceIPAddrs returns the interface's IP addresses.  iface must not be nil.
func IfaceIPAddrs(iface NetIface, ipv IPVersion) (ips []net.IP, err error) {
	switch ipv {
	case IPVersion4, IPVersion6:
		// Go on.
	default:
		return nil, fmt.Errorf("invalid ip version %d", ipv)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	for _, a := range addrs {
		if ip := ipFromAddr(a, ipv); ip != nil {
			ips = append(ips, ip)
		}
	}

	return ips, nil
}

// ipFromAddr converts addr to IP.  addr must not be nil.
func ipFromAddr(addr net.Addr, ipv IPVersion) (ip net.IP) {
	switch addr := addr.(type) {
	case *net.IPAddr:
		ip = addr.IP
	case *net.IPNet:
		ip = addr.IP
	default:
		return nil
	}

	// Assume that net.Addr can only be valid IPv4 or IPv6.  Thus,
	// if it isn't an IPv4 address, it must be an IPv6 one.
	ip4 := ip.To4()
	if ipv == IPVersion4 {
		return ip4
	} else if ip4 == nil {
		return ip
	}

	return nil
}

// IfaceDNSIPAddrs returns IP addresses of the interface suitable to send to
// clients as DNS addresses.  If err is nil, addrs contains either no addresses
// or at least two.  l must not be nil.
//
// It makes up to maxAttempts attempts to get the addresses if there are none,
// each time using the provided backoff.  Sometimes an interface needs a few
// seconds to really initialize.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/2304.
func IfaceDNSIPAddrs(
	ctx context.Context,
	l *slog.Logger,
	iface NetIface,
	ipv IPVersion,
	maxAttempts int,
	backoff time.Duration,
) (addrs []net.IP, err error) {
	var n int
	for n = 1; n <= maxAttempts; n++ {
		addrs, err = IfaceIPAddrs(iface, ipv)
		if err != nil {
			return nil, fmt.Errorf("getting ip addrs: %w", err)
		}

		if len(addrs) > 0 {
			break
		}

		l.DebugContext(ctx, "no ip addresses", "attempt", n, "ipv", ipv)

		time.Sleep(backoff)
	}

	n--

	switch len(addrs) {
	case 0:
		// Don't return errors in case the users want to try and enable the DHCP
		// server later.
		t := time.Duration(n) * backoff
		l.ErrorContext(ctx, "no ip addresses for iface", "attempts", n, "duration", t, "ipv", ipv)

		return nil, nil
	case 1:
		// Some Android devices use 8.8.8.8 if there is not a secondary DNS
		// server.  Fix that by setting the secondary DNS address to the same
		// address.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/1708.
		l.DebugContext(ctx, "setting secondary dns ip to itself", "ipv", ipv)
		addrs = append(addrs, addrs[0])
	default:
		// Go on.
	}

	l.DebugContext(ctx, "got addresses", "addrs", addrs, "attempts", n, "ipv", ipv)

	return addrs, nil
}

// interfaceName is a string containing network interface's name.  The name is
// used in file walking methods.
type interfaceName string

// Use interfaceName in the OS-independent code since it's actually only used in
// several OS-dependent implementations which causes linting issues.
var _ = interfaceName("")
