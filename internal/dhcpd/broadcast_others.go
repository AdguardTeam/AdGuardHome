//go:build darwin || linux

package dhcpd

import (
	"net"
)

// broadcast sends resp to the broadcast address specific for network interface.
func (c *dhcpConn) broadcast(respData []byte, peer *net.UDPAddr) (n int, err error) {
	// This write to 0xffffffff reverts some behavior changes made in
	// https://github.com/AdguardTeam/AdGuardHome/issues/3289.  The DHCP
	// server should broadcast the message to 0xffffffff but it's
	// inconsistent with the actual mental model of DHCP implementation
	// which requires the network interface selection to bind to.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/3480 and
	// https://github.com/AdguardTeam/AdGuardHome/issues/3366.
	//
	// See also https://github.com/AdguardTeam/AdGuardHome/issues/3539.
	if n, err = c.udpConn.WriteTo(respData, peer); err != nil {
		return n, err
	}

	// Broadcast the message one more time using the interface-specific
	// broadcast address.
	peer.IP = c.bcastIP

	return c.udpConn.WriteTo(respData, peer)
}
