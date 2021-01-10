package dhcpd

import (
	"errors"
	"net"

	"golang.org/x/net/ipv4"
)

// Create a socket for receiving broadcast packets
func newBroadcastPacketConn(bindAddr net.IP, port int, ifname string) (*ipv4.PacketConn, error) {
	return nil, errors.New("newBroadcastPacketConn(): not supported on Windows")
}
