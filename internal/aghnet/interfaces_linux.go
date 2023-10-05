//go:build linux

package aghnet

import (
	"net"

	"github.com/AdguardTeam/golibs/netutil"
	"github.com/insomniacslk/dhcp/dhcpv4/nclient4"
)

// listenPacketReusable announces on the local network address additionally
// configuring the socket to have a reusable binding.
func listenPacketReusable(ifaceName, network, address string) (c net.PacketConn, err error) {
	var port uint16
	_, port, err = netutil.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	// TODO(e.burkov):  Inspect nclient4.NewRawUDPConn and implement here.
	return nclient4.NewRawUDPConn(ifaceName, int(port))
}
