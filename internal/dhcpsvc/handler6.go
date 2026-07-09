package dhcpsvc

import (
	"bytes"
	"context"
	"fmt"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// serveEther6 handles the incoming ethernet packets and dispatches them to the
// appropriate handler.  It's used to run in a separate goroutine as it blocks
// until packets channel is closed.  iface and nd must not be nil.  nd must have
// at least a single address returned by its Addresses method.
func (srv *DHCPServer) serveEther6(ctx context.Context, iface *dhcpInterfaceV6, nd NetworkDevice) {
	defer slogutil.RecoverAndLog(ctx, srv.logger)

	src := gopacket.NewPacketSource(nd, nd.LinkType())
	srvDUID := newServerDUID(nd.HardwareAddr())

	for pkt := range src.Packets() {
		fd, err := newFrameData6(pkt, nd, srvDUID)
		if err != nil {
			srv.logger.DebugContext(ctx, "parsing frame data", slogutil.KeyError, err)

			continue
		}

		err = srv.serveV6(ctx, iface, pkt, fd)
		if err != nil {
			srv.logger.ErrorContext(ctx, "serving", slogutil.KeyError, err)
		}
	}
}

// serveV6 handles the ethernet packet of IPv6 type. iface and pkt must not be
// nil.  iface and fd must be valid.  pkt must be an IPv6 packet.
//
//lint:ignore U1000 TODO(e.burkov): Use.
func (srv *DHCPServer) serveV6(
	ctx context.Context,
	iface *dhcpInterfaceV6,
	pkt gopacket.Packet,
	fd *frameData6,
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

	return iface.handleDHCPv6(ctx, msg.MsgType, fd, msg)
}

// handleDHCPv6 handles the DHCPv6 message of the given type.
func (iface *dhcpInterfaceV6) handleDHCPv6(
	ctx context.Context,
	typ layers.DHCPv6MsgType,
	fd *frameData6,
	req *layers.DHCPv6,
) (err error) {
	switch typ {
	case layers.DHCPv6MsgTypeSolicit:
		return iface.handleSolicit(ctx, fd, req)
	case layers.DHCPv6MsgTypeRequest:
		return iface.handleRequest(ctx, fd, req)
	case layers.DHCPv6MsgTypeConfirm:
		return iface.handleConfirm(ctx, fd, req)
	case layers.DHCPv6MsgTypeRenew:
		return iface.handleRenew(ctx, fd, req)
	case layers.DHCPv6MsgTypeRebind:
		return iface.handleRebind(ctx, fd, req)
	case layers.DHCPv6MsgTypeInformationRequest:
		return iface.handleInfo(ctx, fd, req)
	case layers.DHCPv6MsgTypeRelease:
		return iface.handleRelease(ctx, fd, req)
	case layers.DHCPv6MsgTypeDecline:
		return iface.handleDecline(ctx, fd, req)
	default:
		return fmt.Errorf("dhcpv6: request type: %w: %d", errors.ErrBadEnumValue, typ)
	}
}

// handleSolicit handles messages of type SOLICIT.  req must not be nil and must
// be a valid DHCPv6 message of type SOLICIT.  fd must be valid.
func (iface *dhcpInterfaceV6) handleSolicit(
	ctx context.Context,
	fd *frameData6,
	req *layers.DHCPv6,
) (err error) {
	cliID, err := clientIDNoServer(req.Options)
	if err != nil {
		return fmt.Errorf("dhcpv6: %s: %w", req.MsgType, err)
	}

	l := iface.common.logger
	l.DebugContext(ctx, "handling message", "type", req.MsgType, "cli_id", cliID)

	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	lease, iaid := iface.allocateForSolicit(ctx, fd.ether.SrcMAC, req)

	resp := &layers.DHCPv6{
		MsgType:       layers.DHCPv6MsgTypeAdvertise,
		TransactionID: req.TransactionID,
	}

	if lease == nil {
		resp.Options = iface.newSolicitRespOpts(fd, req, cliID, iaid, nil, false)

		return respond6(fd, resp)
	}

	_, isRapidCommit := findOption6(req.Options, layers.DHCPv6OptRapidCommit)

	if !isRapidCommit {
		resp.Options = iface.newSolicitRespOpts(fd, req, cliID, iaid, lease, false)

		return respond6(fd, resp)
	}

	err = iface.commit(ctx, req, lease)
	if err != nil {
		l.WarnContext(ctx, "committing rapid leases", slogutil.KeyError, err)
		isRapidCommit = false
	} else {
		// The server will also send a Reply in response to a Solicit with a
		// Rapid Commit option.
		//
		// See RFC 9915 Section 18.3.
		resp.MsgType = layers.DHCPv6MsgTypeReply
	}

	resp.Options = iface.newSolicitRespOpts(fd, req, cliID, iaid, lease, isRapidCommit)

	return respond6(fd, resp)
}

// handleRequest handles messages of type REQUEST.  req must not be nil and must
// be a valid DHCPv6 message of type REQUEST.  fd must be valid.
func (iface *dhcpInterfaceV6) handleRequest(
	ctx context.Context,
	fd *frameData6,
	req *layers.DHCPv6,
) (err error) {
	cliID, err := clientIDMatchingServer(req.Options, fd.duidData)
	if err != nil {
		return fmt.Errorf("dhcpv6: %s: %w", req.MsgType, err)
	}

	l := iface.common.logger
	l.DebugContext(ctx, "handling message", "type", req.MsgType, "cli_id", cliID)

	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	resp := &layers.DHCPv6{
		MsgType:       layers.DHCPv6MsgTypeReply,
		TransactionID: req.TransactionID,
	}

	iana, ok := iface.firstIANA(ctx, req)
	if !ok {
		// In practice, the DHCPv6 REQUEST with no IA_NA options is not useful,
		// since the standard explicitly defines the INFORMATION-REQUEST message
		// type for that purpose.  However, there are no requirements which
		// invalidate such messages, so the server must respond.
		resp.Options = iface.newRequestRespOpts(fd, req, cliID, layers.DHCPv6Option{})

		return respond6(fd, resp)
	}

	reqIP, hasReqIP := iana.requestedAddr()
	if hasReqIP && !iface.subnetPrefix.Contains(reqIP) {
		respIANA := newIANAWithStatus(iana.ID, layers.DHCPv6StatusCodeNotOnLink)
		resp.Options = iface.newRequestRespOpts(fd, req, cliID, respIANA)

		return respond6(fd, resp)
	}

	var ianaOpt layers.DHCPv6Option

	lease, err := iface.leaseForRequest(ctx, req, fd.ether.SrcMAC)
	if err != nil {
		ianaOpt = newIANAWithStatus(iana.ID, layers.DHCPv6StatusCodeNoAddrsAvail)
	} else {
		ianaOpt = iface.iaNAFromLease(lease, iana.ID)
	}

	resp.Options = iface.newRequestRespOpts(fd, req, cliID, ianaOpt)

	return respond6(fd, resp)
}

// handleConfirm handles messages of type CONFIRM.  req must not be nil and must
// be a valid DHCPv6 message of type CONFIRM.  fd must be valid.
//
// TODO(e.burkov):  Implement.  This is a stub for now.
func (iface *dhcpInterfaceV6) handleConfirm(
	ctx context.Context,
	_ *frameData6,
	req *layers.DHCPv6,
) (err error) {
	cliID, err := clientIDNoServer(req.Options)
	if err != nil {
		return fmt.Errorf("dhcpv6: %s: %w", req.MsgType, err)
	}

	l := iface.common.logger
	l.DebugContext(ctx, "handling message", "type", req.MsgType, "cli_id", cliID)

	return nil
}

// handleRenew handles messages of type RENEW.  req must not be nil and must be
// a valid DHCPv6 message of type RENEW.  fd must be valid.
//
// TODO(e.burkov):  Implement.  This is a stub for now.
func (iface *dhcpInterfaceV6) handleRenew(
	ctx context.Context,
	fd *frameData6,
	req *layers.DHCPv6,
) (err error) {
	cliID, err := clientIDMatchingServer(req.Options, fd.duidData)
	if err != nil {
		return fmt.Errorf("dhcpv6: %s: %w", req.MsgType, err)
	}

	l := iface.common.logger
	l.DebugContext(ctx, "handling message", "type", req.MsgType, "cli_id", cliID)

	return nil
}

// handleRebind handles messages of type REBIND.  req must not be nil and must
// be a valid DHCPv6 message of type REBIND.  fd must be valid.
//
// TODO(e.burkov):  Implement.  This is a stub for now.
func (iface *dhcpInterfaceV6) handleRebind(
	ctx context.Context,
	_ *frameData6,
	req *layers.DHCPv6,
) (err error) {
	cliID, err := clientIDNoServer(req.Options)
	if err != nil {
		return fmt.Errorf("dhcpv6: %s: %w", req.MsgType, err)
	}

	l := iface.common.logger
	l.DebugContext(ctx, "handling message", "type", req.MsgType, "cli_id", cliID)

	return nil
}

// handleInfo handles messages of type INFORMATION-REQUEST.  req must not be nil
// and must be a valid DHCPv6 message of type INFORMATION-REQUEST.  fd must be
// valid.
//
// TODO(e.burkov):  Implement.  This is a stub for now.
func (iface *dhcpInterfaceV6) handleInfo(
	ctx context.Context,
	fd *frameData6,
	req *layers.DHCPv6,
) (err error) {
	if srvID, ok := findOption6(req.Options, layers.DHCPv6OptServerID); ok {
		if !bytes.Equal(srvID, fd.duidData) {
			return fmt.Errorf(
				"dhcpv6: server id: got %v, want %v: %w",
				srvID,
				fd.duidData,
				errors.ErrNotEqual,
			)
		}
	}

	_, ok := findOption6(req.Options, layers.DHCPv6OptIANA)
	if ok {
		return fmt.Errorf("dhcpv6: %s: ia option: %w", req.MsgType, errors.ErrUnexpectedValue)
	}

	_, ok = findOption6(req.Options, layers.DHCPv6OptIATA)
	if ok {
		return fmt.Errorf("dhcpv6: %s: ia option: %w", req.MsgType, errors.ErrUnexpectedValue)
	}

	l := iface.common.logger
	l.DebugContext(ctx, "handling message", "type", req.MsgType)

	return nil
}

// handleRelease handles messages of type RELEASE.  req must not be nil and must
// be a valid DHCPv6 message of type RELEASE.  fd must be valid.
//
// TODO(e.burkov):  Implement.  This is a stub for now.
func (iface *dhcpInterfaceV6) handleRelease(
	ctx context.Context,
	fd *frameData6,
	req *layers.DHCPv6,
) (err error) {
	cliID, err := clientIDMatchingServer(req.Options, fd.duidData)
	if err != nil {
		return fmt.Errorf("dhcpv6: %s: %w", req.MsgType, err)
	}

	l := iface.common.logger
	l.DebugContext(ctx, "handling message", "type", req.MsgType, "cli_id", cliID)

	return nil
}

// handleDecline handles messages of type DECLINE.  req must not be nil and must
// be a valid DHCPv6 message of type DECLINE.  fd must be valid.
//
// TODO(e.burkov):  Implement.  This is a stub for now.
func (iface *dhcpInterfaceV6) handleDecline(
	ctx context.Context,
	fd *frameData6,
	req *layers.DHCPv6,
) (err error) {
	cliID, err := clientIDMatchingServer(req.Options, fd.duidData)
	if err != nil {
		return fmt.Errorf("dhcpv6: %s: %w", req.MsgType, err)
	}

	l := iface.common.logger
	l.DebugContext(ctx, "handling message", "type", req.MsgType, "cli_id", cliID)

	return nil
}
