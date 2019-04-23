package dhcpd

import (
	"net"
	"os"
	"syscall"

	"golang.org/x/net/ipv4"
)

// Create a socket for receiving broadcast packets
func newBroadcastPacketConn(bindAddr net.IP, port int, ifname string) (*ipv4.PacketConn, error) {
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		return nil, err
	}

	if err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1); err != nil {
		return nil, err
	}
	if err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return nil, err
	}
	if err := syscall.SetsockoptString(s, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, ifname); err != nil {
		return nil, err
	}

	addr := syscall.SockaddrInet4{Port: port}
	copy(addr.Addr[:], bindAddr.To4())
	err = syscall.Bind(s, &addr)
	if err != nil {
		syscall.Close(s)
		return nil, err
	}

	f := os.NewFile(uintptr(s), "")
	c, err := net.FilePacketConn(f)
	f.Close()
	if err != nil {
		return nil, err
	}

	p := ipv4.NewPacketConn(c)
	return p, nil
}
