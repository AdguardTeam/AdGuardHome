//go:build freebsd || openbsd

package dhcpd

import (
	"net"
)

// broadcast sends resp to the broadcast address specific for network interface.
func (c *dhcpConn) broadcast(respData []byte, peer *net.UDPAddr) (n int, err error) {
	// Despite the fact that server4.NewIPv4UDPConn explicitly sets socket
	// options to allow broadcasting, it also binds the connection to a specific
	// interface.  On FreeBSD and OpenBSD net.UDPConn.WriteTo causes errors
	// while writing to the addresses that belong to another interface.  So, use
	// the broadcast address specific for the interface bound.
	peer.IP = c.bcastIP

	return c.udpConn.WriteTo(respData, peer)
}
