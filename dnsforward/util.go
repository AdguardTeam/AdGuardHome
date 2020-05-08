package dnsforward

import (
	"net"
	"sort"
	"strings"

	"github.com/AdguardTeam/golibs/utils"
)

func stringArrayDup(a []string) []string {
	a2 := make([]string, len(a))
	copy(a2, a)
	return a2
}

// Get IP address from net.Addr object
// Note: we can't use net.SplitHostPort(a.String()) because of IPv6 zone:
// https://github.com/AdguardTeam/AdGuardHome/issues/1261
func ipFromAddr(a net.Addr) string {
	switch addr := a.(type) {
	case *net.UDPAddr:
		return addr.IP.String()
	case *net.TCPAddr:
		return addr.IP.String()
	}
	return ""
}

// Get IP address from net.Addr
func getIP(addr net.Addr) net.IP {
	switch addr := addr.(type) {
	case *net.UDPAddr:
		return addr.IP
	case *net.TCPAddr:
		return addr.IP
	}
	return nil
}

// Find value in a sorted array
func findSorted(ar []string, val string) int {
	i := sort.SearchStrings(ar, val)
	if i == len(ar) || ar[i] != val {
		return -1
	}
	return i
}

func isWildcard(host string) bool {
	return len(host) >= 2 &&
		host[0] == '*' && host[1] == '.'
}

// Return TRUE if host name matches a wildcard pattern
func matchDomainWildcard(host, wildcard string) bool {
	return isWildcard(wildcard) &&
		strings.HasSuffix(host, wildcard[1:])
}

// Return TRUE if client's SNI value matches DNS names from certificate
func matchDNSName(dnsNames []string, sni string) bool {
	if utils.IsValidHostname(sni) != nil {
		return false
	}
	if findSorted(dnsNames, sni) != -1 {
		return true
	}

	for _, dn := range dnsNames {
		if matchDomainWildcard(sni, dn) {
			return true
		}
	}
	return false
}
