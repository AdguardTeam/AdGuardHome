//go:build aix || darwin || dragonfly || linux || netbsd || solaris
// +build aix darwin dragonfly linux netbsd solaris

package dhcpd

import (
	"net"

	"github.com/AdguardTeam/golibs/log"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

// broadcast sends resp to the broadcast address specific for network interface.
func (s *v4Server) broadcast(peer net.Addr, conn net.PacketConn, resp *dhcpv4.DHCPv4) {
	respData := resp.ToBytes()

	log.Debug("dhcpv4: sending to %s: %s", peer, resp.Summary())

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
	if _, err := conn.WriteTo(respData, peer); err != nil {
		log.Error("dhcpv4: conn.Write to %s failed: %s", peer, err)
	}

	// peer is expected to be of type *net.UDPConn as the server4.NewServer
	// initializes it.
	udpPeer, ok := peer.(*net.UDPAddr)
	if !ok {
		log.Error("dhcpv4: peer is of unexpected type %T", peer)

		return
	}

	// Broadcast the message one more time using the interface-specific
	// broadcast address.
	udpPeer.IP = s.conf.broadcastIP

	log.Debug("dhcpv4: sending to %s: %s", peer, resp.Summary())

	if _, err := conn.WriteTo(respData, peer); err != nil {
		log.Error("dhcpv4: conn.Write to %s failed: %s", peer, err)
	}
}
