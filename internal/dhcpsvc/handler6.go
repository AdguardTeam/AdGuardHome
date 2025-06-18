package dhcpsvc

import (
	"context"
	"fmt"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// serveV6 handles the ethernet packet of IPv6 type.  rw and pkt must not be
// nil.
func (srv *DHCPServer) serveV6(
	ctx context.Context,
	rw responseWriter6,
	pkt gopacket.Packet,
) (err error) {
	defer func() { err = errors.Annotate(err, "serving dhcpv6: %w") }()

	msg, ok := pkt.Layer(layers.LayerTypeDHCPv6).(*layers.DHCPv6)
	if !ok {
		// TODO(e.burkov):  Consider adding some debug information about the
		// actual received packet.
		srv.logger.DebugContext(ctx, "skipping non-dhcpv6 packet")

		return nil
	}

	// TODO(e.burkov):  Handle duplicate TransactionID.

	return srv.handleDHCPv6(ctx, rw, msg.MsgType, msg)
}

// handleDHCPv6 handles the DHCPv6 message of the given type.
func (srv *DHCPServer) handleDHCPv6(
	_ context.Context,
	_ responseWriter6,
	typ layers.DHCPv6MsgType,
	_ *layers.DHCPv6,
) (err error) {
	switch typ {
	case
		layers.DHCPv6MsgTypeSolicit,
		layers.DHCPv6MsgTypeRequest,
		layers.DHCPv6MsgTypeConfirm,
		layers.DHCPv6MsgTypeRenew,
		layers.DHCPv6MsgTypeRebind,
		layers.DHCPv6MsgTypeInformationRequest,
		layers.DHCPv6MsgTypeRelease,
		layers.DHCPv6MsgTypeDecline:
		// TODO(e.burkov):  Handle messages.
	default:
		return fmt.Errorf("dhcpv6: request type: %w: %v", errors.ErrBadEnumValue, typ)
	}

	return nil
}
