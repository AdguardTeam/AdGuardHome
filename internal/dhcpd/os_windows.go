package dhcpd

import (
	"net"

	"github.com/AdguardTeam/golibs/errors"
	"golang.org/x/net/ipv4"
)

// Create a socket for receiving broadcast packets
func newBroadcastPacketConn(bindAddr net.IP, port int, ifname string) (*ipv4.PacketConn, error) {
	return nil, errors.Error("newBroadcastPacketConn(): not supported on Windows")
}
