package dhcpd

import (
	"fmt"
	"net"
	"time"

	"github.com/AdguardTeam/golibs/log"
)

// ipVersion is a documentational alias for int.  Use it when the integer means
// IP version.
type ipVersion = int

// IP version constants.
const (
	ipVersion4 ipVersion = 4
	ipVersion6 ipVersion = 6
)

// netIface is the interface for network interface methods.
type netIface interface {
	Addrs() ([]net.Addr, error)
}

// ifaceIPAddrs returns the interface's IP addresses.
func ifaceIPAddrs(iface netIface, ipv ipVersion) (ips []net.IP, err error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	for _, a := range addrs {
		var ip net.IP
		switch a := a.(type) {
		case *net.IPAddr:
			ip = a.IP
		case *net.IPNet:
			ip = a.IP
		default:
			continue
		}

		// Assume that net.(*Interface).Addrs can only return valid IPv4
		// and IPv6 addresses.  Thus, if it isn't an IPv4 address, it
		// must be an IPv6 one.
		switch ipv {
		case ipVersion4:
			if ip4 := ip.To4(); ip4 != nil {
				ips = append(ips, ip4)
			}
		case ipVersion6:
			if ip6 := ip.To4(); ip6 == nil {
				ips = append(ips, ip)
			}
		default:
			return nil, fmt.Errorf("invalid ip version %d", ipv)
		}
	}

	return ips, nil
}

// Currently used defaults for ifaceDNSAddrs.
const (
	defaultMaxAttempts int = 10

	defaultBackoff time.Duration = 500 * time.Millisecond
)

// ifaceDNSIPAddrs returns IP addresses of the interface suitable to send to
// clients as DNS addresses.  If err is nil, addrs contains either no addresses
// or at least two.
//
// It makes up to maxAttempts attempts to get the addresses if there are none,
// each time using the provided backoff.  Sometimes an interface needs a few
// seconds to really ititialize.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/2304.
func ifaceDNSIPAddrs(
	iface netIface,
	ipv ipVersion,
	maxAttempts int,
	backoff time.Duration,
) (addrs []net.IP, err error) {
	var n int
waitForIP:
	for n = 1; n <= maxAttempts; n++ {
		addrs, err = ifaceIPAddrs(iface, ipv)
		if err != nil {
			return nil, fmt.Errorf("getting ip addrs: %w", err)
		}

		switch len(addrs) {
		case 0:
			log.Debug("dhcpv%d: attempt %d: no ip addresses", ipv, n)

			time.Sleep(backoff)
		case 1:
			// Some Android devices use 8.8.8.8 if there is not
			// a secondary DNS server.  Fix that by setting the
			// secondary DNS address to the same address.
			//
			// See https://github.com/AdguardTeam/AdGuardHome/issues/1708.
			log.Debug("dhcpv%d: setting secondary dns ip to itself", ipv)
			addrs = append(addrs, addrs[0])

			fallthrough
		default:
			break waitForIP
		}
	}

	if len(addrs) == 0 {
		// Don't return errors in case the users want to try and enable
		// the DHCP server later.
		log.Error("dhcpv%d: no ip address for interface after %d attempts and %s", ipv, n, time.Duration(n)*backoff)
	} else {
		log.Debug("dhcpv%d: got addresses %s after %d attempts", ipv, addrs, n)
	}

	return addrs, nil
}
