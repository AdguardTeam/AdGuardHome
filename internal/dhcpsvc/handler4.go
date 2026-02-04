package dhcpsvc

import (
	"context"
	"fmt"
	"net/netip"
	"slices"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// serveV4 handles the ethernet packet of IPv4 type. iface and pkt must not be
// nil.  iface and fd must not be nil.  pkt must be an IPv4 packet.
func (srv *DHCPServer) serveV4(
	ctx context.Context,
	iface *dhcpInterfaceV4,
	pkt gopacket.Packet,
	fd *frameData,
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
		return fmt.Errorf("message type: %w", errors.ErrNoValue)
	}

	return iface.handleDHCPv4(ctx, typ, req, fd)
}

// handleDHCPv4 handles the DHCPv4 message of the given type.  The DHCPDISCOVER
// messages are handled by all interfaces concurrently, as those offer addresses
// for the independent networks.  The DHCPREQUEST, DHCPRELEASE, and DHCPDECLINE
// messages are handled by the appropriate interface according to the client's
// choice.  req and fd must not be nil, typ should be one of:
//   - [layers.DHCPMsgTypeDiscover]
//   - [layers.DHCPMsgTypeRequest]
//   - [layers.DHCPMsgTypeRelease]
//   - [layers.DHCPMsgTypeDecline]
func (iface *dhcpInterfaceV4) handleDHCPv4(
	ctx context.Context,
	typ layers.DHCPMsgType,
	req *layers.DHCPv4,
	fd *frameData,
) (err error) {
	switch typ {
	case layers.DHCPMsgTypeDiscover:
		iface.handleDiscover(ctx, req, fd)
	case layers.DHCPMsgTypeRequest:
		iface.handleRequest(ctx, req, fd)
	case layers.DHCPMsgTypeRelease:
		iface.handleRelease(ctx, req)
	case layers.DHCPMsgTypeDecline:
		iface.handleDecline(ctx, req)
	default:
		// TODO(e.burkov):  Handle DHCPINFORM.
		return fmt.Errorf("dhcpv4: request type: %w: %v", errors.ErrBadEnumValue, typ)
	}

	return nil
}

// handleRequest handles the DHCPv4 message of DHCPREQUEST type.  req must be a
// DHCPREQUEST message.  req and fd must not be nil.
//
// See https://datatracker.ietf.org/doc/html/rfc2131#section-4.3.2.
//
// TODO(e.burkov):  Remove allocated leases after client have chosen one.
func (iface *dhcpInterfaceV4) handleRequest(
	ctx context.Context,
	req *layers.DHCPv4,
	fd *frameData,
) {
	srvID, hasSrvID := serverID4(req)
	reqIP, hasReqIP := requestedIPv4(req)

	l := iface.common.logger

	switch {
	case hasSrvID && !srvID.IsUnspecified():
		// If the DHCPREQUEST message contains a server identifier option, the
		// message is in response to a DHCPOFFER message.  Otherwise, the
		// message is a request to verify or extend an existing lease.
		if !slices.Contains(fd.device.Addresses(), srvID) {
			l.DebugContext(ctx, "skipping selecting request", "serverid", srvID)

			return
		}

		iface.handleSelecting(ctx, req, fd, reqIP)
	case hasReqIP && !reqIP.IsUnspecified():
		// Requested IP address option MUST be filled in with client's notion of
		// its previously assigned address.
		if !iface.subnet.Contains(reqIP) {
			// If the DHCP server detects that the client is on the wrong net
			// then the server SHOULD send a DHCPNAK message to the client.
			l.DebugContext(ctx, "declining init-reboot request", "requestedip", reqIP)
			iface.respondNAK(ctx, req, fd)

			return
		}

		iface.handleInitReboot(ctx, req, fd, reqIP)
	default:
		// Server identifier MUST NOT be filled in, requested IP address option
		// MUST NOT be filled in, 'ciaddr' MUST be filled in with client's
		// notion of its previously assigned address.
		ip, ok := netip.AddrFromSlice(req.ClientIP.To4())
		if !ok || !iface.subnet.Contains(ip) {
			l.DebugContext(ctx, "skipping renew request", "clientip", ip)

			return
		}

		iface.handleRenew(ctx, req, fd, ip)
	}
}

