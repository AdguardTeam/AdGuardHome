package dhcpsvc

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// serveV4 handles the ethernet packet of IPv4 type.
func (srv *DHCPServer) serveV4(
	ctx context.Context,
	rw responseWriter4,
	pkt gopacket.Packet,
) (err error) {
	defer func() { err = errors.Annotate(err, "serving dhcpv4: %w") }()

	req, ok := pkt.Layer(layers.LayerTypeDHCPv4).(*layers.DHCPv4)
	if !ok {
		srv.logger.DebugContext(ctx, "skipping non-dhcpv4 packet")

		return nil
	}

	// TODO(e.burkov):  Handle duplicate Xid.

	if req.Operation != layers.DHCPOpRequest {
		srv.logger.DebugContext(ctx, "skipping non-request dhcpv4 packet")

		return nil
	}

	typ, ok := msg4Type(req)
	if !ok {
		// The "DHCP message type" option - must be included in every DHCP
		// message.
		//
		// See https://datatracker.ietf.org/doc/html/rfc2131#section-3.
		return fmt.Errorf("dhcpv4: message type: %w", errors.ErrNoValue)
	}

	return srv.handleDHCPv4(ctx, rw, typ, req)
}

// handleDHCPv4 handles the DHCPv4 message of the given type.
func (srv *DHCPServer) handleDHCPv4(
	ctx context.Context,
	rw responseWriter4,
	typ layers.DHCPMsgType,
	req *layers.DHCPv4,
) (err error) {
	// Each interface should handle the DISCOVER and REQUEST messages offer and
	// allocate the available leases.  The RELEASE and DECLINE messages should
	// be handled by the server itself as it should remove the lease.
	switch typ {
	case layers.DHCPMsgTypeDiscover:
		srv.handleDiscover(ctx, rw, req)
	case layers.DHCPMsgTypeRequest:
		srv.handleRequest(ctx, rw, req)
	case layers.DHCPMsgTypeRelease:
		// TODO(e.burkov):  !! Remove the lease, either allocated or offered.
	case layers.DHCPMsgTypeDecline:
		// TODO(e.burkov):  !! Remove the allocated lease.  RFC tells it only
		// possible if the client found the address already in use.
	default:
		// TODO(e.burkov):  Handle DHCPINFORM.
		return fmt.Errorf("dhcpv4: request type: %w: %v", errors.ErrBadEnumValue, typ)
	}

	return nil
}

// handleDiscover handles the DHCPv4 message of discover type.
func (srv *DHCPServer) handleDiscover(ctx context.Context, rw responseWriter4, req *layers.DHCPv4) {
	// TODO(e.burkov):  Check existing leases, either allocated or offered.

	for _, iface := range srv.interfaces4 {
		go iface.handleDiscover(ctx, rw, req)
	}
}

// handleRequest handles the DHCPv4 message of request type.
func (srv *DHCPServer) handleRequest(ctx context.Context, rw responseWriter4, req *layers.DHCPv4) {
	srvID, hasSrvID := serverID4(req)
	reqIP, hasReqIP := requestedIPv4(req)

	switch {
	case hasSrvID && !srvID.IsUnspecified():
		// If the DHCPREQUEST message contains a server identifier option, the
		// message is in response to a DHCPOFFER message.  Otherwise, the
		// message is a request to verify or extend an existing lease.
		iface, hasIface := srv.interfaces4.findInterface(srvID)
		if !hasIface {
			srv.logger.DebugContext(ctx, "skipping selecting request", "serverid", srvID)

			return
		}

		iface.handleSelecting(ctx, rw, req, reqIP)
	case hasReqIP && !reqIP.IsUnspecified():
		// Requested IP address option MUST be filled in with client's notion of
		// its previously assigned address.
		iface, hasIface := srv.interfaces4.findInterface(reqIP)
		if !hasIface {
			srv.logger.DebugContext(ctx, "skipping init-reboot request", "requestedip", reqIP)

			return
		}

		iface.handleInitReboot(ctx, rw, req, reqIP)
	default:
		// Server identifier MUST NOT be filled in, requested IP address option
		// MUST NOT be filled in.
		ip, _ := netip.AddrFromSlice(req.ClientIP.To4())
		iface, hasIface := srv.interfaces4.findInterface(ip)
		if !hasIface {
			srv.logger.DebugContext(ctx, "skipping init-reboot request", "clientip", ip)

			return
		}

		iface.handleRenew(ctx, rw, req)
	}
}
