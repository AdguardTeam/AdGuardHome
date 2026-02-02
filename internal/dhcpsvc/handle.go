package dhcpsvc

import (
	"context"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// serveEther4 handles the incoming ethernet packets and dispatches them to the
// appropriate handler.  It's used to run in a separate goroutine as it blocks
// until packets channel is closed.  iface and nd must not be nil.  nd must have
// at least a single address returned by its Addresses method.
func (srv *DHCPServer) serveEther4(ctx context.Context, iface *dhcpInterfaceV4, nd NetworkDevice) {
	defer slogutil.RecoverAndLog(ctx, srv.logger)

	src := gopacket.NewPacketSource(nd, nd.LinkType())

	for pkt := range src.Packets() {
		etherLayer, ok := pkt.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		if !ok {
			actual := pkt.Layers()
			srv.logger.DebugContext(ctx, "skipping non-ethernet packet", "layers", actual)

			continue
		}

		ipLayer, ok := pkt.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		if !ok {
			actual := pkt.Layers()
			srv.logger.DebugContext(ctx, "skipping non-ipv4 packet", "layers", actual)

			continue
		}

		fd := &frameData{
			ether:  etherLayer,
			ip:     ipLayer,
			device: nd,
		}

		err := srv.serveV4(ctx, iface, pkt, fd)
		if err != nil {
			srv.logger.ErrorContext(ctx, "serving", slogutil.KeyError, err)
		}
	}
}

// TODO(e.burkov):  Add DHCPServer.serveEther6.
