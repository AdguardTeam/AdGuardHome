package util

import (
	"net"
	"strings"
)

// convert character to hex number
func charToHex(n byte) int8 {
	if n >= '0' && n <= '9' {
		return int8(n) - '0'
	} else if (n|0x20) >= 'a' && (n|0x20) <= 'f' {
		return (int8(n) | 0x20) - 'a' + 10
	}
	return -1
}

// parse IPv6 reverse address
func ipParseArpa6(s string) net.IP {
	if len(s) != 63 {
		return nil
	}
	ip6 := make(net.IP, 16)

	for i := 0; i != 64; i += 4 {

		// parse "0.1."
		n := charToHex(s[i])
		n2 := charToHex(s[i+2])
		if s[i+1] != '.' || (i != 60 && s[i+3] != '.') ||
			n < 0 || n2 < 0 {
			return nil
		}

		ip6[16-i/4-1] = byte(n2<<4) | byte(n&0x0f)
	}
	return ip6
}

// ipReverse - reverse IP address: 1.0.0.127 -> 127.0.0.1
func ipReverse(ip net.IP) net.IP {
	n := len(ip)
	r := make(net.IP, n)
	for i := 0; i != n; i++ {
		r[i] = ip[n-i-1]
	}
	return r
}

// DNSUnreverseAddr - convert reversed ARPA address to a normal IP address
func DNSUnreverseAddr(s string) net.IP {
	const arpaV4 = ".in-addr.arpa"
	const arpaV6 = ".ip6.arpa"

	if strings.HasSuffix(s, arpaV4) {
		ip := strings.TrimSuffix(s, arpaV4)
		ip4 := net.ParseIP(ip).To4()
		if ip4 == nil {
			return nil
		}

		return ipReverse(ip4)

	} else if strings.HasSuffix(s, arpaV6) {
		ip := strings.TrimSuffix(s, arpaV6)
		return ipParseArpa6(ip)
	}

	return nil // unknown suffix
}
