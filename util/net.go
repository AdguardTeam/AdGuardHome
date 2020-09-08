package util

import (
	"fmt"
	"net"
)

// nolint (gocyclo)
// Return TRUE if IP is within public Internet IP range
func IsPublicIP(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 != nil {
		switch ip4[0] {
		case 0:
			return false //software
		case 10:
			return false //private network
		case 127:
			return false //loopback
		case 169:
			if ip4[1] == 254 {
				return false //link-local
			}
		case 172:
			if ip4[1] >= 16 && ip4[1] <= 31 {
				return false //private network
			}
		case 192:
			if (ip4[1] == 0 && ip4[2] == 0) || //private network
				(ip4[1] == 0 && ip4[2] == 2) || //documentation
				(ip4[1] == 88 && ip4[2] == 99) || //reserved
				(ip4[1] == 168) { //private network
				return false
			}
		case 198:
			if (ip4[1] == 18 || ip4[2] == 19) || //private network
				(ip4[1] == 51 || ip4[2] == 100) { //documentation
				return false
			}
		case 203:
			if ip4[1] == 0 && ip4[2] == 113 { //documentation
				return false
			}
		case 224:
			if ip4[1] == 0 && ip4[2] == 0 { //multicast
				return false
			}
		case 255:
			if ip4[1] == 255 && ip4[2] == 255 && ip4[3] == 255 { //subnet
				return false
			}
		}
	} else {
		if ip.IsLoopback() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
			return false
		}
	}

	return true
}

// CanBindPort - checks if we can bind to this port or not
func CanBindPort(port int) (bool, error) {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return false, err
	}
	_ = l.Close()
	return true, nil
}
