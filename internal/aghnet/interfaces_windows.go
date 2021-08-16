//go:build windows
// +build windows

package aghnet

import (
	"net"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
)

// listenPacketReusable announces on the local network address additionally
// configuring the socket to have a reusable binding.
func listenPacketReusable(_, _, _ string) (c net.PacketConn, err error) {
	// TODO(e.burkov):  Check if we are able to control sockets on Windows
	// in the same way as on Unix.
	return nil, aghos.Unsupported("listening packet reusable")
}
