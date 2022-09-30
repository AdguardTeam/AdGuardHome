//go:build windows

package dhcpd

import (
	"net"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"golang.org/x/net/ipv4"
)

// Create a socket for receiving broadcast packets
func newBroadcastPacketConn(_ net.IP, _ int, _ string) (*ipv4.PacketConn, error) {
	return nil, aghos.Unsupported("newBroadcastPacketConn")
}
