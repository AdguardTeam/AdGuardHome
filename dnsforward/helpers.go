package dnsforward

import "net"

// GetIPString is a helper function that extracts IP address from net.Addr
func GetIPString(addr net.Addr) string {
	switch addr := addr.(type) {
	case *net.UDPAddr:
		return addr.IP.String()
	case *net.TCPAddr:
		return addr.IP.String()
	}
	return ""
}
