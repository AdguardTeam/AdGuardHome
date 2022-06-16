package aghnet

import (
	"net"
	"strconv"
)

// The maximum lengths of generated hostnames for different IP versions.
const (
	ipv4HostnameMaxLen = len("192-168-100-100")
	ipv6HostnameMaxLen = len("ff80-f076-0000-0000-0000-0000-0000-0010")
)

// generateIPv4Hostname generates the hostname by IP address version 4.
func generateIPv4Hostname(ipv4 net.IP) (hostname string) {
	hnData := make([]byte, 0, ipv4HostnameMaxLen)
	for i, part := range ipv4 {
		if i > 0 {
			hnData = append(hnData, '-')
		}
		hnData = strconv.AppendUint(hnData, uint64(part), 10)
	}

	return string(hnData)
}

// generateIPv6Hostname generates the hostname by IP address version 6.
func generateIPv6Hostname(ipv6 net.IP) (hostname string) {
	hnData := make([]byte, 0, ipv6HostnameMaxLen)
	for i, partsNum := 0, net.IPv6len/2; i < partsNum; i++ {
		if i > 0 {
			hnData = append(hnData, '-')
		}
		for _, val := range ipv6[i*2 : i*2+2] {
			if val < 10 {
				hnData = append(hnData, '0')
			}
			hnData = strconv.AppendUint(hnData, uint64(val), 16)
		}
	}

	return string(hnData)
}

// GenerateHostname generates the hostname from ip.  In case of using IPv4 the
// result should be like:
//
//   192-168-10-1
//
// In case of using IPv6, the result is like:
//
//   ff80-f076-0000-0000-0000-0000-0000-0010
//
// ip must be either an IPv4 or an IPv6.
func GenerateHostname(ip net.IP) (hostname string) {
	if ipv4 := ip.To4(); ipv4 != nil {
		return generateIPv4Hostname(ipv4)
	}

	return generateIPv6Hostname(ip)
}