// handleDiscover handles messages of type DHCPDISCOVER.  req must be a
// DHCPDISCOVER message, fd must not be nil.
func (iface *dhcpInterfaceV4) handleDiscover(
	ctx context.Context,
	req *layers.DHCPv4,
	fd *frameData,
) {
	l := iface.common.logger
	mac := req.ClientHWAddr
	mk := macToKey(mac)

	// Check if there's an existing lease for this MAC address.
	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	lease, hasLease := iface.common.leases[mk]
	if hasLease {
		reqIP, hasReqIP := requestedIPv4(req)
		if hasReqIP && reqIP != lease.IP {
			l.DebugContext(ctx, "different requested ip", "requested", reqIP, "lease", lease.IP)
		}

		lease.updateExpiry(iface.clock, iface.common.leaseTTL)
		iface.respondOffer(ctx, req, fd, lease)

		return
	}

	lease, err := iface.allocateLease(ctx, mac)
	if err != nil {
		l.ErrorContext(ctx, "allocating a lease", slogutil.KeyError, err)

		return
	}

	// Send DHCPOFFER with new lease.
	iface.respondOffer(ctx, req, fd, lease)
}

// handleSelecting handles messages of type DHCPREQUEST in SELECTING state.  req
// must be a DHCPREQUEST message, reqIP must be a valid IPv4 address, fd must
// not be nil.
func (iface *dhcpInterfaceV4) handleSelecting(
	ctx context.Context,
	req *layers.DHCPv4,
	fd *frameData,
	reqIP netip.Addr,
) {
	l := iface.common.logger

	ciaddr, ok := netip.AddrFromSlice(req.ClientIP)
	if ok && !ciaddr.IsUnspecified() {
		l.DebugContext(ctx, "non-zero ciaddr in selecting request", "ciaddr", ciaddr)

		return
	}

	mac := req.ClientHWAddr
	mk := macToKey(mac)

	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	lease, hasLease := iface.common.leases[mk]
	if !hasLease {
		l.DebugContext(ctx, "no reserved lease", "clienthwaddr", mac)
		iface.respondNAK(ctx, req, fd)

		return
	}

	if lease.IP != reqIP {
		l.DebugContext(ctx, "selecting request mismatched", "requested", reqIP, "lease", lease.IP)
		iface.respondNAK(ctx, req, fd)

		return
	}

	// Commit the lease and send ACK.
	lease.Hostname = hostname4(req)
	err := iface.updateLease(ctx, lease)
	if err != nil {
		l.ErrorContext(ctx, "selecting request failed", slogutil.KeyError, err)
		iface.respondNAK(ctx, req, fd)

		return
	}

	iface.respondACK(ctx, req, fd, lease)
}

// handleInitReboot handles messages of type DHCPREQUEST in INIT-REBOOT state.
// req must be a DHCPREQUEST message, reqIP must be a valid IPv4 address, fd
// must not be nil.
func (iface *dhcpInterfaceV4) handleInitReboot(
	ctx context.Context,
	req *layers.DHCPv4,
	fd *frameData,
	reqIP netip.Addr,
) {
	l := iface.common.logger

	// ciaddr must be zero.  The client is seeking to verify a previously
	// allocated, cached configuration.
	ciaddr, _ := netip.AddrFromSlice(req.ClientIP)
	if ciaddr.IsValid() && !ciaddr.IsUnspecified() {
		l.DebugContext(ctx, "unexpected ciaddr in init-reboot request", "ciaddr", ciaddr)

		return
	}

	// Check if the lease exists and matches.
	mac := req.ClientHWAddr
	mk := macToKey(mac)

	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	lease, hasLease := iface.common.leases[mk]
	if !hasLease {
		// If the DHCP server has no record of this client, then it MUST remain
		// silent, and MAY output a warning to the network administrator.
		l.WarnContext(ctx, "no existing lease", "mac", mac)

		return
	}

	if lease.IP != reqIP {
		l.WarnContext(ctx, "init-reboot request mismatched", "requested", reqIP, "lease", lease.IP)
		iface.respondNAK(ctx, req, fd)

		return
	}

	iface.updateAndRespond(ctx, l, req, lease, fd)
}

