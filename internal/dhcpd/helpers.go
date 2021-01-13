package dhcpd

import (
	"encoding/binary"
	"fmt"
	"net"
)

func isTimeout(err error) bool {
	operr, ok := err.(*net.OpError)
	if !ok {
		return false
	}
	return operr.Timeout()
}

func tryTo4(ip net.IP) (ip4 net.IP, err error) {
	if ip == nil {
		return nil, fmt.Errorf("%v is not an IP address", ip)
	}

	ip4 = ip.To4()
	if ip4 == nil {
		return nil, fmt.Errorf("%v is not an IPv4 address", ip)
	}

	return ip4, nil
}

// Return TRUE if subnet mask is correct (e.g. 255.255.255.0)
func isValidSubnetMask(mask net.IP) bool {
	var n uint32
	n = binary.BigEndian.Uint32(mask)
	for i := 0; i != 32; i++ {
		if n == 0 {
			break
		}
		if (n & 0x80000000) == 0 {
			return false
		}
		n <<= 1
	}
	return true
}
