package dhcpsvc

import (
	"context"
	"log/slog"
	"net/netip"
	"slices"

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
		fd := newFrameData4(ctx, srv.logger, pkt, nd)
		if fd == nil {
			continue
		}

		err := srv.serveV4(ctx, iface, pkt, fd)
		if err != nil {
			srv.logger.ErrorContext(ctx, "serving", slogutil.KeyError, err)
		}
	}
}

// serveEther6 handles the incoming ethernet packets and dispatches them to the
// appropriate handler.  It's used to run in a separate goroutine as it blocks
// until packets channel is closed.  iface and nd must not be nil.  nd must have
// at least a single address returned by its Addresses method.
//
//lint:ignore U1000 TODO(e.burkov): Use.
func (srv *DHCPServer) serveEther6(ctx context.Context, iface *dhcpInterfaceV6, nd NetworkDevice) {
	defer slogutil.RecoverAndLog(ctx, srv.logger)

	src := gopacket.NewPacketSource(nd, nd.LinkType())
	srvDUID := newServerDUID(nd.HardwareAddr())

	for pkt := range src.Packets() {
		fd := newFrameData6(ctx, srv.logger, pkt, nd, srvDUID)
		if fd == nil {
			continue
		}

		err := srv.serveV6(ctx, iface, pkt, fd)
		if err != nil {
			srv.logger.ErrorContext(ctx, "serving", slogutil.KeyError, err)
		}
	}
}

// newFrameData4 creates a new [frameData4] with layers extracted from pkt.  It
// returns nil if the packet is not an Ethernet or IPv4 packet, or if the
// network device has no addresses.  logger, pkt, and dev must not be nil.
func newFrameData4(
	ctx context.Context,
	logger *slog.Logger,
	pkt gopacket.Packet,
	dev NetworkDevice,
) (fd *frameData4) {
	addrs := dev.Addresses()
	if len(addrs) == 0 {
		logger.ErrorContext(ctx, "no addresses for network device")

		return nil
	}

	etherLayer, ok := pkt.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if !ok {
		actual := pkt.Layers()
		logger.DebugContext(ctx, "skipping non-ethernet packet", "layers", actual)

		return nil
	}

	ipLayer, ok := pkt.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	if !ok {
		actual := pkt.Layers()
		logger.DebugContext(ctx, "skipping non-ipv4 packet", "layers", actual)

		return nil
	}

	addr, ok := netip.AddrFromSlice(ipLayer.DstIP)
	if !ok || !slices.Contains(addrs, addr) {
		addr = addrs[0]
	}

	return &frameData4{
		ether:     etherLayer,
		ip:        ipLayer,
		device:    dev,
		localAddr: addr,
	}
}

// newFrameData6 creates a new [frameData6] with layers extracted from pkt.  It
// returns nil if the packet is not an Ethernet or IPv6 packet, or if the
// network device has no addresses.  logger, pkt, and dev must not be nil.
func newFrameData6(
	ctx context.Context,
	logger *slog.Logger,
	pkt gopacket.Packet,
	dev NetworkDevice,
	duid *layers.DHCPv6DUID,
) (fd *frameData6) {
	addrs := dev.Addresses()
	if len(addrs) == 0 {
		logger.ErrorContext(ctx, "no addresses for network device")

		return nil
	}

	etherLayer, ok := pkt.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if !ok {
		actual := pkt.Layers()
		logger.DebugContext(ctx, "skipping non-ethernet packet", "layers", actual)

		return nil
	}

	ipLayer, ok := pkt.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
	if !ok {
		actual := pkt.Layers()
		logger.DebugContext(ctx, "skipping non-ipv6 packet", "layers", actual)

		return nil
	}

	addr, ok := netip.AddrFromSlice(ipLayer.DstIP)
	if !ok || !slices.Contains(addrs, addr) {
		addr = addrs[0]
	}

	return &frameData6{
		ether:     etherLayer,
		ip:        ipLayer,
		duid:      duid,
		device:    dev,
		localAddr: addr,
	}
}