// handleRenew handles messages of type DHCPREQUEST in RENEWING or REBINDING
// state.  req must be a DHCPREQUEST message, ip should be a previously leased
// address, fd must not be nil.
func (iface *dhcpInterfaceV4) handleRenew(
	ctx context.Context,
	req *layers.DHCPv4,
	fd *frameData,
	ip netip.Addr,
) {
	l := iface.common.logger

	// Check if the lease exists and matches.
	mac := req.ClientHWAddr
	mk := macToKey(mac)

	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	lease, hasLease := iface.common.leases[mk]
	if !hasLease {
		// If the DHCP server has no record of this client, then it MUST remain
		// silent, and MAY output a warning to the network administrator.
		l.InfoContext(ctx, "no existing lease", "mac", mac)

		// TODO(e.burkov):  Investigate if we should respond with NAK.
		return
	}

	if lease.IP != ip {
		l.DebugContext(ctx, "renew request mismatched", "ciaddr", ip, "lease", lease.IP)
		iface.respondNAK(ctx, req, fd)

		return
	}

	iface.updateAndRespond(ctx, l, req, lease, fd)
}

// handleDecline handles messages of type DHCPDECLINE.  req must be a
// DHCPDECLINE message.
//
// TODO(e.burkov):  Log the message option, as the request should include one.
//
// TODO(e.burkov):  Consider DRY'ing this with [dhcpInterfaceV4.handleRelease].
func (iface *dhcpInterfaceV4) handleDecline(ctx context.Context, req *layers.DHCPv4) {
	l := iface.common.logger

	reqIP, hasReqIP := requestedIPv4(req)
	if !hasReqIP {
		l.DebugContext(ctx, "skipping decline message without requested ip")

		return
	}

	if !iface.subnet.Contains(reqIP) {
		l.DebugContext(ctx, "skipping decline message", "requestedip", reqIP)

		return
	}

	// Check if the lease exists and matches.
	mac := req.ClientHWAddr
	mk := macToKey(mac)

	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	lease, hasLease := iface.common.leases[mk]
	if !hasLease {
		l.ErrorContext(ctx, "decline message for non-existing lease", "mac", mac)

		return
	}

	if lease.IP != reqIP {
		l.ErrorContext(ctx, "decline mismatch", "ip", reqIP, "lease", lease.IP)

		return
	}

	l.WarnContext(ctx, "lease reported to be unavailable", "ip", lease.IP)

	err := iface.common.blockLease(ctx, lease, iface.clock)
	if err != nil {
		l.ErrorContext(ctx, "blocking lease", slogutil.KeyError, err)
	}
}

// handleRelease handles messages of type DHCPRELEASE.  req must be a
// DHCPRELEASE message.
func (iface *dhcpInterfaceV4) handleRelease(ctx context.Context, req *layers.DHCPv4) {
	l := iface.common.logger

	ip, _ := netip.AddrFromSlice(req.ClientIP.To4())
	if !iface.subnet.Contains(ip) {
		l.DebugContext(ctx, "skipping release message", "clientip", ip)

		return
	}

	// Check if the lease exists and matches.
	mac := req.ClientHWAddr
	mk := macToKey(mac)

	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	lease, hasLease := iface.common.leases[mk]
	if !hasLease {
		l.WarnContext(ctx, "release message for non-existing lease", "mac", mac)

		return
	}

	if lease.IP != ip {
		l.WarnContext(ctx, "release mismatch", "ip", ip, "lease", lease.IP)

		return
	}

	err := iface.common.index.remove(ctx, l, lease, iface.common)
	if err != nil {
		l.ErrorContext(ctx, "removing lease", slogutil.KeyError, err)

		return
	}
}
