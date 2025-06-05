package dhcpsvc

import (
	"context"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/google/gopacket/layers"
)

// responseWriter4 writes DHCPv4 response to the client.
type responseWriter4 interface {
	// write writes the DHCPv4 response to the client.
	write(ctx context.Context, pkt *layers.DHCPv4) (err error)
}

// responseWriter6 writes DHCPv6 response to the client.
type responseWriter6 interface {
	// write writes the DHCPv6 response to the client.
	write(ctx context.Context, pkt *layers.DHCPv6) (err error)
}

// serve handles the incoming packets and dispatches them to the appropriate
// handler based on the Ethernet type.  It's used to run in a separate goroutine
// as it blocks until packets channel is closed.
func (srv *DHCPServer) serve(ctx context.Context) {
	defer slogutil.RecoverAndLog(ctx, srv.logger)

	for pkt := range srv.packetSource.Packets() {
		etherLayer, ok := pkt.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		if !ok {
			actual := pkt.Layers()
			srv.logger.DebugContext(ctx, "skipping non-ethernet packet", "layers", actual)

			continue
		}

		var err error

		switch typ := etherLayer.EthernetType; typ {
		case layers.EthernetTypeIPv4:
			// TODO(e.burkov):  Set the response writer.
			var rw responseWriter4
			err = srv.serveV4(ctx, rw, pkt)
		case layers.EthernetTypeIPv6:
			// TODO(e.burkov):  Set the response writer.
			var rw responseWriter6
			err = srv.serveV6(ctx, rw, pkt)
		default:
			// TODO(e.burkov):  It seems, there is another standard for Ethernet
			// header, which uses the Length field instead of the EthernetType,
			// so handle it properly.
			srv.logger.DebugContext(ctx, "skipping ethernet packet", "type", typ)

			continue
		}

		if err != nil {
			srv.logger.ErrorContext(ctx, "serving", slogutil.KeyError, err)
		}
	}
}
