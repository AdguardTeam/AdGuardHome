//go:build freebsd || openbsd
// +build freebsd openbsd

package dhcpd

import (
	"net"

	"github.com/AdguardTeam/golibs/log"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

// broadcast sends resp to the broadcast address specific for network interface.
func (s *v4Server) broadcast(peer net.Addr, conn net.PacketConn, resp *dhcpv4.DHCPv4) {
	// peer is expected to be of type *net.UDPConn as the server4.NewServer
	// initializes it.
	udpPeer, ok := peer.(*net.UDPAddr)
	if !ok {
		log.Error("dhcpv4: peer is of unexpected type %T", peer)

		return
	}

	// Despite the fact that server4.NewIPv4UDPConn explicitly sets socket
	// options to allow broadcasting, it also binds the connection to a
	// specific interface.  On FreeBSD and OpenBSD conn.WriteTo causes
	// errors while writing to the addresses that belong to another
	// interface.  So, use the broadcast address specific for the binded
	// interface.
	udpPeer.IP = s.conf.broadcastIP

	log.Debug("dhcpv4: sending to %s: %s", peer, resp.Summary())

	if _, err := conn.WriteTo(resp.ToBytes(), peer); err != nil {
		log.Error("dhcpv4: conn.Write to %s failed: %s", peer, err)
	}
}
