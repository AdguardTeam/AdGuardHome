package dhcpsvc

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// serveV4 handles the ethernet packet of IPv4 type.  rw and pkt must not be
// nil.
func (srv *DHCPServer) serveV4(
	ctx context.Context,
	rw responseWriter4,
	pkt gopacket.Packet,
) (err error) {
	defer func() { err = errors.Annotate(err, "serving dhcpv4: %w") }()

	req, ok := pkt.Layer(layers.LayerTypeDHCPv4).(*layers.DHCPv4)
	if !ok {
		// TODO(e.burkov):  Consider adding some debug information about the
		// actual received packet.
		srv.logger.DebugContext(ctx, "skipping non-dhcpv4 packet")

		return nil
	}

	// TODO(e.burkov):  Handle duplicate Xid.

	if req.Operation != layers.DHCPOpRequest {
		srv.logger.DebugContext(ctx, "skipping non-request dhcpv4 packet", "op", req.Operation)

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

// handleDHCPv4 handles the DHCPv4 message of the given type.  The DHCPDISCOVER
// messages are handled by all interfaces concurrently, as those offer addresses
// for the independent networks.  The DHCPREQUEST, DHCPRELEASE, and DHCPDECLINE
// messages are handled by the server itself as it should pick the appropriate
// interface according to the client's choice.  req must not be nil, typ should
// be one of:
//   - [layers.DHCPMsgTypeDiscover]
//   - [layers.DHCPMsgTypeRequest]
//   - [layers.DHCPMsgTypeRelease]
//   - [layers.DHCPMsgTypeDecline]
func (srv *DHCPServer) handleDHCPv4(
	ctx context.Context,
	rw responseWriter4,
	typ layers.DHCPMsgType,
	req *layers.DHCPv4,
) (err error) {
	switch typ {
	case layers.DHCPMsgTypeDiscover:
		srv.handleDiscover(ctx, rw, req)
	case layers.DHCPMsgTypeRequest:
		srv.handleRequest(ctx, rw, req)
	case layers.DHCPMsgTypeRelease:
		srv.handleRelease(ctx, req)
	case layers.DHCPMsgTypeDecline:
		srv.handleDecline(ctx, req)
	default:
		// TODO(e.burkov):  Handle DHCPINFORM.
		return fmt.Errorf("dhcpv4: request type: %w: %v", errors.ErrBadEnumValue, typ)
	}

	return nil
}

// handleDiscover handles the DHCPv4 message of DHCPDISCOVER type.  rw must not
// be nil, req must be a DHCPDISCOVER message.
func (srv *DHCPServer) handleDiscover(ctx context.Context, rw responseWriter4, req *layers.DHCPv4) {
	// TODO(e.burkov):  Check existing leases, either allocated or offered.

	for _, iface := range srv.interfaces4 {
		go iface.handleDiscover(ctx, rw, req)
	}
}

// handleRequest handles the DHCPv4 message of DHCPREQUEST type.  rw must not be
// nil, req must be a DHCPREQUEST message.
//
// See https://datatracker.ietf.org/doc/html/rfc2131#section-4.3.2.
//
// TODO(e.burkov):  Remove allocated leases after client have chosen one.
func (srv *DHCPServer) handleRequest(ctx context.Context, rw responseWriter4, req *layers.DHCPv4) {
	srvID, hasSrvID := serverID4(req)
	reqIP, hasReqIP := requestedIPv4(req)

	switch {
	case hasSrvID && !srvID.IsUnspecified():
		// If the DHCPREQUEST message contains a server identifier option, the
		// message is in response to a DHCPOFFER message.  Otherwise, the
		// message is a request to verify or extend an existing lease.
		iface, ok := srv.interfaces4.findInterface(srvID)
		if !ok {
			srv.logger.DebugContext(ctx, "skipping selecting request", "serverid", srvID)

			return
		}

		iface.handleSelecting(ctx, rw, req, reqIP)
	case hasReqIP && !reqIP.IsUnspecified():
		// Requested IP address option MUST be filled in with client's notion of
		// its previously assigned address.
		iface, ok := srv.interfaces4.findInterface(reqIP)
		if !ok {
			// If the DHCP server detects that the client is on the wrong net
			// then the server SHOULD send a DHCPNAK message to the client.
			srv.logger.DebugContext(ctx, "declining init-reboot request", "requestedip", reqIP)
			iface.respondNAK(ctx, rw, req)

			return
		}

		iface.handleInitReboot(ctx, rw, req, reqIP)
	default:
		// Server identifier MUST NOT be filled in, requested IP address option
		// MUST NOT be filled in.
		ip, _ := netip.AddrFromSlice(req.ClientIP.To4())
		iface, ok := srv.interfaces4.findInterface(ip)
		if !ok {
			srv.logger.DebugContext(ctx, "skipping renew request", "clientip", ip)

			return
		}

		iface.handleRenew(ctx, rw, req, ip)
	}
}

// handleDecline handles the DHCPv4 message of DHCPDECLINE type.  req must be a
// DHCPDECLINE message.
func (srv *DHCPServer) handleDecline(ctx context.Context, req *layers.DHCPv4) {
	reqIP, hasReqIP := requestedIPv4(req)
	if !hasReqIP {
		srv.logger.DebugContext(ctx, "skipping decline message without requested ip")

		return
	}

	iface, ok := srv.interfaces4.findInterface(reqIP)
	if !ok {
		srv.logger.DebugContext(ctx, "skipping decline message", "requestedip", reqIP)

		return
	}

	iface.handleDecline(ctx, reqIP, req)
}

// handleRelease handles the DHCPv4 message of DHCPRELEASE type.  req must be a
// DHCPRELEASE message.
func (srv *DHCPServer) handleRelease(ctx context.Context, req *layers.DHCPv4) {
	ip, _ := netip.AddrFromSlice(req.ClientIP.To4())
	iface, ok := srv.interfaces4.findInterface(ip)
	if !ok {
		srv.logger.DebugContext(ctx, "skipping release message", "clientip", ip)

		return
	}

	iface.handleRelease(ctx, ip, req)
}
