package logs

import "net"

// AnonymizeIP masks ip to anonymize the client if the ip is a valid one.
func AnonymizeIP(ip net.IP) {
	// zeroes is a slice of zero bytes from which the IP address tail is copied.
	// Using constant string as source of copying is more efficient than byte
	// slice, see https://github.com/golang/go/issues/49997.
	const zeroes = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

	if ip4 := ip.To4(); ip4 != nil {
		copy(ip4[net.IPv4len-2:net.IPv4len], zeroes)
	} else if len(ip) == net.IPv6len {
		copy(ip[net.IPv6len-10:net.IPv6len], zeroes)
	}
}
