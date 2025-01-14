package dhcpsvc

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func (srv *DHCPServer) serve(ctx context.Context) {
	defer slogutil.RecoverAndLog(ctx, srv.logger)

	for pkt := range srv.packetSource.Packets() {
		etherLayer, ok := pkt.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		if !ok {
			srv.logger.DebugContext(ctx, "skipping non-ethernet packet")

			continue
		}

		var err error
		switch typ := etherLayer.EthernetType; typ {
		case layers.EthernetTypeIPv4:
			err = srv.serveV4(ctx, pkt)
		case layers.EthernetTypeIPv6:
			// TODO(e.burkov):  Handle DHCPv6 as well.
		default:
			srv.logger.DebugContext(ctx, "skipping ethernet packet", "type", typ)

			continue
		}

		if err != nil {
			srv.logger.ErrorContext(ctx, "serving", slogutil.KeyError, err)
		}
	}
}

// serveV4 handles the ethernet packet of IPv4 type.
func (srv *DHCPServer) serveV4(ctx context.Context, pkt gopacket.Packet) (err error) {
	defer func() { err = errors.Annotate(err, "serving dhcpv4: %w") }()

	msg, ok := pkt.Layer(layers.LayerTypeDHCPv4).(*layers.DHCPv4)
	if !ok {
		srv.logger.DebugContext(ctx, "skipping non-dhcpv4 packet")

		return nil
	}

	// TODO(e.burkov):  Handle duplicate Xid.

	typ, ok := msgType(msg)
	if !ok {
		return errors.Error("no message type in the dhcpv4 message")
	}

	return srv.handleDHCPv4(ctx, typ, msg)
}

// handleDHCPv4 handles the DHCPv4 message of the given type.
func (srv *DHCPServer) handleDHCPv4(
	ctx context.Context,
	typ layers.DHCPMsgType,
	msg *layers.DHCPv4,
) (err error) {
	// Each interface should handle the DISCOVER and REQUEST messages offer and
	// allocate the available leases.  The RELEASE and DECLINE messages should
	// be handled by the server itself as it should remove the lease.
	switch typ {
	case layers.DHCPMsgTypeDiscover:
		for _, iface := range srv.interfaces4 {
			go iface.handleDiscover(ctx, msg)
		}
	case layers.DHCPMsgTypeRequest:
		for _, iface := range srv.interfaces4 {
			go iface.handleRequest(ctx, msg)
		}
	case layers.DHCPMsgTypeRelease:
		addr, ok := netip.AddrFromSlice(msg.ClientIP)
		if !ok {
			return fmt.Errorf("invalid client ip in the release message")
		}

		return srv.removeLeaseByAddr(ctx, addr)
	case layers.DHCPMsgTypeDecline:
		addr, ok := requestedIP(msg)
		if !ok {
			return fmt.Errorf("no requested ip in the decline message")
		}

		return srv.removeLeaseByAddr(ctx, addr)
	default:
		// TODO(e.burkov):  Handle DHCPINFORM.
		return fmt.Errorf("dhcpv4 message type: %w: %v", errors.ErrBadEnumValue, typ)
	}

	return nil
}
